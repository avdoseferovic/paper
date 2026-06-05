package cellwriter

import (
	"github.com/johnfercher/paper/v2/internal/providers/paper/gofpdfwrapper"
)

type WriterBuilder struct{}

func NewBuilder() *WriterBuilder {
	return &WriterBuilder{}
}

// Build constructs the cellwriter chain. When drawer is non-nil, a gradient
// styler is prepended to the chain so gradient backgrounds render first.
func (c *WriterBuilder) Build(fpdf gofpdfwrapper.Fpdf, drawer ...gradientDrawer) CellWriter {
	cellCreator := NewCellWriter(fpdf)
	borderColorStyle := NewBorderColorStyler(fpdf)
	borderLineStyler := NewBorderLineStyler(fpdf)
	borderThicknessStyler := NewBorderThicknessStyler(fpdf)
	fillColorStyler := NewFillColorStyler(fpdf)
	perSideBorder := NewPerSideBorderStyler(fpdf)
	borderRadius := NewBorderRadiusStyler(fpdf)

	// Chain order (first applied → last):
	//   shadow → perSideBorder → borderRadius → borderThickness → borderLine → borderColor → fillColor → outline → cellWriter
	// shadow: draws behind all decorations.
	// outline: LAST before cellWriter — draws outside the cell box after all fills.
	outlineStyle := NewOutlineStyler(fpdf)
	shadowStyle := NewShadowStyler(fpdf)

	shadowStyle.SetNext(perSideBorder)
	perSideBorder.SetNext(borderRadius)
	borderRadius.SetNext(borderThicknessStyler)
	borderThicknessStyler.SetNext(borderLineStyler)
	borderLineStyler.SetNext(borderColorStyle)
	borderColorStyle.SetNext(fillColorStyler)
	fillColorStyler.SetNext(outlineStyle)
	outlineStyle.SetNext(cellCreator)

	if len(drawer) > 0 && drawer[0] != nil {
		gradientStyle := NewGradientStyler(fpdf, drawer[0])
		gradientStyle.SetNext(shadowStyle)
		return gradientStyle
	}
	return shadowStyle
}
