package table

import (
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/props"
)

const defaultCollapsedBorderThickness = 0.2

func (t *Table) collapsedCellStyle(cell *Cell, row, col int) *props.Cell {
	if t == nil || cell == nil || cell.Style == nil || !t.borderCollapse {
		if cell == nil {
			return nil
		}
		return cell.Style
	}
	style := props.CloneCell(cell.Style)
	expandLegacyBorders(style)
	clearBorderRadius(style)
	if col+max(cell.Colspan, 1) < t.colCount {
		style.BorderRightThickness = 0
		style.BorderRightColor = nil
		style.BorderRightStyle = ""
	}
	if row+max(cell.Rowspan, 1) < t.rowCount {
		style.BorderBottomThickness = 0
		style.BorderBottomColor = nil
		style.BorderBottomStyle = ""
	}
	return style
}

func expandLegacyBorders(style *props.Cell) {
	if style == nil || style.BorderType == border.None {
		return
	}
	thickness := style.BorderThickness
	if thickness <= 0 {
		thickness = defaultCollapsedBorderThickness
	}
	if style.BorderType.HasTop() && style.BorderTopThickness == 0 {
		style.BorderTopThickness = thickness
		style.BorderTopColor = props.CloneColor(style.BorderColor)
		style.BorderTopStyle = style.LineStyle
	}
	if style.BorderType.HasRight() && style.BorderRightThickness == 0 {
		style.BorderRightThickness = thickness
		style.BorderRightColor = props.CloneColor(style.BorderColor)
		style.BorderRightStyle = style.LineStyle
	}
	if style.BorderType.HasBottom() && style.BorderBottomThickness == 0 {
		style.BorderBottomThickness = thickness
		style.BorderBottomColor = props.CloneColor(style.BorderColor)
		style.BorderBottomStyle = style.LineStyle
	}
	if style.BorderType.HasLeft() && style.BorderLeftThickness == 0 {
		style.BorderLeftThickness = thickness
		style.BorderLeftColor = props.CloneColor(style.BorderColor)
		style.BorderLeftStyle = style.LineStyle
	}
	style.BorderType = border.None
	style.BorderThickness = 0
}

func clearBorderRadius(style *props.Cell) {
	if style == nil {
		return
	}
	style.BorderRadius = 0
	style.BorderRadiusTopLeft = 0
	style.BorderRadiusTopRight = 0
	style.BorderRadiusBottomRight = 0
	style.BorderRadiusBottomLeft = 0
}
