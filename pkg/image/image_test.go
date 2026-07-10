package image_test

import (
	"bytes"
	"errors"
	goimage "image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	paperimage "github.com/avdoseferovic/paper/pkg/image"
	"golang.org/x/image/tiff"
)

func TestNewPNG(t *testing.T) {
	t.Parallel()

	data := pngBytes(t, 3, 2)

	img, err := paperimage.NewPNG(data)

	require.NoError(t, err)
	assert.Equal(t, 3, img.Width())
	assert.Equal(t, 2, img.Height())
	assert.Equal(t, data, img.Bytes())
	assert.NotNil(t, img.Dimensions())
	assert.NotNil(t, img.Component())
	assert.NotNil(t, img.Col(6))
	assert.NotNil(t, img.Row(10))
	assert.NotNil(t, img.AutoRow())
}

func TestNewJPEG(t *testing.T) {
	t.Parallel()

	img, err := paperimage.NewJPEG(jpegBytes(t, 4, 3))

	require.NoError(t, err)
	assert.Equal(t, 4, img.Width())
	assert.Equal(t, 3, img.Height())
}

func TestNewGIF(t *testing.T) {
	t.Parallel()

	img, err := paperimage.NewGIF(gifBytes(t, 5, 4))

	require.NoError(t, err)
	assert.Equal(t, 5, img.Width())
	assert.Equal(t, 4, img.Height())
}

func TestNewTIFF(t *testing.T) {
	t.Parallel()

	img, err := paperimage.NewTIFF(tiffBytes(t, 6, 5))

	require.NoError(t, err)
	assert.Equal(t, 6, img.Width())
	assert.Equal(t, 5, img.Height())
}

func TestLoadPNG(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	require.NoError(t, os.WriteFile(path, pngBytes(t, 7, 6), 0o600))

	img, err := paperimage.LoadPNG(path)

	require.NoError(t, err)
	assert.Equal(t, 7, img.Width())
	assert.Equal(t, 6, img.Height())
}

func TestLoadPNG_NotFound(t *testing.T) {
	t.Parallel()

	_, err := paperimage.LoadPNG(filepath.Join(t.TempDir(), "missing.png"))

	require.ErrorIs(t, err, paperimage.ErrCannotReadFile)
}

func TestNewPNG_RejectsWrongFormat(t *testing.T) {
	t.Parallel()

	_, err := paperimage.NewPNG(jpegBytes(t, 1, 1))

	require.ErrorIs(t, err, paperimage.ErrUnexpectedImageType)
}

func TestNewPNG_RejectsInvalidBytes(t *testing.T) {
	t.Parallel()

	_, err := paperimage.NewPNG([]byte("not an image"))

	require.ErrorIs(t, err, paperimage.ErrInvalidImage)
}

func TestNilImageMethods(t *testing.T) {
	t.Parallel()

	var img *paperimage.Image

	assert.Nil(t, img.Bytes())
	assert.Equal(t, 0, img.Width())
	assert.Equal(t, 0, img.Height())
	assert.Nil(t, img.Dimensions())
	assert.NotNil(t, img.Component())
	assert.NotNil(t, img.Col(1))
	assert.NotNil(t, img.Row(1))
	assert.NotNil(t, img.AutoRow())
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	assert.True(t, errors.Is(paperimage.ErrCannotReadFile, paperimage.ErrCannotReadFile))
	assert.True(t, errors.Is(paperimage.ErrInvalidImage, paperimage.ErrInvalidImage))
	assert.True(t, errors.Is(paperimage.ErrInvalidDimensions, paperimage.ErrInvalidDimensions))
	assert.True(t, errors.Is(paperimage.ErrUnexpectedImageType, paperimage.ErrUnexpectedImageType))
}

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, rgba(width, height)))
	return buf.Bytes()
}

func jpegBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, rgba(width, height), nil))
	return buf.Bytes()
}

func gifBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := goimage.NewPaletted(goimage.Rect(0, 0, width, height), []color.Color{color.White, color.Black})
	var buf bytes.Buffer
	require.NoError(t, gif.Encode(&buf, img, nil))
	return buf.Bytes()
}

func tiffBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, tiff.Encode(&buf, rgba(width, height), nil))
	return buf.Bytes()
}

func rgba(width, height int) *goimage.RGBA {
	img := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	for x := range width {
		for y := range height {
			img.SetRGBA(x, y, color.RGBA{R: 20, G: 120, B: 200, A: 255})
		}
	}
	return img
}
