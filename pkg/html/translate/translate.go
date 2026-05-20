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
	sheet              *stylesheet
	gridSize           int     // 0 = use defaultGridSize (12)
	contentWidthMM     float64 // 0 = use default A4 estimate (170mm)
	imageResolver      ImageResolver
	unsupportedHandler func(thing, value string)
}

// WithImageResolver lets callers plug in a custom <img src=…> loader.
func WithImageResolver(fn ImageResolver) Option {
	return func(tr *translator) {
		tr.imageResolver = fn
	}
}

// WithImageBaseDir scopes the default resolver to a single directory.
// Local file reads outside this directory are refused (path-traversal safe).
func WithImageBaseDir(dir string) Option {
	return func(tr *translator) {
		tr.imageResolver = baseDirResolver(dir)
	}
}

// WithUnsupportedHandler registers a callback for unsupported tags/props.
func WithUnsupportedHandler(fn func(thing, value string)) Option {
	return func(tr *translator) {
		tr.unsupportedHandler = fn
	}
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
		return []core.Row{tr.styledHrRow(n)}
	case "table":
		return tr.tableRows(n)
	case "ul", "ol":
		return tr.listRows(n)
	case "dl":
		return tr.definitionListRows(n)
	case "details":
		return tr.detailsRows(n)
	case "img":
		if r, ok := tr.imageRow(n); ok {
			return []core.Row{r}
		}
		return altRow(n)
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
			if isColumnDirection(style.FlexDirection) {
				return tr.flexColumnRows(n, style)
			}
			r := tr.flexRow(n, style)
			if r == nil {
				return nil
			}
			return []core.Row{r}
		}
		// Default: collect children's rows.
		var rows []core.Row
		for _, c := range n.Children() {
			rows = append(rows, tr.blockRows(c)...)
		}
		// When the container has background/border/padding, wrap children
		// in a single styled blockContainer so the styling spans them all.
		if shouldUseContainer(style) && len(rows) > 0 {
			return []core.Row{buildContainerRow(style, rows)}
		}
		return rows
	}
}

// paragraphRow converts a block element with inline content into a single auto-height row.
// When the computed style sets CSS padding, it is passed through as the richtext's
// Top/Right/Bottom/Left offsets so the text is inset from the styled background's
// edges instead of butting against them.
func (tr *translator) paragraphRow(n *dom.Node) core.Row {
	style := computeNodeStyle(tr.sheet, n, nil)
	runs := inlineRuns(n)
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	// User CSS first, then heading-default fallback. applyBlockStyling only
	// sets runs[i].Size when it's still 0, so applying inline CSS first lets a
	// user `h2 { font-size: 12pt }` override the 20pt heading default.
	applyInlineStyleToRuns(style, runs)
	applyBlockStyling(n, runs)
	// text-indent shifts the whole paragraph in v1 (true first-line indent
	// requires renderer-side support; documented limitation).
	rtProp := props.RichText{
		Top:    style.PaddingTop,
		Right:  style.PaddingRight,
		Bottom: style.PaddingBottom,
		Left:   style.PaddingLeft + style.TextIndent,
	}
	rt := richtext.New(runs, rtProp)
	c := col.New().Add(rt)
	r := row.New().Add(c)
	if cellStyle := blockCellStyle(style); cellStyle != nil {
		r = r.WithStyle(cellStyle)
	}
	return r
}

// hrRow produces a thin row containing a horizontal line (default style).
func hrRow() core.Row {
	l := line.New()
	c := col.New().Add(l)
	return row.New(1).Add(c)
}

// styledHrRow honours border-top-width, border-top-style, color on the <hr>
// element. Defaults match the original hrRow behaviour when no style is set.
func (tr *translator) styledHrRow(n *dom.Node) core.Row {
	style := computeNodeStyle(tr.sheet, n, nil)
	lineProp := props.Line{}
	if style.BorderTopWidth > 0 {
		lineProp.Thickness = style.BorderTopWidth
	}
	if c := style.BorderTopColor; c != nil {
		lineProp.Color = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	} else if style.Color != nil {
		lineProp.Color = &props.Color{Red: style.Color.R, Green: style.Color.G, Blue: style.Color.B}
	}
	l := line.New(lineProp)
	c := col.New().Add(l)
	h := 1.0
	if style.BorderTopWidth > 0 {
		h = style.BorderTopWidth + 0.5
	}
	return row.New(h).Add(c)
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
