package translate

import (
	"strconv"

	"github.com/avdoseferovic/paper/v2/pkg/components/col"
	"github.com/avdoseferovic/paper/v2/pkg/components/richtext"
	"github.com/avdoseferovic/paper/v2/pkg/components/row"
	"github.com/avdoseferovic/paper/v2/pkg/components/table"
	"github.com/avdoseferovic/paper/v2/pkg/consts/align"
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/html/css"
	"github.com/avdoseferovic/paper/v2/pkg/html/dom"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

// tableRows converts a <table> element into one Paper row containing a Table component.
// When a <caption> child exists, it is emitted as a centred row above the table.
// When <colgroup>/<col> children exist, they are detected and logged as a feature
// the v1 table builder cannot honour (explicit column widths are not supported).
func (tr *translator) tableRows(n *dom.Node) []core.Row {
	var out []core.Row
	for _, c := range n.Children() {
		switch c.Tag() {
		case "caption":
			out = append(out, tr.captionRow(c))
		case "colgroup":
			tr.unsupported("table.colgroup", "explicit column widths not supported in v1")
		}
	}
	cells := tr.buildTableMatrix(n)
	if len(cells) == 0 {
		return out
	}
	tbl, err := table.New(cells)
	if err != nil {
		return out
	}
	c := col.New().Add(tbl)
	out = append(out, row.New().Add(c))
	return out
}

// captionRow renders a <caption> as a centred row above the table.
func (tr *translator) captionRow(n *dom.Node) core.Row {
	runs := inlineRuns(n)
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	rt := richtext.New(runs, props.RichText{Align: align.Center})
	return row.New().Add(col.New().Add(rt))
}

func (tr *translator) buildTableMatrix(n *dom.Node) [][]table.Cell {
	tableStyle := computeNodeStyleRooted(tr.sheet, n, nil, tr.rootStyle)
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

	runs := inlineRuns(td)
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
	cellProp := &props.Cell{
		PaddingTop:    cellStyle.PaddingTop,
		PaddingRight:  cellStyle.PaddingRight,
		PaddingBottom: cellStyle.PaddingBottom,
		PaddingLeft:   cellStyle.PaddingLeft,
	}

	effectiveBg := cellStyle.BackgroundColor
	if effectiveBg == nil && rowStyle != nil {
		effectiveBg = rowStyle.BackgroundColor
	}
	if effectiveBg != nil {
		cellProp.BackgroundColor = toPropsColor(effectiveBg, effectiveOpacity(cellStyle))
	}

	if cellProp.BackgroundColor == nil &&
		cellProp.PaddingTop == 0 && cellProp.PaddingRight == 0 &&
		cellProp.PaddingBottom == 0 && cellProp.PaddingLeft == 0 {
		return nil
	}
	return cellProp
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
