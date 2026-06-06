package translate

import (
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
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
		rows = append(rows, tr.summaryRow(summary))
	}
	for _, c := range body {
		rows = append(rows, tr.blockRows(c)...)
	}
	return rows
}

func (tr *translator) summaryRow(n *dom.Node) core.Row {
	runs := tr.inlineRuns(n)
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
