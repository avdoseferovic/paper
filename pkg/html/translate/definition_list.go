package translate

import (
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/richtext"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// definitionListRows converts <dl> with <dt>/<dd> children into a sequence of rows.
// <dt> renders bold; <dd> renders indented (~5mm left padding).
func (tr *translator) definitionListRows(n *dom.Node) []core.Row {
	var rows []core.Row
	for _, c := range n.Children() {
		switch c.Tag() {
		case "dt":
			rows = append(rows, dtRow(c))
		case "dd":
			rows = append(rows, ddRow(c))
		}
	}
	return rows
}

func dtRow(n *dom.Node) core.Row {
	runs := inlineRuns(n)
	for i := range runs {
		if runs[i].Style == fontstyle.Normal {
			runs[i].Style = fontstyle.Bold
		}
	}
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	rt := richtext.New(runs)
	return row.New().Add(col.New().Add(rt))
}

func ddRow(n *dom.Node) core.Row {
	runs := inlineRuns(n)
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	rt := richtext.New(runs, props.RichText{Left: 5}) // 5mm indent
	return row.New().Add(col.New().Add(rt))
}
