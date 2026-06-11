package paper

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
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type objectImageCall struct {
	name       string
	x, y, w, h float64
}

// objectBoxPDFStub implements imagePDF, recording draw calls.
type objectBoxPDFStub struct {
	clipRects [][4]float64
	clipEnds  int
	images    []objectImageCall
}

func (s *objectBoxPDFStub) ClipEnd() { s.clipEnds++ }

func (s *objectBoxPDFStub) ClipRect(x, y, w, h float64, _ bool) {
	s.clipRects = append(s.clipRects, [4]float64{x, y, w, h})
}

func (s *objectBoxPDFStub) Image(imageNameStr string, x, y, w, h float64, _ bool, _ string, _ int, _ string) {
	s.images = append(s.images, objectImageCall{imageNameStr, x, y, w, h})
}

func (s *objectBoxPDFStub) RegisterImageOptionsReader(_ string, _ gofpdf.ImageOptions, _ io.Reader) *gofpdf.ImageInfoType {
	return nil
}

// realImageInfo registers a small PNG against a real gofpdf instance, the only
// way to obtain a populated *ImageInfoType (its fields are unexported).
func realImageInfo(t *testing.T) *gofpdf.ImageInfoType {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	pdf := gofpdf.NewCustom(&gofpdf.InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	info := pdf.RegisterImageOptionsReader("object-box-test", gofpdf.ImageOptions{ImageType: "PNG"}, &buf)
	require.NotNil(t, info)
	return info
}

func TestImage_AddObjectImageToPdf(t *testing.T) {
	t.Parallel()

	t.Run("fill fit clips the object box and stretches the image into it", func(t *testing.T) {
		t.Parallel()
		stub := &objectBoxPDFStub{}
		sut := NewImage(stub, nil)
		cell := &entity.Cell{X: 0, Y: 0, Width: 40, Height: 40}
		margins := &entity.Margins{Left: 5, Top: 7}
		prop := &props.Rect{ObjectFit: "fill"}

		sut.addObjectImageToPdf("img", realImageInfo(t), cell, margins, prop, false)

		require.Len(t, stub.clipRects, 1)
		assert.Equal(t, [4]float64{5, 7, 40, 40}, stub.clipRects[0])
		require.Len(t, stub.images, 1)
		assert.Equal(t, objectImageCall{"img", 5, 7, 40, 40}, stub.images[0])
		assert.Equal(t, 1, stub.clipEnds)
	})

	t.Run("prop offsets shrink the object box", func(t *testing.T) {
		t.Parallel()
		stub := &objectBoxPDFStub{}
		sut := NewImage(stub, nil)
		cell := &entity.Cell{X: 0, Y: 0, Width: 40, Height: 40}
		margins := &entity.Margins{}
		prop := &props.Rect{ObjectFit: "fill", Left: 10, Top: 20}

		sut.addObjectImageToPdf("img", realImageInfo(t), cell, margins, prop, false)

		require.Len(t, stub.clipRects, 1)
		assert.Equal(t, [4]float64{10, 20, 30, 20}, stub.clipRects[0])
	})

	t.Run("non-positive object box draws nothing", func(t *testing.T) {
		t.Parallel()
		stub := &objectBoxPDFStub{}
		sut := NewImage(stub, nil)
		margins := &entity.Margins{}
		prop := &props.Rect{ObjectFit: "cover"}

		sut.addObjectImageToPdf("img", nil, &entity.Cell{Width: 0, Height: 10}, margins, prop, false)
		sut.addObjectImageToPdf("img", nil, &entity.Cell{Width: 10, Height: 0}, margins, prop, false)

		assert.Len(t, stub.clipRects, 0)
		assert.Len(t, stub.images, 0)
	})
}

func TestParseObjectPositionLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected float64
		ok       bool
	}{
		{"millimetres", "5mm", 5, true},
		{"centimetres", "2cm", 20, true},
		{"inches", "1in", 25.4, true},
		{"points", "10pt", 3.52778, true},
		{"pixels", "10px", 2.64583, true},
		{"bare number", "7", 7, true},
		{"empty", "", 0, false},
		{"invalid number with unit", "xmm", 0, false},
		{"garbage", "junk", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseObjectPositionLength(tt.value)
			assert.Equal(t, tt.ok, ok)
			assert.InDelta(t, tt.expected, got, 0.0001)
		})
	}
}

func TestObjectPositionOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		free     float64
		expected float64
	}{
		{"left is zero", "left", 10, 0},
		{"top is zero", "top", 10, 0},
		{"right is full", "right", 10, 10},
		{"bottom is full", "bottom", 10, 10},
		{"center is half", "center", 10, 5},
		{"percent of free space", "25%", 8, 2},
		{"invalid percent falls back to center", "bad%", 8, 4},
		{"length offset", "3mm", 8, 3},
		{"garbage falls back to center", "junk", 8, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.expected, objectPositionOffset(tt.value, tt.free), 0.0001)
		})
	}
}

func TestNormalizeObjectPositionTokens(t *testing.T) {
	t.Parallel()

	t.Run("vertical-first pair is swapped", func(t *testing.T) {
		t.Parallel()
		x, y := normalizeObjectPositionTokens("top", "left")
		assert.Equal(t, "left", x)
		assert.Equal(t, "top", y)
	})

	t.Run("both vertical tokens are kept", func(t *testing.T) {
		t.Parallel()
		x, y := normalizeObjectPositionTokens("top", "bottom")
		assert.Equal(t, "top", x)
		assert.Equal(t, "bottom", y)
	})

	t.Run("ordered pair is kept", func(t *testing.T) {
		t.Parallel()
		x, y := normalizeObjectPositionTokens("left", "top")
		assert.Equal(t, "left", x)
		assert.Equal(t, "top", y)
	})
}
