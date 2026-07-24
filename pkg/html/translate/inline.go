package translate

import (
	"context"

	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// inlineRuns walks the inline children of a block element and returns the run list.
// <br> becomes "\n"; <img> becomes an image run when a translator resolver is available.
func inlineRuns(n *dom.Node) []props.RichRun {
	return inlineRunsWithHandler(n, nil)
}

// inlineRunsWithHandler is the same as inlineRuns but surfaces side-channel
// information (e.g. <abbr title="…"> tooltip text, <time datetime="…">) via
// the given unsupportedHandler. nil handler disables side-channel reporting.
func inlineRunsWithHandler(n *dom.Node, h func(thing, value string)) []props.RichRun {
	return inlineRunsWithContext(n, runContext{handler: h})
}

func (tr *translator) inlineRuns(n *dom.Node) []props.RichRun {
	return inlineRunsWithContext(n, runContext{
		handler:       tr.unsupportedHandler,
		inlineImage:   tr.inlineImage,
		inlinePicture: tr.inlinePicture,
		inlineSVG:     tr.inlineSVG,
	})
}

func (tr *translator) inlineRunsStyled(n *dom.Node, style *css.ComputedStyle) []props.RichRun {
	return tr.applyFallbackFontRuns(inlineRunsWithContext(n, tr.styledRunContext(style)))
}

// applyFallbackFontRuns splits runs whose text needs the registered fallback
// font (see fallbackRunsFromRun). A no-op until the fallback font is loaded.
func (tr *translator) applyFallbackFontRuns(runs []props.RichRun) []props.RichRun {
	if tr == nil || !tr.fallbackFontReady {
		return runs
	}
	out := make([]props.RichRun, 0, len(runs))
	for _, run := range runs {
		out = append(out, fallbackRunsFromRun(run, true)...)
	}
	return out
}

// styledRunContextContext is styledRunContext for context-aware call paths.
// The runContext walk itself is synchronous and non-blocking, so ctx is not
// consulted beyond the caller's own cancellation checks.
func (tr *translator) styledRunContextContext(_ context.Context, style *css.ComputedStyle) runContext {
	return tr.styledRunContext(style)
}

func (tr *translator) styledRunContext(style *css.ComputedStyle) runContext {
	counters := tr.counters
	if counters == nil {
		counters = newCounterState()
	}
	quotes := tr.quotes
	if quotes == nil {
		quotes = newQuoteState()
	}
	return runContext{
		handler:        tr.unsupportedHandler,
		inlineImage:    tr.inlineImage,
		inlinePicture:  tr.inlinePicture,
		inlineSVG:      tr.inlineSVG,
		contentImage:   tr.generatedContentImage,
		style:          style,
		counters:       counters,
		quotes:         quotes,
		inlineBoxSeq:   &tr.inlineBoxSeq,
		styleResolver:  tr.computeInlineStyle,
		pseudoResolver: tr.computePseudoStyle,
	}
}

func (tr *translator) computeInlineStyle(n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle {
	return computeInlineNodeStyle(tr.sheet, n, parent)
}

func (tr *translator) computePseudoStyle(n *dom.Node, parent *css.ComputedStyle, pseudo string) *css.ComputedStyle {
	return computePseudoNodeStyle(tr.sheet, n, parent, pseudo)
}

func inlineRunsWithContext(n *dom.Node, ctx runContext) []props.RichRun {
	if ctx.counters == nil {
		ctx.counters = newCounterState()
	}
	if ctx.quotes == nil {
		ctx.quotes = newQuoteState()
	}
	if ctx.inlineBoxSeq == nil {
		seq := 0
		ctx.inlineBoxSeq = &seq
	}
	var runs []props.RichRun
	if n != nil && n.Tag() != "" {
		appendGeneratedContent(n, ctx, "before", &runs)
		for _, c := range n.Children() {
			walkInline(c, ctx, &runs)
		}
		appendGeneratedContent(n, ctx, "after", &runs)
		return runs
	}
	walkInline(n, ctx, &runs)
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
	inlineImage    func(n *dom.Node) (*props.RichImage, bool)
	inlinePicture  func(n *dom.Node) (*props.RichImage, bool)
	inlineSVG      func(n *dom.Node) (*props.RichImage, bool)
	contentImage   func(src string, style *css.ComputedStyle) (*props.RichImage, bool)
	style          *css.ComputedStyle
	counters       *counterState
	quotes         *quoteState
	inlineBox      *inlineBoxStyle
	inlineBoxSeq   *int
	styleResolver  func(n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle
	pseudoResolver func(n *dom.Node, parent *css.ComputedStyle, pseudo string) *css.ComputedStyle
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
	next := ctx
	if ctx.styleResolver != nil {
		next.style = ctx.styleResolver(n, ctx.style)
	}
	if isDisplayNone(n) || (next.style != nil && next.style.Display == displayNone) {
		return
	}
	if box, ok := inlineBoxFromStyle(next.style, ctx.style); ok {
		*next.inlineBoxSeq++
		box.ID = *next.inlineBoxSeq
		next.inlineBox = &box
	}
	counterScope := next.counters.enter(next.style)
	defer next.counters.exit(counterScope)
	if controlRuns, handled := formControlRuns(n, next); handled {
		*runs = append(*runs, controlRuns...)
		return
	}
	if tag == tagPicture && ctx.inlinePicture != nil {
		if img, ok := ctx.inlinePicture(n); ok {
			run := richRunFromContext("", next)
			run.Image = img
			run.InlineBoxWidth = 0
			run.InlineBoxHeight = 0
			*runs = append(*runs, run)
			return
		}
	}
	if handleSelfClosing(tag, n, next, runs) {
		return
	}
	next = mutateContext(tag, n, next)
	appendGeneratedContent(n, next, "before", runs)
	if tag == "q" {
		*runs = append(*runs, richRunFromContext(next.quotes.open(next.style), next))
		walkInlineChildren(n, next, runs)
		*runs = append(*runs, richRunFromContext(next.quotes.close(next.style), next))
		appendGeneratedContent(n, next, "after", runs)
		return
	}
	beforeChildren := len(*runs)
	walkInlineChildren(n, next, runs)
	if len(*runs) == beforeChildren && shouldEmitEmptyInlineBox(next.style) {
		*runs = append(*runs, richRunFromContext("", next))
	}
	appendGeneratedContent(n, next, "after", runs)
}

func walkInlineChildren(n *dom.Node, ctx runContext, runs *[]props.RichRun) {
	children := n.Children()
	for i, c := range children {
		before := len(*runs)
		walkInline(c, ctx, runs)
		if shouldApplyInlineFlexGap(ctx.style, children, i) {
			addInlineGapToLastRun(runs, before, ctx.style.ColumnGap)
		}
	}
}

func shouldApplyInlineFlexGap(style *css.ComputedStyle, children []*dom.Node, idx int) bool {
	if style == nil || style.Display != displayFlex || style.ColumnGap <= 0 {
		return false
	}
	for i := idx + 1; i < len(children); i++ {
		if !isWhitespaceNode(children[i]) {
			return true
		}
	}
	return false
}

func addInlineGapToLastRun(runs *[]props.RichRun, before int, gap float64) {
	if gap <= 0 || len(*runs) <= before {
		return
	}
	for i := len(*runs) - 1; i >= before; i-- {
		if (*runs)[i].ForceBreak {
			continue
		}
		(*runs)[i].InlineMarginRight += gap
		return
	}
}

func shouldEmitEmptyInlineBox(style *css.ComputedStyle) bool {
	if style == nil {
		return false
	}
	if style.Width <= 0 && style.Height <= 0 {
		return false
	}
	box, ok := inlineBoxFromStyle(style, nil)
	return ok && (box.BorderColor != nil || box.Background != nil || len(box.BoxShadows) > 0)
}

func appendGeneratedContent(n *dom.Node, ctx runContext, pseudo string, runs *[]props.RichRun) {
	if ctx.pseudoResolver == nil || n == nil || n.Tag() == "" {
		return
	}
	style := ctx.pseudoResolver(n, ctx.style, pseudo)
	if style == nil || style.Content == "" {
		return
	}
	counterScope := ctx.counters.enter(style)
	defer ctx.counters.exit(counterScope)
	pseudoCtx := ctx
	pseudoCtx.style = style
	generated, ok := generatedContentRuns(style.Content, n, pseudoCtx)
	if !ok || len(generated) == 0 {
		return
	}
	*runs = append(*runs, generated...)
}

func appendTextRun(n *dom.Node, ctx runContext, runs *[]props.RichRun) {
	text := n.TextContent()
	if text == "" {
		return
	}
	run := richRunFromContext(text, ctx)
	*runs = append(*runs, run)
}

func richRunFromContext(text string, ctx runContext) props.RichRun {
	run := props.RichRun{
		Text:          text,
		Family:        ctx.familyOverride,
		Style:         ctx.toStyle(),
		Underline:     ctx.underline,
		Strikethrough: ctx.strike,
		Hyperlink:     ctx.hyperlink,
		LocalAnchor:   ctx.localAnchor,
		SizeScale:     ctx.sizeScale,
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
	applyInlineStyleToRun(ctx.style, &run)
	applyInlineBoxToRun(ctx.inlineBox, &run)
	return run
}

func applyInlineBoxToRun(box *inlineBoxStyle, run *props.RichRun) {
	if box == nil || run == nil {
		return
	}
	run.InlineBoxID = box.ID
	if box.Background != nil && run.Background == nil {
		run.Background = props.CloneColor(box.Background)
	}
	if box.BorderColor != nil && run.BorderColor == nil {
		run.BorderColor = props.CloneColor(box.BorderColor)
	}
	if box.BorderWidth > 0 && run.BorderWidth == 0 {
		run.BorderWidth = box.BorderWidth
	}
	if box.Radius > 0 && run.BgRadius == 0 {
		run.BgRadius = box.Radius
	}
	if box.PadLeft == box.PadRight && box.PadLeft > 0 && run.BgPadX == 0 {
		run.BgPadX = box.PadLeft
	}
	if box.PadLeft > 0 && run.BgPadLeft == 0 {
		run.BgPadLeft = box.PadLeft
	}
	if box.PadRight > 0 && run.BgPadRight == 0 {
		run.BgPadRight = box.PadRight
	}
	if box.PadY > 0 && run.BgPadY == 0 {
		run.BgPadY = box.PadY
	}
	if box.MarginLeft > 0 && run.InlineMarginLeft == 0 {
		run.InlineMarginLeft = box.MarginLeft
	}
	if box.MarginRight > 0 && run.InlineMarginRight == 0 {
		run.InlineMarginRight = box.MarginRight
	}
	if len(box.BoxShadows) > 0 && len(run.BoxShadows) == 0 {
		run.BoxShadows = clonePropShadows(box.BoxShadows)
	}
}

func clonePropShadows(shadows []props.Shadow) []props.Shadow {
	if shadows == nil {
		return nil
	}
	clone := make([]props.Shadow, len(shadows))
	for i, shadow := range shadows {
		clone[i] = props.CloneShadow(shadow)
	}
	return clone
}

// handleSelfClosing handles tags that emit a run directly without recursion.
// Returns true if the tag was handled.
func handleSelfClosing(tag string, n *dom.Node, ctx runContext, runs *[]props.RichRun) bool {
	switch tag {
	case "br":
		*runs = append(*runs, props.RichRun{Text: "\n", ForceBreak: true, Style: ctx.toStyle()})
		return true
	case tagImg:
		if ctx.inlineImage != nil {
			if img, ok := ctx.inlineImage(n); ok {
				run := richRunFromContext("", ctx)
				run.Image = img
				run.InlineBoxWidth = 0
				run.InlineBoxHeight = 0
				*runs = append(*runs, run)
				return true
			}
		}
		if alt := n.Attr("alt"); alt != "" {
			*runs = append(*runs, richRunFromContext(alt, ctx))
		}
		return true
	case tagSVG:
		if ctx.inlineSVG != nil {
			if img, ok := ctx.inlineSVG(n); ok {
				run := richRunFromContext("", ctx)
				run.Image = img
				run.InlineBoxWidth = 0
				run.InlineBoxHeight = 0
				*runs = append(*runs, run)
				return true
			}
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
	case "i", "em", "var", "cite", "dfn":
		next.italic = true
	case "u", "ins":
		next.underline = true
	case "s", "strike", "del":
		next.strike = true
	case verticalAlignSub:
		next.sub = true
		next.sizeScale = scaledRunSize(next.sizeScale, 0.75)
	case "sup":
		next.sup = true
		next.sizeScale = scaledRunSize(next.sizeScale, 0.75)
	case "a":
		if href := n.Attr("href"); href != "" {
			switch link, ok := sanitizedHyperlink(href); {
			case href[0] == '#':
				next.localAnchor = href[1:]
				next.hyperlink = nil
			case ok:
				next.hyperlink = &link
			default:
				// Refused schemes render as plain text rather than as a link.
				next.hyperlink = nil
				if ctx.handler != nil {
					ctx.handler("a.href", href)
				}
			}
		}
	case "mark":
		// Yellow background highlight.
		next.background = &css.RGBColor{R: 255, G: 255, B: 0, A: 1}
	case "small":
		// Render at 0.85x of the parent size. runContext keeps a scale; the
		// absolute size is applied when the parent's computed font-size is known.
		next.sizeScale = scaledRunSize(next.sizeScale, 0.85)
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

func scaledRunSize(current, factor float64) float64 {
	if current == 0 {
		return factor
	}
	return current * factor
}

func vAlign(ctx runContext) string {
	if ctx.sub {
		return verticalAlignSub
	}
	if ctx.sup {
		return verticalAlignSuper
	}
	return verticalAlignBaseline
}
