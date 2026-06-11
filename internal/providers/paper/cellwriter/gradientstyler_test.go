package cellwriter_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// gradientPDFStub implements the gradientStylerPDF interface used by the
// gradient styler (GetXY + GetMargins).
type gradientPDFStub struct {
	x, y      float64
	left, top float64
}

func (g *gradientPDFStub) GetXY() (float64, float64) { return g.x, g.y }

func (g *gradientPDFStub) GetMargins() (float64, float64, float64, float64) {
	return g.left, g.top, 0, 0
}

// gradientDrawerRecorder records DrawGradient calls for behavior assertions.
type gradientDrawerRecorder struct {
	cells     []entity.Cell
	gradients []*props.Gradient
	widths    []float64
	heights   []float64
}

func (d *gradientDrawerRecorder) DrawGradient(cell *entity.Cell, g *props.Gradient, widthMM, heightMM float64) {
	d.cells = append(d.cells, *cell)
	d.gradients = append(d.gradients, g)
	d.widths = append(d.widths, widthMM)
	d.heights = append(d.heights, heightMM)
}

func TestNewGradientStyler(t *testing.T) {
	t.Parallel()
	// Act
	sut := cellwriter.NewGradientStyler(nil, nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "gradientStyler", sut.GetName())
}

func TestGradientStyler_Apply(t *testing.T) {
	t.Parallel()

	t.Run("when prop is nil, should skip drawing and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		width := 100.0
		height := 50.0
		cfg := &entity.Config{}
		var nilProp *props.Cell

		drawer := &gradientDrawerRecorder{}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(width, height, cfg, nilProp).Once()

		sut := cellwriter.NewGradientStyler(nil, drawer)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, nilProp)

		// Assert
		assert.Len(t, drawer.cells, 0)
	})

	t.Run("when prop has no gradient, should skip drawing and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		width := 100.0
		height := 50.0
		cfg := &entity.Config{}
		prop := &props.Cell{}

		drawer := &gradientDrawerRecorder{}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(width, height, cfg, prop).Once()

		sut := cellwriter.NewGradientStyler(&gradientPDFStub{}, drawer)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, prop)

		// Assert
		assert.Len(t, drawer.cells, 0)
	})

	t.Run("when prop has gradient, should draw margin-relative cell and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		width := 40.0
		height := 20.0
		cfg := &entity.Config{}
		gradient := &props.Gradient{
			Kind: props.GradientLinear,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 255}, Position: 0},
				{Color: props.Color{Blue: 255}, Position: 1},
			},
		}
		prop := &props.Cell{BackgroundGradient: gradient}

		fpdf := &gradientPDFStub{x: 12, y: 30, left: 10, top: 15}
		drawer := &gradientDrawerRecorder{}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(width, height, cfg, prop).Once()

		sut := cellwriter.NewGradientStyler(fpdf, drawer)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, prop)

		// Assert
		assert.Len(t, drawer.cells, 1)
		assert.InDelta(t, 2.0, drawer.cells[0].X, 0.001)  // x - left
		assert.InDelta(t, 15.0, drawer.cells[0].Y, 0.001) // y - top
		assert.InDelta(t, width, drawer.cells[0].Width, 0.001)
		assert.InDelta(t, height, drawer.cells[0].Height, 0.001)
		assert.Equal(t, gradient, drawer.gradients[0])
		assert.InDelta(t, width, drawer.widths[0], 0.001)
		assert.InDelta(t, height, drawer.heights[0], 0.001)
	})
}
