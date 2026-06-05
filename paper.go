package paper

import (
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/avdoseferovic/paper/v2/pkg/consts/generation"
	"github.com/f-amaral/go-async/pool"

	"github.com/avdoseferovic/paper/v2/internal/cache"

	paperprovider "github.com/avdoseferovic/paper/v2/internal/providers/paper"

	"github.com/avdoseferovic/paper/v2/pkg/merge"

	"github.com/avdoseferovic/paper/v2/pkg/core/entity"

	"github.com/avdoseferovic/paper/v2/pkg/tree/node"

	"github.com/avdoseferovic/paper/v2/pkg/components/col"
	"github.com/avdoseferovic/paper/v2/pkg/components/page"
	"github.com/avdoseferovic/paper/v2/pkg/components/row"
	"github.com/avdoseferovic/paper/v2/pkg/config"
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/html"
)

var (
	ErrCannotGenerateInLowMemoryMode       = errors.New("an error has occurred while trying to generate PDFs in low memory mode")
	ErrCannotGenerateInParallelMode        = errors.New("an error has occurred while trying to generate PDFs concurrently")
	ErrFooterHeightIsGreaterThanUsefulArea = errors.New("footer height is greater than page useful area")
	ErrHeaderHeightIsGreaterThanUsefulArea = errors.New("header height is greater than page useful area")
)

type Paper struct {
	config   *entity.Config
	provider core.Provider
	cache    cache.Cache

	// Building
	cell          entity.Cell
	pages         []core.Page
	rows          []core.Row
	header        []core.Row
	footer        []core.Row
	headerHeight  float64
	footerHeight  float64
	currentHeight float64
}

// GetCurrentConfig is responsible for returning the current settings from the file
func (m *Paper) GetCurrentConfig() *entity.Config {
	return m.config
}

// New is responsible for create a new instance of core.Paper.
// It's optional to provide an *entity.Config with customizations
// those customization are created by using the config.Builder.
func New(cfgs ...*entity.Config) core.Paper {
	cache := cache.New()
	cfg := getConfig(cfgs...)
	provider := getProvider(cache, cfg)

	m := &Paper{
		provider: provider,
		cell: entity.NewRootCell(cfg.Dimensions.Width, cfg.Dimensions.Height, entity.Margins{
			Left:   cfg.Margins.Left,
			Top:    cfg.Margins.Top,
			Right:  cfg.Margins.Right,
			Bottom: cfg.Margins.Bottom,
		}),
		cache:  cache,
		config: cfg,
	}

	return m
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
	for _, page := range pages {
		if m.currentHeight != m.headerHeight {
			m.fillPageToAddNew()
			m.addHeader()
		}
		m.addRows(page.GetRows()...)
	}
}

// AddRows is responsible for add rows in the current document.
// By adding a row, if the row will extrapolate the useful area of a page,
// paper will automatically add a new page. Paper use the information of
// PageSize, PageMargin, FooterSize and HeaderSize to calculate the useful
// area of a page.
func (m *Paper) AddRows(rows ...core.Row) {
	m.addRows(rows...)
}

// AddRow is responsible for add one row in the current document.
// By adding a row, if the row will extrapolate the useful area of a page,
// paper will automatically add a new page. Paper use the information of
// PageSize, PageMargin, FooterSize and HeaderSize to calculate the useful
// area of a page.
func (m *Paper) AddRow(rowHeight float64, cols ...core.Col) core.Row {
	r := row.New(rowHeight).Add(cols...)
	m.addRow(r)
	return r
}

// AddAutoRow is responsible for adding a line with automatic height to the
// current document.
// The row height will be calculated based on its content.
func (m *Paper) AddAutoRow(cols ...core.Col) core.Row {
	r := row.New().Add(cols...)
	m.addRow(r)
	return r
}

// AddHTML parses an HTML string into Paper rows and adds them to the current document.
// Headers, footers, and pagination continue to work as with manually constructed rows.
// For advanced options (e.g. html.WithImageBaseDir for safe <img> loading), call
// html.FromString directly and append the returned rows via m.AddRows(rows...).
// Supported HTML subset is documented in docs/v2/html-support.md.
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
	contentSize := m.getRowsHeight(m.rows...) + m.footerHeight + m.headerHeight
	return contentSize+heightNewLine < m.cell.Height
}

