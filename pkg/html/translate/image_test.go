package translate

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	svgraster "github.com/avdoseferovic/paper/internal/svg"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

func newSolidRGBA(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
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

func TestSelectSrcsetCandidate(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "large.png", selectSrcsetCandidate("fallback.png", "small.png 1x,large.png 2x", "", 170))
	assert.Equal(t, "wide.png", selectSrcsetCandidate("fallback.png", "narrow.png 400w, wide.png 800w", "", 170))
	assert.Equal(t, "fallback.png", selectSrcsetCandidate("fallback.png", "", "", 170))

	dataURL := "data:image/png;base64,abc"
	assert.Equal(t, dataURL, selectSrcsetCandidate("fallback.png", dataURL+" 2x, low.png 1x", "", 170))
}

func TestSelectSrcsetCandidate_UsesSizesForWidthDescriptors(t *testing.T) {
	t.Parallel()

	srcset := "small.png 400w, medium.png 800w, large.png 1200w"
	assert.Equal(t, "small.png", selectSrcsetCandidate("fallback.png", srcset, "80mm", 170))
	assert.Equal(t, "medium.png", selectSrcsetCandidate("fallback.png", srcset, "180mm", 170))
	assert.Equal(t, "large.png", selectSrcsetCandidate("fallback.png", srcset, "500mm", 170))
	assert.Equal(t, "small.png", selectSrcsetCandidate("fallback.png", srcset, "50%", 170))
	assert.Equal(t, "small.png", selectSrcsetCandidate("fallback.png", srcset, "50vw", 170))
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

func TestBaseDirResolver_RefusesEmptyDir(t *testing.T) {
	t.Parallel()
	r := baseDirResolver("")
	_, _, err := r("anything.png")
	require.ErrorIs(t, err, errBaseDirEmpty)
}

func TestBaseDirResolver_RefusesSymlinkEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.png")
	require.NoError(t, os.WriteFile(secret, minimalPNG(t), 0o600))
	if err := os.Symlink(secret, filepath.Join(dir, "link.png")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	r := baseDirResolver(dir)
	_, _, err := r("link.png")
	require.Error(t, err, "symlink pointing outside base dir must be refused")
}

func TestStylesheetResolver_RefusesEmptyDir(t *testing.T) {
	t.Parallel()
	r := stylesheetBaseDirResolver("")
	_, err := r("anything.css")
	require.ErrorIs(t, err, errBaseDirEmpty)
}

func TestStylesheetResolver_RefusesSymlinkEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.css")
	require.NoError(t, os.WriteFile(secret, []byte("p{color:red}"), 0o600))
	if err := os.Symlink(secret, filepath.Join(dir, "link.css")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	r := stylesheetBaseDirResolver(dir)
	_, err := r("link.css")
	require.Error(t, err, "symlink pointing outside base dir must be refused")
}

func TestRasteriseSVG_RefusesOversizeMM(t *testing.T) {
	t.Parallel()
	_, _, _, err := svgraster.Rasterize([]byte(minimalSVG), 1600.0, 1600.0)
	require.ErrorIs(t, err, svgraster.ErrSVGTooLarge)
}

func TestRasteriseSVG_RefusesOversizeViewBox(t *testing.T) {
	t.Parallel()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10000 10000"><rect width="10" height="10"/></svg>`
	_, _, _, err := svgraster.Rasterize([]byte(svg), 0, 0)
	require.ErrorIs(t, err, svgraster.ErrSVGTooLarge)
}

func TestRasteriseSVG_ProducesPNGAtRequestedSize(t *testing.T) {
	t.Parallel()
	pngBytes, pxW, pxH, err := svgraster.Rasterize([]byte(minimalSVG), 10.0, 10.0) // 10mm × 10mm
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

func TestImageRow_DisplayNoneSkipsImageWithoutAltFallback(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><head><style>.hidden{display:none}</style></head><body><img class="hidden" src="local.png" alt="fallback text"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	require.Empty(t, rows)
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

func TestImageRow_SrcsetSelectsHighestDensityCandidate(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><img src="small.png" srcset="small.png 1x, large.png 2x" width="10mm" height="10mm"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "large.png", calledSrc)
}

func TestImageRow_SrcsetUsesSizesForWidthDescriptors(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><img src="fallback.png" srcset="small.png 400w, medium.png 800w, large.png 1200w" sizes="80mm" width="10mm" height="10mm"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "small.png", calledSrc)
}

func TestPictureRow_UsesFirstSourceSrcsetCandidate(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><picture><source srcset="hero-small.png 1x, hero-large.png 2x"><img src="fallback.png" width="8mm" height="6mm" alt="hero"></picture></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "hero-large.png", calledSrc)
}

