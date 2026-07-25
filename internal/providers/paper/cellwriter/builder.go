package cellwriter

// WriterBuilder assembles the chain of cell writers in the right order.
type WriterBuilder struct{}

// NewBuilder creates a WriterBuilder.
func NewBuilder() *WriterBuilder {
	return &WriterBuilder{}
}

// Build constructs the cellwriter chain. When drawer is non-nil, a gradient
// styler is prepended to the chain so gradient backgrounds render first.
func (c *WriterBuilder) Build(fpdf any, drawer ...gradientDrawer) CellWriter {
	cellCreator := NewCellWriter(fpdf)
	borderColorStyle := NewBorderColorStyler(fpdf)
	borderLineStyler := NewBorderLineStyler(fpdf)
	borderThicknessStyler := NewBorderThicknessStyler(fpdf)
	fillColorStyler := NewFillColorStyler(fpdf)
	perSideBorder := NewPerSideBorderStyler(fpdf)
	borderRadius := NewBorderRadiusStyler(fpdf)
	backgroundImageStyler := NewBackgroundImageStyler(fpdf)

	// Chain order starts with shadow, then outline, then borders/fills/background
	// image, then cellWriter.
	// shadow: draws behind all decorations.
	// outline: draws outside the cell box before fills so it can act as a halo.
	outlineStyle := NewOutlineStyler(fpdf)
	shadowStyle := NewShadowStyler(fpdf)

	shadowStyle.SetNext(outlineStyle)
	outlineStyle.SetNext(perSideBorder)
	perSideBorder.SetNext(borderRadius)
	borderRadius.SetNext(borderThicknessStyler)
	borderThicknessStyler.SetNext(borderLineStyler)
	borderLineStyler.SetNext(borderColorStyle)
	borderColorStyle.SetNext(fillColorStyler)
	fillColorStyler.SetNext(backgroundImageStyler)
	backgroundImageStyler.SetNext(cellCreator)

	if len(drawer) > 0 && drawer[0] != nil {
		gradientStyle := NewGradientStyler(fpdf, drawer[0])
		gradientStyle.SetNext(shadowStyle)
		return gradientStyle
	}
	return shadowStyle
}
