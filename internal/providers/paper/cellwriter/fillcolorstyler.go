package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// FillColorStyler sets the cell's background fill color before passing the cell to the next writer.
type FillColorStyler struct {
	stylerTemplate
	defaultFillColor *props.Color
}

// NewFillColorStyler creates a FillColorStyler that draws on the given PDF writer.
func NewFillColorStyler(fpdf any) *FillColorStyler {
	defaultFillColor := props.White()
	return &FillColorStyler{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "fillColorStyler",
		},
		defaultFillColor: &defaultFillColor,
	}
}

// Apply sets the cell's background fill color from prop, then continues down the chain.
func (f *FillColorStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil {
		f.GoToNext(width, height, config, prop)
		return
	}

	applyColorStyler(&f.stylerTemplate, prop.BackgroundColor, f.defaultFillColor, setFillColor, width, height, config, prop)
}

// clampAlpha clamps an alpha value to [0, 1].
func clampAlpha(a float64) float64 {
	return min(max(a, 0), 1)
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
