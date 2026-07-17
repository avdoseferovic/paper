package svg

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
)

func TestRasterize_ProducesPNGAtRequestedSize(t *testing.T) {
	t.Parallel()
	pngBytes, pxW, pxH, err := Rasterize([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
  <rect x="0" y="0" width="32" height="32" fill="#ff0000"/>
</svg>`), 10, 10)
	require.NoError(t, err)
	assert.InDelta(t, 59, pxW, 2)
	assert.InDelta(t, 59, pxH, 2)

	img, err := png.Decode(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	assert.Equal(t, pxW, img.Bounds().Dx())
	assert.Equal(t, pxH, img.Bounds().Dy())
}

func TestRasterize_DrawsBasicText(t *testing.T) {
	t.Parallel()
	pngBytes, _, _, err := Rasterize([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 60">
  <style>
    .label { font-size: 24px; font-weight: 700; fill: #12355b; }
  </style>
  <g transform="translate(12 36)">
    <text x="0" y="0" class="label">Paper</text>
  </g>
</svg>`), 0, 0)
	require.NoError(t, err)

	img, err := png.Decode(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	assert.Greater(t, countOpaquePixels(img), 50)
}

func TestRasterize_DrawsShapesPathsAndTransforms(t *testing.T) {
	t.Parallel()

	pngBytes, width, height, err := Rasterize([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <rect x="10" y="10" width="20" height="20" fill="#ff0000"/>
  <circle cx="60" cy="20" r="10" fill="#0000ff"/>
  <path d="M 10 60 L 30 60 L 20 80 z" fill="#00ff00"/>
  <g transform="translate(50 50)"><rect width="10" height="10" fill="#000000"/></g>
</svg>`), 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 100, width)
	assert.Equal(t, 100, height)

	img, err := png.Decode(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	assert.Equal(t, color.RGBA{R: 255, A: 255}, rgbaAt(img, 15, 15))
	assert.Equal(t, color.RGBA{B: 255, A: 255}, rgbaAt(img, 60, 20))
	assert.Equal(t, color.RGBA{G: 255, A: 255}, rgbaAt(img, 20, 65))
	assert.Equal(t, color.RGBA{A: 255}, rgbaAt(img, 55, 55))
}

func TestRasterize_RejectsMalformedXML(t *testing.T) {
	t.Parallel()

	_, _, _, err := Rasterize([]byte(`<svg><rect></svg>`), 0, 0)
	require.Error(t, err)
}

func TestRasterize_RefusesOversize(t *testing.T) {
	t.Parallel()
	_, _, _, err := Rasterize([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10000 10000">
  <rect width="10" height="10"/>
</svg>`), 0, 0)
	require.ErrorIs(t, err, ErrSVGTooLarge)
}

func TestRasterizeWithLimit_RefusesCustomLimit(t *testing.T) {
	t.Parallel()
	_, _, _, err := RasterizeWithLimit([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 500 500">
  <rect width="10" height="10"/>
</svg>`), 0, 0, 1_000)
	require.ErrorIs(t, err, ErrSVGTooLarge)
}

func countOpaquePixels(img image.Image) int {
	bounds := img.Bounds()
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				count++
			}
		}
	}
	return count
}

func rgbaAt(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{R: byte(r >> 8), G: byte(g >> 8), B: byte(b >> 8), A: byte(a >> 8)}
}

func FuzzSVGPath(f *testing.F) {
	for _, seed := range []string{"M0 0 L10 10 Z", "M10 10 h20 v20 z", "M0,0 C10,0 10,20 20,20"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		dst := image.NewRGBA(image.Rect(0, 0, 32, 32))
		path := newSVGPath(svgRenderer{dst: dst, viewBox: svgViewBox{w: 32, h: 32}, scaleX: 1, scaleY: 1}, identity())
		_ = path.parse(data)
	})
}
