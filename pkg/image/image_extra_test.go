package image_test

import (
	"bytes"
	goimage "image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/image"
	"golang.org/x/image/tiff"
)

func encodeGIF(t *testing.T) []byte {
	t.Helper()
	img := goimage.NewPaletted(goimage.Rect(0, 0, 2, 2), []color.Color{color.Black, color.White})
	var buf bytes.Buffer
	require.NoError(t, gif.Encode(&buf, img, nil))
	return buf.Bytes()
}

func encodeTIFF(t *testing.T) []byte {
	t.Helper()
	img := goimage.NewRGBA(goimage.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	require.NoError(t, tiff.Encode(&buf, img, nil))
	return buf.Bytes()
}

func TestNewGIFAndLoadGIF(t *testing.T) {
	t.Parallel()

	data := encodeGIF(t)
	img, err := image.NewGIF(data)
	require.NoError(t, err)
	assert.Equal(t, extension.Gif, img.Extension())

	path := filepath.Join(t.TempDir(), "img.gif")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	loaded, err := image.LoadGIF(path)
	require.NoError(t, err)
	assert.Equal(t, extension.Gif, loaded.Extension())
}

func TestNewTIFFAndLoadTIFF(t *testing.T) {
	t.Parallel()

	data := encodeTIFF(t)
	img, err := image.NewTIFF(data)
	require.NoError(t, err)
	assert.Equal(t, extension.Tiff, img.Extension())

	path := filepath.Join(t.TempDir(), "img.tiff")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	loaded, err := image.LoadTIFF(path)
	require.NoError(t, err)
	assert.Equal(t, extension.Tiff, loaded.Extension())
}

func TestLoadJPEG(t *testing.T) {
	t.Parallel()

	img, err := image.LoadJPEG("../../test/assets/images/biplane.jpg")
	require.NoError(t, err)
	assert.Equal(t, extension.Jpg, img.Extension())
	assert.NotEmpty(t, img.Bytes())
}

func TestNewWebPAndLoadWebP_InvalidData(t *testing.T) {
	t.Parallel()

	_, err := image.NewWebP([]byte("not a webp"))
	assert.Error(t, err)

	path := filepath.Join(t.TempDir(), "img.webp")
	require.NoError(t, os.WriteFile(path, []byte("not a webp"), 0o600))
	_, err = image.LoadWebP(path)
	assert.Error(t, err)

	_, err = image.LoadWebP(filepath.Join(t.TempDir(), "missing.webp"))
	assert.Error(t, err)
}
