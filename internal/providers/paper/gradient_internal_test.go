package paper

import (
	"image/color"
	"testing"

	"github.com/avdoseferovic/paper/pkg/props"
)

func TestRasteriseLinearGradient_UsesFullStopRangeForHorizontalGradient(t *testing.T) {
	t.Parallel()

	g := &props.Gradient{
		Kind:     props.GradientLinear,
		AngleDeg: 90,
		Stops: []props.GradientStop{
			{Color: props.Color{Red: 255, Green: 0, Blue: 0}, Position: 0},
			{Color: props.Color{Red: 0, Green: 0, Blue: 255}, Position: 1},
		},
	}
	img := rasteriseGradient(g, 8, 1)

	left := color.RGBAModel.Convert(img.At(0, 0)).(color.RGBA)
	right := color.RGBAModel.Convert(img.At(7, 0)).(color.RGBA)
	if left.R <= 200 || left.B >= 60 {
		t.Fatalf("left edge = %#v, want close to first stop red", left)
	}
	if right.B <= 200 || right.R >= 60 {
		t.Fatalf("right edge = %#v, want close to final stop blue", right)
	}
}

func TestRasteriseLinearGradient_UsesFullStopRangeForVerticalGradient(t *testing.T) {
	t.Parallel()

	g := &props.Gradient{
		Kind:     props.GradientLinear,
		AngleDeg: 180,
		Stops: []props.GradientStop{
			{Color: props.Color{Red: 255, Green: 255, Blue: 255}, Position: 0},
			{Color: props.Color{Red: 0, Green: 0, Blue: 0}, Position: 1},
		},
	}
	img := rasteriseGradient(g, 1, 8)

	top := color.RGBAModel.Convert(img.At(0, 0)).(color.RGBA)
	bottom := color.RGBAModel.Convert(img.At(0, 7)).(color.RGBA)
	if top.R <= 200 || top.G <= 200 || top.B <= 200 {
		t.Fatalf("top edge = %#v, want close to first stop white", top)
	}
	if bottom.R >= 60 || bottom.G >= 60 || bottom.B >= 60 {
		t.Fatalf("bottom edge = %#v, want close to final stop black", bottom)
	}
}
