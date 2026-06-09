// nolint: dupl
package cellwriter

import (
	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type BorderColorStyler struct {
	stylerTemplate
	defaultColor *props.Color
}

func NewBorderColorStyler(fpdf gofpdfwrapper.PDF) *BorderColorStyler {
	defaultColor := props.Black()
	return &BorderColorStyler{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "borderColorStyler",
		},
		defaultColor: &defaultColor,
	}
}

func (b *BorderColorStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil {
		b.GoToNext(width, height, config, prop)
		return
	}

	if prop.BorderColor == nil {
		b.GoToNext(width, height, config, prop)
		return
	}

	b.fpdf.SetDrawColor(prop.BorderColor.Red, prop.BorderColor.Green, prop.BorderColor.Blue)
	if a := prop.BorderColor.Alpha; a != nil && *a < 1 {
		b.fpdf.SetAlpha(clampAlpha(*a), "Normal")
		defer b.fpdf.SetAlpha(1, "Normal")
	}
	b.GoToNext(width, height, config, prop)
	b.fpdf.SetDrawColor(b.defaultColor.Red, b.defaultColor.Green, b.defaultColor.Blue)
}
