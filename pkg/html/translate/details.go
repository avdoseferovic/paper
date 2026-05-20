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

// detailsRows treats <details> as always expanded: <summary> renders as a bold
// row followed by the rest of the body's rows. <summary> is bold and slightly
// larger; the open attribute is ignored (PDFs have no interactive toggle).
func (tr *translator) detailsRows(n *dom.Node) []core.Row {
	var summary *dom.Node
	var body []*dom.Node
	for _, c := range n.Children() {
		if summary == nil && c.Tag() == "summary" {
			summary = c
			continue
		}
		body = append(body, c)
	}

	var rows []core.Row
	if summary != nil {
		rows = append(rows, summaryRow(summary))
	}
	for _, c := range body {
		rows = append(rows, tr.blockRows(c)...)
	}
	return rows
}

func summaryRow(n *dom.Node) core.Row {
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
