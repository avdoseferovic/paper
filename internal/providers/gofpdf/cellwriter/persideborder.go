package cellwriter

import (
	"github.com/johnfercher/maroto/v2/internal/providers/gofpdf/gofpdfwrapper"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type perSideBorderStyler struct {
	stylerTemplate
}

// NewPerSideBorderStyler creates a CellWriter chain node that handles per-side borders.
// When any of the Cell's BorderXThickness fields are non-zero it draws raw gofpdf Line
// calls per side (with per-side color/thickness) and clears BorderType before passing
// through — so the downstream CellFormat call does not double-draw borders.
// When no per-side fields are set it passes through unchanged (zero cost for legacy callers).
func NewPerSideBorderStyler(fpdf gofpdfwrapper.Fpdf) CellWriter {
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

	// Save and defer-restore draw state so we don't leak colour/thickness.
	origLineWidth := p.fpdf.GetLineWidth()
	origR, origG, origB := p.fpdf.GetDrawColor()
	defer func() {
		p.fpdf.SetLineWidth(origLineWidth)
		p.fpdf.SetDrawColor(origR, origG, origB)
	}()

	// Snapshot the pen position at the start of Apply — gofpdf places the pen at the
	// cell's top-left before any drawing calls, so GetXY gives the true cell origin.
	x, y := p.fpdf.GetXY()

	p.drawSide(prop.BorderTopThickness, prop.BorderTopColor, prop,
		x, y, x+width, y)
	p.drawSide(prop.BorderRightThickness, prop.BorderRightColor, prop,
		x+width, y, x+width, y+height)
	p.drawSide(prop.BorderBottomThickness, prop.BorderBottomColor, prop,
		x, y+height, x+width, y+height)
	p.drawSide(prop.BorderLeftThickness, prop.BorderLeftColor, prop,
		x, y, x, y+height)

	// Clear BorderType so downstream CellFormat doesn't also draw borders.
	modified := *prop
	modified.BorderType = border.None
	p.GoToNext(width, height, config, &modified)
}

func (p *perSideBorderStyler) drawSide(thickness float64, color *props.Color, prop *props.Cell, x1, y1, x2, y2 float64) {
	if thickness <= 0 {
		return
	}

	p.fpdf.SetLineWidth(thickness)

	if color != nil {
		p.fpdf.SetDrawColor(color.Red, color.Green, color.Blue)
	} else if prop.BorderColor != nil {
		p.fpdf.SetDrawColor(prop.BorderColor.Red, prop.BorderColor.Green, prop.BorderColor.Blue)
	}

	p.fpdf.Line(x1, y1, x2, y2)
}
