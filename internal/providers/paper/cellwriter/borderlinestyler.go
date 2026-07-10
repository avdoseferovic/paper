package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type BorderLineStyler struct {
	stylerTemplate
}

func NewBorderLineStyler(fpdf any) *BorderLineStyler {
	return &BorderLineStyler{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "borderLineStyler",
		},
	}
}

func (b *BorderLineStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil {
		b.GoToNext(width, height, config, prop)
		return
	}

	if prop.LineStyle == consts.LineStyleSolid || prop.LineStyle == "" {
		b.GoToNext(width, height, config, prop)
		return
	}

	fpdf := asPDF[dashPatternPDF](b.fpdf)
	// Dotted uses {0.4,0.4} and dashed {1,1}, matching line.go,
	// persideborder.go and outlinestyler.go.
	if prop.LineStyle == consts.LineStyleDotted {
		fpdf.SetDashPattern([]float64{0.4, 0.4}, 0)
	} else {
		fpdf.SetDashPattern([]float64{1, 1}, 0)
	}
	b.GoToNext(width, height, config, prop)
	fpdf.SetDashPattern([]float64{1, 0}, 0)
}
