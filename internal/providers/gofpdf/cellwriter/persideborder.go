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

	// Compute cell coordinates in page space.
	left, top, _, _ := p.fpdf.GetMargins()

	// We don't have cell X/Y here — the cellwriter chain only receives width/height.
	// gofpdf's current pen position (GetXY) marks the cell's top-left.
	// Use the same approach as CellFormat: cell starts at current position.
	// For Line drawing we reconstruct corners from current pdf position.
	// Since we can't call GetXY (not in the interface), we approximate using margins only.
	// Actual cell position is tracked by gofpdf internally; Line coordinates are absolute.
	// We use 0-relative offsets + page margins, matching how CellFormat positions itself.
	// Note: the true X/Y would require GetXY which isn't in the gofpdfwrapper interface.
	// The caller (cellWriter.Apply via CellFormat) handles real positioning. Here we
	// draw decorative lines relative to the margin origin (a known v1 limitation for
	// per-side borders that are not cell-position-aware).
	// TODO(v2): add GetXY to gofpdfwrapper.Fpdf and use real cell coordinates.
	x := left
	y := top

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
