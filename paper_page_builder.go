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

	pages         []core.Page
	rows          []core.Row
	header        []core.Row
	footer        []core.Row
	headerHeight  float64
	footerHeight  float64
	currentHeight float64
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
	// PageBreaker rows signal a hard page break; they are not placed on any page.
	if pb, ok := r.(core.PageBreaker); ok && pb.IsPageBreak() {
		b.fillPageToAddNew()
		b.addHeader()
		return
	}

	if len(r.GetColumns()) == 0 {
		r.Add(col.New())
	}

	maxHeight := b.cell.Height

	r.SetConfig(b.config)
	rowHeight := r.GetHeight(b.provider, &b.cell)
	sumHeight := rowHeight + b.currentHeight + b.footerHeight

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
	remaining := maxHeight - b.currentHeight - b.footerHeight
	first, rest, didSplit := sp.SplitAt(b.provider, remaining)
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
		b.addHeader()
		b.addRow(rest)
		return true
	}
	first.SetConfig(b.config)
	b.currentHeight += first.GetHeight(b.provider, &b.cell)
	b.rows = append(b.rows, first)
	b.fillPageToAddNew()
	b.addHeader()
	if rest != nil {
		b.addRow(rest)
	}
	return true
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
	space := b.cell.Height - b.currentHeight - b.footerHeight

	// Truncate space to 9 decimal places to avoid rounding errors.
	space = math.Floor(space*math.Pow10(9)) / math.Pow10(9)

	c := col.New(b.config.MaxGridSize)
	spaceRow := row.New(space)
	spaceRow.Add(c)

	b.rows = append(b.rows, spaceRow)
	b.rows = append(b.rows, b.footer...)

	var p core.Page
	if b.config.PageNumber != nil {
		p = page.New(*b.config.PageNumber)
	} else {
		p = page.New()
	}

	p.SetConfig(b.config)
	p.Add(b.rows...)

	b.pages = append(b.pages, p)
	b.rows = nil
	b.currentHeight = 0
}

func (b *pageBuilder) setConfig() {
	for i, page := range b.pages {
		page.SetConfig(b.config)
		page.SetNumber(i+1, len(b.pages))
	}
}

func (b *pageBuilder) fitInCurrentPage(heightNewLine float64) bool {
	contentSize := b.getRowsHeight(b.rows...) + b.footerHeight + b.headerHeight
	return contentSize+heightNewLine < b.cell.Height
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
