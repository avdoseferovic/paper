package translate

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

type crossAxisBox struct {
	child core.Component
	align string
}

func (c *crossAxisBox) SetConfig(config *entity.Config) {
	if c.child != nil {
		c.child.SetConfig(config)
	}
}

func (c *crossAxisBox) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:    "cross_axis_box",
		Details: map[string]any{"align": c.align},
	}
	n := node.New(str)
	if c.child != nil {
		n.AddNext(c.child.GetStructure())
	}
	return n
}

func (c *crossAxisBox) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if c.child == nil {
		return 0
	}
	return c.child.GetHeight(provider, cell)
}

func (c *crossAxisBox) Render(provider core.Provider, cell *entity.Cell) {
	if c.child == nil {
		return
	}
	childCell := cell.Copy()
	childHeight := c.child.GetHeight(provider, &childCell)
	switch c.align {
	case "center":
		if cell.Height > childHeight {
			childCell.Y += (cell.Height - childHeight) / 2
			childCell.Height = childHeight
		}
	case "flex-end":
		if cell.Height > childHeight {
			childCell.Y += cell.Height - childHeight
			childCell.Height = childHeight
		}
	case "flex-start":
		if childHeight > 0 {
			childCell.Height = childHeight
		}
	}
	c.child.Render(provider, &childCell)
}

func flexItemCrossAxisBox(child core.Component, containerStyle, itemStyle *css.ComputedStyle) core.Component {
	align := effectiveCrossAxisAlign(containerStyle, itemStyle)
	switch align {
	case "center", "flex-end", "flex-start":
		return &crossAxisBox{child: child, align: align}
	default:
		return child
	}
}

func effectiveCrossAxisAlign(containerStyle, itemStyle *css.ComputedStyle) string {
	align := ""
	if itemStyle != nil {
		align = normalizeCrossAxisAlign(itemStyle.AlignSelf)
	}
	if align == "" || align == "auto" {
		if containerStyle != nil {
			align = normalizeCrossAxisAlign(containerStyle.AlignItems)
		}
	}
	return align
}

func normalizeCrossAxisAlign(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "start", "self-start":
		return "flex-start"
	case "end", "self-end":
		return "flex-end"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}
