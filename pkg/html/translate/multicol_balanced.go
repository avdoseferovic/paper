package translate

import (
	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

type balancedMultiColumnRow struct {
	columnCount int
	gapCols     int
	columnSizes []int
	rows        []core.Row
	config      *entity.Config
	style       *props.Cell

	cachedWidth float64
	cached      core.Row
}

func newBalancedMultiColumnRow(columnCount, gapCols int, columnSizes []int, rows []core.Row) core.Row {
	return &balancedMultiColumnRow{
		columnCount: columnCount,
		gapCols:     gapCols,
		columnSizes: append([]int(nil), columnSizes...),
		rows:        append([]core.Row(nil), rows...),
	}
}

func (r *balancedMultiColumnRow) SetConfig(config *entity.Config) {
	r.config = config
	for _, child := range r.rows {
		child.SetConfig(config)
	}
	if r.cached != nil {
		r.cached.SetConfig(config)
	}
}

func (r *balancedMultiColumnRow) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type: "multi_column_row",
		Details: map[string]any{
			"column_count": r.columnCount,
			"gap_columns":  r.gapCols,
		},
	}
	n := node.New(str)
	if child := r.cachedOrFallback(); child != nil {
		n.AddNext(child.GetStructure())
	}
	return n
}

func (r *balancedMultiColumnRow) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	child := r.build(provider, cell)
	if child == nil {
		return 0
	}
	return child.GetHeight(provider, cell)
}

func (r *balancedMultiColumnRow) Render(provider core.Provider, cell entity.Cell) {
	child := r.build(provider, &cell)
	if child == nil {
		return
	}
	child.Render(provider, cell)
}

func (r *balancedMultiColumnRow) Add(...core.Col) core.Row { return r }

func (r *balancedMultiColumnRow) WithStyle(style *props.Cell) core.Row {
	r.style = props.CloneCell(style)
	r.cached = nil
	r.cachedWidth = 0
	return r
}

func (r *balancedMultiColumnRow) GetColumns() []core.Col {
	child := r.cachedOrFallback()
	if child == nil {
		return nil
	}
	return child.GetColumns()
}

func (r *balancedMultiColumnRow) cachedOrFallback() core.Row {
	if r.cached != nil {
		return r.cached
	}
	return r.build(nil, &entity.Cell{Width: defaultFlexContentWidthMM})
}

func (r *balancedMultiColumnRow) build(provider core.Provider, cell *entity.Cell) core.Row {
	if r.columnCount <= 0 {
		return nil
	}
	width := defaultFlexContentWidthMM
	if cell != nil && cell.Width > 0 {
		width = cell.Width
	}
	if r.cached != nil && r.cachedWidth == width {
		return r.cached
	}

	columnWidth := r.measureColumnWidth(width)
	var columns [][]core.Row
	if provider == nil {
		columns = distributeColumnRows(r.rows, r.columnCount)
	} else {
		columns = distributeBalancedColumnRows(r.rows, r.columnCount, provider, &entity.Cell{
			Width:  columnWidth,
			Height: cellHeight(cell),
		})
	}

	built := row.New()
	for i, rows := range columns {
		if i > 0 && r.gapCols > 0 {
			built = built.Add(col.New(r.gapCols))
		}
		size := 1
		if i < len(r.columnSizes) && r.columnSizes[i] > 0 {
			size = r.columnSizes[i]
		}
		built = built.Add(col.New(size).Add(newFlexCellContent(rows)))
	}
	if r.style != nil {
		built = built.WithStyle(r.style)
	}
	if r.config != nil {
		built.SetConfig(r.config)
	}
	r.cached = built
	r.cachedWidth = width
	return built
}

func (r *balancedMultiColumnRow) measureColumnWidth(width float64) float64 {
	if width <= 0 {
		width = defaultFlexContentWidthMM
	}
	if len(r.columnSizes) == 0 || r.columnSizes[0] <= 0 {
		return width / float64(max(r.columnCount, 1))
	}
	return layout.UnitWidth(width, r.columnSizes[0], r.gridSize())
}

func (r *balancedMultiColumnRow) gridSize() int {
	if r.config == nil {
		return layout.DefaultGridSize()
	}
	return layout.NormalizeGridSize(r.config.MaxGridSize)
}

func cellHeight(cell *entity.Cell) float64 {
	if cell == nil {
		return 0
	}
	return cell.Height
}

func distributeBalancedColumnRows(
	rows []core.Row,
	columnCount int,
	provider core.Provider,
	measureCell *entity.Cell,
) [][]core.Row {
	columns := make([][]core.Row, columnCount)
	if columnCount <= 0 || len(rows) == 0 {
		return columns
	}

	heights := make([]float64, len(rows))
	total := 0.0
	for i, r := range rows {
		heights[i] = r.GetHeight(provider, measureCell)
		total += heights[i]
	}
	if total <= 0 {
		return distributeColumnRows(rows, columnCount)
	}

	target := total / float64(columnCount)
	column := 0
	columnHeight := 0.0
	for i, r := range rows {
		if columnHeight+heights[i] > target && column < columnCount-1 && columnHeight > 0 {
			column++
			columnHeight = 0
		}
		columns[column] = append(columns[column], r)
		columnHeight += heights[i]
	}
	return columns
}
