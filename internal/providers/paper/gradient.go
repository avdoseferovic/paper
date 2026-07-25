package paper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"

	gofpdflib "github.com/avdoseferovic/paper/internal/pdf"

	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"

	"github.com/avdoseferovic/paper/internal/pngcodec"
)

const gradientDPI = 75.0

// GradientRenderer handles gradient rasterisation and caching.
type GradientRenderer struct {
	pdf     gradientPDF
	mu      sync.Mutex
	nameMap map[string]string // cacheKey → registered imgName
}

// NewGradientRenderer creates a GradientRenderer that uses pdf for drawing.
func NewGradientRenderer(pdf gradientPDF) *GradientRenderer {
	return &GradientRenderer{pdf: pdf, nameMap: map[string]string{}}
}

// DrawGradient rasterises g as a PNG and embeds it at the cell position.
// The cell coordinates are margin-relative; we add page margins before calling
// fpdf.Image so the image lands at the correct absolute page position.
func (gr *GradientRenderer) DrawGradient(cell *entity.Cell, g *props.Gradient, widthMM, heightMM float64) {
	if g == nil || len(g.Stops) < 2 || widthMM <= 0 || heightMM <= 0 {
		return
	}

	left, top, _, _ := gr.pdf.GetMargins()
	x := cell.X + left
	y := cell.Y + top
	pxW, pxH := gradientRasterDimensions(g, widthMM, heightMM)

	key := gradientCacheKey(g, pxW, pxH)

	gr.mu.Lock()
	imgName, cached := gr.nameMap[key]
	gr.mu.Unlock()

	if !cached {
		img := rasteriseGradient(g, pxW, pxH)
		var buf bytes.Buffer
		// This PNG is a throwaway transport buffer: RegisterImageOptionsReader
		// immediately decodes it back to raw pixel planes, which the PDF writer
		// re-compresses with FlateDecode. Compressing here is pure waste, so we
		// skip it — NoCompression avoids the LZ77 pass and the ~128KB deflate
		// table allocation. The embedded PDF stream is byte-identical.
		err := pngcodec.EncodeUncompressed(&buf, img)
		if err != nil {
			return
		}
		imgName = "gradient-" + key[:16]
		r := bytes.NewReader(buf.Bytes())
		gr.pdf.RegisterImageOptionsReader(imgName, gofpdflib.ImageOptions{ImageType: "PNG"}, r)

		gr.mu.Lock()
		gr.nameMap[key] = imgName
		gr.mu.Unlock()
	}

	gr.pdf.Image(imgName, x, y, widthMM, heightMM, false, "PNG", 0, "")
}

func gradientRasterDimensions(g *props.Gradient, widthMM, heightMM float64) (int, int) {
	pxW := int(math.Round(widthMM * gradientDPI / 25.4))
	pxH := int(math.Round(heightMM * gradientDPI / 25.4))
	if pxW < 1 {
		pxW = 1
	}
	if pxH < 1 {
		pxH = 1
	}
	if isHorizontalLinearGradient(g) {
		pxH = 1
	}
	return pxW, pxH
}

func isHorizontalLinearGradient(g *props.Gradient) bool {
	if g == nil || g.Kind != props.GradientLinear {
		return false
	}
	deg := math.Mod(g.AngleDeg, 360)
	if deg < 0 {
		deg += 360
	}
	const epsilon = 0.000001
	return math.Abs(deg-90) <= epsilon || math.Abs(deg-270) <= epsilon
}

// gradientCacheKey returns a stable hex string for the given gradient + dimensions.
func gradientCacheKey(g *props.Gradient, pxW, pxH int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d,%.1f,%.4f", g.Kind, g.AngleDeg, g.CX)
	fmt.Fprintf(&sb, ",%.4f,%v", g.CY, g.Circle)
	fmt.Fprintf(&sb, ",%dx%d@%d", pxW, pxH, int(gradientDPI))
	for _, s := range g.Stops {
		fmt.Fprintf(&sb, ",(%d,%d,%d,%.2f@%.4f)",
			s.Color.Red, s.Color.Green, s.Color.Blue, 0.0, s.Position)
	}
	h := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(h[:])
}

// rasteriseGradient draws the gradient into a new RGBA image.
func rasteriseGradient(g *props.Gradient, w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	switch g.Kind {
	case props.GradientLinear:
		rasteriseLinear(img, g, w, h)
	case props.GradientRadial:
		rasteriseRadial(img, g, w, h)
	case props.GradientConic:
		rasteriseConic(img, g, w, h)
	}
	return img
}

