package cellwriter

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	gofpdf "github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// chainRecorder is a minimal CellWriter that records Apply calls. The
// generated mocks package cannot be used from inside package cellwriter
// (import cycle), so a hand-rolled fake is used here.
type chainRecorder struct {
	stylerTemplate
	calls int
}

func (c *chainRecorder) Apply(_, _ float64, _ *entity.Config, _ *props.Cell) {
	c.calls++
}

type bgImageCall struct {
	name       string
	x, y, w, h float64
}

type bgAlphaCall struct {
	alpha float64
	mode  string
}

type bgClipCall struct {
	x, y, w, h float64
}

// bgPDFStub implements backgroundImagePDF. Image registration is delegated to
// a real gofpdf instance because ImageInfoType cannot be constructed directly.
type bgPDFStub struct {
	reg        *gofpdf.PDF
	x, y       float64
	alpha      float64
	blendMode  string
	alphaCalls []bgAlphaCall
	clipRects  []bgClipCall
	clipEnds   int
	images     []bgImageCall
}

func newBgPDFStub(x, y float64) *bgPDFStub {
	return &bgPDFStub{
		reg:       gofpdf.NewCustom(&gofpdf.InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"}),
		x:         x,
		y:         y,
		alpha:     1,
		blendMode: "Normal",
	}
}

func (s *bgPDFStub) ClipEnd() { s.clipEnds++ }

func (s *bgPDFStub) ClipRect(x, y, w, h float64, _ bool) {
	s.clipRects = append(s.clipRects, bgClipCall{x, y, w, h})
}

func (s *bgPDFStub) GetAlpha() (float64, string) { return s.alpha, s.blendMode }

func (s *bgPDFStub) GetXY() (float64, float64) { return s.x, s.y }

func (s *bgPDFStub) Image(imageNameStr string, x, y, w, h float64, _ bool, _ string, _ int, _ string) {
	s.images = append(s.images, bgImageCall{imageNameStr, x, y, w, h})
}

func (s *bgPDFStub) RegisterImageOptionsReader(imgName string, options gofpdf.ImageOptions, r io.Reader) *gofpdf.ImageInfoType {
	return s.reg.RegisterImageOptionsReader(imgName, options, r)
}

func (s *bgPDFStub) SetAlpha(alpha float64, blendModeStr string) {
	s.alphaCalls = append(s.alphaCalls, bgAlphaCall{alpha, blendModeStr})
}

func testPNGBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.SetRGBA(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

const validSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" fill="#f00"/></svg>`

// invalidSVG exceeds the rasterizer's pixel limit, forcing a rasterize error.
const invalidSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100000 100000"><rect width="1" height="1"/></svg>`

func bgProp(imageBytes []byte, ext extension.Type, size, position, repeat string) *props.Cell {
	return &props.Cell{
		BackgroundImage: &props.CellBackgroundImage{
			Bytes:     imageBytes,
			Extension: ext,
			Size:      size,
			Position:  position,
			Repeat:    repeat,
		},
	}
}

func TestBackgroundImageStyler_Apply(t *testing.T) {
	t.Parallel()

	t.Run("when prop is nil, should call next and draw nothing", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(0, 0)
		next := &chainRecorder{}
		sut := NewBackgroundImageStyler(stub)
		sut.SetNext(next)

		sut.Apply(10, 10, &entity.Config{}, nil)

		assert.Equal(t, 1, next.calls)
		assert.Len(t, stub.images, 0)
		assert.Len(t, stub.clipRects, 0)
	})

	t.Run("when background image bytes are empty, should call next and draw nothing", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(0, 0)
		next := &chainRecorder{}
		sut := NewBackgroundImageStyler(stub)
		sut.SetNext(next)

		sut.Apply(10, 10, &entity.Config{}, &props.Cell{BackgroundImage: &props.CellBackgroundImage{}})

		assert.Equal(t, 1, next.calls)
		assert.Len(t, stub.images, 0)
	})

	t.Run("when width is zero, should not register or draw", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(0, 0)
		sut := NewBackgroundImageStyler(stub)

		sut.Apply(0, 10, &entity.Config{}, bgProp(testPNGBytes(t, 4, 2), extension.Png, "", "", ""))

		assert.Len(t, stub.images, 0)
		assert.Len(t, stub.clipRects, 0)
	})

	t.Run("when image bytes are invalid, should not draw", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(0, 0)
		sut := NewBackgroundImageStyler(stub)

		sut.Apply(10, 10, &entity.Config{}, bgProp([]byte{1, 2, 3}, extension.Png, "", "", ""))

		assert.Len(t, stub.images, 0)
		assert.Len(t, stub.clipRects, 0)
	})

	t.Run("when no-repeat, should clip cell and draw a single tile at position", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(10, 20)
		next := &chainRecorder{}
		sut := NewBackgroundImageStyler(stub)
		sut.SetNext(next)

		sut.Apply(8, 4, &entity.Config{}, bgProp(testPNGBytes(t, 4, 2), extension.Png, "4mm 2mm", "", "no-repeat"))

		assert.Equal(t, 1, next.calls)
		require.Len(t, stub.clipRects, 1)
		assert.Equal(t, bgClipCall{10, 20, 8, 4}, stub.clipRects[0])
		assert.Equal(t, 1, stub.clipEnds)
		require.Len(t, stub.images, 1)
		assert.InDelta(t, 10.0, stub.images[0].x, 0.001)
		assert.InDelta(t, 20.0, stub.images[0].y, 0.001)
		assert.InDelta(t, 4.0, stub.images[0].w, 0.001)
		assert.InDelta(t, 2.0, stub.images[0].h, 0.001)
		// Full alpha: no temporary alpha override needed.
		assert.Len(t, stub.alphaCalls, 0)
	})

	t.Run("when repeating, should tile the whole cell", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(10, 20)
		sut := NewBackgroundImageStyler(stub)

		sut.Apply(8, 4, &entity.Config{}, bgProp(testPNGBytes(t, 4, 2), extension.Png, "4mm 2mm", "", "repeat"))

		require.Len(t, stub.images, 4)
		got := map[bgImageCall]bool{}
		for _, img := range stub.images {
			got[bgImageCall{"", img.x, img.y, img.w, img.h}] = true
		}
		assert.True(t, got[bgImageCall{"", 10, 20, 4, 2}])
		assert.True(t, got[bgImageCall{"", 14, 20, 4, 2}])
		assert.True(t, got[bgImageCall{"", 10, 22, 4, 2}])
		assert.True(t, got[bgImageCall{"", 14, 22, 4, 2}])
	})

	t.Run("when repeating with offset position, should backfill tiles before the origin", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(10, 20)
		sut := NewBackgroundImageStyler(stub)

		sut.Apply(8, 4, &entity.Config{}, bgProp(testPNGBytes(t, 4, 2), extension.Png, "4mm 2mm", "3mm 1mm", "repeat"))

		// startX = 13 backfilled to 9; startY = 21 backfilled to 19.
		require.Len(t, stub.images, 9)
		assert.InDelta(t, 9.0, stub.images[0].x, 0.001)
		assert.InDelta(t, 19.0, stub.images[0].y, 0.001)
	})

	t.Run("when repeat-x, should tile a single row", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(10, 20)
		sut := NewBackgroundImageStyler(stub)

		sut.Apply(8, 4, &entity.Config{}, bgProp(testPNGBytes(t, 4, 2), extension.Png, "4mm 2mm", "", "repeat-x"))

		require.Len(t, stub.images, 2)
		assert.InDelta(t, 20.0, stub.images[0].y, 0.001)
		assert.InDelta(t, 20.0, stub.images[1].y, 0.001)
	})

	t.Run("when alpha is below one, should draw at full alpha and restore", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(0, 0)
		stub.alpha = 0.5
		stub.blendMode = "Multiply"
		sut := NewBackgroundImageStyler(stub)

		sut.Apply(8, 4, &entity.Config{}, bgProp(testPNGBytes(t, 4, 2), extension.Png, "4mm 2mm", "", "no-repeat"))

		require.Len(t, stub.alphaCalls, 2)
		assert.Equal(t, bgAlphaCall{1, "Normal"}, stub.alphaCalls[0])
		assert.Equal(t, bgAlphaCall{0.5, "Multiply"}, stub.alphaCalls[1])
	})

	t.Run("when extension is svg, should rasterize and draw", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(0, 0)
		sut := NewBackgroundImageStyler(stub)

		sut.Apply(8, 4, &entity.Config{}, bgProp([]byte(validSVG), extension.Svg, "4mm 2mm", "", "no-repeat"))

		require.Len(t, stub.images, 1)
		assert.InDelta(t, 4.0, stub.images[0].w, 0.001)
	})

	t.Run("when svg is invalid, should not draw", func(t *testing.T) {
		t.Parallel()
		stub := newBgPDFStub(0, 0)
		sut := NewBackgroundImageStyler(stub)

		sut.Apply(8, 4, &entity.Config{}, bgProp([]byte(invalidSVG), extension.Svg, "", "", ""))

		assert.Len(t, stub.images, 0)
		assert.Len(t, stub.clipRects, 0)
	})
}

