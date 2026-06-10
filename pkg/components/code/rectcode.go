package code

import (
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

type rectCodeRenderer func(provider core.Provider, code string, cell *entity.Cell, prop *props.Rect)

type rectCodeDimensions func(provider core.Provider, code string) (*entity.Dimensions, error)

type rectCode struct {
	code          string
	prop          props.Rect
	config        *entity.Config
	structureType string
	render        rectCodeRenderer
	dimensions    rectCodeDimensions
}

func newRectCode(
	structureType string,
	code string,
	render rectCodeRenderer,
	dimensions rectCodeDimensions,
	rectProps ...props.Rect,
) *rectCode {
	prop := props.Rect{}
	if len(rectProps) > 0 {
		prop = rectProps[0]
	}
	prop.MakeValid()

	return &rectCode{
		code:          code,
		prop:          prop,
		structureType: structureType,
		render:        render,
		dimensions:    dimensions,
	}
}

func newRectCodeCol(size int, component core.Component) core.Col {
	return col.New(size).Add(component)
}

func newRectComponentCol(size int, constructor rectCodeConstructor, code string, ps ...props.Rect) core.Col {
	return newRectCodeCol(size, constructor(code, ps...))
}

func newAutoRectCodeRow(component core.Component) core.Row {
	return row.New().Add(col.New().Add(component))
}

func newAutoRectComponentRow(constructor rectCodeConstructor, code string, ps ...props.Rect) core.Row {
	return newAutoRectCodeRow(constructor(code, ps...))
}

func newRectCodeRow(height float64, component core.Component) core.Row {
	return row.New(height).Add(col.New().Add(component))
}

func newRectComponentRow(height float64, constructor rectCodeConstructor, code string, ps ...props.Rect) core.Row {
	return newRectCodeRow(height, constructor(code, ps...))
}

func (c *rectCode) Render(provider core.Provider, cell *entity.Cell) {
	c.render(provider, c.code, cell, &c.prop)
}

func (c *rectCode) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:    c.structureType,
		Value:   c.code,
		Details: c.prop.ToMap(),
	}

	return node.New(str)
}

func (c *rectCode) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	dimensions, err := c.dimensions(provider, c.code)
	if err != nil {
		return 0
	}
	proportion := dimensions.Height / dimensions.Width
	width := (c.prop.Percent / 100) * cell.Width
	return proportion * width
}

func (c *rectCode) SetConfig(config *entity.Config) {
	c.config = config
}
