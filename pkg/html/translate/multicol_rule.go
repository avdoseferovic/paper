package translate

import (
	"strings"

	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

type columnRuleRow struct {
	child       core.Row
	columnCount int
	gapCols     int
	ruleWidth   float64
	ruleStyle   string
	ruleColor   *props.Color
	config      *entity.Config
}

func newColumnRuleRow(child core.Row, columnCount, gapCols int, ruleWidth float64, ruleStyle string, ruleColor *props.Color) core.Row {
	if child == nil || columnCount <= 1 || ruleWidth <= 0 || isColumnRuleSuppressed(ruleStyle) {
		return child
	}
	if ruleStyle == "" {
		ruleStyle = "solid"
	}
	return &columnRuleRow{
		child:       child,
		columnCount: columnCount,
		gapCols:     gapCols,
		ruleWidth:   ruleWidth,
		ruleStyle:   strings.ToLower(strings.TrimSpace(ruleStyle)),
		ruleColor:   props.CloneColor(ruleColor),
	}
}

func (r *columnRuleRow) SetConfig(config *entity.Config) {
	r.config = config
	if r.child != nil {
		r.child.SetConfig(config)
	}
}

func (r *columnRuleRow) GetStructure() *node.Node[core.Structure] {
	details := map[string]any{
		"column_count": r.columnCount,
		"gap_columns":  r.gapCols,
		"rule_width":   r.ruleWidth,
		"rule_style":   r.ruleStyle,
	}
	if r.ruleColor != nil {
		details["rule_color"] = r.ruleColor.ToString()
	}
	n := node.New(core.Structure{
		Type:    "column_rule_row",
		Details: details,
	})
	if r.child != nil {
		n.AddNext(r.child.GetStructure())
	}
	return n
}

func (r *columnRuleRow) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if r.child == nil {
		return 0
	}
	return r.child.GetHeight(provider, cell)
}

func (r *columnRuleRow) Render(provider core.Provider, cell entity.Cell) {
	if r.child == nil {
		return
	}
	height := r.child.GetHeight(provider, &cell)
	renderCell := cell
	renderCell.Height = height
	r.child.Render(provider, renderCell)
	r.renderRules(provider, renderCell)
}

func (r *columnRuleRow) Add(cols ...core.Col) core.Row {
	if r.child != nil {
		r.child = r.child.Add(cols...)
	}
	return r
}

func (r *columnRuleRow) WithStyle(style *props.Cell) core.Row {
	if r.child != nil {
		r.child = r.child.WithStyle(style)
	}
	return r
}

func (r *columnRuleRow) GetColumns() []core.Col {
	if r.child == nil {
		return nil
	}
	return r.child.GetColumns()
}

func (r *columnRuleRow) renderRules(provider core.Provider, cell entity.Cell) {
	if provider == nil || cell.Height <= 0 || r.ruleWidth <= 0 || isColumnRuleSuppressed(r.ruleStyle) {
		return
	}
	for _, x := range r.rulePositions(cell.Width) {
		provider.AddLine(&entity.Cell{
			X:      cell.X + x,
			Y:      cell.Y,
			Width:  0,
			Height: cell.Height,
		}, &props.Line{
			Color:         props.CloneColor(r.ruleColor),
			Style:         cssBorderStyleToLineStyle(r.ruleStyle),
			Thickness:     r.ruleWidth,
			Orientation:   consts.OrientationVertical,
			OffsetPercent: 0,
			SizePercent:   100,
		})
	}
}

func (r *columnRuleRow) rulePositions(width float64) []float64 {
	if r.child == nil || r.columnCount <= 1 || width <= 0 {
		return nil
	}
	cols := r.child.GetColumns()
	if len(cols) == 0 {
		return nil
	}
	units := make([]int, len(cols))
	for i, c := range cols {
		units[i] = c.GetSize()
	}
	plan := layout.ManualUnits(units, r.gridSize())

	positions := make([]float64, 0, r.columnCount-1)
	x := 0.0
	unitIdx := 0
	for column := 0; column < r.columnCount-1 && unitIdx < len(plan.Units); column++ {
		x += layout.UnitWidth(width, plan.Units[unitIdx], plan.GridSize)
		unitIdx++
		if r.gapCols > 0 && unitIdx < len(plan.Units) {
			gapWidth := layout.UnitWidth(width, plan.Units[unitIdx], plan.GridSize)
			positions = append(positions, x+gapWidth/2)
			x += gapWidth
			unitIdx++
			continue
		}
		positions = append(positions, x)
	}
	return positions
}

func (r *columnRuleRow) gridSize() int {
	if r.config == nil {
		return layout.DefaultGridSize()
	}
	return layout.NormalizeGridSize(r.config.MaxGridSize)
}

func isColumnRuleSuppressed(style string) bool {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case cssValueNone, cssValueHidden:
		return true
	default:
		return false
	}
}
