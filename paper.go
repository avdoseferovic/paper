package paper

import (
	"errors"
	"fmt"
	"io"

	"github.com/avdoseferovic/paper/internal/cache"

	paperprovider "github.com/avdoseferovic/paper/internal/providers/paper"

	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html"
)

var (
	ErrCannotGenerateInLowMemoryMode       = errors.New("an error has occurred while trying to generate PDFs in low memory mode")
	ErrCannotGenerateInParallelMode        = errors.New("an error has occurred while trying to generate PDFs concurrently")
	ErrFooterHeightIsGreaterThanUsefulArea = errors.New("footer height is greater than page useful area")
	ErrHeaderHeightIsGreaterThanUsefulArea = errors.New("header height is greater than page useful area")
)

type Paper struct {
	config      *entity.Config
	provider    core.Provider
	cache       cache.Cache
	pageBuilder *pageBuilder
}

// New is responsible for create a new instance of core.Paper.
// It's optional to provide an *entity.Config with customizations
// those customization are created by using the config.Builder.
func New(cfgs ...*entity.Config) core.Paper {
	return NewPaper(cfgs...)
}

// NewPaper creates a concrete Paper instance. Use New when the public
// core.Paper interface is enough.
func NewPaper(cfgs ...*entity.Config) *Paper {
	cache := cache.New()
	cfg := getConfig(cfgs...)
	provider := getProvider(cache, cfg)

	m := &Paper{
		provider:    provider,
		cache:       cache,
		config:      cfg,
		pageBuilder: newPageBuilder(cfg, provider),
	}

	return m
}

// GetCurrentConfig is responsible for returning the current settings from the file
func (m *Paper) GetCurrentConfig() *entity.Config {
	return m.config
}

// FromHTML converts an HTML string directly into a PDF document.
// It is the shortest path for callers that only need HTML-to-PDF output.
// Optional configs are the same configs accepted by New.
func FromHTML(htmlStr string, cfgs ...*entity.Config) (core.Document, error) {
	m := New(cfgs...)
	err := m.AddHTML(htmlStr)
	if err != nil {
		return nil, err
	}
	return m.Generate()
}

// FromHTMLReader reads HTML from r and converts it directly into a PDF document.
// Optional configs are the same configs accepted by New.
func FromHTMLReader(r io.Reader, cfgs ...*entity.Config) (core.Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("paper: reading HTML: %w", err)
	}
	return FromHTML(string(data), cfgs...)
}

// AddPages is responsible for add pages directly in the document.
// By adding a page directly, the current cursor will reset and the
// new page will appear as the next. If the page provided have
// more rows than the maximum useful area of a page, paper will split
// that page in more than one.
func (m *Paper) AddPages(pages ...core.Page) {
	m.pageBuilder.addPages(pages...)
}

// AddRows is responsible for add rows in the current document.
// By adding a row, if the row will extrapolate the useful area of a page,
// paper will automatically add a new page. Paper use the information of
// PageSize, PageMargin, FooterSize and HeaderSize to calculate the useful
// area of a page.
func (m *Paper) AddRows(rows ...core.Row) {
	m.pageBuilder.addRows(rows...)
}

// AddRow is responsible for add one row in the current document.
// By adding a row, if the row will extrapolate the useful area of a page,
// paper will automatically add a new page. Paper use the information of
// PageSize, PageMargin, FooterSize and HeaderSize to calculate the useful
// area of a page.
func (m *Paper) AddRow(rowHeight float64, cols ...core.Col) core.Row {
	r := row.New(rowHeight).Add(cols...)
	m.pageBuilder.addRow(r)
	return r
}

// AddAutoRow is responsible for adding a line with automatic height to the
// current document.
// The row height will be calculated based on its content.
func (m *Paper) AddAutoRow(cols ...core.Col) core.Row {
	r := row.New().Add(cols...)
	m.pageBuilder.addRow(r)
	return r
}

// AddHTML parses an HTML string into Paper rows and adds them to the current document.
// Headers, footers, and pagination continue to work as with manually constructed rows.
// For advanced options (e.g. html.WithImageBaseDir for safe <img> loading), call
// html.FromString directly and append the returned rows via m.AddRows(rows...).
// Supported HTML subset is documented in docs/html-support.md.
func (m *Paper) AddHTML(htmlStr string) error {
	opts := []html.Option{html.WithGridSize(m.config.MaxGridSize)}
	if m.config.Dimensions != nil {
		contentWidth := m.config.Dimensions.Width - m.config.Margins.Left - m.config.Margins.Right
		if contentWidth > 0 {
			opts = append(opts, html.WithContentWidth(contentWidth))
		}
	}
	rows, err := html.FromString(htmlStr, opts...)
	if err != nil {
		return err
	}
	m.AddRows(rows...)
	return nil
}

// FitlnCurrentPage is responsible to validating whether a line fits on
// the current page.
func (m *Paper) FitlnCurrentPage(heightNewLine float64) bool {
	return m.pageBuilder.fitInCurrentPage(heightNewLine)
}

// RegisterHeader is responsible to define a set of rows as a header
// of the document. The header will appear in every new page of the document.
// The header cannot occupy an area greater than the useful area of the page,
// it this case the method will return an error.
func (m *Paper) RegisterHeader(rows ...core.Row) error {
	return m.pageBuilder.registerHeader(rows...)
}

// RegisterFooter is responsible to define a set of rows as a footer
// of the document. The footer will appear in every new page of the document.
// The footer cannot occupy an area greater than the useful area of the page,
// it this case the method will return an error.
func (m *Paper) RegisterFooter(rows ...core.Row) error {
	return m.pageBuilder.registerFooter(rows...)
}

func getConfig(configs ...*entity.Config) *entity.Config {
	if len(configs) > 0 {
		return configs[0]
	}

	return config.NewBuilder().Build()
}

func getProvider(cache cache.Cache, cfg *entity.Config) core.Provider {
	deps := paperprovider.NewBuilder().Build(cfg, cache)
	provider := paperprovider.New(deps)
	provider.SetMetadata(cfg.Metadata)
	provider.SetCompression(cfg.Compression)
	provider.SetProtection(cfg.Protection)
	return provider
}
