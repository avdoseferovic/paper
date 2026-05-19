package translate

import (
	"strconv"

	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/richtext"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/table"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// tableRows converts a <table> element into one Maroto row containing a Table component.
func tableRows(n *dom.Node) []core.Row {
	cells := buildTableMatrix(n)
	if len(cells) == 0 {
		return nil
	}
	tbl, err := table.New(cells)
	if err != nil {
		return nil
	}
	c := col.New().Add(tbl)
	return []core.Row{row.New().Add(c)}
}

func buildTableMatrix(n *dom.Node) [][]table.Cell {
	var matrix [][]table.Cell
	for _, child := range n.Children() {
		switch child.Tag() {
		case "thead", "tbody", "tfoot":
			matrix = append(matrix, collectRows(child)...)
		case "tr":
			if rowCells := buildRow(child); rowCells != nil {
				matrix = append(matrix, rowCells)
			}
		}
	}
	return matrix
}

func collectRows(parent *dom.Node) [][]table.Cell {
	var rows [][]table.Cell
	for _, c := range parent.Children() {
		if c.Tag() == "tr" {
			if cells := buildRow(c); cells != nil {
				rows = append(rows, cells)
			}
		}
	}
	return rows
}

func buildRow(trNode *dom.Node) []table.Cell {
	var cells []table.Cell
	for _, c := range trNode.Children() {
		tag := c.Tag()
		if tag != "td" && tag != "th" {
			continue
		}
		cells = append(cells, buildCell(c))
	}
	return cells
}

func buildCell(td *dom.Node) table.Cell {
	colspan := atoiOr(td.Attr("colspan"), 1)
	rowspan := atoiOr(td.Attr("rowspan"), 1)

	runs := inlineRuns(td)
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	content := richtext.New(runs)
	return table.Cell{
		Content: content,
		Colspan: colspan,
		Rowspan: rowspan,
	}
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
