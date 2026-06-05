// Package table implements a PDF table component with colspan, rowspan, and per-cell styling.
package table

import (
	"errors"

	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/paper/v2/pkg/core"
	"github.com/johnfercher/paper/v2/pkg/core/entity"
	"github.com/johnfercher/paper/v2/pkg/props"
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

// Render draws the table into the PDF cell.
func (t *Table) Render(provider core.Provider, cell *entity.Cell) {
	t.computeRowHeights(provider, cell)
	colWidth := cell.Width / float64(t.colCount)
	y := cell.Y

	for r := range t.rowCount {
		rowH := t.rowHeights[r]
		x := cell.X
		for c := range t.colCount {
			slot := t.grid[r][c]
			// Skip empty (-1) and spanned slots (spannedMarker) — only render at the
			// declared cell's origin position so a colspan/rowspan cell draws once.
			if slot < 0 {
				x += colWidth
				continue
			}
			declCell := t.cellAtFlatIndex(slot)
			if declCell == nil {
				x += colWidth
				continue
			}
			w := colWidth * float64(declCell.Colspan)
			innerCell := paddedTableCell(x, y, w, rowH, declCell.Style)
			if declCell.Style != nil {
				// Reset the pen to the cell origin before CreateCol so CellFormat
				// paints the styled background at the right (x, y) — otherwise the
				// pen drifts across rows and cells when only some cells are styled.
				if pp, ok := provider.(core.PositionProvider); ok {
					pp.SetCursor(x, y)
				}
				provider.CreateCol(w, rowH, t.config, declCell.Style)
			}
			if declCell.Content != nil {
				declCell.Content.Render(provider, &innerCell)
			}
			x += colWidth
		}
		provider.CreateRow(rowH)
		y += rowH
	}
}

// computeRowHeights runs the two-pass rowspan height algorithm and caches results.
func (t *Table) computeRowHeights(provider core.Provider, cell *entity.Cell) {
	if len(t.rowHeights) == t.rowCount {
		return
	}
	colWidth := cell.Width / float64(t.colCount)
	heights := t.passOne(provider, cell, colWidth)
	t.passTwo(provider, cell, colWidth, heights)
	t.rowHeights = heights
}

// passOne computes heights for single-row cells.
func (t *Table) passOne(provider core.Provider, cell *entity.Cell, colWidth float64) []float64 {
	heights := make([]float64, t.rowCount)
	defH := t.defaultRowHeight(provider)
	for r, row := range t.declared {
		for _, c := range row {
			if c.Rowspan > 1 || c.Content == nil {
				continue
			}
			inner := paddedTableCell(0, 0, colWidth*float64(c.Colspan), cell.Height, c.Style)
			if h := c.Content.GetHeight(provider, &inner) + verticalPadding(c.Style); h > heights[r] {
				heights[r] = h
			}
		}
		if heights[r] == 0 {
			heights[r] = defH
		}
	}
	return heights
}

// passTwo distributes rowspan cells' height surplus across spanned rows.
func (t *Table) passTwo(provider core.Provider, cell *entity.Cell, colWidth float64, heights []float64) {
	for r, row := range t.declared {
		for _, c := range row {
			if c.Rowspan <= 1 || c.Content == nil {
				continue
			}
			inner := paddedTableCell(0, 0, colWidth*float64(c.Colspan), cell.Height, c.Style)
			needed := c.Content.GetHeight(provider, &inner) + verticalPadding(c.Style)
			spanEnd := min(r+c.Rowspan, t.rowCount)
			sum := 0.0
			for i := r; i < spanEnd; i++ {
				sum += heights[i]
			}
			if needed <= sum {
				continue
			}
			delta := needed - sum
			if sum > 0 {
				for i := r; i < spanEnd; i++ {
					heights[i] += delta * heights[i] / sum
				}
			} else {
				each := delta / float64(spanEnd-r)
				for i := r; i < spanEnd; i++ {
					heights[i] += each
				}
			}
		}
	}
}

func paddedTableCell(x, y, width, height float64, style *props.Cell) entity.Cell {
	inner := entity.Cell{X: x, Y: y, Width: width, Height: height}
	if style == nil {
		return inner
	}
	inner.X += style.PaddingLeft
	inner.Y += style.PaddingTop
	inner.Width -= style.PaddingLeft + style.PaddingRight
	inner.Height -= style.PaddingTop + style.PaddingBottom
	if inner.Width < 0 {
		inner.Width = 0
	}
	if inner.Height < 0 {
		inner.Height = 0
	}
	return inner
}

func verticalPadding(style *props.Cell) float64 {
	if style == nil {
		return 0
	}
	return style.PaddingTop + style.PaddingBottom
}

func (t *Table) defaultRowHeight(provider core.Provider) float64 {
	if t.config == nil || t.config.DefaultFont == nil {
		return 5.0
	}
	return provider.GetFontHeight(t.config.DefaultFont)
}

func (t *Table) cellAtFlatIndex(idx int) *Cell {
	flat := 0
	for r := range t.declared {
		for c := range t.declared[r] {
			if flat == idx {
				return &t.declared[r][c]
			}
			flat++
		}
	}
	return nil
}

// normaliseSpans ensures all cells have Colspan >= 1 and Rowspan >= 1.
func normaliseSpans(cells [][]Cell) {
	for r := range cells {
		for c := range cells[r] {
			if cells[r][c].Colspan < 1 {
				cells[r][c].Colspan = 1
			}
			if cells[r][c].Rowspan < 1 {
				cells[r][c].Rowspan = 1
			}
		}
	}
}

// deriveColCount scans all rows to find the true column count accounting for colspans.
func deriveColCount(cells [][]Cell) (int, error) {
	maxCols := 0
	for _, row := range cells {
		total := 0
		for _, cell := range row {
			total += cell.Colspan
		}
		if total > maxCols {
			maxCols = total
		}
	}
	if maxCols == 0 {
		return 0, ErrTableEmpty
	}
	return maxCols, nil
}

// buildGrid fills an occupation matrix: grid[r][c] = flat index of source cell, -1 if spanned.
func buildGrid(cells [][]Cell, rowCount, colCount int) ([][]int, error) {
	occ := make([][]int, rowCount)
	for r := range rowCount {
		occ[r] = make([]int, colCount)
		for c := range colCount {
			occ[r][c] = -1
		}
	}

	flat := 0
	for r, row := range cells {
		col := 0
		for _, cell := range row {
			for col < colCount && occ[r][col] != -1 {
				col++
			}
			if col >= colCount {
				return nil, ErrTableSpanOverlap
			}
			err := markOccupied(occ, r, col, cell, flat, rowCount, colCount)
			if err != nil {
				return nil, err
			}
			col += cell.Colspan
			flat++
		}
	}
	return occ, nil
}

// spannedMarker indicates a grid slot occupied by a cell whose origin is elsewhere.
// Distinct from empty (-1) so Render can skip rendering at non-origin slots.
const spannedMarker = -2

func markOccupied(occ [][]int, startR, startC int, cell Cell, flatIdx, rowCount, colCount int) error {
	for dr := range cell.Rowspan {
		if startR+dr >= rowCount {
			break
		}
		for dc := range cell.Colspan {
			if startC+dc >= colCount {
				break
			}
			if dr == 0 && dc == 0 {
				occ[startR][startC] = flatIdx
				continue
			}
			if occ[startR+dr][startC+dc] != -1 {
				return ErrTableSpanOverlap
			}
			occ[startR+dr][startC+dc] = spannedMarker
		}
	}
	return nil
}
