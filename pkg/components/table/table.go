// Package table implements a PDF table component with colspan, rowspan, and per-cell styling.
package table

import (
	"errors"

	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
	"github.com/avdoseferovic/paper/v2/pkg/tree/node"
)

var (
	// ErrTableSpanOverlap is returned when colspan/rowspan cells overlap each other.
	ErrTableSpanOverlap = errors.New("table: cell spans overlap")
	// ErrTableEmpty is returned when the cell matrix is empty.
	ErrTableEmpty = errors.New("table: empty cell matrix")
)

// Cell is one entry in the table grid.
type Cell struct {
	Content core.Component
	Colspan int
	Rowspan int
	Style   *props.Cell
}

// Table is a core.Component that renders a grid with span support.
type Table struct {
	declared   [][]Cell // original declaration
	grid       [][]int  // normalized: flat index into declared cells; -1 = occupied by span
	rowCount   int
	colCount   int
	config     *entity.Config
	rowHeights []float64 // computed by two-pass algorithm
}

// New validates spans, normalises the grid, and builds the Table component.
func New(cells [][]Cell, _ ...any) (*Table, error) {
	cells = cloneCells(cells)
	normaliseSpans(cells)

	colCount, err := deriveColCount(cells)
	if err != nil {
		return nil, err
	}

	grid, err := buildGrid(cells, len(cells), colCount)
	if err != nil {
		return nil, err
	}

	return &Table{
		declared: cells,
		grid:     grid,
		rowCount: len(cells),
		colCount: colCount,
	}, nil
}

func cloneCells(cells [][]Cell) [][]Cell {
	if cells == nil {
		return nil
	}
	clone := make([][]Cell, len(cells))
	for r := range cells {
		clone[r] = make([]Cell, len(cells[r]))
		for c := range cells[r] {
			clone[r][c] = cells[r][c]
			clone[r][c].Style = props.CloneCell(cells[r][c].Style)
		}
	}
	return clone
}

// ColCount returns the number of columns determined from the normalised grid.
func (t *Table) ColCount() int { return t.colCount }

// SetConfig propagates Paper config to all cell components.
func (t *Table) SetConfig(config *entity.Config) {
	t.config = config
	for _, row := range t.declared {
		for _, c := range row {
			if c.Content != nil {
				c.Content.SetConfig(config)
			}
		}
	}
}

// GetStructure returns the component node for snapshots/debugging.
func (t *Table) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type: "table",
		Details: map[string]any{
			"rows": t.rowCount,
			"cols": t.colCount,
		},
	}
	return node.New(str)
}

// GetHeight computes and returns the total table height.
func (t *Table) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	t.computeRowHeights(provider, cell)
	total := 0.0
	for _, h := range t.rowHeights {
		total += h
	}
	return total
}
