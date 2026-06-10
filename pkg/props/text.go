package props

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
)

// Text represents properties from a Text inside a cell.
type Text struct {
	// Top is the amount of space between the upper cell limit and the text.
	Top float64
	// Bottom is the amount of space between the lower cell limit and the text. (Used by auto row only)
	Bottom float64
	// Left is the minimal amount of space between the left cell boundary and the text.
	Left float64
	// Right is the minimal amount of space between the right cell boundary and the text.
	Right float64
	// Family of the text, ex: consts.Arial, helvetica and etc.
	Family string
	// Style of the text, ex: consts.Normal, bold and etc.
	Style fontstyle.Type
	// Size of the text.
	Size float64
	// Align of the text.
	Align consts.Align
	// BreakLineStrategy define the break line strategy.
	BreakLineStrategy consts.BreakLineStrategy
	// VerticalPadding define an additional space between linet.
	VerticalPadding float64
	// Color define the font style color.
	Color *Color
	// Hyperlink define a link to be opened when the text is clicked.
	Hyperlink *string
}

// ToMap converts a Text to a map.
func (t *Text) ToMap() map[string]any {
	m := make(map[string]any)
	if t.Top != 0 {
		m["prop_top"] = t.Top
	}
	if t.Bottom != 0 {
		m["prop_bottom"] = t.Bottom
	}

	if t.Left != 0 {
		m["prop_left"] = t.Left
	}

	if t.Right != 0 {
		m["prop_right"] = t.Right
	}

	if t.Family != "" {
		m["prop_font_family"] = t.Family
	}

	if t.Style != "" {
		m["prop_font_style"] = t.Style
	}

	if t.Size != 0 {
		m["prop_font_size"] = t.Size
	}

	if t.Align != "" {
		m["prop_align"] = t.Align
	}

	if t.BreakLineStrategy != "" {
		m["prop_breakline_strategy"] = t.BreakLineStrategy
	}

	if t.VerticalPadding != 0 {
		m["prop_vertical_padding"] = t.VerticalPadding
	}

	if t.Color != nil {
		m["prop_color"] = t.Color.ToString()
	}

	if t.Hyperlink != nil {
		m["prop_hyperlink"] = *t.Hyperlink
	}

	return m
}

// MakeValid from Text define default values for a Text.
func (t *Text) MakeValid(font *Font) {
	*t = NormalizeText(*t, font)
}

// NormalizeText returns a defaulted copy of t.
func NormalizeText(t Text, font *Font) Text {
	minValue := 0.0
	undefinedValue := 0.0

	defaultFont := Font{}
	if font != nil {
		defaultFont = NormalizeFont(*font, "")
	}

	if t.Family == "" {
		t.Family = defaultFont.Family
	}

	if t.Style == "" {
		t.Style = defaultFont.Style
	}

	if t.Size == undefinedValue {
		t.Size = defaultFont.Size
	}

	if t.Color == nil {
		t.Color = CloneColor(defaultFont.Color)
	} else {
		t.Color = CloneColor(t.Color)
	}

	if t.Align == "" {
		t.Align = consts.AlignLeft
	}

	if t.Top < minValue {
		t.Top = minValue
	}

	if t.Bottom < minValue {
		t.Bottom = minValue
	}

	if t.Left < minValue {
		t.Left = minValue
	}

	if t.Right < minValue {
		t.Right = minValue
	}

	if t.VerticalPadding < 0 {
		t.VerticalPadding = 0
	}

	if t.BreakLineStrategy == "" {
		t.BreakLineStrategy = consts.BreakLineEmptySpace
	}

	return t
}
