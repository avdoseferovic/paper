// Package translate converts a parsed HTML DOM into a slice of Paper rows.
package translate

import (
	"context"
	"fmt"
	"strings"

	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

const defaultContentWidthMM = 170.0

// translator threads parsed stylesheet rules through the recursive walker.
type translator struct {
	sheet              *stylesheet
	gridSize           int     // 0 = use defaultGridSize (12)
	contentWidthMM     float64 // 0 = use default A4 estimate (170mm)
	imageResolver      ImageResolver
	imageBaseDir       string
	stylesheetResolver StylesheetResolver
	limits             htmllimits.Limits
	unsupportedHandler func(thing, value string)
	err                error
	anchorReg          *anchorRegistry     // shared id→linkID map (Task 6)
	anchorIDs          map[string]struct{} // pre-collected id values (forward refs)
	loadedFonts        []loadedFont        // @font-face fonts (Task 10)
	rootStyle          *css.ComputedStyle  // seed for body-level cascade (:root vars)
	counters           *counterState       // document-order CSS counter state
	quotes             *quoteState         // document-order CSS quote depth
	inlineBoxSeq       int                 // monotonically assigns grouped inline boxes

	// outlineFromHeadings, when true, marks h1-h6 paragraphs with a
	// props.Outline so they appear in the PDF document outline.
	outlineFromHeadings bool
}

// Translate walks the styled DOM and emits Paper rows. It observes ctx at
// cheap phase and recursive traversal boundaries.
func Translate(ctx context.Context, doc *dom.Document, opts ...Option) ([]core.Row, error) {
	document, err := translateDocument(ctx, doc, false, opts...)
	if err != nil || document == nil {
		return nil, err
	}
	return document.Rows, nil
}

// TranslateDocument walks the styled DOM and returns the full Document
// result: content rows plus @page options. It observes ctx at cheap phase and
// recursive traversal boundaries. Unlike Translate, the first top-level
// <header>/<footer> elements are extracted into HeaderRows/FooterRows instead
// of rendering inline.
func TranslateDocument(ctx context.Context, doc *dom.Document, opts ...Option) (*Document, error) {
	return translateDocument(ctx, doc, true, opts...)
}

func translateDocument(ctx context.Context, doc *dom.Document, extractBands bool, opts ...Option) (*Document, error) {
	if doc == nil {
		return &Document{}, nil
	}
	tr := &translator{
		anchorReg: newAnchorRegistry(),
		limits:    htmllimits.Default(),
	}
	for _, opt := range opts {
		opt(tr)
	}
	err := translationCanceled(ctx)
	if err != nil {
		return nil, err
	}
	err = doc.ValidateLimits(tr.limits)
	if err != nil {
		return nil, err
	}
	err = translationCanceled(ctx)
	if err != nil {
		return nil, err
	}
	// External stylesheets load BEFORE inline <style> so browser-style
	// cascade order applies. defer/recover is inside safeLoadStylesheet so
	// resolver bugs cannot crash Translate.
	inlineCSS, hrefs := doc.StyleSources()
	resolver := tr.stylesheetResolver
	if resolver == nil {
		resolver = safeDefaultStylesheetResolver
	}
	var combined []byte
	for _, href := range hrefs {
		err = translationCanceled(ctx)
		if err != nil {
			return nil, err
		}
		data, ok := safeLoadStylesheet(resolver, href)
		if !ok {
			if tr.unsupportedHandler != nil {
				tr.unsupportedHandler("link.skipped", href)
			}
			continue
		}
		combined = append(combined, data...)
		combined = append(combined, '\n')
	}
	combined = append(combined, []byte(inlineCSS)...)
	err = translationCanceled(ctx)
	if err != nil {
		return nil, err
	}
	sheet, err := parseStylesheetWithLimits(string(combined), tr.availableContentWidth(), tr.limits)
	if err != nil {
		return nil, err
	}
	tr.sheet = sheet
	if tr.unsupportedHandler != nil {
		for _, prelude := range sheet.skippedPages {
			tr.unsupportedHandler("page-rule.skipped", prelude)
		}
	}
	err = translationCanceled(ctx)
	if err != nil {
		return nil, err
	}

	// Process @font-face declarations: resolve src URLs via the stylesheet
	// resolver and emit a fontRegistration row that registers the bytes via
	// LateFontProvider at render time. defer/recover inside processFontFace
	// ensures a malformed font cannot crash translation.
	tr.registerFontFaces(resolver)
	err = translationCanceled(ctx)
	if err != nil {
		return nil, err
	}
	body := findBody(doc)
	if body == nil {
		return &Document{Page: tr.pageOptions()}, nil
	}
	// Pre-pass: collect all id values so forward references (link before
	// target) resolve correctly at render time via the shared anchor registry.
	tr.anchorIDs = collectAnchorIDs(body)
	// Prepend font-face registration rows (zero-height) so any subsequent
	// row that uses font-family: "MyFont" finds the font already registered.
	rows := tr.fontRegistrationRows()
	// Seed the cascade so :root and html-level rules (CSS variables, etc)
	// propagate into body's descendants. Without this, `:root { --x: red }`
	// is parsed but never inherited because computeNodeStyle is called with
	// parent=nil for top-level body children.
	tr.rootStyle = tr.seedRootStyle(doc)
	tr.counters = newCounterState()
	tr.quotes = newQuoteState()
	rootCounters := tr.counters.enter(tr.rootStyle)
	document, err := tr.walkBody(ctx, body, rows, extractBands)
	if err != nil {
		return nil, err
	}
	tr.counters.exit(rootCounters)
	if tr.err != nil {
		return nil, tr.err
	}
	document.Page = tr.pageOptions()
	return document, nil
}

// walkBody converts the body's children into rows. When extractBands is set,
// the first top-level <header>/<footer> land in HeaderRows/FooterRows instead
// of the content rows.
func (tr *translator) walkBody(ctx context.Context, body *dom.Node, rows []core.Row, extractBands bool) (*Document, error) {
	var headerRows, footerRows []core.Row
	headerTaken, footerTaken := false, false
	for _, child := range body.Children() {
		err := translationCanceled(ctx)
		if err != nil {
			return nil, err
		}
		childRows := tr.blockRowsWithParent(ctx, child, tr.rootStyle)
		if tr.err != nil {
			return nil, tr.err
		}
		switch {
		case extractBands && !headerTaken && child.Tag() == "header":
			headerRows = childRows
			headerTaken = true
		case extractBands && !footerTaken && child.Tag() == "footer":
			footerRows = childRows
			footerTaken = true
		default:
			rows = append(rows, childRows...)
		}
	}
	return &Document{Rows: rows, HeaderRows: headerRows, FooterRows: footerRows}, nil
}

// seedRootStyle computes the ComputedStyle of the html element so :root vars
// and inherited CSS properties propagate to the body's descendants. Falls back
// to body's style when no html element exists.
func (tr *translator) seedRootStyle(doc *dom.Document) *css.ComputedStyle {
	if html := doc.HTMLElement(); html != nil {
		htmlStyle := computeNodeStyle(tr.sheet, html, nil)
		// Continue computing body's style with html as the parent so body
		// rules and any inherited vars merge correctly.
		if body := findBody(doc); body != nil {
			return computeNodeStyle(tr.sheet, body, htmlStyle)
		}
		return htmlStyle
	}
	if body := findBody(doc); body != nil {
		return computeNodeStyle(tr.sheet, body, nil)
	}
	return nil
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
func (tr *translator) blockRows(ctx context.Context, n *dom.Node) []core.Row {
	return tr.blockRowsWithParent(ctx, n, nil)
}

func (tr *translator) blockRowsWithParent(ctx context.Context, n *dom.Node, parent *css.ComputedStyle) []core.Row {
	if n == nil || tr.err != nil {
		return nil
	}
	err := translationCanceled(ctx)
	if err != nil {
		tr.err = err
		return nil
	}
	if isDisplayNone(n) {
		return nil
	}
	var style *css.ComputedStyle
	if n.Tag() != "" {
		style = tr.computeBlockStyle(n, parent)
		if style.Display == displayNone {
			return nil
		}
	} else {
		style = parent
	}
	counterScope := tr.counters.enter(style)
	defer tr.counters.exit(counterScope)

	rows := tr.dispatchBlockRowsWithStyle(ctx, n, style)
	control, hasControl := pageControlFromStyle(style, parent)
	if hasControl && control.Blank {
		rows = []core.Row{NewBlankPageRow(control)}
	} else {
		rows = tr.decorateBlockRows(n, style, rows, control, hasControl)
	}
	return withPageBreakRows(style, rows)
}

func (tr *translator) decorateBlockRows(
	n *dom.Node,
	style *css.ComputedStyle,
	rows []core.Row,
	control core.PageControl,
	hasControl bool,
) []core.Row {
	// If the element has an id, wrap its first row in an anchorTarget so the
	// PDF destination registers at the element's actual Y position.
	if id := n.Attr("id"); id != "" && len(rows) > 0 && tr.anchorReg != nil {
		rows[0] = wrapRowAnchorTarget(rows[0], id, tr.anchorReg)
	}
	rows = applyBlockMargins(n, style, rows)
	if hasControl {
		rows = append([]core.Row{NewPageControlRow(control)}, rows...)
	}
	return rows
}

func applyBlockMargins(n *dom.Node, style *css.ComputedStyle, rows []core.Row) []core.Row {
	if style == nil || n.Tag() == "" || len(rows) == 0 || selfHandlesMargin(n.Tag()) {
		return rows
	}
	// Honor block-level margins outside the element's content box. Horizontal
	// margins narrow/offset each produced content row; vertical margins remain
	// flow spacers so they can collapse with adjacent block margins.
	if style.MarginLeft != 0 || style.MarginRight != 0 {
		rows = wrapRowsWithHorizontalMargins(rows, style.MarginLeft, style.MarginRight)
	}
	if style.MarginTop > 0 {
		rows = append([]core.Row{newMarginSpacer(style.MarginTop)}, rows...)
	}
	if style.MarginBottom > 0 {
		rows = append(rows, newMarginSpacer(style.MarginBottom))
	}
	// CSS margins collapse: adjacent/last-child margins reduce to the larger,
	// they do not sum. Merge touching margin spacers so stacked block margins
	// don't accumulate. (Containers with padding/border are opaque rows, so
	// margins never collapse through them — matching CSS.)
	return collapseMarginSpacers(rows)
}

func withPageBreakRows(style *css.ComputedStyle, rows []core.Row) []core.Row {
	if style != nil {
		if style.PageBreakBefore == "always" {
			rows = append([]core.Row{NewPageBreakRow()}, rows...)
		}
		if style.PageBreakAfter == "always" {
			rows = append(rows, NewPageBreakRow())
		}
	}
	return rows
}

func wrapRowsWithHorizontalMargins(rows []core.Row, marginLeft, marginRight float64) []core.Row {
	out := make([]core.Row, 0, len(rows))
	for _, r := range rows {
		if _, ok := r.(marginSpacerRow); ok {
			out = append(out, r)
			continue
		}
		out = append(out, &horizontalMarginRow{
			child:       r,
			marginLeft:  marginLeft,
			marginRight: marginRight,
		})
	}
	return out
}

func pageControlFromStyle(style, parent *css.ComputedStyle) (core.PageControl, bool) {
	if style == nil || len(style.Vars) == 0 {
		return core.PageControl{}, false
	}
	pageNumberCount := cssVarDeclaredEquals(style, parent, "--paper-page-number", "count")
	control := core.PageControl{
		Blank:                     cssVarDeclaredEquals(style, parent, "--paper-page", "blank"),
		SuppressFooter:            cssVarDeclaredEquals(style, parent, "--paper-page-footer", "none"),
		SuppressPageNumber:        cssVarDeclaredEquals(style, parent, "--paper-page-number", "none") || pageNumberCount,
		CountPageNumber:           pageNumberCount,
		DecorFirstPageOnly:        cssVarDeclaredEquals(style, parent, "--paper-page-decor-scope", "first-page"),
		TopMarginFirstPageOnly:    cssVarDeclaredEquals(style, parent, "--paper-page-top-margin-scope", "first-page"),
		RenderOffsetFirstPageOnly: cssVarDeclaredEquals(style, parent, "--paper-page-render-offset-scope", "first-page"),
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-top-margin"); ok {
		top := css.ParseLength(value, style.FontSize)
		control.TopMargin = &top
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-top-margin-continuation"); ok {
		top := css.ParseLength(value, style.FontSize)
		control.ContinuationTopMargin = &top
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-render-offset-x"); ok {
		x := css.ParseLength(value, style.FontSize)
		control.RenderOffsetX = &x
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-render-offset-y"); ok {
		y := css.ParseLength(value, style.FontSize)
		control.RenderOffsetY = &y
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-render-offset-x-continuation"); ok {
		x := css.ParseLength(value, style.FontSize)
		control.ContinuationRenderOffsetX = &x
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-render-offset-y-continuation"); ok {
		y := css.ParseLength(value, style.FontSize)
		control.ContinuationRenderOffsetY = &y
	}
	return control, pageControlHasEffect(control)
}

func pageControlHasEffect(control core.PageControl) bool {
	return control.Blank ||
		control.SuppressFooter ||
		control.SuppressPageNumber ||
		control.CountPageNumber ||
		control.DecorFirstPageOnly ||
		control.TopMargin != nil ||
		control.TopMarginFirstPageOnly ||
		control.ContinuationTopMargin != nil ||
		control.RenderOffsetX != nil ||
		control.RenderOffsetY != nil ||
		control.RenderOffsetFirstPageOnly ||
		control.ContinuationRenderOffsetX != nil ||
		control.ContinuationRenderOffsetY != nil
}

func cssVarDeclaredEquals(style, parent *css.ComputedStyle, name, want string) bool {
	value, ok := cssVarDeclaredValue(style, parent, name)
	return ok && strings.EqualFold(strings.TrimSpace(value), want)
}

func cssVarDeclaredValue(style, parent *css.ComputedStyle, name string) (string, bool) {
	value, ok := style.Vars[name]
	if !ok {
		return "", false
	}
	if parent == nil || len(parent.Vars) == 0 {
		return value, true
	}
	parentValue, inherited := parent.Vars[name]
	return value, !inherited || parentValue != value
}

// marginSpacerRow is an empty flow spacer emitted for a block's vertical margin.
// It is a distinct type so collapseMarginSpacers can recognise and merge
// adjacent ones (CSS margin collapsing).
type marginSpacerRow struct {
	core.Row
	height float64
}

func newMarginSpacer(h float64) marginSpacerRow {
	return marginSpacerRow{Row: spacerRow(h), height: h}
}

// collapseMarginSpacers merges each run of adjacent margin spacers into a single
// spacer of the largest height, approximating CSS adjacent and parent/last-child
// margin collapsing so stacked block margins reduce to the larger rather than
// summing. Non-margin rows (content, page breaks, flex gaps) are untouched.
func collapseMarginSpacers(rows []core.Row) []core.Row {
	out := make([]core.Row, 0, len(rows))
	for _, r := range rows {
		if ms, ok := r.(marginSpacerRow); ok && len(out) > 0 {
			if prev, prevOK := out[len(out)-1].(marginSpacerRow); prevOK {
				if ms.height > prev.height {
					out[len(out)-1] = ms
				}
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// selfHandlesMargin reports whether a tag's dedicated translation already
// applies its own CSS margins (via marginBox), so blockRowsWithParent must not
// add margin spacers on top (which would double them). Lists wrap their content
// in a marginBox that also carries horizontal indent.
func selfHandlesMargin(tag string) bool {
	switch tag {
	case "ul", "ol", "dl":
		return true
	default:
		return false
	}
}

// dispatchBlockRows is the original blockRows tag switch (split out so the
// outer blockRows can handle anchor wrapping uniformly).
func (tr *translator) dispatchBlockRowsWithStyle(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	tag := n.Tag()
	// Drop metadata tags that may appear in the body — their text content
	// (CSS source, script source, meta values) must not render as visible
	// document text. <style>/<link> CSS is already extracted via StyleSources.
	if isMetadataBlockTag(tag) {
		return nil
	}
	switch tag {
	case "":
		// Text node at block level — wrap into a paragraph-like row.
		return wrapTextRowStyled(n.TextContent(), style)
	case "p", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "pre":
		return []core.Row{tr.paragraphRowStyled(n, style)}
	case "hr":
		return []core.Row{tr.styledHrRowWithStyle(n, style)}
	case tagTable:
		return tr.tableRowsWithStyle(n, style)
	case "ul", "ol":
		return tr.listRows(n)
	case "dl":
		return tr.definitionListRows(n)
	case "details":
		return tr.detailsRows(ctx, n)
	case tagImg:
		return tr.imageBlockRowsWithStyle(n, style)
	case tagPicture:
		return tr.pictureRowWithStyle(n, style)
	case tagSVG:
		return tr.svgBlockRowsWithStyle(n, style)
	case "br":
		return nil // top-level <br> is a no-op
	default:
		return tr.containerRows(ctx, n, style)
	}
}

func isMetadataBlockTag(tag string) bool {
	switch tag {
	case "style", "link", "script", "meta", "head", "title":
		return true
	default:
		return false
	}
}

func (tr *translator) imageBlockRowsWithStyle(n *dom.Node, style *css.ComputedStyle) []core.Row {
	if r, ok := tr.imageRowWithStyle(n, style); ok {
		return []core.Row{r}
	}
	if tr.err != nil {
		return nil
	}
	return altRowStyled(n, style)
}

func (tr *translator) svgBlockRowsWithStyle(n *dom.Node, style *css.ComputedStyle) []core.Row {
	if r, ok := tr.svgRowWithStyle(n, style); ok {
		return []core.Row{r}
	}
	if tr.err != nil {
		return nil
	}
	return nil
}

func (tr *translator) containerRows(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	// Container (div, section, article, header, footer, nav, etc.).
	// Compute style to detect class-based display:flex and display:none.
	if style.Display == displayNone {
		return nil
	}
	if style.Display == displayFlex {
		return tr.containerFlexRows(ctx, n, style)
	}
	rows := tr.containerChildRows(ctx, n, style)
	if shouldUseContainer(style) && len(rows) > 0 {
		return []core.Row{tr.buildContainerRow(style, rows)}
	}
	return rows
}

func (tr *translator) containerFlexRows(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	if isColumnDirection(style.FlexDirection) {
		return tr.flexColumnRows(ctx, n, style)
	}
	rows := tr.flexRows(ctx, n, style)
	if len(rows) == 0 {
		return nil
	}
	return rows
}

func (tr *translator) containerChildRows(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	// Consecutive inline-level children (text, <strong>, <a>, <span>, …) are
	// grouped into a single anonymous paragraph row — a CSS inline formatting
	// context — so mixed inline content like "<strong>1</strong> = text" stays
	// on one line. Block children flush the pending inline group.
	var rows []core.Row
	var inlineGroup []*dom.Node
	flushInline := func() {
		if len(inlineGroup) == 0 {
			return
		}
		if r, ok := tr.inlineGroupRow(inlineGroup, style); ok {
			rows = append(rows, r)
		}
		inlineGroup = nil
	}
	for _, c := range n.Children() {
		err := translationCanceled(ctx)
		if err != nil {
			tr.err = err
			return nil
		}
		if isInlineGroupableChild(c) {
			inlineGroup = append(inlineGroup, c)
			continue
		}
		flushInline()
		rows = append(rows, tr.blockRowsWithParent(ctx, c, style)...)
	}
	flushInline()
	return rows
}

// isInlineGroupableChild reports whether a child node participates in its
// parent block's inline formatting context (so consecutive such children share
// one paragraph row). Text nodes and inline formatting elements qualify; block
// tags and the replaced elements the engine renders as their own block rows
// (img/picture/svg) do not.
func isInlineGroupableChild(n *dom.Node) bool {
	tag := n.Tag()
	if tag == "" {
		return true // text node
	}
	switch tag {
	case tagImg, tagPicture, tagSVG:
		return false
	}
	return !dom.IsBlockTag(tag)
}

func translationCanceled(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("html: translation canceled: %w", err)
	}
	return nil
}
