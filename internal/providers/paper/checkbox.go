package paper

import (
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

const labelGap = 1.0

// checkboxLabelBaselineRatio places the label baseline so the cap height is
// visually centered on the checkbox: the middle of a glyph's cap box sits
// ~0.35 * fontHeight above the baseline (consistent with the vertical
// centering in provider_capabilities.go and richtext_render.go).
const checkboxLabelBaselineRatio = 0.35

type Checkbox struct {
	pdf                   checkboxPDF
	font                  core.Font
	defaultCodeTranslator func(string) string
}

// NewCheckbox create a Checkbox.
func NewCheckbox(pdf checkboxPDF, font core.Font) *Checkbox {
	return &Checkbox{pdf: pdf, font: font}
}

// Add a checkbox with a label inside a cell.
func (c *Checkbox) Add(label string, cell *entity.Cell, prop *props.Checkbox) {
	left, top, _, _ := c.pdf.GetMargins()

	x := cell.X + prop.Left + left
	y := cell.Y + prop.Top + top

	// Draw the checkbox square border
	c.pdf.Rect(x, y, prop.Size, prop.Size, "D")

	if prop.Checked {
		// Draw X mark inside the box
		c.pdf.Line(x, y, x+prop.Size, y+prop.Size)
		c.pdf.Line(x+prop.Size, y, x, y+prop.Size)
	}

	// Draw label to the right of the checkbox, vertically centered
	if label != "" {
		family, style, size := c.font.GetFont()
		fontHeight := c.font.GetHeight(family, style, size)

		labelX := x + prop.Size + labelGap
		labelY := y + prop.Size/2 + fontHeight*checkboxLabelBaselineRatio

		c.pdf.Text(labelX, labelY, c.translateLabel(label, family))
	}
}

// translateLabel applies the code page translation for core font families so
// non-ASCII labels don't render as mojibake, mirroring Text.translateUnicode.
// Custom (UTF-8) fonts pass through unchanged.
func (c *Checkbox) translateLabel(label, family string) string {
	if !isCoreFontFamily(family) {
		return label
	}
	if c.defaultCodeTranslator == nil {
		c.defaultCodeTranslator = c.pdf.UnicodeTranslatorFromDescriptor("")
	}
	return c.defaultCodeTranslator(label)
}
