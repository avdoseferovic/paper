package cellwriter

import (
	"github.com/johnfercher/maroto/v2/internal/providers/gofpdf/gofpdfwrapper"
)

type WriterBuilder struct{}

func NewBuilder() *WriterBuilder {
	return &WriterBuilder{}
}

func (c *WriterBuilder) Build(fpdf gofpdfwrapper.Fpdf) CellWriter {
	cellCreator := NewCellWriter(fpdf)
	borderColorStyle := NewBorderColorStyler(fpdf)
	borderLineStyler := NewBorderLineStyler(fpdf)
	borderThicknessStyler := NewBorderThicknessStyler(fpdf)
	fillColorStyler := NewFillColorStyler(fpdf)
	perSideBorder := NewPerSideBorderStyler(fpdf)

	// perSideBorder is first: intercepts when per-side fields are set, passes through otherwise.
	perSideBorder.SetNext(borderThicknessStyler)
	borderThicknessStyler.SetNext(borderLineStyler)
	borderLineStyler.SetNext(borderColorStyle)
	borderColorStyle.SetNext(fillColorStyler)
	fillColorStyler.SetNext(cellCreator)

	return perSideBorder
}
