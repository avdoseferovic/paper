package translate

import (
	"strconv"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// tableRows converts a <table> element into one Paper row containing a Table component.
// When a <caption> child exists, it is emitted as a centred row above the table.
// When <colgroup>/<col> children provide widths, they are mapped to relative
// table column widths.
func (tr *translator) tableRows(n *dom.Node) []core.Row {
	var out []core.Row
	for _, c := range n.Children() {
		if c.Tag() == "caption" {
			out = append(out, tr.captionRow(c))
		}
	}
	cells := tr.buildTableMatrix(n)
	if len(cells) == 0 {
		return out
	}
	var opts []table.Option
	if widths := tr.tableColumnWidths(n); len(widths) > 0 {
		opts = append(opts, table.WithColumnWidths(widths))
	}
	tbl, err := table.New(cells, opts...)
	if err != nil {
		return out
	}
	c := col.New().Add(tbl)
	out = append(out, row.New().Add(c))
	return out
}

func (tr *translator) tableColumnWidths(n *dom.Node) []float64 {
	tableStyle := computeNodeStyleRooted(tr.sheet, n, tr.rootStyle)
	refWidth := tableStyle.Width
	if refWidth <= 0 {
		refWidth = tr.contentWidthMM
	}
	if refWidth <= 0 {
		refWidth = 100
	}

	var widths []float64
	for _, group := range n.Children() {
		if group.Tag() != "colgroup" {
			continue
		}
		groupStyle := computeNodeStyleCtx(tr.sheet, group, tableStyle, refWidth)
		groupWidth := tableColumnWidth(group, groupStyle, refWidth)
		groupHasCol := false
		for _, child := range group.Children() {
			if child.Tag() != "col" {
				continue
			}
			groupHasCol = true
			colStyle := computeNodeStyleCtx(tr.sheet, child, groupStyle, refWidth)
			width := tableColumnWidth(child, colStyle, refWidth)
			if width <= 0 {
				width = groupWidth
			}
			span := atoiOr(child.Attr("span"), 1)
			for range span {
				widths = append(widths, width)
			}
		}
		if !groupHasCol {
			span := atoiOr(group.Attr("span"), 1)
			for range span {
				widths = append(widths, groupWidth)
			}
		}
	}
	for _, width := range widths {
		if width > 0 {
			return widths
		}
	}
	return nil
}

func tableColumnWidth(n *dom.Node, style *css.ComputedStyle, refWidth float64) float64 {
	if style != nil && style.Width > 0 {
		return style.Width
	}
	if n == nil {
		return 0
	}
	if width := n.Attr("width"); width != "" {
		fontSize := 0.0
		if style != nil {
			fontSize = style.FontSize
		}
		return css.ParseLengthCtx(width, fontSize, refWidth)
	}
	return 0
}

// captionRow renders a <caption> as a centred row above the table.
func (tr *translator) captionRow(n *dom.Node) core.Row {
	style := computeNodeStyleRooted(tr.sheet, n, tr.rootStyle)
	runs := tr.inlineRunsStyled(n, blockInlineStyle(style))
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	rt := richtext.New(runs, props.RichText{Align: consts.AlignCenter})
	return row.New().Add(col.New().Add(rt))
}

func (tr *translator) buildTableMatrix(n *dom.Node) [][]table.Cell {
	tableStyle := computeNodeStyleRooted(tr.sheet, n, tr.rootStyle)
	var matrix [][]table.Cell
	for _, child := range n.Children() {
		switch child.Tag() {
		case "thead", "tbody", "tfoot":
			matrix = append(matrix, tr.collectRows(child, tableStyle)...)
		case "tr":
			if rowCells := tr.buildRow(child, tableStyle); rowCells != nil {
				matrix = append(matrix, rowCells)
			}
		}
	}
	return matrix
}

func (tr *translator) collectRows(parent *dom.Node, tableStyle *css.ComputedStyle) [][]table.Cell {
	var rows [][]table.Cell
	for _, c := range parent.Children() {
		if c.Tag() == "tr" {
			if cells := tr.buildRow(c, tableStyle); cells != nil {
				rows = append(rows, cells)
			}
		}
	}
	return rows
}

// buildRow builds the cells for a <tr>, propagating the row's computed style as a
// fallback for background-color and color when individual cells have no own style.
func (tr *translator) buildRow(trNode *dom.Node, parentStyle *css.ComputedStyle) []table.Cell {
	rowStyle := computeNodeStyle(tr.sheet, trNode, parentStyle)
	var cells []table.Cell
	for _, c := range trNode.Children() {
		tag := c.Tag()
		if tag != "td" && tag != "th" {
			continue
		}
		cells = append(cells, tr.buildCell(c, rowStyle))
	}
	return cells
}

// buildCell builds a single table.Cell, using rowStyle as a background/color fallback.
func (tr *translator) buildCell(td *dom.Node, rowStyle *css.ComputedStyle) table.Cell {
	colspan := atoiOr(td.Attr("colspan"), 1)
	rowspan := atoiOr(td.Attr("rowspan"), 1)

	cellStyle := computeNodeStyle(tr.sheet, td, rowStyle)

	runs := tr.inlineRunsStyled(td, blockInlineStyle(cellStyle))
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}

	// Propagate row-level color to runs that have no own color.
	effectiveColor := cellStyle.Color
	if effectiveColor == nil {
		effectiveColor = rowStyle.Color
	}
	if effectiveColor != nil {
		op := effectiveOpacity(cellStyle)
		for i := range runs {
			if runs[i].Color == nil {
				runs[i].Color = toPropsColor(effectiveColor, op)
			}
		}
	}

	content := richtext.New(runs)

	cellProp := tableCellStyle(cellStyle, rowStyle)

	return table.Cell{
		Content: content,
		Colspan: colspan,
		Rowspan: rowspan,
		Style:   cellProp,
	}
}

