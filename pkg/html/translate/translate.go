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
	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// Option configures translator behaviour.
type Option func(*translator)

// WithGridSize overrides the default 12-column grid size used for flex quantization.
func WithGridSize(n int) Option {
	return func(tr *translator) {
		if n > 0 {
			tr.gridSize = n
		}
	}
}

// WithContentWidth sets the content width in mm, used for gap-to-col approximation.
func WithContentWidth(mm float64) Option {
	return func(tr *translator) {
		if mm > 0 {
			tr.contentWidthMM = mm
		}
	}
}

// translator threads parsed stylesheet rules through the recursive walker.
type translator struct {
	sheet          *stylesheet
	gridSize       int     // 0 = use defaultGridSize (12)
	contentWidthMM float64 // 0 = use default A4 estimate (170mm)
}

// Translate walks the styled DOM and emits Maroto rows.
func Translate(doc *dom.Document, opts ...Option) ([]core.Row, error) {
	if doc == nil {
		return nil, nil
	}
	tr := &translator{sheet: parseStylesheet(doc.StyleText())}
	for _, opt := range opts {
		opt(tr)
	}
	var rows []core.Row
	body := findBody(doc)
	if body == nil {
		return rows, nil
	}
	for _, child := range body.Children() {
		rows = append(rows, tr.blockRows(child)...)
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
func (tr *translator) blockRows(n *dom.Node) []core.Row {
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
		return []core.Row{tr.paragraphRow(n)}
	case "hr":
		return []core.Row{hrRow()}
	case "table":
		return tableRows(n)
	case "ul", "ol":
		return listRows(n)
	case "br":
		return nil // top-level <br> is a no-op
	default:
		// Container (div, section, article, header, footer, nav, etc.).
		// Compute style to detect class-based display:flex and display:none.
		style := computeNodeStyle(tr.sheet, n, nil)
		if style.Display == "none" {
			return nil
		}
		if style.Display == "flex" {
			r := tr.flexRow(n, style)
			if r == nil {
				return nil
			}
			return []core.Row{r}
		}
		// Default: flatten children into rows.
		var rows []core.Row
		for _, c := range n.Children() {
			rows = append(rows, tr.blockRows(c)...)
		}
		return rows
	}
}

// paragraphRow converts a block element with inline content into a single auto-height row.
func (tr *translator) paragraphRow(n *dom.Node) core.Row {
	style := computeNodeStyle(tr.sheet, n, nil)
	runs := inlineRuns(n)
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	applyBlockStyling(n, runs)
	applyInlineStyleToRuns(style, runs)
	rt := richtext.New(runs)
	c := col.New().Add(rt)
	r := row.New().Add(c)
	if cellStyle := blockCellStyle(style); cellStyle != nil {
		r = r.WithStyle(cellStyle)
	}
	return r
}

// applyInlineStyleToRuns applies CSS-computed font size and color to every run
// whose own field is unset (run-level styling wins over block-level).
func applyInlineStyleToRuns(style *css.ComputedStyle, runs []props.RichRun) {
	if style == nil {
		return
	}
	for i := range runs {
		if style.FontSize > 0 && runs[i].Size == 0 {
			// FontSize is in mm; props.RichRun expects pt — convert.
			runs[i].Size = style.FontSize / 0.352778
		}
		if style.Color != nil && runs[i].Color == nil {
			runs[i].Color = &props.Color{Red: style.Color.R, Green: style.Color.G, Blue: style.Color.B}
		}
	}
}

// blockCellStyle converts a ComputedStyle's background and border fields
// into a Maroto props.Cell. Returns nil if no decorative styling is set.
func blockCellStyle(style *css.ComputedStyle) *props.Cell {
	if style == nil || (style.BackgroundColor == nil && style.BorderTopWidth == 0 &&
		style.BorderRightWidth == 0 && style.BorderBottomWidth == 0 && style.BorderLeftWidth == 0) {
		return nil
	}
	cell := &props.Cell{}
	if style.BackgroundColor != nil {
		cell.BackgroundColor = &props.Color{
			Red: style.BackgroundColor.R, Green: style.BackgroundColor.G, Blue: style.BackgroundColor.B,
		}
	}
	if c := style.BorderTopColor; c != nil {
		cell.BorderTopColor = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
	if c := style.BorderRightColor; c != nil {
		cell.BorderRightColor = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
	if c := style.BorderBottomColor; c != nil {
		cell.BorderBottomColor = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
	if c := style.BorderLeftColor; c != nil {
		cell.BorderLeftColor = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
	cell.BorderTopThickness = style.BorderTopWidth
	cell.BorderRightThickness = style.BorderRightWidth
	cell.BorderBottomThickness = style.BorderBottomWidth
	cell.BorderLeftThickness = style.BorderLeftWidth
	return cell
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
