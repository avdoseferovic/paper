package paper

import (
	"errors"

	"github.com/avdoseferovic/paper/internal/cache"

	paperprovider "github.com/avdoseferovic/paper/internal/providers/paper"

	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

var (
	ErrCannotGenerateInLowMemoryMode       = errors.New("an error has occurred while trying to generate PDFs in low memory mode")
	ErrCannotGenerateInParallelMode        = errors.New("an error has occurred while trying to generate PDFs concurrently")
	ErrFooterHeightIsGreaterThanUsefulArea = errors.New("footer height is greater than page useful area")
	ErrHeaderHeightIsGreaterThanUsefulArea = errors.New("header height is greater than page useful area")
	// ErrHTMLHeaderAfterContent is returned by AddHTML when the HTML contains
	// a top-level <header>/<footer> but the document already has rows or a
	// registered header/footer: late registration would not reserve space on
	// already-built pages.
	ErrHTMLHeaderAfterContent = errors.New("html top-level header/footer must be added before any other content")
)

type Paper struct {
	config      *entity.Config
	provider    core.Provider
	cache       cache.Cache
	pageBuilder *pageBuilder
}

// New creates a concrete Paper instance.
// It's optional to provide an *entity.Config with customizations
// those customization are created by using the config.Builder.
func New(cfgs ...*entity.Config) *Paper {
	return NewPaper(cfgs...)
}

// NewPaper creates a concrete Paper instance.
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

// FitInCurrentPage reports whether a row of the given height fits in the remaining useful area of the current page.
func (m *Paper) FitInCurrentPage(heightNewLine float64) bool {
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

// GetStructure is responsible for return the component tree, this is useful
// on unit tests cases.
func (m *Paper) GetStructure() *node.Node[core.Structure] {
	return m.pageBuilder.getStructure()
}

func getConfig(configs ...*entity.Config) *entity.Config {
	if len(configs) > 0 {
		return config.NormalizeConfig(configs[0])
	}

	return config.NormalizeConfig(nil)
}

func getProvider(cache cache.Cache, cfg *entity.Config) core.Provider {
	deps := paperprovider.NewBuilder().Build(cfg, cache)
	provider := paperprovider.New(deps)
	provider.SetMetadata(cfg.Metadata)
	provider.SetCompression(cfg.Compression)
	provider.SetProtection(cfg.Protection)
	return provider
}