// RegisterHeader is responsible to define a set of rows as a header
// of the document. The header will appear in every new page of the document.
// The header cannot occupy an area greater than the useful area of the page,
// it this case the method will return an error.
func (m *Paper) RegisterHeader(rows ...core.Row) error {
	height := m.getRowsHeight(rows...)
	if height+m.footerHeight > m.config.Dimensions.Height {
		return ErrHeaderHeightIsGreaterThanUsefulArea
	}

	m.headerHeight = height
	m.header = rows

	for _, headerRow := range rows {
		m.addRow(headerRow)
	}

	return nil
}

// RegisterFooter is responsible to define a set of rows as a footer
// of the document. The footer will appear in every new page of the document.
// The footer cannot occupy an area greater than the useful area of the page,
// it this case the method will return an error.
func (m *Paper) RegisterFooter(rows ...core.Row) error {
	height := m.getRowsHeight(rows...)
	if height > m.config.Dimensions.Height {
		return ErrFooterHeightIsGreaterThanUsefulArea
	}

	m.footerHeight = height
	m.footer = rows
	return nil
}

// Generate is responsible to compute the component tree created by
// the usage of all other Paper methods, and generate the PDF document.
func (m *Paper) Generate() (core.Document, error) {
	m.fillPageToAddNew()
	m.setConfig()

	if m.config.Protection != nil {
		return m.generate()
	}

	if m.config.GenerationMode == generation.Concurrent {
		return m.generateConcurrently()
	}

	if m.config.GenerationMode == generation.SequentialLowMemory {
		return m.generateLowMemory()
	}

	return m.generate()
}

// GetStructure is responsible for return the component tree, this is useful
// on unit tests cases.
func (m *Paper) GetStructure() *node.Node[core.Structure] {
	m.fillPageToAddNew()

	str := core.Structure{
		Type:    "paper",
		Details: m.config.ToMap(),
	}
	node := node.New(str)

	for _, p := range m.pages {
		inner := p.GetStructure()
		node.AddNext(inner)
	}

	return node
}

func (m *Paper) addRows(rows ...core.Row) {
	for _, row := range rows {
		m.addRow(row)
	}
}

func (m *Paper) addRow(r core.Row) {
	// PageBreaker rows signal a hard page break; they are not placed on any page.
	if pb, ok := r.(core.PageBreaker); ok && pb.IsPageBreak() {
		m.fillPageToAddNew()
		m.addHeader()
		return
	}

	if len(r.GetColumns()) == 0 {
		r.Add(col.New())
	}

	maxHeight := m.cell.Height

	r.SetConfig(m.config)
	rowHeight := r.GetHeight(m.provider, &m.cell)
	sumHeight := rowHeight + m.currentHeight + m.footerHeight

	// Row smaller than the remaining space on page
	if sumHeight <= maxHeight {
		m.currentHeight += rowHeight
		m.rows = append(m.rows, r)
		return
	}

	// Row is too tall. Check if it implements Splittable for cross-page splitting.
	if sp, ok := r.(core.Splittable); ok && m.addSplittableRow(sp, maxHeight) {
		return
	}

	// As row will extrapolate page, we will add empty space
	// on the page to force a new page
	m.fillPageToAddNew()

	m.addHeader()

	// AddRows row on the new page
	m.currentHeight += rowHeight
	m.rows = append(m.rows, r)
}

// addSplittableRow handles cross-page splitting for a row that implements
// core.Splittable. Returns true when the split was performed (caller should return).
func (m *Paper) addSplittableRow(sp core.Splittable, maxHeight float64) bool {
	remaining := maxHeight - m.currentHeight - m.footerHeight
	first, rest, didSplit := sp.SplitAt(m.provider, remaining)
	if !didSplit {
		return false
	}
	if first != nil {
		first.SetConfig(m.config)
		m.currentHeight += first.GetHeight(m.provider, &m.cell)
		m.rows = append(m.rows, first)
	}
	m.fillPageToAddNew()
	m.addHeader()
	if rest != nil {
		m.addRow(rest)
	}
	return true
}

