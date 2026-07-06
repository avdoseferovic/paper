package paper

import (
	"math"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

type pageBuilder struct {
	config   *entity.Config
	provider core.Provider
	cell     entity.Cell

	pages             []core.Page
	pageNumbered      []bool
	pageNumberCounted []bool
	rows              []core.Row
	header            []core.Row
	footer            []core.Row
	headerHeight      float64
	footerHeight      float64
	currentHeight     float64
	currentControl    core.PageControl
}

func newPageBuilder(config *entity.Config, provider core.Provider) *pageBuilder {
	return &pageBuilder{
		config:   config,
		provider: provider,
		cell: entity.NewRootCell(config.Dimensions.Width, config.Dimensions.Height, entity.Margins{
			Left:   config.Margins.Left,
			Top:    config.Margins.Top,
			Right:  config.Margins.Right,
			Bottom: config.Margins.Bottom,
		}),
	}
}

func (b *pageBuilder) getStructure() *node.Node[core.Structure] {
	b.finalize()

	str := core.Structure{
		Type:    "paper",
		Details: b.config.ToMap(),
	}
	node := node.New(str)

	for _, p := range b.pages {
		inner := p.GetStructure()
		node.AddNext(inner)
	}

	return node
}

func (b *pageBuilder) finalize() {
	if b.shouldCommitPendingPage() {
		b.fillPageToAddNew()
	}
	b.setConfig()
}

func (b *pageBuilder) shouldCommitPendingPage() bool {
	if len(b.pages) == 0 {
		return true
	}
	return len(b.rows) > 0 || b.currentHeight != 0
}

func (b *pageBuilder) addPages(pages ...core.Page) {
	for _, page := range pages {
		if b.currentHeight != b.headerHeight {
			b.fillPageToAddNew()
			b.addHeader()
		}
		b.addRows(page.GetRows()...)
	}
}

func (b *pageBuilder) addRows(rows ...core.Row) {
	for _, row := range rows {
		b.addRow(row)
	}
}

func (b *pageBuilder) addRow(r core.Row) {
	if pc, ok := r.(core.PageController); ok {
		b.applyPageControl(pc.PageControl())
		return
	}

	// PageBreaker rows signal a hard page break; they are not placed on any page.
	if pb, ok := r.(core.PageBreaker); ok && pb.IsPageBreak() {
		// Collapse hard breaks at the top of a page. CSS page-break-before on
		// the first element and consecutive hard breaks must not emit blanks.
		if !b.isAtTopOfUsablePage() {
			b.fillPageToAddNew()
			b.addHeader()
		}
		return
	}

	if len(r.GetColumns()) == 0 {
		r.Add(col.New())
	}

	maxHeight := b.effectiveCellHeight()

	r.SetConfig(b.config)
	rowHeight := r.GetHeight(b.provider, &b.cell)
	sumHeight := rowHeight + b.currentHeight + b.effectiveFooterHeight()

	// Row smaller than the remaining space on page.
	if sumHeight <= maxHeight {
		b.currentHeight += rowHeight
		b.rows = append(b.rows, r)
		return
	}

	// Row is too tall. Check if it implements Splittable for cross-page splitting.
	if sp, ok := r.(core.Splittable); ok && b.addSplittableRow(r, sp, maxHeight) {
		return
	}

	// As row will extrapolate page, add empty space on the page to force a new page.
	b.fillPageToAddNew()

	b.addHeader()

	// AddRows row on the new page.
	b.currentHeight += rowHeight
	b.rows = append(b.rows, r)
}

// addSplittableRow handles cross-page splitting for a row that implements
// core.Splittable. Returns true when the split was performed (caller should return).
func (b *pageBuilder) addSplittableRow(row core.Row, sp core.Splittable, maxHeight float64) bool {
	splitControl := clonePageControl(b.currentControl)
	remaining := maxHeight - b.currentHeight - b.effectiveFooterHeight()
	first, rest, didSplit := sp.SplitAt(b.provider, remaining, b.cell.Width)
	if !didSplit {
		return false
	}
	if first == nil {
		if rest == nil {
			rest = row
		}
		if b.isAtTopOfUsablePage() {
			b.appendOversizedRow(rest)
			return true
		}
		b.fillPageToAddNew()
		b.currentControl = continuationPageControl(splitControl)
		b.addHeader()
		b.addRow(rest)
		return true
	}
	first.SetConfig(b.config)
	b.currentHeight += first.GetHeight(b.provider, &b.cell)
	b.rows = append(b.rows, first)
	b.fillPageToAddNew()
	b.currentControl = continuationPageControl(splitControl)
	b.addHeader()
	if rest != nil {
		b.addRow(rest)
	}
	return true
}

func continuationPageControl(control core.PageControl) core.PageControl {
	if control.DecorFirstPageOnly {
		control.SuppressFooter = false
		control.SuppressPageNumber = false
	}
	if control.ContinuationTopMargin != nil {
		top := *control.ContinuationTopMargin
		control.TopMargin = &top
	} else if control.TopMarginFirstPageOnly {
		control.TopMargin = nil
	}
	if control.ContinuationRenderOffsetX != nil {
		x := *control.ContinuationRenderOffsetX
		control.RenderOffsetX = &x
	} else if control.RenderOffsetFirstPageOnly {
		control.RenderOffsetX = nil
	}
	if control.ContinuationRenderOffsetY != nil {
		y := *control.ContinuationRenderOffsetY
		control.RenderOffsetY = &y
	} else if control.RenderOffsetFirstPageOnly {
		control.RenderOffsetY = nil
	}
	return control
}

func clonePageControl(control core.PageControl) core.PageControl {
	if control.TopMargin != nil {
		top := *control.TopMargin
		control.TopMargin = &top
	}
	if control.ContinuationTopMargin != nil {
		top := *control.ContinuationTopMargin
		control.ContinuationTopMargin = &top
	}
	if control.RenderOffsetX != nil {
		x := *control.RenderOffsetX
		control.RenderOffsetX = &x
	}
	if control.RenderOffsetY != nil {
		y := *control.RenderOffsetY
		control.RenderOffsetY = &y
	}
	if control.ContinuationRenderOffsetX != nil {
		x := *control.ContinuationRenderOffsetX
		control.ContinuationRenderOffsetX = &x
	}
	if control.ContinuationRenderOffsetY != nil {
		y := *control.ContinuationRenderOffsetY
		control.ContinuationRenderOffsetY = &y
	}
	return control
}

func (b *pageBuilder) isAtTopOfUsablePage() bool {
	const heightEpsilon = 0.000001
	return b.currentHeight <= b.headerHeight+heightEpsilon
}

func (b *pageBuilder) appendOversizedRow(r core.Row) {
	if len(r.GetColumns()) == 0 {
		r.Add(col.New())
	}
	r.SetConfig(b.config)
	b.currentHeight += r.GetHeight(b.provider, &b.cell)
	b.rows = append(b.rows, r)
}

func (b *pageBuilder) addHeader() {
	for _, headerRow := range b.header {
		b.currentHeight += headerRow.GetHeight(b.provider, &b.cell)
		b.rows = append(b.rows, headerRow)
	}
}

func (b *pageBuilder) fillPageToAddNew() {
	space := b.effectiveCellHeight() - b.currentHeight - b.effectiveFooterHeight()

	// Truncate space to 9 decimal places to avoid rounding errors.
	space = math.Floor(space*math.Pow10(9)) / math.Pow10(9)

	c := col.New(b.config.MaxGridSize)
	spaceRow := row.New(space)
	spaceRow.Add(c)

	b.rows = append(b.rows, spaceRow)
	if !b.currentControl.SuppressFooter {
		b.rows = append(b.rows, b.footer...)
	}

	var p core.Page
	numbered := b.config.PageNumber != nil && !b.currentControl.SuppressPageNumber
	counted := numbered || (b.config.PageNumber != nil && b.currentControl.CountPageNumber)
	if numbered {
		p = page.New(*b.config.PageNumber)
	} else {
		p = page.New()
	}

	p.SetConfig(b.config)
	if controlledPage, ok := p.(interface {
		SetPageControl(control core.PageControl)
	}); ok {
		controlledPage.SetPageControl(b.currentControl)
	}
	p.Add(b.rows...)

	b.appendPage(p, numbered, counted)
	b.rows = nil
	b.currentHeight = 0
	b.currentControl = core.PageControl{}
}

func (b *pageBuilder) setConfig() {
	numberedTotal := 0
	for _, counted := range b.pageNumberCounted {
		if counted {
			numberedTotal++
		}
	}
	numberedCurrent := 0
	for i, page := range b.pages {
		page.SetConfig(b.config)
		if indexed, ok := page.(interface{ SetPageIndex(index int) }); ok {
			indexed.SetPageIndex(i + 1)
		}
		if i < len(b.pageNumberCounted) && b.pageNumberCounted[i] {
			numberedCurrent++
		}
		if i < len(b.pageNumbered) && b.pageNumbered[i] {
			page.SetNumber(numberedCurrent, numberedTotal)
		} else {
			page.SetNumber(0, numberedTotal)
		}
	}
}

func (b *pageBuilder) fitInCurrentPage(heightNewLine float64) bool {
	contentSize := b.getRowsHeight(b.rows...) + b.effectiveFooterHeight() + b.headerHeight
	return contentSize+heightNewLine < b.effectiveCellHeight()
}

func (b *pageBuilder) applyPageControl(control core.PageControl) {
	if control.Blank {
		b.addBlankPage(control)
		return
	}
	if control.TopMargin != nil && !b.isAtTopOfUsablePage() {
		b.fillPageToAddNew()
		b.addHeader()
	}
	b.currentControl.SuppressFooter = b.currentControl.SuppressFooter || control.SuppressFooter
	b.currentControl.SuppressPageNumber = b.currentControl.SuppressPageNumber || control.SuppressPageNumber
	b.currentControl.CountPageNumber = b.currentControl.CountPageNumber || control.CountPageNumber
	b.currentControl.DecorFirstPageOnly = b.currentControl.DecorFirstPageOnly || control.DecorFirstPageOnly
	b.currentControl.TopMarginFirstPageOnly = b.currentControl.TopMarginFirstPageOnly || control.TopMarginFirstPageOnly
	b.currentControl.RenderOffsetFirstPageOnly = b.currentControl.RenderOffsetFirstPageOnly || control.RenderOffsetFirstPageOnly
	if control.ContinuationTopMargin != nil {
		top := *control.ContinuationTopMargin
		b.currentControl.ContinuationTopMargin = &top
	}
	if control.ContinuationRenderOffsetX != nil {
		x := *control.ContinuationRenderOffsetX
		b.currentControl.ContinuationRenderOffsetX = &x
	}
	if control.ContinuationRenderOffsetY != nil {
		y := *control.ContinuationRenderOffsetY
		b.currentControl.ContinuationRenderOffsetY = &y
	}
	if control.TopMargin != nil {
		top := *control.TopMargin
		b.currentControl.TopMargin = &top
	}
	if control.RenderOffsetX != nil {
		x := *control.RenderOffsetX
		b.currentControl.RenderOffsetX = &x
	}
	if control.RenderOffsetY != nil {
		y := *control.RenderOffsetY
		b.currentControl.RenderOffsetY = &y
	}
}

func (b *pageBuilder) addBlankPage(control core.PageControl) {
	if !b.isAtTopOfUsablePage() {
		b.fillPageToAddNew()
	}
	b.rows = nil
	b.currentHeight = 0
	numbered := b.config.PageNumber != nil && !control.SuppressPageNumber
	counted := numbered || (b.config.PageNumber != nil && control.CountPageNumber)
	var p core.Page
	if numbered {
		p = page.New(*b.config.PageNumber)
	} else {
		p = page.New()
	}
	p.SetConfig(b.config)
	footerHeight := b.footerHeight
	if control.SuppressFooter {
		footerHeight = 0
	}
	space := math.Floor((b.cell.Height-footerHeight)*math.Pow10(9)) / math.Pow10(9)
	spaceRow := row.New(space).Add(col.New(b.config.MaxGridSize))
	p.Add(spaceRow)
	if !control.SuppressFooter {
		p.Add(b.footer...)
	}
	b.appendPage(p, numbered, counted)
	b.addHeader()
}

func (b *pageBuilder) effectiveFooterHeight() float64 {
	if b.currentControl.SuppressFooter {
		return 0
	}
	return b.footerHeight
}

func (b *pageBuilder) effectiveCellHeight() float64 {
	height := b.cell.Height
	if b.currentControl.TopMargin == nil || b.config == nil || b.config.Margins == nil {
		return height
	}
	height += b.config.Margins.Top - *b.currentControl.TopMargin
	if height < 0 {
		return 0
	}
	return height
}

func (b *pageBuilder) appendPage(p core.Page, numbered bool, counted bool) {
	b.pages = append(b.pages, p)
	b.pageNumbered = append(b.pageNumbered, numbered)
	b.pageNumberCounted = append(b.pageNumberCounted, counted)
}

// hasContent reports whether any pages or rows were already added.
func (b *pageBuilder) hasContent() bool {
	return len(b.pages) > 0 || len(b.rows) > 0
}

// hasHeaderOrFooter reports whether a header or footer is already registered.
func (b *pageBuilder) hasHeaderOrFooter() bool {
	return len(b.header) > 0 || len(b.footer) > 0
}

func (b *pageBuilder) registerHeader(rows ...core.Row) error {
	height := b.getRowsHeight(rows...)
	if height+b.footerHeight > b.cell.Height {
		return ErrHeaderHeightIsGreaterThanUsefulArea
	}

	b.headerHeight = height
	b.header = rows

	for _, headerRow := range rows {
		b.addRow(headerRow)
	}

	return nil
}

func (b *pageBuilder) registerFooter(rows ...core.Row) error {
	height := b.getRowsHeight(rows...)
	if height+b.headerHeight > b.cell.Height {
		return ErrFooterHeightIsGreaterThanUsefulArea
	}

	b.footerHeight = height
	b.footer = rows
	return nil
}

func (b *pageBuilder) getRowsHeight(rows ...core.Row) float64 {
	var height float64
	for _, r := range rows {
		r.SetConfig(b.config)
		height += r.GetHeight(b.provider, &b.cell)
	}

	return height
}
