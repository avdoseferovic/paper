package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type perSideBorderStyler struct {
	stylerTemplate
}

// NewPerSideBorderStyler creates a CellWriter chain node that handles per-side borders.
// When any of the Cell's BorderXThickness fields are non-zero it draws raw gofpdf Line
// calls per side (with per-side color/thickness) and clears BorderType before passing
// through — so the downstream CellFormat call does not double-draw borders.
// When no per-side fields are set it passes through unchanged (zero cost for legacy callers).
func NewPerSideBorderStyler(fpdf any) CellWriter {
	return &perSideBorderStyler{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "perSideBorderStyler",
		},
	}
}

func (p *perSideBorderStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil || !prop.HasPerSideBorders() {
		p.GoToNext(width, height, config, prop)
		return
	}
	// When border-radius is set, defer the entire border render to borderRadiusStyler
	// so rounded corners and rectangular per-side lines don't fight each other.
	if prop.HasBorderRadius() {
		p.GoToNext(width, height, config, prop)
		return
	}

	// Save and defer-restore draw state so we don't leak colour/thickness.
	fpdf := asPDF[perSideBorderPDF](p.fpdf)
	origLineWidth := fpdf.GetLineWidth()
	origR, origG, origB := fpdf.GetDrawColor()
	defer func() {
		fpdf.SetLineWidth(origLineWidth)
		fpdf.SetDrawColor(origR, origG, origB)
	}()

	// Snapshot the pen position at the start of Apply — gofpdf places the pen at the
	// cell's top-left before any drawing calls, so GetXY gives the true cell origin.
	x, y := fpdf.GetXY()

	p.drawSide(fpdf, prop.BorderTopThickness, prop.BorderTopColor, prop.BorderTopStyle, prop, x, y, x+width, y)
	p.drawSide(fpdf, prop.BorderRightThickness, prop.BorderRightColor, prop.BorderRightStyle, prop, x+width, y, x+width, y+height)
	p.drawSide(fpdf, prop.BorderBottomThickness, prop.BorderBottomColor, prop.BorderBottomStyle, prop, x, y+height, x+width, y+height)
	p.drawSide(fpdf, prop.BorderLeftThickness, prop.BorderLeftColor, prop.BorderLeftStyle, prop, x, y, x, y+height)

	// Clear BorderType so downstream CellFormat doesn't also draw borders.
	modified := *prop
	modified.BorderType = border.None
	p.GoToNext(width, height, config, &modified)
}

func (p *perSideBorderStyler) drawSide(
	fpdf perSideBorderPDF,
	thickness float64,
	color *props.Color,
	style consts.LineStyle,
	prop *props.Cell,
	x1, y1, x2, y2 float64,
) {
	if thickness <= 0 {
		return
	}

	fpdf.SetLineWidth(thickness)

	if color != nil {
		fpdf.SetDrawColor(color.Red, color.Green, color.Blue)
	} else if prop.BorderColor != nil {
		fpdf.SetDrawColor(prop.BorderColor.Red, prop.BorderColor.Green, prop.BorderColor.Blue)
	}

	switch style {
	case consts.LineStyleSolid:
	case consts.LineStyleDashed:
		fpdf.SetDashPattern([]float64{1, 1}, 0)
	case consts.LineStyleDotted:
		fpdf.SetDashPattern([]float64{0.4, 0.4}, 0)
	}

	fpdf.Line(x1, y1, x2, y2)

	if style != consts.LineStyleSolid && style != "" {
		fpdf.SetDashPattern([]float64{1, 0}, 0)
	}
}
