package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/consts/linestyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type outlineStyler struct {
	stylerTemplate
}

// NewOutlineStyler creates a CellWriter chain node that draws an outline
// OUTSIDE the cell box (does not affect layout). It must be the LAST node in
// the chain so it can read the final cell position from GetXY after other nodes
// have drawn. The cursor position is saved and restored before forwarding.
func NewOutlineStyler(fpdf any) CellWriter {
	return &outlineStyler{
		stylerTemplate: stylerTemplate{fpdf: fpdf, name: "outlineStyler"},
	}
}

func (o *outlineStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	needOutline := prop != nil && prop.OutlineWidth > 0

	// Capture the cell origin BEFORE the downstream chain moves the cursor
	// (cellWriter's CellFormat advances X to the cell's right edge).
	fpdf := asPDF[outlinePDF](o.fpdf)
	var x, y float64
	if needOutline {
		x, y = fpdf.GetXY()
	}

	o.GoToNext(width, height, config, prop)

	if !needOutline {
		return
	}

	// Save state.
	origWidth := fpdf.GetLineWidth()
	origR, origG, origB := fpdf.GetDrawColor()
	defer func() {
		fpdf.SetLineWidth(origWidth)
		fpdf.SetDrawColor(origR, origG, origB)
	}()

	fpdf.SetLineWidth(prop.OutlineWidth)
	if prop.OutlineColor != nil {
		fpdf.SetDrawColor(prop.OutlineColor.Red, prop.OutlineColor.Green, prop.OutlineColor.Blue)
	}

	// Outline rect sits outside the cell: expanded by (outlineOffset + width/2).
	expansion := prop.OutlineOffset + prop.OutlineWidth/2
	rx := x - expansion
	ry := y - expansion
	rw := width + 2*expansion
	rh := height + 2*expansion

	switch prop.OutlineStyle {
	case linestyle.Solid:
	case linestyle.Dashed:
		fpdf.SetDashPattern([]float64{1, 1}, 0)
	case linestyle.Dotted:
		fpdf.SetDashPattern([]float64{0.4, 0.4}, 0)
	}

	fpdf.Rect(rx, ry, rw, rh, "D")

	if prop.OutlineStyle != linestyle.Solid && prop.OutlineStyle != "" {
		fpdf.SetDashPattern([]float64{1, 0}, 0)
	}
}
