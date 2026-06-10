package svg

import (
	"bytes"
	"image"
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
