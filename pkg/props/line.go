package props

import (
	"cmp"

	"github.com/avdoseferovic/paper/pkg/consts"
)

// Line represents properties from a Line inside a cell.
type Line struct {
	// Color define the line color.
	Color *Color
	// Style define the line style (solid or dashed).
	Style consts.LineStyle
	// Thickness define the line thicknesl.
	Thickness float64
	// Orientation define if line would be horizontal or vertical.
	Orientation consts.Orientation
	// OffsetPercent define where the line would be placed, 0 is the start of cell, 50 the middle and 100 the end.
	OffsetPercent float64
	// SizePercent define the size of the line inside cell.
	SizePercent float64
}

// ToMap returns a map with the Line fields.
func (l *Line) ToMap() map[string]any {
	if l == nil {
		return nil
	}

	m := make(map[string]any)

	if l.Color != nil {
		m["prop_color"] = l.Color.ToString()
	}

	if l.Style != "" {
		m["prop_style"] = l.Style
	}

	if l.Thickness != 0 {
		m["prop_thickness"] = l.Thickness
	}

	if l.Orientation != "" {
		m["prop_orientation"] = l.Orientation
	}

	if l.OffsetPercent != 0 {
		m["prop_offset_percent"] = l.OffsetPercent
	}

	if l.SizePercent != 0 {
		m["prop_size_percent"] = l.SizePercent
	}

	return m
}

// MakeValid from Line define default values for a Line.
func (l *Line) MakeValid() {
	*l = NormalizeLine(*l)
}

// NormalizeLine returns a defaulted copy of l.
func NormalizeLine(l Line) Line {
	l.Style = cmp.Or(l.Style, consts.LineStyleSolid)

	l.Thickness = cmp.Or(l.Thickness, consts.DefaultLineThickness)

	l.Orientation = cmp.Or(l.Orientation, consts.OrientationHorizontal)

	l.OffsetPercent = max(l.OffsetPercent, 5)

	l.OffsetPercent = min(l.OffsetPercent, 95)

	if l.SizePercent <= 0 {
		l.SizePercent = 90
	}

	l.SizePercent = min(l.SizePercent, 100)

	l.Color = CloneColor(l.Color)
	return l
}
