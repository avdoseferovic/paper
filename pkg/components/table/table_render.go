package table

import (
	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// Render draws the table into the PDF cell.
func (t *Table) Render(provider core.Provider, cell *entity.Cell) {
	tableCell := t.tableCell(cell)
	t.computeRowHeights(provider, &tableCell)
	y := tableCell.Y

	for r := range t.rowCount {
		rowH := t.rowHeights[r]
		x := tableCell.X
		for c := range t.colCount {
			colWidth := t.columnWidth(tableCell.Width, c)
			advanceColumn := func() {
				x += colWidth
				if c < t.colCount-1 {
					x += t.borderSpacingX
				}
			}
			slot := t.grid[r][c]
			// Skip empty and spanned slots. Render each declared cell only at its origin.
			if slot < 0 {
				advanceColumn()
				continue
			}
			declCell := t.cellAtFlatIndex(slot)
			if declCell == nil {
				advanceColumn()
				continue
			}
			w := t.columnSpanWidth(tableCell.Width, c, declCell.Colspan)
			outerCell := explicitCellBox(x, y, w, rowH, declCell)
			innerCell := paddedTableCell(outerCell.X, outerCell.Y, outerCell.Width, outerCell.Height, declCell.Style)
			if declCell.Style != nil {
				paintCell := layout.ApplyCellMargins(outerCell, declCell.Style)
				if pp, ok := provider.(core.PositionProvider); ok {
					pp.SetCursor(paintCell.X, paintCell.Y)
				}
				provider.CreateCol(paintCell.Width, paintCell.Height, t.config, declCell.Style)
			}
			if declCell.Content != nil {
				declCell.Content.Render(provider, &innerCell)
			}
			advanceColumn()
		}
		provider.CreateRow(rowH)
		y += rowH
		if r < t.rowCount-1 {
			y += t.borderSpacingY
		}
	}
}

func explicitCellBox(x, y, width, rowHeight float64, cell *Cell) entity.Cell {
	height := rowHeight
	if cell != nil && cell.Height > 0 && cell.Height < rowHeight {
		height = cell.Height
		switch cell.VerticalAlign {
		case "middle":
			y += (rowHeight - height) / 2
		case "bottom":
			y += rowHeight - height
		}
	}
	return entity.Cell{X: x, Y: y, Width: width, Height: height}
}
