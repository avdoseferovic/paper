// Package svg provides pure-Go SVG rasterisation helpers for PDF embedding.
package svg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"io"
	"strings"

	"github.com/avdoseferovic/paper/internal/htmllimits"
	"golang.org/x/image/vector"

	"github.com/avdoseferovic/paper/internal/pngcodec"
)

// DPIForRaster is the resolution SVGs are rasterised at, and MaxRasterPixels the
// largest raster the renderer will allocate.
const (
	DPIForRaster    = 150.0
	MaxRasterPixels = 50_000_000
	svgFill         = "fill"

	// rasterPixelBudget caps the total pixels rasterised across every shape of
	// one SVG. Each shape only costs the area of its own bounding box, so real
	// artwork stays far below the budget while a document that repeats
	// full-canvas shapes thousands of times is refused instead of pinning a CPU.
	// 400M pixels is roughly a second of rasterisation, versus the unbounded
	// minutes a few kilobytes of hostile SVG used to buy.
	rasterPixelBudget = 400_000_000

	// minCanvasBudgetFactor lets any SVG paint at least this many full-canvas
	// shapes, however large its canvas, so the budget never rejects a document
	// that the pixel limit already accepted.
	minCanvasBudgetFactor = 8
)

// Errors reported when an SVG cannot be rasterised.
var (
	ErrSVGHasZeroDimensions = errors.New("svg has zero dimensions")
	ErrSVGTooLarge          = htmllimits.ErrSVGTooLarge

	// ErrSVGTooComplex reports an SVG whose shapes need more rasterisation work
	// than the pixel budget allows. It wraps ErrSVGTooLarge so callers that gate
	// on the SVG limit keep treating it as a limit failure.
	ErrSVGTooComplex = fmt.Errorf("%w: rasterization budget exhausted", htmllimits.ErrSVGTooLarge)

	errMissingSVGRoot = errors.New("svg parse: missing <svg> root")
)

type svgViewBox struct {
	x, y float64
	w, h float64
}

// Rasterize converts SVG bytes into PNG bytes at the requested mm dimensions.
func Rasterize(svgBytes []byte, widthMM, heightMM float64) ([]byte, int, int, error) {
	return RasterizeWithLimit(svgBytes, widthMM, heightMM, MaxRasterPixels)
}

// RasterizeWithLimit converts SVG bytes into PNG bytes with a pixel cap. A
// maxPixels of zero or less disables both the canvas cap and the rasterisation
// work budget, and is only safe for trusted input.
func RasterizeWithLimit(svgBytes []byte, widthMM, heightMM float64, maxPixels int64) ([]byte, int, int, error) {
	viewBox, err := readSVGViewBox(svgBytes)
	if err != nil {
		return nil, 0, 0, err
	}
	pxW, pxH := targetPixels(viewBox, widthMM, heightMM)
	if pxW <= 0 || pxH <= 0 {
		return nil, 0, 0, ErrSVGHasZeroDimensions
	}
	if maxPixels > 0 && int64(pxW) > maxPixels/int64(pxH) {
		return nil, 0, 0, fmt.Errorf("%w: %dx%d exceeds limit %d", ErrSVGTooLarge, pxW, pxH, maxPixels)
	}

	dst := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	renderer := newSVGRenderer(dst, viewBox, svgBytes, maxPixels)
	err = drawSVG(renderer, svgBytes)
	if err != nil {
		return nil, 0, 0, err
	}
	drawText(renderer, svgBytes)
	if renderer.exhausted {
		return nil, 0, 0, fmt.Errorf("%w after %d pixels", ErrSVGTooComplex, renderer.spent)
	}

	var encoded bytes.Buffer
	err = pngcodec.EncodeUncompressed(&encoded, dst)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("png encode: %w", err)
	}
	return encoded.Bytes(), pxW, pxH, nil
}

func pxFromMM(mm float64) int {
	px := int(mm / 25.4 * DPIForRaster)
	if px < 1 {
		return 1
	}
	return px
}

func targetPixels(viewBox svgViewBox, widthMM, heightMM float64) (int, int) {
	switch {
	case widthMM > 0 && heightMM > 0:
		return pxFromMM(widthMM), pxFromMM(heightMM)
	case widthMM > 0:
		width := pxFromMM(widthMM)
		if viewBox.w > 0 && viewBox.h > 0 {
			return width, max(1, int(float64(width)*viewBox.h/viewBox.w))
		}
		return width, width
	case heightMM > 0:
		height := pxFromMM(heightMM)
		if viewBox.w > 0 && viewBox.h > 0 {
			return max(1, int(float64(height)*viewBox.w/viewBox.h)), height
		}
		return height, height
	default:
		if viewBox.w > 0 && viewBox.h > 0 {
			return max(1, int(viewBox.w)), max(1, int(viewBox.h))
		}
		return 32, 32
	}
}

func readSVGViewBox(data []byte) (svgViewBox, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return svgViewBox{}, errMissingSVGRoot
		}
		if err != nil {
			return svgViewBox{}, fmt.Errorf("svg parse: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || !strings.EqualFold(start.Name.Local, "svg") {
			continue
		}
		viewBox := svgViewBox{w: parseNumber(attrValue(start.Attr, "width")), h: parseNumber(attrValue(start.Attr, "height"))}
		if parts := parseNumberList(attrValue(start.Attr, "viewBox")); len(parts) == 4 {
			viewBox = svgViewBox{x: parts[0], y: parts[1], w: parts[2], h: parts[3]}
		}
		if viewBox.w < 0 || viewBox.h < 0 {
			return svgViewBox{}, ErrSVGHasZeroDimensions
		}
		return viewBox, nil
	}
}

