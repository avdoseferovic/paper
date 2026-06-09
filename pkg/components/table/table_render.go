package table

import (
	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// Render draws the table into the PDF cell.
func (t *Table) Render(provider core.Provider, cell *entity.Cell) {
	t.computeRowHeights(provider, cell)
	y := cell.Y

	for r := range t.rowCount {
		rowH := t.rowHeights[r]
		x := cell.X
		for c := range t.colCount {
			colWidth := t.columnWidth(cell.Width, c)
			slot := t.grid[r][c]
			// Skip empty and spanned slots. Render each declared cell only at its origin.
			if slot < 0 {
				x += colWidth
				continue
			}
			declCell := t.cellAtFlatIndex(slot)
			if declCell == nil {
				x += colWidth
				continue
			}
			w := t.columnSpanWidth(cell.Width, c, declCell.Colspan)
			innerCell := paddedTableCell(x, y, w, rowH, declCell.Style)
			if declCell.Style != nil {
				paintCell := layout.ApplyCellMargins(entity.Cell{X: x, Y: y, Width: w, Height: rowH}, declCell.Style)
				if pp, ok := provider.(core.PositionProvider); ok {
					pp.SetCursor(paintCell.X, paintCell.Y)
				}
				provider.CreateCol(paintCell.Width, paintCell.Height, t.config, declCell.Style)
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
