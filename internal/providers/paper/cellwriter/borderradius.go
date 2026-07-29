package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// bezierArcMagic is the standard cubic-Bezier approximation factor for a
// circular quarter-arc (k ≈ 0.5522847498). All mainstream rendering engines
// use this constant for rounded rectangles.
const bezierArcMagic = 0.5522847498

// accentCornerInsetFactor lets thick per-side accents enter rounded corners
// without painting a square full-height bar over the radius.
const accentCornerInsetFactor = 0.35

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
func NewBorderRadiusStyler(fpdf any) CellWriter {
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
	fpdf := asPDF[borderRadiusPDF](b.fpdf)
	origLineWidth := fpdf.GetLineWidth()
	origDR, origDG, origDB := fpdf.GetDrawColor()
	origFR, origFG, origFB := fpdf.GetFillColor()
	defer func() {
		fpdf.SetLineWidth(origLineWidth)
		fpdf.SetDrawColor(origDR, origDG, origDB)
		fpdf.SetFillColor(origFR, origFG, origFB)
	}()

	x, y := fpdf.GetXY()

	tl, tr, br, bl := prop.EffectiveRadii()
	tl = clampRadius(tl, width, height)
	tr = clampRadius(tr, width, height)
	br = clampRadius(br, width, height)
	bl = clampRadius(bl, width, height)

	hasFill := prop.BackgroundColor != nil
	// Use the thinnest (base) border for the rounded stroke; sides thicker than
	// the base (e.g. a coloured left accent bar) are overlaid as straight lines
	// afterwards so a rounded card can still carry a per-side accent.
	thickness := baseBorderThickness(prop)
	hasStroke := thickness > 0

	if !hasFill && !hasStroke {
		// Radius set but nothing to paint — just pass through with bg/border cleared.
		b.passCleared(width, height, config, prop)
		return
	}

	if hasFill {
		fpdf.SetFillColor(prop.BackgroundColor.Red, prop.BackgroundColor.Green, prop.BackgroundColor.Blue)
	}
	if hasStroke {
		fpdf.SetLineWidth(thickness)
		if c := pickStrokeColor(prop); c != nil {
			fpdf.SetDrawColor(c.Red, c.Green, c.Blue)
		}
	}

	style := drawStyle(hasFill, hasStroke)
	restoreJoin := b.useRoundJoinForStroke(hasStroke)
	defer restoreJoin()
	if isCompactFullRoundCell(width, height, tl, tr, br, bl) {
		b.drawEllipse(fpdf, prop, x, y, width, height, style)
		b.drawAccentSides(fpdf, prop, x, y, width, height, tl, tr, br, bl, thickness)
		b.passCleared(width, height, config, prop)
		return
	}

	b.tracePath(fpdf, x, y, width, height, tl, tr, br, bl)
	// If fill or stroke has alpha < 1, wrap DrawPath in SetAlpha; restore on
	// return. Fill alpha takes precedence when both are translucent.
	if a := effectiveAlpha(prop); a < 1 {
		fpdf.SetAlpha(clampAlpha(a), "Normal")
		fpdf.DrawPath(style)
		fpdf.SetAlpha(1, "Normal")
	} else {
		fpdf.DrawPath(style)
	}

	// Overlay any side thicker than the base stroke (e.g. a coloured left accent
	// bar) as a straight line between its adjacent corner radii.
	b.drawAccentSides(fpdf, prop, x, y, width, height, tl, tr, br, bl, thickness)

	b.passCleared(width, height, config, prop)
}

func (b *borderRadiusStyler) drawEllipse(fpdf borderRadiusPDF, prop *props.Cell, x, y, w, h float64, style string) {
	if shouldExpandFilledEllipse(prop, style) {
		x -= filledEllipseExpansionMM
		y -= filledEllipseExpansionMM
		w += 2 * filledEllipseExpansionMM
		h += 2 * filledEllipseExpansionMM
	}
	draw := func() {
		fpdf.Ellipse(x+w/2, y+h/2, w/2, h/2, 0, style)
	}
	if a := effectiveAlpha(prop); a < 1 {
		fpdf.SetAlpha(clampAlpha(a), "Normal")
		draw()
		fpdf.SetAlpha(1, "Normal")
		return
	}
	draw()
}

// gofpdf's filled ellipse rasterizes slightly inside Chrome's CSS full-circle
// fill at small sizes. Compensate only fill-only compact circles; stroked
// circles keep exact geometry so borders stay aligned.
const filledEllipseExpansionMM = 0.08

func shouldExpandFilledEllipse(prop *props.Cell, style string) bool {
	return style == "F" && prop != nil && prop.BackgroundColor != nil && baseBorderThickness(prop) == 0
}

