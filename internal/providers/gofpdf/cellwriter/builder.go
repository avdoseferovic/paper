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
	borderRadius := NewBorderRadiusStyler(fpdf)

	// Chain order:
	//   perSideBorder → borderRadius → borderThickness → borderLine → borderColor → fillColor → cellWriter
	// perSideBorder runs first; it skips itself when BorderRadius is set so that
	// borderRadius owns the entire rounded fill + stroke without overlap.
	perSideBorder.SetNext(borderRadius)
	borderRadius.SetNext(borderThicknessStyler)
	borderThicknessStyler.SetNext(borderLineStyler)
	borderLineStyler.SetNext(borderColorStyle)
	borderColorStyle.SetNext(fillColorStyler)
	fillColorStyler.SetNext(cellCreator)

	return perSideBorder
}
