package cellwriter

import (
	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/consts/linestyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type BorderLineStyler struct {
	stylerTemplate
}

func NewBorderLineStyler(fpdf gofpdfwrapper.Fpdf) *BorderLineStyler {
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

	if prop.LineStyle == linestyle.Solid || prop.LineStyle == "" {
		b.GoToNext(width, height, config, prop)
		return
	}

	b.fpdf.SetDashPattern([]float64{1, 1}, 0)
	b.GoToNext(width, height, config, prop)
	b.fpdf.SetDashPattern([]float64{1, 0}, 0)
}
