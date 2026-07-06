package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type outlineStyler struct {
	stylerTemplate
}

// NewOutlineStyler creates a CellWriter chain node that draws an outline
// OUTSIDE the cell box (does not affect layout). It draws before downstream
// fills/content so thick decorative outlines can act as halos behind the cell.
func NewOutlineStyler(fpdf any) CellWriter {
	return &outlineStyler{
		stylerTemplate: stylerTemplate{fpdf: fpdf, name: "outlineStyler"},
	}
}

func (o *outlineStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	needOutline := prop != nil && prop.OutlineWidth > 0
	if !needOutline {
		o.GoToNext(width, height, config, prop)
		return
	}

	fpdf := asPDF[outlinePDF](o.fpdf)
	x, y := fpdf.GetXY()

	// Save state.
	origWidth := fpdf.GetLineWidth()
	origR, origG, origB := fpdf.GetDrawColor()

	colorR, colorG, colorB := origR, origG, origB
	if prop.OutlineColor != nil {
		colorR, colorG, colorB = prop.OutlineColor.Red, prop.OutlineColor.Green, prop.OutlineColor.Blue
	}

	fpdf.SetLineWidth(prop.OutlineWidth)
	fpdf.SetDrawColor(colorR, colorG, colorB)

	// Outline rect sits outside the cell: expanded by (outlineOffset + width/2).
	expansion := prop.OutlineOffset + prop.OutlineWidth/2
	rx := x - expansion
	ry := y - expansion
	rw := width + 2*expansion
	rh := height + 2*expansion

	switch prop.OutlineStyle {
	case consts.LineStyleSolid:
	case consts.LineStyleDashed:
		fpdf.SetDashPattern([]float64{1, 1}, 0)
	case consts.LineStyleDotted:
		fpdf.SetDashPattern([]float64{0.4, 0.4}, 0)
	}

	if radius := outlineRadius(prop, expansion, rw, rh); radius > 0 {
		if isCompactFullRoundCell(rw, rh, radius, radius, radius, radius) {
			fpdf.Ellipse(rx+rw/2, ry+rh/2, rw/2, rh/2, 0, "D")
		} else {
			fpdf.RoundedRect(rx, ry, rw, rh, radius, "1234", "D")
		}
	} else {
		fpdf.Rect(rx, ry, rw, rh, "D")
	}

	if prop.OutlineStyle != consts.LineStyleSolid && prop.OutlineStyle != "" {
		fpdf.SetDashPattern([]float64{1, 0}, 0)
	}

	fpdf.SetLineWidth(origWidth)
	fpdf.SetDrawColor(origR, origG, origB)

	o.GoToNext(width, height, config, prop)
}

func outlineRadius(prop *props.Cell, expansion, width, height float64) float64 {
	if prop == nil || !prop.HasBorderRadius() {
		return 0
	}
	tl, tr, br, bl := prop.EffectiveRadii()
	radius := max(max(tl, tr), max(br, bl)) + expansion
	return clampRadius(radius, width, height)
}
