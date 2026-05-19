// Package translate converts a parsed HTML DOM into a slice of Maroto rows.
package translate

import (
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/richtext"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// Translate walks the styled DOM and emits Maroto rows.
func Translate(doc *dom.Document) ([]core.Row, error) {
	if doc == nil {
		return nil, nil
	}
	var rows []core.Row
	doc.Walk(func(n *dom.Node) bool {
		// We only handle top-level block iteration via walk; recursion handled in handleBlock.
		return true
	})
	// Walk body children explicitly to maintain order and handle nesting.
	body := findBody(doc)
	if body == nil {
		return rows, nil
	}
	for _, child := range body.Children() {
		rows = append(rows, blockRows(child)...)
	}
	return rows, nil
}

func findBody(doc *dom.Document) *dom.Node {
	var body *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "body" {
			body = n
			return false
		}
		return true
	})
	if body != nil {
		return body
	}
	// Fallback: iterate root children.
	return nil
}

// blockRows recursively converts a node into block-level Rows.
func blockRows(n *dom.Node) []core.Row {
	if n == nil {
		return nil
	}
	if isDisplayNone(n) {
		return nil
	}

	tag := n.Tag()
	switch tag {
	case "":
		// Text node at block level — wrap into a paragraph-like row.
		return wrapTextRow(n.TextContent())
	case "p", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "pre":
		return []core.Row{paragraphRow(n)}
	case "hr":
		return []core.Row{hrRow()}
	case "table":
		return tableRows(n)
	case "ul", "ol":
		return listRows(n)
	case "br":
		return nil // top-level <br> is a no-op
	default:
		// Container (div, section, article, header, footer, nav, etc.) — flatten children.
		var rows []core.Row
		for _, c := range n.Children() {
			rows = append(rows, blockRows(c)...)
		}
		return rows
	}
}

// paragraphRow converts a block element with inline content into a single auto-height row.
func paragraphRow(n *dom.Node) core.Row {
	runs := inlineRuns(n)
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	applyBlockStyling(n, runs)
	rt := richtext.New(runs)
	c := col.New().Add(rt)
	return row.New().Add(c)
}

// hrRow produces a thin row containing a horizontal line.
func hrRow() core.Row {
	l := line.New()
	c := col.New().Add(l)
	return row.New(1).Add(c)
}

// wrapTextRow handles raw text nodes at block level.
func wrapTextRow(text string) []core.Row {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	rt := richtext.New([]props.RichRun{{Text: text}})
	c := col.New().Add(rt)
	return []core.Row{row.New().Add(c)}
}

// isDisplayNone checks for the display:none inline-style override.
func isDisplayNone(n *dom.Node) bool {
	return strings.Contains(n.InlineStyle(), "display:none") ||
		strings.Contains(n.InlineStyle(), "display: none")
}

// applyBlockStyling applies block-level heading defaults to the first run.
func applyBlockStyling(n *dom.Node, runs []props.RichRun) {
	tag := n.Tag()
	headingSizes := map[string]float64{
		"h1": 24, "h2": 20, "h3": 16, "h4": 14, "h5": 12, "h6": 10,
	}
	if size, ok := headingSizes[tag]; ok {
		for i := range runs {
			if runs[i].Size == 0 {
				runs[i].Size = size
			}
			if runs[i].Style == "" {
				runs[i].Style = fontstyle.Bold
			}
		}
	}
}
