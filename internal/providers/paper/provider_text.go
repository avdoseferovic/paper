package paper

import (
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

func (g *provider) AddText(text string, cell *entity.Cell, prop *props.Text) {
	g.text.Add(text, cell, prop)
}

func (g *provider) GetLinesQuantity(text string, textProp *props.Text, colWidth float64) int {
	return g.text.GetLinesQuantity(text, textProp, colWidth)
}

func (g *provider) GetFontHeight(prop *props.Font) float64 {
	return g.font.GetHeight(prop.Family, prop.Style, prop.Size)
}

func (g *provider) AddLine(cell *entity.Cell, prop *props.Line) {
	g.line.Add(cell, prop)
}

func (g *provider) AddCheckbox(label string, cell *entity.Cell, prop *props.Checkbox) {
	g.checkbox.Add(label, cell, prop)
}
