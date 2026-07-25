package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// BorderColorStyler sets the border color before passing the cell to the next writer.
type BorderColorStyler struct {
	stylerTemplate
	defaultColor *props.Color
}

// NewBorderColorStyler creates a BorderColorStyler that draws on the given PDF writer.
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

// Apply sets the border color from prop, then continues down the chain.
func (b *BorderColorStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil {
		b.GoToNext(width, height, config, prop)
		return
	}

	applyColorStyler(&b.stylerTemplate, prop.BorderColor, b.defaultColor, setDrawColor, width, height, config, prop)
}
