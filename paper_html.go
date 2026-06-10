package paper

import (
	"context"
	"fmt"
	"io"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html"
)

// FromHTML converts an HTML string directly into a PDF document.
// It is the shortest path for callers that only need HTML-to-PDF output.
// Optional configs are the same configs accepted by New.
func FromHTML(htmlStr string, cfgs ...*entity.Config) (*core.Pdf, error) {
	return FromHTMLCtx(context.TODO(), htmlStr, cfgs...)
}

// FromHTMLCtx converts an HTML string directly into a PDF document.
// It observes ctx while parsing/translating HTML and while generating the PDF.
//
// When no config argument is given, a plain `@page { size; margin }` rule in
// the HTML configures the page; an explicit config argument disables @page
// handling entirely (documented precedence rule, no field-level merging).
func FromHTMLCtx(ctx context.Context, htmlStr string, cfgs ...*entity.Config) (*core.Pdf, error) {
	if len(cfgs) > 0 {
		m := New(cfgs...)
		err := m.AddHTMLCtx(ctx, htmlStr)
		if err != nil {
			return nil, err
		}
		return m.GenerateCtx(ctx)
	}

	// No explicit config: pre-parse with default-config options so @page can
	// shape the document. When @page is present, the HTML is re-translated
	// against the resulting page geometry (content width affects layout).
	m := New()
	doc, err := html.DocumentFromStringCtx(ctx, htmlStr, m.htmlOptions()...)
	if err != nil {
		return nil, err
	}
	if doc.Page != nil {
		m = New(configFromPageOptions(doc.Page))
		err = m.AddHTMLCtx(ctx, htmlStr)
		if err != nil {
			return nil, err
		}
		return m.GenerateCtx(ctx)
	}
	err = m.addHTMLDocument(doc)
	if err != nil {
		return nil, err
	}
	return m.GenerateCtx(ctx)
}

// configFromPageOptions builds an entity.Config from a parsed @page rule.
func configFromPageOptions(page *html.PageOptions) *entity.Config {
	builder := config.NewBuilder()
	switch {
	case page.Width > 0 && page.Height > 0:
		width, height := page.Width, page.Height
		if page.Landscape {
			width, height = height, width
		}
		builder.WithDimensions(width, height)
	case page.PageSize != "":
		builder.WithPageSize(pagesize.Type(page.PageSize))
		if page.Landscape {
			builder.WithOrientation(consts.OrientationHorizontal)
		}
	case page.Landscape:
		builder.WithOrientation(consts.OrientationHorizontal)
	}
	if page.MarginLeft >= 0 {
		builder.WithLeftMargin(page.MarginLeft)
	}
	if page.MarginTop >= 0 {
		builder.WithTopMargin(page.MarginTop)
	}
	if page.MarginRight >= 0 {
		builder.WithRightMargin(page.MarginRight)
	}
	if page.MarginBottom >= 0 {
		builder.WithBottomMargin(page.MarginBottom)
	}
	return builder.Build()
}

// FromHTMLReader reads HTML from r and converts it directly into a PDF document.
// Optional configs are the same configs accepted by New.
func FromHTMLReader(r io.Reader, cfgs ...*entity.Config) (*core.Pdf, error) {
	return FromHTMLReaderCtx(context.TODO(), r, cfgs...)
}

// FromHTMLReaderCtx reads HTML from r and converts it directly into a PDF document.
// It observes ctx before and after reading, while translating HTML, and while
// generating the PDF.
func FromHTMLReaderCtx(ctx context.Context, r io.Reader, cfgs ...*entity.Config) (*core.Pdf, error) {
	err := generationCanceled(ctx)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("paper: reading HTML: %w", err)
	}
	err = generationCanceled(ctx)
	if err != nil {
		return nil, err
	}
	return FromHTMLCtx(ctx, string(data), cfgs...)
}

// AddHTML parses an HTML string into Paper rows and adds them to the current document.
// Headers, footers, and pagination continue to work as with manually constructed rows.
// For advanced options (e.g. html.WithImageBaseDir for safe <img> loading), call
// html.FromString directly and append the returned rows via m.AddRows(rows...).
// Supported HTML subset is documented in docs/html-support.md.
func (m *Paper) AddHTML(htmlStr string) error {
	return m.AddHTMLCtx(context.TODO(), htmlStr)
}

// AddHTMLCtx parses an HTML string into Paper rows and adds them to the current
// document. It observes ctx while parsing and translating the HTML.
func (m *Paper) AddHTMLCtx(ctx context.Context, htmlStr string) error {
	doc, err := html.DocumentFromStringCtx(ctx, htmlStr, m.htmlOptions()...)
	if err != nil {
		return err
	}
	err = generationCanceled(ctx)
	if err != nil {
		return err
	}
	return m.addHTMLDocument(doc)
}

// addHTMLDocument registers the document's repeating header/footer (ordering
// is load-bearing: registration changes the useful-area math and AddRows
// paginates incrementally) and then adds the content rows.
func (m *Paper) addHTMLDocument(doc *html.Document) error {
	err := m.registerHTMLBands(doc)
	if err != nil {
		return err
	}
	m.AddRows(doc.Rows...)
	return nil
}

// registerHTMLBands registers the first top-level <header>/<footer> rows as
// the repeating page header/footer. Bands are rejected once the document has
// content: late registration cannot reserve space on already-built pages.
func (m *Paper) registerHTMLBands(doc *html.Document) error {
	if len(doc.HeaderRows) == 0 && len(doc.FooterRows) == 0 {
		return nil
	}
	if m.pageBuilder.hasContent() || m.pageBuilder.hasHeaderOrFooter() {
		return ErrHTMLHeaderAfterContent
	}
	if len(doc.FooterRows) > 0 {
		err := m.RegisterFooter(doc.FooterRows...)
		if err != nil {
			return err
		}
	}
	if len(doc.HeaderRows) > 0 {
		err := m.RegisterHeader(doc.HeaderRows...)
		if err != nil {
			return err
		}
	}
	return nil
}

// htmlOptions derives the html conversion options from the document config.
func (m *Paper) htmlOptions() []html.Option {
	opts := []html.Option{html.WithGridSize(m.config.MaxGridSize)}
	if m.config.HTMLLimits != (entity.HTMLLimits{}) {
		opts = append(opts, html.WithLimits(m.config.HTMLLimits))
	}
	if m.config.OutlineFromHeadings {
		opts = append(opts, html.WithOutlineFromHeadings())
	}
	if m.config.Dimensions != nil {
		contentWidth := m.config.Dimensions.Width - m.config.Margins.Left - m.config.Margins.Right
		if contentWidth > 0 {
			opts = append(opts, html.WithContentWidth(contentWidth))
		}
	}
	return opts
}
