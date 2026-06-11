package paper_test

import (
	"testing"

	mock "github.com/avdoseferovic/paper/internal/mocktest"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestGradientRenderer_DrawGradient_Kinds(t *testing.T) {
	t.Parallel()

	stops := []props.GradientStop{
		{Color: props.Color{Red: 300, Green: -5, Blue: 0}, Position: 0},
		{Color: props.Color{Red: 128, Green: 128, Blue: 128}, Position: 0.5},
		{Color: props.Color{Red: 0, Green: 0, Blue: 255}, Position: 1},
	}

	rasterise := func(t *testing.T, g *props.Gradient) {
		t.Helper()
		pdf := newPDF(t)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		pdf.EXPECT().RegisterImageOptionsReader(mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil).Once()
		pdf.EXPECT().Image(mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything, false, "PNG", 0, "").Once()

		gr := gofpdf.NewGradientRenderer(pdf)
		gr.DrawGradient(&entity.Cell{X: 0, Y: 0, Width: 4, Height: 4}, g, 4, 4)
	}

	t.Run("radial gradient with centred origin", func(t *testing.T) {
		t.Parallel()
		rasterise(t, &props.Gradient{Kind: props.GradientRadial, CX: 0.5, CY: 0.5, Stops: stops})
	})

	t.Run("radial gradient with zero-centre falls back to half radius", func(t *testing.T) {
		t.Parallel()
		rasterise(t, &props.Gradient{Kind: props.GradientRadial, CX: 0, CY: 0, Stops: stops})
	})

	t.Run("conic gradient with starting angle", func(t *testing.T) {
		t.Parallel()
		rasterise(t, &props.Gradient{Kind: props.GradientConic, CX: 0.5, CY: 0.5, AngleDeg: 90, Stops: stops})
	})

	t.Run("duplicate stop positions do not divide by zero", func(t *testing.T) {
		t.Parallel()
		rasterise(t, &props.Gradient{
			Kind: props.GradientLinear,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 10}, Position: 0},
				{Color: props.Color{Red: 20}, Position: 0.5},
				{Color: props.Color{Red: 30}, Position: 0.5},
				{Color: props.Color{Red: 40}, Position: 1},
			},
		})
	})
}

func TestGradientRenderer_DrawGradient_Guards(t *testing.T) {
	t.Parallel()

	stops := []props.GradientStop{
		{Color: props.Color{Red: 255}, Position: 0},
		{Color: props.Color{Blue: 255}, Position: 1},
	}
	cell := &entity.Cell{Width: 10, Height: 10}

	t.Run("nil gradient draws nothing", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		gofpdf.NewGradientRenderer(pdf).DrawGradient(cell, nil, 10, 10)
		pdf.AssertNotCalled(t, "Image")
	})

	t.Run("fewer than two stops draws nothing", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		g := &props.Gradient{Kind: props.GradientLinear, Stops: stops[:1]}
		gofpdf.NewGradientRenderer(pdf).DrawGradient(cell, g, 10, 10)
		pdf.AssertNotCalled(t, "Image")
	})

	t.Run("non-positive dimensions draw nothing", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		g := &props.Gradient{Kind: props.GradientLinear, Stops: stops}
		gofpdf.NewGradientRenderer(pdf).DrawGradient(cell, g, 0, 10)
		gofpdf.NewGradientRenderer(pdf).DrawGradient(cell, g, 10, -1)
		pdf.AssertNotCalled(t, "Image")
	})
}
