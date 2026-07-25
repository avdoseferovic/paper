package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// BorderThicknessStyler sets the border line width before passing the cell to the next writer.
type BorderThicknessStyler struct {
	stylerTemplate
	defaultLineThickness float64
}

// NewBorderThicknessStyler creates a BorderThicknessStyler that draws on the given PDF writer.
func NewBorderThicknessStyler(fpdf any) *BorderThicknessStyler {
	return &BorderThicknessStyler{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "borderThicknessStyler",
		},
		defaultLineThickness: consts.DefaultLineThickness,
	}
}

// Apply sets the border line width from prop, then continues down the chain.
func (b *BorderThicknessStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil {
		b.GoToNext(width, height, config, prop)
		return
	}

	if prop.BorderThickness == 0 {
		b.GoToNext(width, height, config, prop)
		return
	}

	fpdf := asPDF[lineWidthPDF](b.fpdf)
	fpdf.SetLineWidth(prop.BorderThickness)
	b.GoToNext(width, height, config, prop)
	fpdf.SetLineWidth(b.defaultLineThickness)
}
