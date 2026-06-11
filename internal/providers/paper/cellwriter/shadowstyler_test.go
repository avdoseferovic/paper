package cellwriter_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type shadowRect struct {
	x, y, w, h float64
	style      string
}

type shadowAlpha struct {
	alpha float64
	mode  string
}

// shadowPDFStub implements the shadowPDF interface used by the shadow styler,
// recording every call for behavior assertions.
type shadowPDFStub struct {
	x, y    float64
	rects   []shadowRect
	alphas  []shadowAlpha
	fills   [][3]int
	xyCalls [][2]float64
}

func (s *shadowPDFStub) GetXY() (float64, float64) { return s.x, s.y }

func (s *shadowPDFStub) Rect(x, y, w, h float64, styleStr string) {
	s.rects = append(s.rects, shadowRect{x, y, w, h, styleStr})
}

func (s *shadowPDFStub) SetAlpha(alpha float64, blendModeStr string) {
	s.alphas = append(s.alphas, shadowAlpha{alpha, blendModeStr})
}

func (s *shadowPDFStub) SetFillColor(r, g, b int) {
	s.fills = append(s.fills, [3]int{r, g, b})
}

func (s *shadowPDFStub) SetXY(x, y float64) {
	s.xyCalls = append(s.xyCalls, [2]float64{x, y})
}

func floatPtr(v float64) *float64 { return &v }

func TestNewShadowStyler(t *testing.T) {
	t.Parallel()
	// Act
	sut := cellwriter.NewShadowStyler(nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "shadowStyler", sut.GetName())
}

func TestShadowStyler_Apply(t *testing.T) {
	t.Parallel()

	t.Run("when prop is nil, should skip drawing and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		var nilProp *props.Cell

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(20.0, 10.0, cfg, nilProp).Once()

		sut := cellwriter.NewShadowStyler(nil)
		sut.SetNext(inner)

		// Act
		sut.Apply(20, 10, cfg, nilProp)
	})

	t.Run("when prop has no shadows, should skip drawing and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		prop := &props.Cell{}
		stub := &shadowPDFStub{}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(20.0, 10.0, cfg, prop).Once()

		sut := cellwriter.NewShadowStyler(stub)
		sut.SetNext(inner)

		// Act
		sut.Apply(20, 10, cfg, prop)

		// Assert
		assert.Len(t, stub.rects, 0)
	})

	t.Run("when shadow has no blur, should draw a single offset rect and restore state", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		prop := &props.Cell{
			BoxShadow: []props.Shadow{{
				OffsetX: 2,
				OffsetY: 3,
				Spread:  1,
				Color:   &props.Color{Red: 10, Green: 20, Blue: 30, Alpha: floatPtr(0.6)},
			}},
		}
		stub := &shadowPDFStub{x: 5, y: 7}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(20.0, 10.0, cfg, prop).Once()

		sut := cellwriter.NewShadowStyler(stub)
		sut.SetNext(inner)

		// Act
		sut.Apply(20, 10, cfg, prop)

		// Assert
		require.Len(t, stub.rects, 1)
		assert.Equal(t, shadowRect{6, 9, 22, 12, "F"}, stub.rects[0])
		require.Len(t, stub.alphas, 2)
		assert.Equal(t, shadowAlpha{0.6, "Normal"}, stub.alphas[0])
		assert.Equal(t, shadowAlpha{1, "Normal"}, stub.alphas[1])
		require.Len(t, stub.fills, 2)
		assert.Equal(t, [3]int{10, 20, 30}, stub.fills[0])
		assert.Equal(t, [3]int{255, 255, 255}, stub.fills[1])
		// Cursor restored to its original position before forwarding.
		require.Len(t, stub.xyCalls, 1)
		assert.Equal(t, [2]float64{5, 7}, stub.xyCalls[0])
	})

	t.Run("when shadow color is nil, should default to black with full alpha", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		prop := &props.Cell{BoxShadow: []props.Shadow{{OffsetX: 1, OffsetY: 1}}}
		stub := &shadowPDFStub{}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(20.0, 10.0, cfg, prop).Once()

		sut := cellwriter.NewShadowStyler(stub)
		sut.SetNext(inner)

		// Act
		sut.Apply(20, 10, cfg, prop)

		// Assert
		require.Len(t, stub.rects, 1)
		assert.Equal(t, [3]int{0, 0, 0}, stub.fills[0])
		assert.Equal(t, shadowAlpha{1, "Normal"}, stub.alphas[0])
	})

	t.Run("when shadow has blur, should draw three expanding rects with rising alpha", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		prop := &props.Cell{BoxShadow: []props.Shadow{{BlurRadius: 3}}}
		stub := &shadowPDFStub{x: 5, y: 7}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(20.0, 10.0, cfg, prop).Once()

		sut := cellwriter.NewShadowStyler(stub)
		sut.SetNext(inner)

		// Act
		sut.Apply(20, 10, cfg, prop)

		// Assert
		require.Len(t, stub.rects, 3)
		assert.Equal(t, shadowRect{2, 4, 26, 16, "F"}, stub.rects[0])
		assert.Equal(t, shadowRect{3, 5, 24, 14, "F"}, stub.rects[1])
		assert.Equal(t, shadowRect{4, 6, 22, 12, "F"}, stub.rects[2])
		require.Len(t, stub.alphas, 4)
		assert.InDelta(t, 0.3, stub.alphas[0].alpha, 0.001)
		assert.InDelta(t, 0.5, stub.alphas[1].alpha, 0.001)
		assert.InDelta(t, 0.8, stub.alphas[2].alpha, 0.001)
		assert.InDelta(t, 1.0, stub.alphas[3].alpha, 0.001)
		// Fill colour reset to white after the blur loop.
		assert.Equal(t, [3]int{255, 255, 255}, stub.fills[len(stub.fills)-1])
	})

	t.Run("when shadow is inset, should draw inside the cell with inverted offset", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		prop := &props.Cell{
			BoxShadow: []props.Shadow{{
				OffsetX: 2,
				OffsetY: 3,
				Inset:   true,
				Color:   &props.Color{Red: 1, Green: 2, Blue: 3},
			}},
		}
		stub := &shadowPDFStub{x: 5, y: 7}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(20.0, 10.0, cfg, prop).Once()

		sut := cellwriter.NewShadowStyler(stub)
		sut.SetNext(inner)

		// Act
		sut.Apply(20, 10, cfg, prop)

		// Assert
		require.Len(t, stub.rects, 1)
		assert.Equal(t, shadowRect{3, 4, 20, 10, "F"}, stub.rects[0])
		require.Len(t, stub.alphas, 2)
		assert.InDelta(t, 0.3, stub.alphas[0].alpha, 0.001) // default inset alpha
		assert.InDelta(t, 1.0, stub.alphas[1].alpha, 0.001)
		require.Len(t, stub.fills, 1)
		assert.Equal(t, [3]int{1, 2, 3}, stub.fills[0])
	})

	t.Run("when inset shadow has explicit alpha, should use it", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		prop := &props.Cell{
			BoxShadow: []props.Shadow{{
				Inset: true,
				Color: &props.Color{Alpha: floatPtr(0.9)},
			}},
		}
		stub := &shadowPDFStub{}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(20.0, 10.0, cfg, prop).Once()

		sut := cellwriter.NewShadowStyler(stub)
		sut.SetNext(inner)

		// Act
		sut.Apply(20, 10, cfg, prop)

		// Assert
		require.Len(t, stub.alphas, 2)
		assert.InDelta(t, 0.9, stub.alphas[0].alpha, 0.001)
	})
}
