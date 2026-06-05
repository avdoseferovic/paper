package cellwriter

import (
	"github.com/johnfercher/maroto/v2/internal/providers/paper/gofpdfwrapper"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// gradientDrawer is the narrow interface of GradientRenderer used here,
// avoiding a direct import of the gofpdf package from within its own sub-package.
type gradientDrawer interface {
	DrawGradient(cell *entity.Cell, g *props.Gradient, widthMM, heightMM float64)
}

type gradientStyler struct {
	stylerTemplate
	drawer gradientDrawer
}

// NewGradientStyler creates a CellWriter chain node that paints gradient
// backgrounds before the solid fill colour styler runs.
func NewGradientStyler(fpdf gofpdfwrapper.Fpdf, drawer gradientDrawer) CellWriter {
	return &gradientStyler{
		stylerTemplate: stylerTemplate{fpdf: fpdf, name: "gradientStyler"},
		drawer:         drawer,
	}
}

func (g *gradientStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop != nil && prop.BackgroundGradient != nil {
		x, y := g.fpdf.GetXY()
		left, top, _, _ := g.fpdf.GetMargins()
		// Build a margin-relative cell for DrawGradient (it adds margins internally).
		cell := &entity.Cell{
			X:      x - left,
			Y:      y - top,
			Width:  width,
			Height: height,
		}
		g.drawer.DrawGradient(cell, prop.BackgroundGradient, width, height)
	}
	g.GoToNext(width, height, config, prop)
}
