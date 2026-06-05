package paper

import (
	"math"

	"github.com/avdoseferovic/paper/v2/pkg/components/col"
	"github.com/avdoseferovic/paper/v2/pkg/components/page"
	"github.com/avdoseferovic/paper/v2/pkg/components/row"
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/tree/node"
)

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

func (m *Paper) getRowsHeight(rows ...core.Row) float64 {
	var height float64
	for _, r := range rows {
		r.SetConfig(m.config)
		height += r.GetHeight(m.provider, &m.cell)
	}

	return height
}