func TestBackgroundImageStyler_DrawTiles_InvalidImageSize(t *testing.T) {
	t.Parallel()
	stub := newBgPDFStub(0, 0)
	sut := &backgroundImageStyler{stylerTemplate: stylerTemplate{fpdf: stub, name: "backgroundImageStyler"}}

	sut.drawTiles(stub, "img", 0, 0, 0, 0, 10, 10, 0, 2, true, true)
	sut.drawTiles(stub, "img", 0, 0, 0, 0, 10, 10, 2, 0, true, true)

	assert.Len(t, stub.images, 0)
}

func TestNormalizeBackgroundImageBytes(t *testing.T) {
	t.Parallel()

	t.Run("non-svg bytes pass through unchanged", func(t *testing.T) {
		t.Parallel()
		input := []byte{9, 8, 7}

		out, ext, err := normalizeBackgroundImageBytes(input, extension.Jpg)

		assert.Nil(t, err)
		assert.Equal(t, input, out)
		assert.Equal(t, extension.Jpg, ext)
	})

	t.Run("valid svg is rasterized to png", func(t *testing.T) {
		t.Parallel()
		out, ext, err := normalizeBackgroundImageBytes([]byte(validSVG), extension.Svg)

		assert.Nil(t, err)
		assert.Equal(t, extension.Png, ext)
		assert.True(t, len(out) > 0)
	})

	t.Run("invalid svg returns error", func(t *testing.T) {
		t.Parallel()
		out, _, err := normalizeBackgroundImageBytes([]byte(invalidSVG), extension.Svg)

		assert.NotNil(t, err)
		assert.Nil(t, out)
	})
}
