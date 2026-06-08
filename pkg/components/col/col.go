// Package col implements creation of columns.
package col

import (
	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/tree/node"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type Col struct {
	size       int
	isMax      bool
	components []core.Component
	config     *entity.Config
	style      *props.Cell
}

// New is responsible to create an instance of core.Col.
func New(size ...int) core.Col {
	if len(size) == 0 {
		return &Col{isMax: true}
	}

	return &Col{size: size[0]}
}

// Add is responsible to add a component to a core.Col.
func (c *Col) Add(components ...core.Component) core.Col {
	c.components = append(c.components, components...)
	return c
}

// GetSize returns the size of a core.Col.
func (c *Col) GetSize() int {
	if c.isMax {
		return c.maxGridSize()
	}

	return c.size
}

// GetStructure returns the Structure of a core.Col.
func (c *Col) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:    "col",
		Value:   c.size,
		Details: c.style.ToMap(),
	}

	if c.isMax {
		if len(str.Details) == 0 {
			str.Details = make(map[string]any)
		}
		str.Details["is_max"] = true
	}

	node := node.New(str)

	for _, c := range c.components {
		inner := c.GetStructure()
		node.AddNext(inner)
	}

	return node
}

// Render renders a core.Col into a PDF context.
func (c *Col) Render(provider core.Provider, cell entity.Cell, createCell bool) {
	if createCell {
		provider.CreateCol(cell.Width, cell.Height, c.config, c.style)
	}

	for _, component := range c.components {
		component.Render(provider, &cell)
	}
}

// SetConfig set the config for the component.
func (c *Col) SetConfig(config *entity.Config) {
	c.config = config
	for _, component := range c.components {
		component.SetConfig(config)
	}
}

// WithStyle sets the style for the column.
func (c *Col) WithStyle(style *props.Cell) core.Col {
	c.style = props.CloneCell(style)
	return c
}

// GetHeight returns the height of the column content
func (c *Col) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	innerCell := cell.Copy()
	plan := layout.ManualUnits([]int{c.GetSize()}, c.maxGridSize())
	innerCell.Width = layout.UnitWidth(innerCell.Width, plan.Units[0], plan.GridSize)

	greaterHeight := 0.0
	for _, component := range c.components {
		height := component.GetHeight(provider, &innerCell)
		if greaterHeight < height {
			greaterHeight = height
		}
	}
	return greaterHeight
}

func (c *Col) maxGridSize() int {
	if c.config == nil {
		return layout.DefaultGridSize()
	}
	return layout.NormalizeGridSize(c.config.MaxGridSize)
}
