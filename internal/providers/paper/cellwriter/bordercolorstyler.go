package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type BorderColorStyler struct {
	stylerTemplate
	defaultColor *props.Color
}

func NewBorderColorStyler(fpdf any) *BorderColorStyler {
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

	applyColorStyler(&b.stylerTemplate, prop.BorderColor, b.defaultColor, setDrawColor, width, height, config, prop)
}
