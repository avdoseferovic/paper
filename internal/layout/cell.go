package layout

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// ApplyCellMargins returns the visible content box inside the allocated cell.
func ApplyCellMargins(cell entity.Cell, style *props.Cell) entity.Cell {
	if style == nil {
		return cell
	}

	left := nonNegative(style.MarginLeft)
	right := nonNegative(style.MarginRight)
	top := nonNegative(style.MarginTop)
	bottom := nonNegative(style.MarginBottom)

	cell.X += left
	cell.Y += top
	cell.Width -= left + right
	cell.Height -= top + bottom
	if cell.Width < 0 {
		cell.Width = 0
	}
	if cell.Height < 0 {
		cell.Height = 0
	}
	return cell
}

func VerticalCellMargins(style *props.Cell) float64 {
	if style == nil {
		return 0
	}
	return nonNegative(style.MarginTop) + nonNegative(style.MarginBottom)
}

func nonNegative(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}
