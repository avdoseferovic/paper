// Package translate converts a parsed HTML DOM into a slice of Paper rows.
package translate

import (
	"context"
	"fmt"

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
	// If the element has an id, wrap its first row in an anchorTarget so the
	// PDF destination registers at the element's actual Y position.
	if id := n.Attr("id"); id != "" && len(rows) > 0 && tr.anchorReg != nil {
		rows[0] = wrapRowAnchorTarget(rows[0], id, tr.anchorReg)
	}
	// Prepend/append page-break markers when CSS requests them.
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

// dispatchBlockRows is the original blockRows tag switch (split out so the
// outer blockRows can handle anchor wrapping uniformly).
func (tr *translator) dispatchBlockRowsWithStyle(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	tag := n.Tag()
	// Drop metadata tags that may appear in the body — their text content
	// (CSS source, script source, meta values) must not render as visible
	// document text. <style>/<link> CSS is already extracted via StyleSources.
	switch tag {
	case "style", "link", "script", "meta", "head", "title":
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
		return tr.tableRows(n)
	case "ul", "ol":
		return tr.listRows(n)
	case "dl":
		return tr.definitionListRows(n)
	case "details":
		return tr.detailsRows(ctx, n)
	case tagImg:
		if r, ok := tr.imageRowWithStyle(n, style); ok {
			return []core.Row{r}
		}
		if tr.err != nil {
			return nil
		}
		return altRowStyled(n, style)
	case "picture":
		return tr.pictureRowWithStyle(n, style)
	case tagSVG:
		if r, ok := tr.svgRowWithStyle(n, style); ok {
			return []core.Row{r}
		}
		if tr.err != nil {
			return nil
		}
		return nil
	case "br":
		return nil // top-level <br> is a no-op
	default:
		// Container (div, section, article, header, footer, nav, etc.).
		// Compute style to detect class-based display:flex and display:none.
		if style.Display == displayNone {
			return nil
		}
		if style.Display == displayFlex {
			if isColumnDirection(style.FlexDirection) {
				return tr.flexColumnRows(ctx, n, style)
			}
			rows := tr.flexRows(ctx, n, style)
			if len(rows) == 0 {
				return nil
			}
			return rows
		}
		// Default: collect children's rows.
		var rows []core.Row
		for _, c := range n.Children() {
			err := translationCanceled(ctx)
			if err != nil {
				tr.err = err
				return nil
			}
			rows = append(rows, tr.blockRowsWithParent(ctx, c, style)...)
		}
		// When the container has background/border/padding, wrap children
		// in a single styled blockContainer so the styling spans them all.
		if shouldUseContainer(style) && len(rows) > 0 {
			return []core.Row{tr.buildContainerRow(style, rows)}
		}
		return rows
	}
}

func translationCanceled(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("html: translation canceled: %w", err)
	}
	return nil
}
