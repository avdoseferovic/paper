// Package cellwriter draws the background and border of a single cell.
//
// Each visual concern (fill color, border color, thickness, radius, shadow,
// gradient, background image) is a small CellWriter, and they are chained so
// every writer applies its own styling and then passes the cell along.
package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// CellWriter is one link in the cell-drawing chain. Apply draws this writer's
// part of the cell, then hands off to the next writer.
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

// NewCellWriter creates the last writer in the chain, which draws the cell box
// itself once the stylers before it have set up the colors and borders.
func NewCellWriter(fpdf any) CellWriter {
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
	fpdf := asPDF[cellFormatPDF](c.fpdf)
	if prop == nil {
		bd := border.None
		if config.Debug {
			bd = border.Full
		}

		fpdf.CellFormat(width, height, "", bd.String(), 0, "C", false, 0, "")
		return
	}

	bd := prop.BorderType
	if config.Debug {
		bd = border.Full
	}

	fpdf.CellFormat(width, height, "", bd.String(), 0, "C", prop.BackgroundColor != nil, 0, "")
}