func rasteriseLinear(img *image.RGBA, g *props.Gradient, w, h int) {
	// Convert CSS angle to unit direction vector.
	// CSS 0deg = to top, 90deg = to right.
	rad := g.AngleDeg * math.Pi / 180.0
	dx := math.Sin(rad)
	dy := -math.Cos(rad)
	minProj, maxProj := linearGradientProjectionBounds(dx, dy, w, h)
	span := maxProj - minProj
	if span == 0 {
		span = 1
	}

	for py := range h {
		for px := range w {
			// Project the pixel centre onto the gradient direction, then
			// normalise across the projected rectangle corners. This makes
			// axis-aligned CSS gradients use the full stop range across the box.
			proj := (float64(px)+0.5)*dx + (float64(py)+0.5)*dy
			t := clamp01((proj - minProj) / span)
			c := interpolateStops(g.Stops, t)
			img.SetRGBA(px, py, c)
		}
	}
}

func linearGradientProjectionBounds(dx, dy float64, w, h int) (float64, float64) {
	fw := float64(w)
	fh := float64(h)
	projections := [...]float64{
		0,
		fw * dx,
		fh * dy,
		fw*dx + fh*dy,
	}
	minProj, maxProj := projections[0], projections[0]
	for _, p := range projections[1:] {
		if p < minProj {
			minProj = p
		}
		if p > maxProj {
			maxProj = p
		}
	}
	return minProj, maxProj
}

func rasteriseRadial(img *image.RGBA, g *props.Gradient, w, h int) {
	for py := range h {
		for px := range w {
			nx := (float64(px) + 0.5) / float64(w)
			ny := (float64(py) + 0.5) / float64(h)
			dx := nx - g.CX
			dy := ny - g.CY
			// Normalise by the half-size so the gradient reaches the edge.
			maxR := math.Sqrt(g.CX*g.CX + g.CY*g.CY)
			if maxR == 0 {
				maxR = 0.5
			}
			r := math.Sqrt(dx*dx+dy*dy) / maxR
			t := clamp01(r)
			c := interpolateStops(g.Stops, t)
			img.SetRGBA(px, py, c)
		}
	}
}

func rasteriseConic(img *image.RGBA, g *props.Gradient, w, h int) {
	for py := range h {
		for px := range w {
			nx := (float64(px) + 0.5) / float64(w)
			ny := (float64(py) + 0.5) / float64(h)
			dx := nx - g.CX
			dy := ny - g.CY
			angleDeg := math.Atan2(dx, -dy) * 180 / math.Pi
			if angleDeg < 0 {
				angleDeg += 360
			}
			t := math.Mod(angleDeg-g.AngleDeg, 360)
			if t < 0 {
				t += 360
			}
			c := interpolateStops(g.Stops, t/360)
			img.SetRGBA(px, py, c)
		}
	}
}

func interpolateStops(stops []props.GradientStop, t float64) color.RGBA {
	if len(stops) == 0 {
		return color.RGBA{A: 255}
	}
	if t <= stops[0].Position {
		s := stops[0].Color
		return color.RGBA{R: toColorByte(s.Red), G: toColorByte(s.Green), B: toColorByte(s.Blue), A: 255}
	}
	last := stops[len(stops)-1]
	if t >= last.Position {
		return color.RGBA{R: toColorByte(last.Color.Red), G: toColorByte(last.Color.Green), B: toColorByte(last.Color.Blue), A: 255}
	}
	for i := 1; i < len(stops); i++ {
		if t > stops[i].Position {
			continue
		}
		a, b := stops[i-1], stops[i]
		span := b.Position - a.Position
		if span <= 0 {
			span = 1
		}
		frac := (t - a.Position) / span
		return color.RGBA{
			R: toColorByte(lerp(a.Color.Red, b.Color.Red, frac)),
			G: toColorByte(lerp(a.Color.Green, b.Color.Green, frac)),
			B: toColorByte(lerp(a.Color.Blue, b.Color.Blue, frac)),
			A: 255,
		}
	}
	s := last.Color
	return color.RGBA{R: toColorByte(s.Red), G: toColorByte(s.Green), B: toColorByte(s.Blue), A: 255}
}

func lerp(a, b int, t float64) int {
	return int(math.Round(float64(a) + float64(b-a)*t))
}

// toColorByte clamps v to [0,255] and returns it as a byte. The clamp avoids
// a gosec G115 integer-overflow warning for int→uint8 conversions.
func toColorByte(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
