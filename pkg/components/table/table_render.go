package table

import (
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
)

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
			w := colWidth * float64(declCell.Colspan)
			innerCell := paddedTableCell(x, y, w, rowH, declCell.Style)
			if declCell.Style != nil {
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
