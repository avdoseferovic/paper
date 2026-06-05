// nolint: dupl
package cellwriter

import (
	"github.com/avdoseferovic/paper/v2/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

type FillColorStyler struct {
	stylerTemplate
	defaultFillColor *props.Color
}

func NewFillColorStyler(fpdf gofpdfwrapper.Fpdf) *FillColorStyler {
	defaultFillColor := props.White()
	return &FillColorStyler{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "fillColorStyler",
		},
		defaultFillColor: &defaultFillColor,
	}
}

func (f *FillColorStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil {
		f.GoToNext(width, height, config, prop)
		return
	}

	if prop.BackgroundColor == nil {
		f.GoToNext(width, height, config, prop)
		return
	}

	f.fpdf.SetFillColor(prop.BackgroundColor.Red, prop.BackgroundColor.Green, prop.BackgroundColor.Blue)
	if a := prop.BackgroundColor.Alpha; a != nil && *a < 1 {
		// Wrap fill drawing in SetAlpha; always restore to 1.0 via defer so
		// alpha cannot leak into the chain's downstream nodes or native rows.
		f.fpdf.SetAlpha(clampAlpha(*a), "Normal")
		defer f.fpdf.SetAlpha(1, "Normal")
	}
	f.GoToNext(width, height, config, prop)
	f.fpdf.SetFillColor(f.defaultFillColor.Red, f.defaultFillColor.Green, f.defaultFillColor.Blue)
}

// clampAlpha clamps an alpha value to [0, 1].
func clampAlpha(a float64) float64 {
	if a < 0 {
		return 0
	}
	if a > 1 {
		return 1
	}
	return a
}

// effectiveAlpha returns the minimum of fill and border color alphas, or 1
// when neither is set. Used by render nodes that paint both fill and stroke
// in a single primitive (e.g. borderRadius.DrawPath).
func effectiveAlpha(prop *props.Cell) float64 {
	if prop == nil {
		return 1
	}
	a := 1.0
	if c := prop.BackgroundColor; c != nil && c.Alpha != nil {
		a = clampAlpha(*c.Alpha)
	}
	if c := prop.BorderColor; c != nil && c.Alpha != nil {
		if v := clampAlpha(*c.Alpha); v < a {
			a = v
		}
	}
	return a
}
