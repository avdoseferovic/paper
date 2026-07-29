package table

import (
	"cmp"

	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// computeRowHeights runs the two-pass rowspan height algorithm and caches results.
func (t *Table) computeRowHeights(provider core.Provider, cell *entity.Cell) {
	tableCell := t.tableCell(cell)
	if len(t.rowHeights) == t.rowCount && t.rowHeightWidth == tableCell.Width {
		return
	}
	heights := t.passOne(provider, &tableCell)
	t.passTwo(provider, &tableCell, heights)
	t.rowHeights = heights
	t.rowHeightWidth = tableCell.Width
}

// passOne computes heights for single-row cells.
func (t *Table) passOne(provider core.Provider, cell *entity.Cell) []float64 {
	heights := make([]float64, t.rowCount)
	defH := t.defaultRowHeight(provider)
	flat := 0
	for r, row := range t.declared {
		for _, c := range row {
			originCol := t.originColumn(flat)
			flat++
			if c.Rowspan > 1 {
				continue
			}
			needed := c.Height
			if c.Content != nil {
				width := t.columnSpanWidth(cell.Width, originCol, c.Colspan)
				inner := paddedTableCell(0, 0, width, cell.Height, c.Style)
				if h := c.Content.GetHeight(provider, &inner) + verticalPadding(c.Style); h > needed {
					needed = h
				}
			}
			heights[r] = max(heights[r], needed)
		}
		heights[r] = cmp.Or(heights[r], defH)
	}
	return heights
}

// passTwo distributes rowspan cells' height surplus across spanned rows.
func (t *Table) passTwo(provider core.Provider, cell *entity.Cell, heights []float64) {
	flat := 0
	for r, row := range t.declared {
		for _, c := range row {
			originCol := t.originColumn(flat)
			flat++
			if c.Rowspan <= 1 {
				continue
			}
			needed := c.Height
			if c.Content != nil {
				width := t.columnSpanWidth(cell.Width, originCol, c.Colspan)
				inner := paddedTableCell(0, 0, width, cell.Height, c.Style)
				if h := c.Content.GetHeight(provider, &inner) + verticalPadding(c.Style); h > needed {
					needed = h
				}
			}
			if needed <= 0 {
				continue
			}
			spanEnd := min(r+c.Rowspan, t.rowCount)
			sum := 0.0
			for i := r; i < spanEnd; i++ {
				sum += heights[i]
			}
			if needed <= sum {
				continue
			}
			distributeSpanSurplus(heights[r:spanEnd], needed-sum, sum)
		}
	}
}

func distributeSpanSurplus(heights []float64, delta, sum float64) {
	if sum > 0 {
		for i := range heights {
			heights[i] += delta * heights[i] / sum
		}
		return
	}

	each := delta / float64(len(heights))
	for i := range heights {
		heights[i] += each
	}
}

func paddedTableCell(x, y, width, height float64, style *props.Cell) entity.Cell {
	inner := layout.ApplyCellMargins(entity.Cell{X: x, Y: y, Width: width, Height: height}, style)
	if style == nil {
		return inner
	}
	inner.X += style.PaddingLeft
	inner.Y += style.PaddingTop
	inner.Width -= style.PaddingLeft + style.PaddingRight
	inner.Height -= style.PaddingTop + style.PaddingBottom
	inner.Width = max(inner.Width, 0)
	inner.Height = max(inner.Height, 0)
	return inner
}

func verticalPadding(style *props.Cell) float64 {
	if style == nil {
		return 0
	}
	return style.PaddingTop + style.PaddingBottom + layout.VerticalCellMargins(style)
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