func TestPictureRow_SourceSizesDriveWidthDescriptorSelection(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><picture>
<source media="print" sizes="80mm" srcset="small.png 400w, medium.png 800w, large.png 1200w">
<img src="fallback.png" sizes="200mm" width="8mm" height="6mm" alt="hero">
</picture></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "small.png", calledSrc)
}

func TestPictureRow_SelectsPrintMediaAndSupportedTypeSource(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		if strings.HasSuffix(src, ".svg") {
			return []byte(minimalSVG), "svg", nil
		}
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><picture>
<source media="screen" srcset="screen.png">
<source type="image/webp" srcset="webp.webp">
<source media="print" type="image/svg+xml" srcset="print.svg">
<img src="fallback.png" width="8mm" height="6mm" alt="hero">
</picture></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "print.svg", calledSrc)
}

func TestPictureRow_EvaluatesSourceWidthMediaAgainstContentWidth(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><picture>
<source media="print and (max-width: 500px)" srcset="narrow.png">
<source media="print and (min-width: 600px)" srcset="wide.png">
<img src="fallback.png" width="8mm" height="6mm" alt="hero">
</picture></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "wide.png", calledSrc)
}

func TestPictureRow_SourceWidthMediaUsesConfiguredContentWidth(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><picture>
<source media="print and (max-width: 500px)" srcset="narrow.png">
<source media="print and (min-width: 600px)" srcset="wide.png">
<img src="fallback.png" width="8mm" height="6mm" alt="hero">
</picture></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver), WithContentWidth(80))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "narrow.png", calledSrc)
}

func TestImageRow_CSSDimensionsOverrideAttributes(t *testing.T) {
	t.Parallel()

	resolver := func(src string) ([]byte, string, error) {
		assert.Equal(t, "logo.png", src)
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><head><style>.logo { width: 12mm; height: 7mm }</style></head><body><img class="logo" src="logo.png" width="1mm" height="1mm"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.InDelta(t, 7.0, rows[0].GetHeight(nil, &entity.Cell{}), 0.001)
}

func TestImageRow_MaxWidthClampsAutoHeight(t *testing.T) {
	t.Parallel()

	resolver := func(src string) ([]byte, string, error) {
		assert.Equal(t, "logo.png", src)
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><head><style>.logo { width: 120mm; max-width: 25% }</style></head><body><img class="logo" src="logo.png"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver), WithContentWidth(80))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.InDelta(t, 20.0, rows[0].GetHeight(nil, &entity.Cell{}), 0.001)
}

