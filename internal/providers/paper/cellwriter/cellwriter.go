package cellwriter

import (
	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type CellWriter interface {
	SetNext(next CellWriter)
	GetNext() CellWriter
	GetName() string
	Apply(width, height float64, config *entity.Config, prop *props.Cell)
}

type cellWriter struct {
	stylerTemplate
	defaultColor *props.Color
}

func NewCellWriter(fpdf gofpdfwrapper.Fpdf) CellWriter {
	defaultColor := props.Black()
	return &cellWriter{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "cellWriter",
		},
		defaultColor: &defaultColor,
	}
}

func (c *cellWriter) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil {
		bd := border.None
		if config.Debug {
			bd = border.Full
		}

		c.fpdf.CellFormat(width, height, "", bd.String(), 0, "C", false, 0, "")
		return
	}

	bd := prop.BorderType
	if config.Debug {
		bd = border.Full
	}

	c.fpdf.CellFormat(width, height, "", bd.String(), 0, "C", prop.BackgroundColor != nil, 0, "")
}