// drawAccentSides overlays straight border lines for the sides whose thickness
// exceeds the rounded stroke's base thickness, so a rounded card can carry a
// per-side accent (typically a coloured left bar). Each line runs between the
// side's two corner radii so the rounded corners stay clean.
func (b *borderRadiusStyler) drawAccentSides(fpdf borderRadiusPDF, prop *props.Cell, x, y, w, h, tl, tr, br, bl, base float64) {
	origLW := fpdf.GetLineWidth()
	origR, origG, origB := fpdf.GetDrawColor()
	defer func() {
		fpdf.SetLineWidth(origLW)
		fpdf.SetDrawColor(origR, origG, origB)
	}()
	leftInset := tl * accentCornerInsetFactor
	leftBottomInset := bl * accentCornerInsetFactor
	rightInset := tr * accentCornerInsetFactor
	rightBottomInset := br * accentCornerInsetFactor
	// left
	b.accentLine(
		fpdf, prop.BorderLeftThickness, base, prop.BorderLeftColor, prop,
		x+prop.BorderLeftThickness/2, y+leftInset,
		x+prop.BorderLeftThickness/2, y+h-leftBottomInset,
	)
	// right
	b.accentLine(
		fpdf, prop.BorderRightThickness, base, prop.BorderRightColor, prop,
		x+w-prop.BorderRightThickness/2, y+rightInset,
		x+w-prop.BorderRightThickness/2, y+h-rightBottomInset,
	)
	// top
	b.accentLine(
		fpdf, prop.BorderTopThickness, base, prop.BorderTopColor, prop,
		x+tl, y+prop.BorderTopThickness/2,
		x+w-tr, y+prop.BorderTopThickness/2,
	)
	// bottom
	b.accentLine(
		fpdf, prop.BorderBottomThickness, base, prop.BorderBottomColor, prop,
		x+bl, y+h-prop.BorderBottomThickness/2,
		x+w-br, y+h-prop.BorderBottomThickness/2,
	)
}

func (b *borderRadiusStyler) accentLine(
	fpdf borderRadiusPDF,
	thickness, base float64,
	color *props.Color,
	prop *props.Cell,
	x1, y1, x2, y2 float64,
) {
	if thickness <= base {
		return // not an accent — already covered by the rounded stroke
	}
	fpdf.SetLineWidth(thickness)
	switch {
	case color != nil:
		fpdf.SetDrawColor(color.Red, color.Green, color.Blue)
	case prop.BorderColor != nil:
		fpdf.SetDrawColor(prop.BorderColor.Red, prop.BorderColor.Green, prop.BorderColor.Blue)
	}
	fpdf.Line(x1, y1, x2, y2)
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

func (b *borderRadiusStyler) useRoundJoinForStroke(hasStroke bool) func() {
	if !hasStroke {
		return func() {}
	}
	fpdf, ok := b.fpdf.(lineJoinPDF)
	if !ok {
		return func() {}
	}
	orig := fpdf.GetLineJoinStyle()
	fpdf.SetLineJoinStyle("round")
	return func() {
		fpdf.SetLineJoinStyle(orig)
	}
}

// tracePath draws a rounded-rectangle path starting at the top-left corner end
// of the top edge, going clockwise. Each corner uses one cubic Bezier curve.
func (b *borderRadiusStyler) tracePath(fpdf borderRadiusPDF, x, y, w, h, tl, tr, br, bl float64) {
	k := bezierArcMagic

	// Top edge starts at (x+tl, y) and ends at (x+w-tr, y).
	fpdf.MoveTo(x+tl, y)
	fpdf.LineTo(x+w-tr, y)

	// Top-right corner: arc from (x+w-tr, y) down to (x+w, y+tr).
	fpdf.CurveBezierCubicTo(
		x+w-tr+tr*k, y,
		x+w, y+tr-tr*k,
		x+w, y+tr,
	)
	fpdf.LineTo(x+w, y+h-br)

	// Bottom-right corner: arc from (x+w, y+h-br) to (x+w-br, y+h).
	fpdf.CurveBezierCubicTo(
		x+w, y+h-br+br*k,
		x+w-br+br*k, y+h,
		x+w-br, y+h,
	)
	fpdf.LineTo(x+bl, y+h)

	// Bottom-left corner: arc from (x+bl, y+h) to (x, y+h-bl).
	fpdf.CurveBezierCubicTo(
		x+bl-bl*k, y+h,
		x, y+h-bl+bl*k,
		x, y+h-bl,
	)
	fpdf.LineTo(x, y+tl)

	// Top-left corner: arc from (x, y+tl) to (x+tl, y).
	fpdf.CurveBezierCubicTo(
		x, y+tl-tl*k,
		x+tl-tl*k, y,
		x+tl, y,
	)
	fpdf.ClosePath()
}

func isCompactFullRoundCell(w, h, tl, tr, br, bl float64) bool {
	if w <= 0 || h <= 0 {
		return false
	}
	shortSide := min(w, h)
	longSide := max(w, h)
	if longSide/shortSide > 1.25 {
		return false
	}
	r := shortSide / 2
	return nearlyEqual(tl, r) && nearlyEqual(tr, r) && nearlyEqual(br, r) && nearlyEqual(bl, r)
}

func nearlyEqual(a, b float64) bool {
	const epsilon = 0.000001
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= epsilon
}

func clampRadius(r, w, h float64) float64 {
	return min(max(r, 0), min(w/2, h/2))
}

// baseBorderThickness picks the thickness for the rounded stroke: BorderThickness
// if set, else the thinnest non-zero per-side thickness. Sides thicker than this
// base are overlaid separately by drawAccentSides.
func baseBorderThickness(prop *props.Cell) float64 {
	if prop.BorderThickness > 0 {
		return prop.BorderThickness
	}
	minT := 0.0
	for _, t := range []float64{
		prop.BorderTopThickness, prop.BorderRightThickness,
		prop.BorderBottomThickness, prop.BorderLeftThickness,
	} {
		if t > 0 && (minT == 0 || t < minT) {
			minT = t
		}
	}
	return minT
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