func TestImageRow_ObjectFitAndPositionMappedFromCSS(t *testing.T) {
	t.Parallel()

	resolver := func(src string) ([]byte, string, error) {
		assert.Equal(t, "photo.png", src)
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><head><style>.photo { width: 40mm; height: 20mm; object-fit: cover; object-position: right bottom }</style></head><body><img class="photo" src="photo.png"></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	var details map[string]any
	walkStructure(rows[0].GetStructure(), func(s core.Structure) {
		if s.Type == "bytesImage" {
			details = s.Details
		}
	})
	require.NotNil(t, details)
	assert.Equal(t, "cover", details["prop_object_fit"])
	assert.Equal(t, "right bottom", details["prop_object_position"])
}

func TestSVGElement_BlockRendersAsImageRow(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><head><style>.mark { width: 12mm; height: 6mm }</style></head><body><svg class="mark" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 16"><rect width="32" height="16" fill="red"/></svg></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.InDelta(t, 6.0, rows[0].GetHeight(nil, &entity.Cell{}), 0.001)

	var found bool
	walkStructure(rows[0].GetStructure(), func(s core.Structure) {
		if s.Type == "bytesImage" {
			found = true
			assert.Equal(t, extension.Png, s.Details["extension"])
		}
	})
	assert.True(t, found, "expected SVG element to become a raster image component")
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

func TestInlineImage_SrcsetSelectsHighestDensityCandidate(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><p>A <img src="small.png" srcset="small.png 1x, large.png 2x" width="4mm" height="3mm" alt="logo"> B</p></body></html>`)
	require.NoError(t, err)

	p := firstNodeByTag(doc, "p")
	require.NotNil(t, p)

	tr := &translator{imageResolver: resolver}
	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image != nil {
			found = true
		}
	}
	assert.True(t, found, "expected inline image run")
	assert.Equal(t, "large.png", calledSrc)
}

func TestInlinePicture_UsesSourceSrcsetCandidate(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><p>A <picture><source srcset="small.png 1x, large.png 2x"><img src="fallback.png" width="4mm" height="3mm" alt="hero"></picture> B</p></body></html>`)
	require.NoError(t, err)

	p := firstNodeByTag(doc, "p")
	require.NotNil(t, p)

	tr := &translator{imageResolver: resolver}
	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image != nil {
			found = true
		}
	}
	assert.True(t, found, "expected inline picture to produce image run")
	assert.Equal(t, "large.png", calledSrc)
}

func TestInlinePicture_SkipsScreenAndUnsupportedTypeSources(t *testing.T) {
	t.Parallel()
	var calledSrc string
	resolver := func(src string) ([]byte, string, error) {
		calledSrc = src
		return minimalPNG(t), "png", nil
	}

	doc, err := dom.Parse(`<html><body><p>A <picture>
<source media="screen" srcset="screen.png">
<source type="image/avif" srcset="modern.avif">
<source media="all" type="image/png" srcset="print.png">
<img src="fallback.png" width="4mm" height="3mm" alt="hero">
</picture> B</p></body></html>`)
	require.NoError(t, err)

	p := firstNodeByTag(doc, "p")
	require.NotNil(t, p)

	tr := &translator{imageResolver: resolver}
	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image != nil {
			found = true
		}
	}
	assert.True(t, found, "expected inline picture to produce image run")
	assert.Equal(t, "print.png", calledSrc)
}

func TestInlineImage_CSSDimensionsFromStylesheet(t *testing.T) {
	t.Parallel()

	pngBytes := minimalPNG(t)
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	doc, err := dom.Parse(`<html><head><style>.icon { width: 25%; height: 3mm }</style></head><body><p>A <img class="icon" src="` + uri + `" alt="logo"> B</p></body></html>`)
	require.NoError(t, err)

	inlineCSS, _ := doc.StyleSources()
	tr := &translator{
		sheet:          parseStylesheet(inlineCSS),
		contentWidthMM: 80,
	}
	p := firstNodeByTag(doc, "p")
	require.NotNil(t, p)

	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image == nil {
			continue
		}
		assert.InDelta(t, 20.0, run.Image.Width, 0.001)
		assert.InDelta(t, 3.0, run.Image.Height, 0.001)
		found = true
	}
	assert.True(t, found, "expected inline image run")
}

