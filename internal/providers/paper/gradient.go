package paper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"sync"

	gofpdflib "github.com/avdoseferovic/paper/internal/paperpdf"

	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

const gradientDPI = 75.0

// GradientRenderer handles gradient rasterisation and caching.
type GradientRenderer struct {
	pdf     gofpdfwrapper.Fpdf
	mu      sync.Mutex
	nameMap map[string]string // cacheKey → registered imgName
}

// NewGradientRenderer creates a GradientRenderer that uses pdf for drawing.
func NewGradientRenderer(pdf gofpdfwrapper.Fpdf) *GradientRenderer {
	return &GradientRenderer{pdf: pdf, nameMap: map[string]string{}}
}

// DrawGradient rasterises g as a PNG and embeds it at the cell position.
// The cell coordinates are margin-relative; we add page margins before calling
// fpdf.Image so the image lands at the correct absolute page position.
func (gr *GradientRenderer) DrawGradient(cell *entity.Cell, g *props.Gradient, widthMM, heightMM float64) {
	if g == nil || len(g.Stops) < 2 || widthMM <= 0 || heightMM <= 0 {
		return
	}

	key := gradientCacheKey(g, widthMM, heightMM)

	gr.mu.Lock()
	imgName, cached := gr.nameMap[key]
	gr.mu.Unlock()

	if !cached {
		pxW := int(math.Round(widthMM * gradientDPI / 25.4))
		pxH := int(math.Round(heightMM * gradientDPI / 25.4))
		if pxW < 1 {
			pxW = 1
		}
		if pxH < 1 {
			pxH = 1
		}
		img := rasteriseGradient(g, pxW, pxH)
		var buf bytes.Buffer
		err := png.Encode(&buf, img)
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

	left, top, _, _ := gr.pdf.GetMargins()
	gr.pdf.Image(imgName, cell.X+left, cell.Y+top, widthMM, heightMM, false, "PNG", 0, "")
}

// gradientCacheKey returns a stable hex string for the given gradient + dimensions.
func gradientCacheKey(g *props.Gradient, widthMM, heightMM float64) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d,%.1f,%.4f", g.Kind, g.AngleDeg, g.CX)
	fmt.Fprintf(&sb, ",%.4f,%v", g.CY, g.Circle)
	fmt.Fprintf(&sb, ",%.1fx%.1f@%d", widthMM, heightMM, int(gradientDPI))
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
	}
	return img
}

func rasteriseLinear(img *image.RGBA, g *props.Gradient, w, h int) {
	// Convert CSS angle to unit direction vector.
	// CSS 0deg = to top, 90deg = to right.
	rad := g.AngleDeg * math.Pi / 180.0
	dx := math.Sin(rad)
	dy := -math.Cos(rad)

	for py := range h {
		for px := range w {
			// Normalised [0,1] coordinates of the pixel centre.
			nx := (float64(px) + 0.5) / float64(w)
			ny := (float64(py) + 0.5) / float64(h)
			// Project onto gradient direction.
			t := nx*dx + ny*dy
			// Clamp t to [0,1] range from stop0.pos to stopN.pos.
			t = clamp01((t + 1) / 2) // shift from [-1,1] to [0,1]
			c := interpolateStops(g.Stops, t)
			img.SetRGBA(px, py, c)
		}
	}
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
		if t <= stops[i].Position {
			a, b := stops[i-1], stops[i]
			span := b.Position - a.Position
			if span <= 0 {
				span = 1
			}
			frac := (t - a.Position) / span
			r := lerp(a.Color.Red, b.Color.Red, frac)
			g2 := lerp(a.Color.Green, b.Color.Green, frac)
			bl := lerp(a.Color.Blue, b.Color.Blue, frac)
			return color.RGBA{R: toColorByte(r), G: toColorByte(g2), B: toColorByte(bl), A: 255}
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