// svgRenderer holds the shared rasterisation state for one SVG. The rasterizer
// is reused across shapes: allocating one per shape, sized to the whole canvas,
// made rendering cost scale with (shape count x canvas area).
type svgRenderer struct {
	dst               *image.RGBA
	viewBox           svgViewBox
	scaleX, scaleY    float64
	classDeclarations map[string]map[string]string

	rasterizer vector.Rasterizer
	warmed     bool
	budget     int64
	unlimited  bool
	spent      int64
	exhausted  bool
}

func newSVGRenderer(dst *image.RGBA, viewBox svgViewBox, data []byte, maxPixels int64) *svgRenderer {
	scaleX, scaleY := svgScale(viewBox, dst.Bounds().Dx(), dst.Bounds().Dy())
	renderer := &svgRenderer{
		dst:               dst,
		viewBox:           viewBox,
		scaleX:            scaleX,
		scaleY:            scaleY,
		classDeclarations: svgClassDeclarations(data),
		unlimited:         maxPixels <= 0,
	}
	canvas := int64(dst.Bounds().Dx()) * int64(dst.Bounds().Dy())
	renderer.budget = max(rasterPixelBudget, minCanvasBudgetFactor*canvas)
	return renderer
}

// spend charges area against the work budget, reporting false once the budget
// is exhausted so callers stop rasterising.
func (renderer *svgRenderer) spend(area int64) bool {
	if renderer.unlimited {
		return true
	}
	if renderer.exhausted || area > renderer.budget {
		renderer.exhausted = true
		return false
	}
	renderer.budget -= area
	renderer.spent += area
	return true
}

// maskFor charges rect against the budget and returns the shared rasterizer
// reset to rect's size. Mask coordinates are relative to rect.Min.
func (renderer *svgRenderer) maskFor(rect image.Rectangle) (*vector.Rasterizer, bool) {
	if rect.Empty() {
		return nil, false
	}
	if !renderer.spend(int64(rect.Dx()) * int64(rect.Dy())) {
		return nil, false
	}
	if !renderer.warmed {
		// Size the mask buffer for the canvas once. Reset only reallocates when
		// it has to grow, so every later shape reuses this allocation instead of
		// churning a fresh buffer whenever a bigger shape comes along.
		renderer.rasterizer.Reset(renderer.dst.Bounds().Dx(), renderer.dst.Bounds().Dy())
		renderer.warmed = true
	}
	renderer.rasterizer.Reset(rect.Dx(), rect.Dy())
	return &renderer.rasterizer, true
}

func drawSVG(renderer *svgRenderer, data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	states := []svgRenderState{{transform: identity(), paint: defaultPaint()}}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("svg parse: %w", err)
		}
		switch current := token.(type) {
		case xml.StartElement:
			state := applySVGState(states[len(states)-1], current.Attr, renderer.classDeclarations)
			states = append(states, state)
			renderer.drawElement(strings.ToLower(current.Name.Local), current.Attr, state)
			if renderer.exhausted {
				return nil
			}
		case xml.EndElement:
			if len(states) > 1 {
				states = states[:len(states)-1]
			}
		}
	}
	return nil
}

func (renderer *svgRenderer) drawElement(name string, attrs []xml.Attr, state svgRenderState) {
	if !state.paint.display {
		return
	}
	path := newSVGPath(renderer, state.transform)
	switch name {
	case "rect":
		x, y := parseNumber(attrValue(attrs, "x")), parseNumber(attrValue(attrs, "y"))
		width, height := parseNumber(attrValue(attrs, "width")), parseNumber(attrValue(attrs, "height"))
		if width <= 0 || height <= 0 {
			return
		}
		path.moveTo(x, y)
		path.lineTo(x+width, y)
		path.lineTo(x+width, y+height)
		path.lineTo(x, y+height)
		path.close()
	case "circle":
		cx, cy := parseNumber(attrValue(attrs, "cx")), parseNumber(attrValue(attrs, "cy"))
		radius := parseNumber(attrValue(attrs, "r"))
		path.ellipse(cx, cy, radius, radius)
	case "ellipse":
		cx, cy := parseNumber(attrValue(attrs, "cx")), parseNumber(attrValue(attrs, "cy"))
		path.ellipse(cx, cy, parseNumber(attrValue(attrs, "rx")), parseNumber(attrValue(attrs, "ry")))
	case "line":
		path.moveTo(parseNumber(attrValue(attrs, "x1")), parseNumber(attrValue(attrs, "y1")))
		path.lineTo(parseNumber(attrValue(attrs, "x2")), parseNumber(attrValue(attrs, "y2")))
	case "polyline", "polygon":
		points := parseNumberList(attrValue(attrs, "points"))
		if len(points) < 2 {
			return
		}
		path.moveTo(points[0], points[1])
		for index := 2; index+1 < len(points); index += 2 {
			path.lineTo(points[index], points[index+1])
		}
		if name == "polygon" {
			path.close()
		}
	case "path":
		if !path.parse(attrValue(attrs, "d")) {
			return
		}
	default:
		return
	}
	path.render(state.paint)
}

func svgScale(viewBox svgViewBox, widthPx, heightPx int) (float64, float64) {
	if viewBox.w <= 0 {
		viewBox.w = float64(widthPx)
	}
	if viewBox.h <= 0 {
		viewBox.h = float64(heightPx)
	}
	return float64(widthPx) / viewBox.w, float64(heightPx) / viewBox.h
}