func (m *Paper) addHeader() {
	for _, headerRow := range m.header {
		m.currentHeight += headerRow.GetHeight(m.provider, &m.cell)
		m.rows = append(m.rows, headerRow)
	}
}

func (m *Paper) fillPageToAddNew() {
	space := m.cell.Height - m.currentHeight - m.footerHeight

	// Truncate space to 9 decimal places to avoid rounding errors
	space = math.Floor(space*math.Pow10(9)) / math.Pow10(9)

	c := col.New(m.config.MaxGridSize)
	spaceRow := row.New(space)
	spaceRow.Add(c)

	m.rows = append(m.rows, spaceRow)
	m.rows = append(m.rows, m.footer...)

	var p core.Page
	if m.config.PageNumber != nil {
		p = page.New(*m.config.PageNumber)
	} else {
		p = page.New()
	}

	p.SetConfig(m.config)
	p.Add(m.rows...)

	m.pages = append(m.pages, p)
	m.rows = nil
	m.currentHeight = 0
}

func (m *Paper) setConfig() {
	for i, page := range m.pages {
		page.SetConfig(m.config)
		page.SetNumber(i+1, len(m.pages))
	}
}

func (m *Paper) generate() (core.Document, error) {
	innerCtx := m.cell.Copy()

	for i, page := range m.pages {
		ensureProviderPage(m.provider, i+1)
		page.Render(m.provider, innerCtx)
	}

	documentBytes, err := m.provider.GenerateBytes()
	if err != nil {
		return nil, err
	}

	return core.NewPDF(documentBytes, nil), nil
}

func (m *Paper) generateConcurrently() (core.Document, error) {
	p := pool.NewPool[[]core.Page, []byte](m.config.ChunkWorkers, m.processPage,
		pool.WithSortingOutput[[]core.Page, []byte]())
	defer p.Close()
	chunks := len(m.pages) / m.config.ChunkWorkers
	if chunks == 0 {
		chunks = 1
	}
	pageGroups := make([][]core.Page, 0)
	for i := 0; i < len(m.pages); i += chunks {
		end := min(i+chunks, len(m.pages))
		pageGroups = append(pageGroups, m.pages[i:end])
	}

	processed := p.Process(pageGroups)
	if processed.HasError {
		return nil, ErrCannotGenerateInParallelMode
	}

	pdfs := make([][]byte, len(processed.Results))
	for i, result := range processed.Results {
		bytes, _ := result.Output.([]byte)
		pdfs[i] = bytes
	}

	mergedBytes, err := merge.Bytes(pdfs...)
	if err != nil {
		return nil, err
	}

	return core.NewPDF(mergedBytes, nil), nil
}

func (m *Paper) generateLowMemory() (core.Document, error) {
	chunks := len(m.pages) / m.config.ChunkWorkers
	if chunks == 0 {
		chunks = 1
	}

	pageGroups := make([][]core.Page, 0)
	for i := 0; i < len(m.pages); i += chunks {
		end := min(i+chunks, len(m.pages))
		pageGroups = append(pageGroups, m.pages[i:end])
	}

	var pdfResults [][]byte
	for _, pageGroup := range pageGroups {
		bytes, err := m.processPage(pageGroup)
		if err != nil {
			return nil, ErrCannotGenerateInLowMemoryMode
		}

		pdfResults = append(pdfResults, bytes)
	}

	mergedBytes, err := merge.Bytes(pdfResults...)
	if err != nil {
		return nil, err
	}

	return core.NewPDF(mergedBytes, nil), nil
}

func (m *Paper) processPage(pages []core.Page) ([]byte, error) {
	innerCtx := m.cell.Copy()

	innerProvider := getProvider(cache.NewMutexDecorator(cache.New()), m.config)
	for i, page := range pages {
		ensureProviderPage(innerProvider, i+1)
		page.Render(innerProvider, innerCtx)
	}

	return innerProvider.GenerateBytes()
}

func ensureProviderPage(provider core.Provider, pageNumber int) {
	if pp, ok := provider.(core.PageProvider); ok {
		pp.EnsurePage(pageNumber)
	}
}

func (m *Paper) getRowsHeight(rows ...core.Row) float64 {
	var height float64
	for _, r := range rows {
		r.SetConfig(m.config)
		height += r.GetHeight(m.provider, &m.cell)
	}

	return height
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
