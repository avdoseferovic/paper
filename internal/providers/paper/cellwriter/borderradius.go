package cellwriter

import (
	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// bezierArcMagic is the standard cubic-Bezier approximation factor for a
// circular quarter-arc (k ≈ 0.5522847498). All mainstream rendering engines
// use this constant for rounded rectangles.
const bezierArcMagic = 0.5522847498

type borderRadiusStyler struct {
	stylerTemplate
}

// NewBorderRadiusStyler creates a CellWriter that draws filled and stroked
// rounded rectangles when prop.HasBorderRadius() is true. When active it owns
// the entire fill + border render and clears BackgroundColor / BorderType /
// per-side thicknesses on the downstream prop so subsequent stylers (fill,
// border, cellWriter) do not redraw rectangles on top of the rounded path.
//
// Current limitation: mixed per-side border widths use the averaged thickness
// as a single stroke width. Uniform borders are unaffected.
func NewBorderRadiusStyler(fpdf gofpdfwrapper.Fpdf) CellWriter {
	return &borderRadiusStyler{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "borderRadiusStyler",
		},
	}
}

func (b *borderRadiusStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	if prop == nil || !prop.HasBorderRadius() {
		b.GoToNext(width, height, config, prop)
		return
	}

	// Save and restore draw state so we don't leak.
	origLineWidth := b.fpdf.GetLineWidth()
	origDR, origDG, origDB := b.fpdf.GetDrawColor()
	origFR, origFG, origFB := b.fpdf.GetFillColor()
	defer func() {
		b.fpdf.SetLineWidth(origLineWidth)
		b.fpdf.SetDrawColor(origDR, origDG, origDB)
		b.fpdf.SetFillColor(origFR, origFG, origFB)
	}()

	x, y := b.fpdf.GetXY()

	tl, tr, br, bl := prop.EffectiveRadii()
	tl = clampRadius(tl, width, height)
	tr = clampRadius(tr, width, height)
	br = clampRadius(br, width, height)
	bl = clampRadius(bl, width, height)

	hasFill := prop.BackgroundColor != nil
	thickness := averageBorderThickness(prop)
	hasStroke := thickness > 0

	if !hasFill && !hasStroke {
		// Radius set but nothing to paint — just pass through with bg/border cleared.
		b.passCleared(width, height, config, prop)
		return
	}

	if hasFill {
		b.fpdf.SetFillColor(prop.BackgroundColor.Red, prop.BackgroundColor.Green, prop.BackgroundColor.Blue)
	}
	if hasStroke {
		b.fpdf.SetLineWidth(thickness)
		if c := pickStrokeColor(prop); c != nil {
			b.fpdf.SetDrawColor(c.Red, c.Green, c.Blue)
		}
	}

	b.tracePath(x, y, width, height, tl, tr, br, bl)
	style := drawStyle(hasFill, hasStroke)
	// If fill or stroke has alpha < 1, wrap DrawPath in SetAlpha; restore on
	// return. Fill alpha takes precedence when both are translucent.
	if a := effectiveAlpha(prop); a < 1 {
		b.fpdf.SetAlpha(clampAlpha(a), "Normal")
		b.fpdf.DrawPath(style)
		b.fpdf.SetAlpha(1, "Normal")
	} else {
		b.fpdf.DrawPath(style)
	}

	b.passCleared(width, height, config, prop)
}

// passCleared forwards to the next styler with background/border/per-side fields
// zeroed so downstream nodes don't redraw rectangular fill/stroke on top.
func (b *borderRadiusStyler) passCleared(width, height float64, config *entity.Config, prop *props.Cell) {
	modified := *prop
	modified.BackgroundColor = nil
	modified.BorderType = border.None
	modified.BorderTopThickness = 0
	modified.BorderRightThickness = 0
	modified.BorderBottomThickness = 0
	modified.BorderLeftThickness = 0
	b.GoToNext(width, height, config, &modified)
}

// tracePath draws a rounded-rectangle path starting at the top-left corner end
// of the top edge, going clockwise. Each corner uses one cubic Bezier curve.
func (b *borderRadiusStyler) tracePath(x, y, w, h, tl, tr, br, bl float64) {
	k := bezierArcMagic

	// Top edge starts at (x+tl, y) and ends at (x+w-tr, y).
	b.fpdf.MoveTo(x+tl, y)
	b.fpdf.LineTo(x+w-tr, y)

	// Top-right corner: arc from (x+w-tr, y) down to (x+w, y+tr).
	b.fpdf.CurveBezierCubicTo(
		x+w-tr+tr*k, y,
		x+w, y+tr-tr*k,
		x+w, y+tr,
	)
	b.fpdf.LineTo(x+w, y+h-br)

	// Bottom-right corner: arc from (x+w, y+h-br) to (x+w-br, y+h).
	b.fpdf.CurveBezierCubicTo(
		x+w, y+h-br+br*k,
		x+w-br+br*k, y+h,
		x+w-br, y+h,
	)
	b.fpdf.LineTo(x+bl, y+h)

	// Bottom-left corner: arc from (x+bl, y+h) to (x, y+h-bl).
	b.fpdf.CurveBezierCubicTo(
		x+bl-bl*k, y+h,
		x, y+h-bl+bl*k,
		x, y+h-bl,
	)
	b.fpdf.LineTo(x, y+tl)

	// Top-left corner: arc from (x, y+tl) to (x+tl, y).
	b.fpdf.CurveBezierCubicTo(
		x, y+tl-tl*k,
		x+tl-tl*k, y,
		x+tl, y,
	)
	b.fpdf.ClosePath()
}

func clampRadius(r, w, h float64) float64 {
	maxR := w / 2
	if h/2 < maxR {
		maxR = h / 2
	}
	if r > maxR {
		return maxR
	}
	if r < 0 {
		return 0
	}
	return r
}

// averageBorderThickness picks a single thickness for the rounded stroke:
// uses BorderThickness if set, else averages the non-zero per-side thicknesses.
func averageBorderThickness(prop *props.Cell) float64 {
	if prop.BorderThickness > 0 {
		return prop.BorderThickness
	}
	sum := 0.0
	count := 0
	for _, t := range []float64{
		prop.BorderTopThickness, prop.BorderRightThickness,
		prop.BorderBottomThickness, prop.BorderLeftThickness,
	} {
		if t > 0 {
			sum += t
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func pickStrokeColor(prop *props.Cell) *props.Color {
	if prop.BorderColor != nil {
		return prop.BorderColor
	}
	for _, c := range []*props.Color{
		prop.BorderTopColor, prop.BorderRightColor,
		prop.BorderBottomColor, prop.BorderLeftColor,
	} {
		if c != nil {
			return c
		}
	}
	return nil
}

func drawStyle(fill, stroke bool) string {
	switch {
	case fill && stroke:
		return "DF"
	case fill:
		return "F"
	case stroke:
		return "D"
	default:
		return ""
	}
}
