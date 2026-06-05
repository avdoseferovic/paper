package translate

import (
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// inlineRuns walks the inline children of a block element and returns the run list.
// Inline images and <br> are flattened into the run sequence (br = "\n", img omitted).
func inlineRuns(n *dom.Node) []props.RichRun {
	return inlineRunsWithHandler(n, nil)
}

// inlineRunsWithHandler is the same as inlineRuns but surfaces side-channel
// information (e.g. <abbr title="…"> tooltip text, <time datetime="…">) via
// the given unsupportedHandler. nil handler disables side-channel reporting.
func inlineRunsWithHandler(n *dom.Node, h func(thing, value string)) []props.RichRun {
	var runs []props.RichRun
	walkInline(n, runContext{handler: h}, &runs)
	return runs
}

// runContext threads inline styling state through the recursive walk.
// Fields beyond bold/italic/underline support inline tags that change
// font scale (<small>), background (<mark>, <kbd>), or family (<code> outside <pre>).
type runContext struct {
	bold           bool
	italic         bool
	underline      bool
	strike         bool
	hyperlink      *string
	localAnchor    string
	sub            bool
	sup            bool
	sizeScale      float64       // multiplier on inherited font size (0 = inherit unchanged)
	monospace      bool          // pick a monospace family at render
	background     *css.RGBColor // run-level background fill
	familyOverride string
	handler        func(thing, value string) // optional unsupportedHandler for side-channel data
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
	// <q>: wrap children with ASCII quotes.
	if tag == "q" {
		*runs = append(*runs, props.RichRun{Text: `"`, Style: next.toStyle()})
		for _, c := range n.Children() {
			walkInline(c, next, runs)
		}
		*runs = append(*runs, props.RichRun{Text: `"`, Style: next.toStyle()})
		return
	}
	for _, c := range n.Children() {
		walkInline(c, next, runs)
	}
}

func appendTextRun(n *dom.Node, ctx runContext, runs *[]props.RichRun) {
	text := n.TextContent()
	if text == "" {
		return
	}
	run := props.RichRun{
		Text:          text,
		Family:        ctx.familyOverride,
		Style:         ctx.toStyle(),
		Underline:     ctx.underline,
		Strikethrough: ctx.strike,
		Hyperlink:     ctx.hyperlink,
		LocalAnchor:   ctx.localAnchor,
		VerticalAlign: vAlign(ctx),
	}
	if ctx.monospace && run.Family == "" {
		run.Family = "courier"
	}
	if ctx.background != nil {
		run.Background = &props.Color{
			Red:   ctx.background.R,
			Green: ctx.background.G,
			Blue:  ctx.background.B,
		}
		if ctx.background.A < 1 {
			a := ctx.background.A
			run.Background.Alpha = &a
		}
	}
	*runs = append(*runs, run)
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
	case "i", "em", "var", "cite":
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
			if len(href) > 0 && href[0] == '#' {
				next.localAnchor = href[1:]
				next.hyperlink = nil
			} else {
				next.hyperlink = &href
			}
		}
	case "mark":
		// Yellow background highlight.
		next.background = &css.RGBColor{R: 255, G: 255, B: 0, A: 1}
	case "small":
		// Render at 0.85x of the parent size. runContext keeps a scale; the
		// absolute size is applied when the parent's computed font-size is known.
		next.sizeScale = 0.85
	case "code", "kbd", "samp":
		next.monospace = true
		if tag != "samp" {
			// Light grey background for code/kbd.
			next.background = &css.RGBColor{R: 240, G: 240, B: 240, A: 1}
		}
	case "abbr":
		// Solid underline (dotted not supported by the current renderer).
		next.underline = true
		if title := n.Attr("title"); title != "" && ctx.handler != nil {
			// Surface the tooltip text via the unsupported handler so callers
			// know the title attribute is not rendered as a PDF tooltip.
			ctx.handler("abbr.title", title)
		}
	case "time":
		if dt := n.Attr("datetime"); dt != "" && ctx.handler != nil {
			// Surface the machine-readable datetime so downstream consumers
			// can access it (PDFs have no semantic markup for time values).
			ctx.handler("time.datetime", dt)
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
