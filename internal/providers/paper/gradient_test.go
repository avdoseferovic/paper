package paper_test

import (
	"testing"

	mock "github.com/avdoseferovic/paper/internal/mocktest"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestGradientRenderer_DrawGradient(t *testing.T) {
	t.Parallel()

	t.Run("linear gradient calls RegisterImageOptionsReader once and Image once", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		pdf.EXPECT().GetMargins().Return(15.0, 10.0, 15.0, 10.0).Maybe()
		pdf.EXPECT().RegisterImageOptionsReader(
			mock.AnythingOfType("string"),
			mock.Anything,
			mock.Anything,
		).Return(nil).Once()
		pdf.EXPECT().Image(
			mock.AnythingOfType("string"),
			mock.AnythingOfType("float64"), // x + leftMargin
			mock.AnythingOfType("float64"), // y + topMargin
			mock.AnythingOfType("float64"), // width
			mock.AnythingOfType("float64"), // height
			false,
			"PNG",
			0,
			"",
		).Once()

		g := &props.Gradient{
			Kind:     props.GradientLinear,
			AngleDeg: 90,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 255, Green: 0, Blue: 0}, Position: 0},
				{Color: props.Color{Red: 0, Green: 0, Blue: 255}, Position: 1},
			},
		}
		cell := &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}
		gr := gofpdf.NewGradientRenderer(pdf)
		gr.DrawGradient(cell, g, 50, 20)
	})

	t.Run("identical gradient on second call reuses imgName (no second RegisterImageOptionsReader)", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		// RegisterImageOptionsReader called exactly ONCE even though DrawGradient is called twice
		pdf.EXPECT().RegisterImageOptionsReader(mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil).Once()
		pdf.EXPECT().Image(mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything, false, "PNG", 0, "").Times(2)

		g := &props.Gradient{
			Kind:     props.GradientLinear,
			AngleDeg: 45,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 0, Green: 0, Blue: 0}, Position: 0},
				{Color: props.Color{Red: 255, Green: 255, Blue: 255}, Position: 1},
			},
		}
		cell := &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}
		gr := gofpdf.NewGradientRenderer(pdf)
		gr.DrawGradient(cell, g, 50, 20)
		gr.DrawGradient(cell, g, 50, 20)
	})

	t.Run("gradient Image x/y includes margin offsets", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		pdf.EXPECT().GetMargins().Return(15.0, 10.0, 15.0, 10.0).Maybe()
		pdf.EXPECT().RegisterImageOptionsReader(mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil).Maybe()
		// x should be cell.X + leftMargin = 5 + 15 = 20
		// y should be cell.Y + topMargin  = 3 + 10 = 13
		pdf.EXPECT().Image(mock.AnythingOfType("string"), 20.0, 13.0, 50.0, 20.0, false, "PNG", 0, "").Once()

		g := &props.Gradient{
			Kind: props.GradientLinear, AngleDeg: 90,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 255, Green: 0, Blue: 0}, Position: 0},
				{Color: props.Color{Red: 0, Green: 0, Blue: 255}, Position: 1},
			},
		}
		cell := &entity.Cell{X: 5, Y: 3, Width: 50, Height: 20}
		gr := gofpdf.NewGradientRenderer(pdf)
		gr.DrawGradient(cell, g, 50, 20)
	})
}