func TestInlineImage_MinWidthClampsAutoHeight(t *testing.T) {
	t.Parallel()

	pngBytes := minimalPNG(t)
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	doc, err := dom.Parse(`<html><head><style>.icon { width: 5mm; min-width: 25% }</style></head><body><p>A <img class="icon" src="` + uri + `" alt="logo"> B</p></body></html>`)
	require.NoError(t, err)

	inlineCSS, _ := doc.StyleSources()
	tr := &translator{
		sheet:          parseStylesheet(inlineCSS),
		contentWidthMM: 80,
	}
	p := firstNodeByTag(doc, "p")
	require.NotNil(t, p)

	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image == nil {
			continue
		}
		assert.InDelta(t, 20.0, run.Image.Width, 0.001)
		assert.InDelta(t, 20.0, run.Image.Height, 0.001)
		found = true
	}
	assert.True(t, found, "expected inline image run")
}

func TestInlineImage_ObjectFitAndPositionMappedFromCSS(t *testing.T) {
	t.Parallel()

	pngBytes := minimalPNG(t)
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	doc, err := dom.Parse(`<html><head><style>.icon { width: 8mm; height: 4mm; object-fit: cover; object-position: left top }</style></head><body><p>A <img class="icon" src="` + uri + `" alt="logo"> B</p></body></html>`)
	require.NoError(t, err)

	inlineCSS, _ := doc.StyleSources()
	tr := &translator{
		sheet:          parseStylesheet(inlineCSS),
		contentWidthMM: 80,
	}
	p := firstNodeByTag(doc, "p")
	require.NotNil(t, p)

	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image == nil {
			continue
		}
		assert.Equal(t, "cover", run.Image.ObjectFit)
		assert.Equal(t, "left top", run.Image.ObjectPosition)
		assert.InDelta(t, 8.0, run.Image.Width, 0.001)
		assert.InDelta(t, 4.0, run.Image.Height, 0.001)
		found = true
	}
	assert.True(t, found, "expected inline image run")
}

func TestInlineSVGElement_RendersAsRichImageRun(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><p>A <svg xmlns="http://www.w3.org/2000/svg" width="4mm" height="3mm" viewBox="0 0 40 30"><rect width="40" height="30" fill="red"/></svg> B</p></body></html>`)
	require.NoError(t, err)

	p := firstNodeByTag(doc, "p")
	require.NotNil(t, p)

	tr := &translator{}
	runs := tr.inlineRuns(p)

	var found bool
	for _, run := range runs {
		if run.Image == nil {
			continue
		}
		found = true
		assert.Equal(t, extension.Png, run.Image.Extension)
		assert.InDelta(t, 4.0, run.Image.Width, 0.001)
		assert.InDelta(t, 3.0, run.Image.Height, 0.001)
	}
	assert.True(t, found, "expected inline SVG element to become a RichImage run")
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

func TestBackgroundImage_URLProducesContainerBackgroundImage(t *testing.T) {
	t.Parallel()

	pngBytes := minimalPNG(t)
	resolver := func(src string) ([]byte, string, error) {
		assert.Equal(t, "bg.png", src)
		return pngBytes, "png", nil
	}
	doc, err := dom.Parse(`<html><head><style>.bg { background-image:url("bg.png"); background-size:cover; background-position:center bottom; background-repeat:no-repeat; padding:1mm }</style></head><body><div class="bg"><p>A</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	containerRow, ok := rows[0].(*splittableContainerRow)
	require.True(t, ok)
	require.NotNil(t, containerRow.container)
	require.NotNil(t, containerRow.container.style)
	require.NotNil(t, containerRow.container.style.BackgroundImage)
	assert.Equal(t, extension.Png, containerRow.container.style.BackgroundImage.Extension)
	assert.Equal(t, pngBytes, containerRow.container.style.BackgroundImage.Bytes)
	assert.Equal(t, "cover", containerRow.container.style.BackgroundImage.Size)
	assert.Equal(t, "center bottom", containerRow.container.style.BackgroundImage.Position)
	assert.Equal(t, "no-repeat", containerRow.container.style.BackgroundImage.Repeat)
}

