package translate

import (
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// inlineRuns walks the inline children of a block element and returns the run list.
// Inline images and <br> are flattened into the run sequence (br = "\n", img omitted).
func inlineRuns(n *dom.Node) []props.RichRun {
	var runs []props.RichRun
	walkInline(n, runContext{}, &runs)
	return runs
}

type runContext struct {
	bold      bool
	italic    bool
	underline bool
	strike    bool
	hyperlink *string
	sub       bool
	sup       bool
}

func (c runContext) toStyle() fontstyle.Type {
	switch {
	case c.bold && c.italic:
		return fontstyle.BoldItalic
	case c.bold:
		return fontstyle.Bold
	case c.italic:
		return fontstyle.Italic
	default:
		return fontstyle.Normal
	}
}

func walkInline(n *dom.Node, ctx runContext, runs *[]props.RichRun) {
	if n == nil {
		return
	}
	tag := n.Tag()
	if tag == "" {
		appendTextRun(n, ctx, runs)
		return
	}
	if handleSelfClosing(tag, n, ctx, runs) {
		return
	}
	next := mutateContext(tag, n, ctx)
	for _, c := range n.Children() {
		walkInline(c, next, runs)
	}
}

func appendTextRun(n *dom.Node, ctx runContext, runs *[]props.RichRun) {
	text := n.TextContent()
	if text == "" {
		return
	}
	*runs = append(*runs, props.RichRun{
		Text:          text,
		Style:         ctx.toStyle(),
		Underline:     ctx.underline,
		Strikethrough: ctx.strike,
		Hyperlink:     ctx.hyperlink,
		VerticalAlign: vAlign(ctx),
	})
}

// handleSelfClosing handles tags that emit a run directly without recursion.
// Returns true if the tag was handled.
func handleSelfClosing(tag string, n *dom.Node, ctx runContext, runs *[]props.RichRun) bool {
	switch tag {
	case "br":
		*runs = append(*runs, props.RichRun{Text: "\n", Style: ctx.toStyle()})
		return true
	case "img":
		if alt := n.Attr("alt"); alt != "" {
			*runs = append(*runs, props.RichRun{Text: alt, Style: ctx.toStyle()})
		}
		return true
	}
	return false
}

// mutateContext returns a new runContext with the styling effect of the tag applied.
func mutateContext(tag string, n *dom.Node, ctx runContext) runContext {
	next := ctx
	switch tag {
	case "b", "strong":
		next.bold = true
	case "i", "em":
		next.italic = true
	case "u":
		next.underline = true
	case "s", "strike":
		next.strike = true
	case "sub":
		next.sub = true
	case "sup":
		next.sup = true
	case "a":
		if href := n.Attr("href"); href != "" {
			next.hyperlink = &href
		}
	}
	return next
}

func vAlign(ctx runContext) string {
	if ctx.sub {
		return "sub"
	}
	if ctx.sup {
		return "super"
	}
	return "baseline"
}
