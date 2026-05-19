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