func TestBackgroundImage_ShorthandURLProducesContainerBackgroundImage(t *testing.T) {
	t.Parallel()

	pngBytes := minimalPNG(t)
	resolver := func(src string) ([]byte, string, error) {
		assert.Equal(t, "bg.png", src)
		return pngBytes, "png", nil
	}
	doc, err := dom.Parse(`<html><head><style>.bg { background:#112233 url("bg.png") center bottom / cover no-repeat; padding:1mm }</style></head><body><div class="bg"><p>A</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	containerRow, ok := rows[0].(*splittableContainerRow)
	require.True(t, ok)
	require.NotNil(t, containerRow.container)
	require.NotNil(t, containerRow.container.style)
	require.NotNil(t, containerRow.container.style.BackgroundColor)
	assert.Equal(t, 0x11, containerRow.container.style.BackgroundColor.Red)
	require.NotNil(t, containerRow.container.style.BackgroundImage)
	assert.Equal(t, extension.Png, containerRow.container.style.BackgroundImage.Extension)
	assert.Equal(t, pngBytes, containerRow.container.style.BackgroundImage.Bytes)
	assert.Equal(t, "cover", containerRow.container.style.BackgroundImage.Size)
	assert.Equal(t, "center bottom", containerRow.container.style.BackgroundImage.Position)
	assert.Equal(t, "no-repeat", containerRow.container.style.BackgroundImage.Repeat)
}

func TestBackgroundImage_DataURIInlineStyleProducesContainerBackgroundImage(t *testing.T) {
	t.Parallel()

	pngBytes := minimalPNG(t)
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	doc, err := dom.Parse(`<html><body><div style="background-image:url('` + uri + `'); padding:1mm"><p>A</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	containerRow, ok := rows[0].(*splittableContainerRow)
	require.True(t, ok)
	require.NotNil(t, containerRow.container)
	require.NotNil(t, containerRow.container.style)
	require.NotNil(t, containerRow.container.style.BackgroundImage)
	assert.Equal(t, extension.Png, containerRow.container.style.BackgroundImage.Extension)
	assert.Equal(t, pngBytes, containerRow.container.style.BackgroundImage.Bytes)
}

func TestBackgroundImage_SVGRasterisesToPNG(t *testing.T) {
	t.Parallel()

	resolver := func(src string) ([]byte, string, error) {
		assert.Equal(t, "bg.svg", src)
		return []byte(minimalSVG), "svg", nil
	}
	doc, err := dom.Parse(`<html><head><style>.bg { background-image:url(bg.svg); padding:1mm }</style></head><body><div class="bg"><p>A</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc, WithImageResolver(resolver))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	containerRow, ok := rows[0].(*splittableContainerRow)
	require.True(t, ok)
	require.NotNil(t, containerRow.container)
	require.NotNil(t, containerRow.container.style)
	require.NotNil(t, containerRow.container.style.BackgroundImage)
	assert.Equal(t, extension.Png, containerRow.container.style.BackgroundImage.Extension)
	assert.NotEmpty(t, containerRow.container.style.BackgroundImage.Bytes)
}

func TestBackgroundImage_ConicGradientProducesContainerGradient(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><head><style>.bg { background:conic-gradient(from 90deg at center, red 0deg, blue 360deg); padding:1mm }</style></head><body><div class="bg"><p>A</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	containerRow, ok := rows[0].(*splittableContainerRow)
	require.True(t, ok)
	require.NotNil(t, containerRow.container)
	require.NotNil(t, containerRow.container.style)
	require.NotNil(t, containerRow.container.style.BackgroundGradient)
	assert.Equal(t, props.GradientConic, containerRow.container.style.BackgroundGradient.Kind)
	assert.InDelta(t, 90.0, containerRow.container.style.BackgroundGradient.AngleDeg, 0.001)
	assert.InDelta(t, 0.5, containerRow.container.style.BackgroundGradient.CX, 0.001)
	assert.InDelta(t, 0.5, containerRow.container.style.BackgroundGradient.CY, 0.001)
	require.Len(t, containerRow.container.style.BackgroundGradient.Stops, 2)
}

func firstNodeByTag(doc *dom.Document, tag string) *dom.Node {
	var found *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == tag {
			found = n
			return false
		}
		return true
	})
	return found
}
