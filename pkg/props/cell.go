package props

import (
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/linestyle"
)

// Cell is the representation of a cell in the grid system.
// This can be applied to Col or Row.
type Cell struct {
	// BackgroundColor defines which color will be applied to a cell.
	// Default: nil
	BackgroundColor *Color
	// BorderColor defines which color will be applied to a border cell
	// Default: nil
	BorderColor *Color
	// BorderType defines which kind of border will be applied to a cell.
	// Default: border.None
	BorderType border.Type
	// BorderThickness defines the border thickness applied to a cell.
	// Default: 0.2
	BorderThickness float64
	// LineStyle defines which line style will be applied to a cell.
	// Default: Solid
	LineStyle linestyle.Type

	// Per-side padding (mm). Applied by components to shift content inward.
	PaddingTop    float64
	PaddingRight  float64
	PaddingBottom float64
	PaddingLeft   float64

	// Per-side border colors (nil = no override; falls back to BorderColor).
	BorderTopColor    *Color
	BorderRightColor  *Color
	BorderBottomColor *Color
	BorderLeftColor   *Color

	// Per-side border thickness (0 = no border for that side).
	// When any of these is non-zero the PerSideBorderStyler activates and
	// draws raw Line calls instead of CellFormat borders.
	BorderTopThickness    float64
	BorderRightThickness  float64
	BorderBottomThickness float64
	BorderLeftThickness   float64

	// Per-side border line style ("solid", "dashed", "dotted"). Empty = solid.
	BorderTopStyle    linestyle.Type
	BorderRightStyle  linestyle.Type
	BorderBottomStyle linestyle.Type
	BorderLeftStyle   linestyle.Type

	// BackgroundGradient, when non-nil, paints a gradient behind the cell
	// (overrides BackgroundColor when both are set).
	BackgroundGradient *Gradient

	// BoxShadow holds the shadows to paint behind the cell (up to 4).
	BoxShadow []Shadow

	// Outline fields. Outline is drawn OUTSIDE the cell box (does not affect layout).
	OutlineWidth  float64
	OutlineStyle  linestyle.Type
	OutlineColor  *Color
	OutlineOffset float64 // mm; positive = further out, negative = inside border

	// BorderRadius is the uniform corner radius in mm. When per-corner radii
	// below are set, they override this uniform value for that corner.
	// When any radius is set, the borderRadiusStyler owns the entire border
	// render and per-side stroke thicknesses are averaged into a single width.
	BorderRadius float64

	// Per-corner border radius (mm). 0 = inherit from BorderRadius.
	BorderRadiusTopLeft     float64
	BorderRadiusTopRight    float64
	BorderRadiusBottomLeft  float64
	BorderRadiusBottomRight float64
}

// HasBorderRadius reports whether any uniform or per-corner radius is non-zero.
func (c *Cell) HasBorderRadius() bool {
	if c == nil {
		return false
	}
	return c.BorderRadius > 0 ||
		c.BorderRadiusTopLeft > 0 ||
		c.BorderRadiusTopRight > 0 ||
		c.BorderRadiusBottomLeft > 0 ||
		c.BorderRadiusBottomRight > 0
}

// EffectiveRadii returns the four corner radii (top-left, top-right, bottom-right, bottom-left)
// applying the precedence: per-corner > uniform > 0.
func (c *Cell) EffectiveRadii() (tl, tr, br, bl float64) {
	if c == nil {
		return 0, 0, 0, 0
	}
	tl = c.BorderRadiusTopLeft
	if tl == 0 {
		tl = c.BorderRadius
	}
	tr = c.BorderRadiusTopRight
	if tr == 0 {
		tr = c.BorderRadius
	}
	br = c.BorderRadiusBottomRight
	if br == 0 {
		br = c.BorderRadius
	}
	bl = c.BorderRadiusBottomLeft
	if bl == 0 {
		bl = c.BorderRadius
	}
	return tl, tr, br, bl
}

// HasPerSideBorders reports whether any per-side border thickness is set.
func (c *Cell) HasPerSideBorders() bool {
	return c.BorderTopThickness > 0 || c.BorderRightThickness > 0 ||
		c.BorderBottomThickness > 0 || c.BorderLeftThickness > 0
}

// ToMap adds the Cell fields to the map.
func (c *Cell) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	m := make(map[string]any)

	if c.BorderType != border.None {
		m["prop_border_type"] = c.BorderType
	}

	if c.BorderThickness != 0 {
		m["prop_border_thickness"] = c.BorderThickness
	}

	if c.LineStyle != "" {
		m["prop_border_line_style"] = c.LineStyle
	}

	if c.BackgroundColor != nil {
		m["prop_background_color"] = c.BackgroundColor.ToString()
	}

	if c.BorderColor != nil {
		m["prop_border_color"] = c.BorderColor.ToString()
	}

	return m
}
