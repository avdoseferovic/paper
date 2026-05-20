package cellwriter

import (
	"github.com/johnfercher/maroto/v2/internal/providers/gofpdf/gofpdfwrapper"
	"github.com/johnfercher/maroto/v2/pkg/consts/linestyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type outlineStyler struct {
	stylerTemplate
}

// NewOutlineStyler creates a CellWriter chain node that draws an outline
// OUTSIDE the cell box (does not affect layout). It must be the LAST node in
// the chain so it can read the final cell position from GetXY after other nodes
// have drawn. The cursor position is saved and restored before forwarding.
func NewOutlineStyler(fpdf gofpdfwrapper.Fpdf) CellWriter {
	return &outlineStyler{
		stylerTemplate: stylerTemplate{fpdf: fpdf, name: "outlineStyler"},
	}
}

func (o *outlineStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	o.GoToNext(width, height, config, prop)

	if prop == nil || prop.OutlineWidth <= 0 {
		return
	}

	// Save state.
	origWidth := o.fpdf.GetLineWidth()
	origR, origG, origB := o.fpdf.GetDrawColor()
	defer func() {
		o.fpdf.SetLineWidth(origWidth)
		o.fpdf.SetDrawColor(origR, origG, origB)
	}()

	// Capture the cell origin from the cursor (GetXY after all prior nodes ran).
	x, y := o.fpdf.GetXY()

	o.fpdf.SetLineWidth(prop.OutlineWidth)
	if prop.OutlineColor != nil {
		o.fpdf.SetDrawColor(prop.OutlineColor.Red, prop.OutlineColor.Green, prop.OutlineColor.Blue)
	}

	// Outline rect sits outside the cell: expanded by (outlineOffset + width/2).
	expansion := prop.OutlineOffset + prop.OutlineWidth/2
	rx := x - expansion
	ry := y - expansion
	rw := width + 2*expansion
	rh := height + 2*expansion

	switch prop.OutlineStyle {
	case linestyle.Dashed:
		o.fpdf.SetDashPattern([]float64{1, 1}, 0)
	case linestyle.Dotted:
		o.fpdf.SetDashPattern([]float64{0.4, 0.4}, 0)
	}

	o.fpdf.Rect(rx, ry, rw, rh, "D")

	if prop.OutlineStyle != linestyle.Solid && prop.OutlineStyle != "" {
		o.fpdf.SetDashPattern([]float64{1, 0}, 0)
	}
}
