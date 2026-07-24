package svg

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"runtime"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
)

// fullCanvasShapes builds an SVG whose every shape covers the whole canvas. It
// is the shape of input that used to allocate a canvas-sized mask per shape.
func fullCanvasShapes(count int) []byte {
	var builder strings.Builder
	builder.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000">`)
	for i := range count {
		fmt.Fprintf(&builder, `<rect x="0" y="0" width="1000" height="1000" fill="#%02x0000"/>`, i%256)
	}
	builder.WriteString(`</svg>`)
	return []byte(builder.String())
}

func gridLinesSVG(count int) []byte {
	var builder strings.Builder
	builder.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000">`)
	for i := range count {
		fmt.Fprintf(&builder, `<line x1="%d" y1="0" x2="%d" y2="1000" stroke="#333333"/>`, i%1000, i%1000)
	}
	builder.WriteString(`</svg>`)
	return []byte(builder.String())
}

func TestRasterize_RefusesShapeFlood(t *testing.T) {
	t.Parallel()

	_, _, _, err := Rasterize(fullCanvasShapes(1000), 0, 0)

	require.ErrorIs(t, err, ErrSVGTooComplex)
	// Callers gating on the SVG limit must keep seeing a limit failure.
	require.ErrorIs(t, err, ErrSVGTooLarge)
}

func TestRasterize_AllowsDenseArtwork(t *testing.T) {
	t.Parallel()

	pngBytes, _, _, err := Rasterize(gridLinesSVG(500), 0, 0)

	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	assert.Greater(t, countOpaquePixels(img), 1000)
}

func TestNewSVGRenderer_ZeroLimitDisablesBudget(t *testing.T) {
	t.Parallel()

	// A zero pixel limit is the documented trusted-input escape hatch, so the
	// work budget has to step aside with it.
	renderer := newSVGRenderer(image.NewRGBA(image.Rect(0, 0, 8, 8)), svgViewBox{w: 8, h: 8}, nil, 0)

	assert.True(t, renderer.spend(1<<62))
	assert.False(t, renderer.exhausted)
}

func TestSVGRenderer_SpendExhaustsBudgetOnce(t *testing.T) {
	t.Parallel()

	renderer := newSVGRenderer(image.NewRGBA(image.Rect(0, 0, 8, 8)), svgViewBox{w: 8, h: 8}, nil, MaxRasterPixels)
	renderer.budget = 100

	assert.True(t, renderer.spend(60))
	assert.False(t, renderer.spend(60))
	assert.True(t, renderer.exhausted)
	// Once exhausted the renderer stays exhausted, even for work it could afford.
	assert.False(t, renderer.spend(1))
}

func TestDrawSVG_StopsAtExhaustedBudget(t *testing.T) {
	t.Parallel()

	dst := image.NewRGBA(image.Rect(0, 0, 64, 64))
	renderer := newSVGRenderer(dst, svgViewBox{w: 64, h: 64}, nil, MaxRasterPixels)
	renderer.budget = 100

	require.NoError(t, drawSVG(renderer, fullCanvasShapes(50)))

	assert.True(t, renderer.exhausted)
	assert.Equal(t, 0, countOpaquePixels(dst))
}

func TestRasterize_ClampsOversizedFontSize(t *testing.T) {
	t.Parallel()

	svgBytes := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">` +
		`<text x="1" y="5" font-size="2000000" fill="#000000">A</text></svg>`)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, _, _, err := Rasterize(svgBytes, 20, 20)
	runtime.ReadMemStats(&after)

	require.NoError(t, err)
	// An unclamped face rasterises a mask of its own em size: this input used to
	// allocate ~2.7GB for a single glyph on a 118x118 canvas.
	allocatedMB := (after.TotalAlloc - before.TotalAlloc) / (1 << 20)
	assert.Less(t, int(allocatedMB), 64)
}
