package cellwriter

import (
	"github.com/johnfercher/maroto/v2/internal/providers/gofpdf/gofpdfwrapper"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

const (
	shadowBlurIterations = 3
	shadowAlphaOuter     = 0.3
	shadowAlphaMid       = 0.5
	shadowAlphaInner     = 0.8
)

type shadowStyler struct {
	stylerTemplate
}

// NewShadowStyler creates a CellWriter chain node that renders box-shadows
// behind the cell. It must be the FIRST node in the chain so it draws beneath
// all other decorations. The cursor position is saved and restored so
// downstream nodes see the original coordinates.
func NewShadowStyler(fpdf gofpdfwrapper.Fpdf) CellWriter {
	return &shadowStyler{
		stylerTemplate: stylerTemplate{fpdf: fpdf, name: "shadowStyler"},
	}
}

func (s *shadowStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil || len(prop.BoxShadow) == 0 {
		s.GoToNext(width, height, config, prop)
		return
	}

	// Save cursor so downstream chain nodes draw at the correct position.
	origX, origY := s.fpdf.GetXY()
	left, top, _, _ := s.fpdf.GetMargins()

	for _, shadow := range prop.BoxShadow {
		s.drawShadow(origX, origY, width, height, left, top, shadow)
	}

	// Restore cursor before forwarding.
	s.fpdf.SetXY(origX, origY)
	s.GoToNext(width, height, config, prop)
}

func (s *shadowStyler) drawShadow(cellX, cellY, width, height, left, top float64, sh props.Shadow) {
	cr, cg, cb := 0, 0, 0
	if sh.Color != nil {
		cr, cg, cb = sh.Color.Red, sh.Color.Green, sh.Color.Blue
	}

	// Base rect — offset from cell by (offsetX, offsetY) and expanded by spread.
	rx := cellX + sh.OffsetX - sh.Spread
	ry := cellY + sh.OffsetY - sh.Spread
	rw := width + 2*sh.Spread
	rh := height + 2*sh.Spread

	if sh.Inset {
		// Inset: offset inverted, drawn inside the cell.
		rx = cellX - sh.OffsetX
		ry = cellY - sh.OffsetY
		rw = width
		rh = height
	}

	if sh.BlurRadius <= 0 || shadowBlurIterations == 1 {
		s.fpdf.SetFillColor(cr, cg, cb)
		a := 1.0
		if sh.Color != nil && sh.Color.Alpha != nil {
			a = *sh.Color.Alpha
		}
		s.fpdf.SetAlpha(a, "Normal")
		s.fpdf.Rect(rx+left, ry+top, rw, rh, "F")
		s.fpdf.SetAlpha(1, "Normal")
	} else {
		alphas := [shadowBlurIterations]float64{shadowAlphaOuter, shadowAlphaMid, shadowAlphaInner}
		step := sh.BlurRadius / float64(shadowBlurIterations)
		for i := range shadowBlurIterations {
			expand := sh.BlurRadius - float64(i)*step
			s.fpdf.SetFillColor(cr, cg, cb)
			s.fpdf.SetAlpha(alphas[i], "Normal")
			s.fpdf.Rect(rx-expand+left, ry-expand+top, rw+2*expand, rh+2*expand, "F")
		}
		s.fpdf.SetAlpha(1, "Normal")
	}
	// Reset fill colour to avoid leaking into downstream nodes.
	s.fpdf.SetFillColor(255, 255, 255)
}