func tableCellStyle(cellStyle, rowStyle *css.ComputedStyle) *props.Cell {
	if cellStyle == nil {
		return nil
	}
	cellProp := baseBlockCellStyle(cellStyle)
	if cellProp == nil {
		cellProp = &props.Cell{}
	}
	cellProp.PaddingTop = cellStyle.PaddingTop
	cellProp.PaddingRight = cellStyle.PaddingRight
	cellProp.PaddingBottom = cellStyle.PaddingBottom
	cellProp.PaddingLeft = cellStyle.PaddingLeft

	if cellProp.BackgroundColor == nil && cellProp.BackgroundGradient == nil && rowStyle != nil && rowStyle.BackgroundColor != nil {
		cellProp.BackgroundColor = toPropsColor(rowStyle.BackgroundColor, effectiveOpacity(cellStyle))
	}

	if isEmptyTableCellStyle(cellProp) {
		return nil
	}
	return cellProp
}

func isEmptyTableCellStyle(cell *props.Cell) bool {
	return cell == nil ||
		(cell.BackgroundColor == nil &&
			cell.BorderColor == nil &&
			cell.BorderType == border.None &&
			cell.BorderThickness == 0 &&
			cell.PaddingTop == 0 &&
			cell.PaddingRight == 0 &&
			cell.PaddingBottom == 0 &&
			cell.PaddingLeft == 0 &&
			cell.BorderTopColor == nil &&
			cell.BorderRightColor == nil &&
			cell.BorderBottomColor == nil &&
			cell.BorderLeftColor == nil &&
			cell.BorderTopThickness == 0 &&
			cell.BorderRightThickness == 0 &&
			cell.BorderBottomThickness == 0 &&
			cell.BorderLeftThickness == 0 &&
			cell.BackgroundGradient == nil &&
			cell.BackgroundImage == nil &&
			len(cell.BoxShadow) == 0 &&
			cell.OutlineWidth == 0 &&
			cell.BorderRadius == 0 &&
			cell.BorderRadiusTopLeft == 0 &&
			cell.BorderRadiusTopRight == 0 &&
			cell.BorderRadiusBottomLeft == 0 &&
			cell.BorderRadiusBottomRight == 0)
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
