package translate

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSolidRGBA(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	return img
}

// minimalSVG is a tiny valid SVG used as a rasterisation fixture.
const minimalSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
  <rect x="0" y="0" width="32" height="32" fill="#ff0000"/>
</svg>`

// minimalPNG returns a 2x2 PNG byte slice (used as an <img src=…> fixture).
func minimalPNG(t *testing.T) []byte {
	t.Helper()
	// 2x2 RGBA: synthesised via image/png to avoid embedding binary fixtures.
	rgba := newSolidRGBA(2, 2)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, rgba))
	return buf.Bytes()
}

func TestSafeDefaultResolver_RefusesLocalPaths(t *testing.T) {
	t.Parallel()
	cases := []string{"/etc/passwd", "../../secret", "icon.svg", "./foo.png"}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			t.Parallel()
			_, _, err := safeDefaultResolver(src)
			require.Error(t, err)
		})
	}
}

func TestSafeDefaultResolver_DataURI_PNG(t *testing.T) {
	t.Parallel()
	pngBytes := minimalPNG(t)
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	data, ext, err := safeDefaultResolver(uri)
	require.NoError(t, err)
	assert.Equal(t, "png", ext)
	assert.Equal(t, pngBytes, data)
}

func TestSafeDefaultResolver_DataURI_SVG(t *testing.T) {
	t.Parallel()
	uri := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(minimalSVG))
	data, ext, err := safeDefaultResolver(uri)
	require.NoError(t, err)
	assert.Equal(t, "svg", ext)
	assert.Equal(t, []byte(minimalSVG), data)
}

func TestBaseDirResolver_AllowsInside(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pngBytes := minimalPNG(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "icon.png"), pngBytes, 0o600))

	r := baseDirResolver(dir)
	data, ext, err := r("icon.png")
	require.NoError(t, err)
	assert.Equal(t, "png", ext)
	assert.Equal(t, pngBytes, data)
}

func TestBaseDirResolver_RefusesEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r := baseDirResolver(dir)
	_, _, err := r("../escape.png")
	require.Error(t, err)
	_, _, err = r("/abs/escape.png")
	require.Error(t, err)
}

func TestRasteriseSVG_ProducesPNGAtRequestedSize(t *testing.T) {
	t.Parallel()
	pngBytes, pxW, pxH, err := rasteriseSVG([]byte(minimalSVG), 10.0, 10.0) // 10mm × 10mm
	require.NoError(t, err)
	// 10mm @ 150 DPI ≈ 59 px
	assert.InDelta(t, 59, pxW, 2)
	assert.InDelta(t, 59, pxH, 2)
	// Verify the PNG actually decodes.
	img, err := png.Decode(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	assert.Equal(t, pxW, img.Bounds().Dx())
	assert.Equal(t, pxH, img.Bounds().Dy())
}

func TestImageRow_DefaultResolverRefusesLocalPath_FallsBackToAlt(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><img src="local.png" alt="fallback text"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc) // no resolver configured → default safe resolver
	require.NoError(t, err)
	// Fall back to a text row containing the alt label.
	require.Len(t, rows, 1)
}

func TestImageRow_WithImageBaseDir_LoadsLocalPNG(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "logo.png"), minimalPNG(t), 0o600))

	doc, err := dom.Parse(`<html><body><img src="logo.png" width="20mm" height="20mm" alt="logo"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageBaseDir(dir))
	require.NoError(t, err)
	require.Len(t, rows, 1, "img should produce one row")
}

func TestImageRow_CustomResolverIsInvoked(t *testing.T) {
	t.Parallel()
	called := false
	resolver := func(src string) ([]byte, string, error) {
		called = true
		assert.Equal(t, "any.png", src)
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><img src="any.png" width="10mm" height="10mm"></body></html>`)
	require.NoError(t, err)

	_, err = Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	assert.True(t, called, "custom resolver should be invoked")
}

func TestImageRow_UnsupportedSVGFallsBackToAlt(t *testing.T) {
	t.Parallel()
	// oksvg's IgnoreErrorMode lets junk parse-but-render-nothing; we still
	// exercise the err path via an obviously broken payload.
	resolver := func(_ string) ([]byte, string, error) {
		return []byte("<<<not svg>>>"), "svg", nil
	}

	doc, err := dom.Parse(`<html><body><img src="x.svg" alt="alt"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	// alt fallback → one text row.
	require.Len(t, rows, 1)
}

func TestInlineImage_DataURIProducesRichImageRun(t *testing.T) {
	t.Parallel()

	pngBytes := minimalPNG(t)
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	doc, err := dom.Parse(`<html><body><p>A <img src="` + uri + `" width="4mm" height="3mm" alt="logo"> B</p></body></html>`)
	require.NoError(t, err)

	var p *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			p = n
		}
		return true
	})
	require.NotNil(t, p)

	tr := &translator{}
	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image == nil {
			continue
		}
		assert.Equal(t, "logo", run.Image.Alt)
		assert.InDelta(t, 4.0, run.Image.Width, 0.001)
		assert.InDelta(t, 3.0, run.Image.Height, 0.001)
		found = true
	}
	assert.True(t, found, "expected inline image run")
}

func TestInlineImage_SVGRasterisesToRichImageRun(t *testing.T) {
	t.Parallel()

	uri := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(minimalSVG))
	doc, err := dom.Parse(`<html><body><p>Icon <img src="` + uri + `" width="5mm" height="5mm" alt="svg icon"></p></body></html>`)
	require.NoError(t, err)

	var p *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			p = n
		}
		return true
	})
	require.NotNil(t, p)

	tr := &translator{}
	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image == nil {
			continue
		}
		assert.Equal(t, "svg icon", run.Image.Alt)
		assert.Equal(t, extension.Png, run.Image.Extension)
		assert.NotEmpty(t, run.Image.Bytes)
		found = true
	}
	assert.True(t, found, "expected inline SVG image run")
}
