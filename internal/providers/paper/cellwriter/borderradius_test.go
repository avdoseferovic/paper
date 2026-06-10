package cellwriter_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	mock "github.com/avdoseferovic/paper/internal/mocktest"
	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestNewBorderRadiusStyler(t *testing.T) {
	t.Parallel()
	sut := cellwriter.NewBorderRadiusStyler(newPDF(t))
	assert.NotNil(t, sut)
}

func TestBorderRadiusStyler_Apply(t *testing.T) {
	t.Parallel()
	const w, h = 100.0, 50.0
	config := &entity.Config{}

	t.Run("when no border radius set, should pass through", func(t *testing.T) {
		t.Parallel()
		next := mocks.NewCellWriter(t)
		next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell"))

		fpdf := newPDF(t)
		sut := cellwriter.NewBorderRadiusStyler(fpdf)
		sut.SetNext(next)

		sut.Apply(w, h, config, &props.Cell{})
	})

	t.Run("when nil prop, should pass through", func(t *testing.T) {
		t.Parallel()
		next := mocks.NewCellWriter(t)
		next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell"))

		fpdf := newPDF(t)
		sut := cellwriter.NewBorderRadiusStyler(fpdf)
		sut.SetNext(next)

		sut.Apply(w, h, config, nil)
	})

	t.Run("when fill only, should draw rounded fill path and clear background", func(t *testing.T) {
		t.Parallel()
		next := mocks.NewCellWriter(t)
		var captured *props.Cell
		next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell")).
			Run(func(_w, _h float64, _c *entity.Config, p *props.Cell) {
				captured = p
			})

		fpdf := newPDF(t)
		fpdf.EXPECT().GetLineWidth().Return(0.2)
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0)
		fpdf.EXPECT().GetFillColor().Return(0, 0, 0)
		fpdf.EXPECT().GetXY().Return(10.0, 20.0)
		fpdf.EXPECT().SetFillColor(255, 100, 50)
		fpdf.EXPECT().MoveTo(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
		fpdf.EXPECT().LineTo(mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).Times(4)
		fpdf.EXPECT().CurveBezierCubicTo(
			mock.AnythingOfType("float64"), mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"), mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"), mock.AnythingOfType("float64"),
		).Times(4)
		fpdf.EXPECT().ClosePath()
		fpdf.EXPECT().DrawPath("F")
		// Restore
		fpdf.EXPECT().SetLineWidth(0.2)
		fpdf.EXPECT().SetDrawColor(0, 0, 0)
		fpdf.EXPECT().SetFillColor(0, 0, 0)

		sut := cellwriter.NewBorderRadiusStyler(fpdf)
		sut.SetNext(next)

		sut.Apply(w, h, config, &props.Cell{
			BorderRadius:    4,
			BackgroundColor: &props.Color{Red: 255, Green: 100, Blue: 50},
		})

		assert.NotNil(t, captured)
		assert.Nil(t, captured.BackgroundColor, "downstream prop should have BackgroundColor cleared")
	})

	t.Run("when fill + stroke, should DrawPath DF and average mixed per-side thicknesses", func(t *testing.T) {
		t.Parallel()
		next := mocks.NewCellWriter(t)
		next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell"))

		fpdf := newPDF(t)
		fpdf.EXPECT().GetLineWidth().Return(0.2)
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0)
		fpdf.EXPECT().GetFillColor().Return(0, 0, 0)
		fpdf.EXPECT().GetXY().Return(0.0, 0.0)
		fpdf.EXPECT().SetFillColor(1, 2, 3)
		// Mixed: top=2pt, bottom=0.5pt → average = (2+0.5)/2 = 1.25
		fpdf.EXPECT().SetLineWidth(1.25)
		fpdf.EXPECT().SetDrawColor(10, 20, 30)
		fpdf.EXPECT().MoveTo(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
		fpdf.EXPECT().LineTo(mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).Times(4)
		fpdf.EXPECT().CurveBezierCubicTo(
			mock.AnythingOfType("float64"), mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"), mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"), mock.AnythingOfType("float64"),
		).Times(4)
		fpdf.EXPECT().ClosePath()
		fpdf.EXPECT().DrawPath("DF")
		// Restore
		fpdf.EXPECT().SetLineWidth(0.2)
		fpdf.EXPECT().SetDrawColor(0, 0, 0)
		fpdf.EXPECT().SetFillColor(0, 0, 0)

		sut := cellwriter.NewBorderRadiusStyler(fpdf)
		sut.SetNext(next)

		sut.Apply(w, h, config, &props.Cell{
			BorderRadius:          4,
			BackgroundColor:       &props.Color{Red: 1, Green: 2, Blue: 3},
			BorderTopThickness:    2.0,
			BorderBottomThickness: 0.5,
			BorderColor:           &props.Color{Red: 10, Green: 20, Blue: 30},
		})
	})
}

// Per-side border styler should skip itself when border-radius is active.
func TestPerSideBorderStyler_SkipsWhenBorderRadiusSet(t *testing.T) {
	t.Parallel()
	const w, h = 100.0, 50.0
	config := &entity.Config{}

	next := mocks.NewCellWriter(t)
	next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell"))

	fpdf := newPDF(t)
	// No GetXY, no Line, no SetDrawColor — perSideBorder skips entirely.

	sut := cellwriter.NewPerSideBorderStyler(fpdf)
	sut.SetNext(next)

	sut.Apply(w, h, config, &props.Cell{
		BorderRadius:       4,
		BorderTopThickness: 1.0,
	})
}
