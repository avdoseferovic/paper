package translate

import (
	"maps"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// breakInsideAvoid is the CSS break-inside / page-break-inside value that keeps
// a block from being divided across a page boundary.
const breakInsideAvoid = "avoid"

// blockContainer wraps multiple child Rows under a single background+border+padding box.
// Mirrors flexCellContent's structure but adds styled fill/border painting on Render.
type blockContainer struct {
	rows          []core.Row
	style         *props.Cell
	paddingTop    float64
	paddingRight  float64
	paddingBottom float64
	paddingLeft   float64
	height        float64
	minHeight     float64
	maxHeight     float64
	// breakInside mirrors CSS break-inside: avoid. When true the container is
	// never split across a page boundary — SplitAt pushes the whole box to the
	// next page instead of placing the rows that happen to fit.
	breakInside  bool
	config       *entity.Config
	cachedHeight float64
	cachedWidth  float64
}

type marginBox struct {
	child        core.Component
	marginTop    float64
	marginRight  float64
	marginBottom float64
	marginLeft   float64
}

type horizontalMarginRow struct {
	child       core.Row
	marginLeft  float64
	marginRight float64
}

func (r *horizontalMarginRow) SetConfig(config *entity.Config) {
	if r.child != nil {
		r.child.SetConfig(config)
	}
}

func (r *horizontalMarginRow) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type: "horizontal_margin_row",
		Details: map[string]any{
			"margin_left":  r.marginLeft,
			"margin_right": r.marginRight,
		},
	}
	n := node.New(str)
	if r.child != nil {
		n.AddNext(r.child.GetStructure())
	}
	return n
}

func (r *horizontalMarginRow) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if r.child == nil {
		return 0
	}
	inner := r.innerCell(cell)
	return r.child.GetHeight(provider, &inner)
}

func (r *horizontalMarginRow) Render(provider core.Provider, cell entity.Cell) {
	if r.child == nil {
		return
	}
	r.child.Render(provider, r.innerCell(&cell))
}

func (r *horizontalMarginRow) Add(cols ...core.Col) core.Row {
	if r.child != nil {
		r.child = r.child.Add(cols...)
	}
	return r
}

func (r *horizontalMarginRow) WithStyle(style *props.Cell) core.Row {
	if r.child != nil {
		r.child = r.child.WithStyle(style)
	}
	return r
}

func (r *horizontalMarginRow) GetColumns() []core.Col {
	if r.child == nil {
		return nil
	}
	return r.child.GetColumns()
}

func (r *horizontalMarginRow) SplitAt(provider core.Provider, remainingHeight float64, width float64) (core.Row, core.Row, bool) {
	splittable, ok := r.child.(core.Splittable)
	if !ok {
		return nil, nil, false
	}
	innerWidth := width
	if innerWidth > 0 {
		innerWidth -= r.marginLeft + r.marginRight
		if innerWidth < 0 {
			innerWidth = 0
		}
	}
	first, rest, didSplit := splittable.SplitAt(provider, remainingHeight, innerWidth)
	if !didSplit {
		return nil, nil, false
	}
	return r.wrapSplit(first), r.wrapSplit(rest), true
}

func (r *horizontalMarginRow) wrapSplit(row core.Row) core.Row {
	if row == nil {
		return nil
	}
	return &horizontalMarginRow{
		child:       row,
		marginLeft:  r.marginLeft,
		marginRight: r.marginRight,
	}
}

func (r *horizontalMarginRow) innerCell(cell *entity.Cell) entity.Cell {
	inner := cell.Copy()
	inner.X += r.marginLeft
	inner.Width -= r.marginLeft + r.marginRight
	if inner.Width < 0 {
		inner.Width = 0
	}
	return inner
}

func (m *marginBox) SetConfig(config *entity.Config) {
	if m.child != nil {
		m.child.SetConfig(config)
	}
}

func (m *marginBox) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type: "margin_box",
		Details: map[string]any{
			"margin_top":    m.marginTop,
			"margin_right":  m.marginRight,
			"margin_bottom": m.marginBottom,
			"margin_left":   m.marginLeft,
		},
	}
	n := node.New(str)
	if m.child != nil {
		n.AddNext(m.child.GetStructure())
	}
	return n
}

func (m *marginBox) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if m.child == nil {
		return m.marginTop + m.marginBottom
	}
	inner := m.innerCell(cell)
	return m.child.GetHeight(provider, &inner) + m.marginTop + m.marginBottom
}

func (m *marginBox) Render(provider core.Provider, cell *entity.Cell) {
	if m.child == nil {
		return
	}
	inner := m.innerCell(cell)
	m.child.Render(provider, &inner)
}

func (m *marginBox) innerCell(cell *entity.Cell) entity.Cell {
	inner := cell.Copy()
	inner.X += m.marginLeft
	inner.Y += m.marginTop
	inner.Width -= m.marginLeft + m.marginRight
	inner.Height -= m.marginTop + m.marginBottom
	if inner.Width < 0 {
		inner.Width = 0
	}
	if inner.Height < 0 {
		inner.Height = 0
	}
	return inner
}

// SetConfig propagates the config to every child row.
func (b *blockContainer) SetConfig(config *entity.Config) {
	b.config = config
	for _, r := range b.rows {
		r.SetConfig(config)
	}
	// Invalidate the height cache when config changes.
	b.cachedHeight = 0
	b.cachedWidth = 0
}

// GetStructure returns a "container" structure with all child row structures attached.
func (b *blockContainer) GetStructure() *node.Node[core.Structure] {
	details := map[string]any{"rows": len(b.rows)}
	if b.style != nil {
		maps.Copy(details, b.style.ToMap())
	}
	str := core.Structure{Type: "container", Details: details}
	n := node.New(str)
	for _, r := range b.rows {
		n.AddNext(r.GetStructure())
	}
	return n
}

// GetHeight returns the sum of child row heights plus top+bottom padding.
func (b *blockContainer) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if b.cachedHeight > 0 && b.cachedWidth == cell.Width {
		return b.cachedHeight
	}
	inner := cell.Copy()
	inner.Width = cell.Width - b.paddingLeft - b.paddingRight
	if inner.Width < 0 {
		inner.Width = 0
	}
	total := b.paddingTop + b.paddingBottom
	for _, r := range b.rows {
		total += r.GetHeight(provider, &inner)
	}
	if b.height > 0 {
		total = b.height
	} else {
		if b.minHeight > 0 && total < b.minHeight {
			total = b.minHeight
		}
		if b.maxHeight > 0 && total > b.maxHeight {
			total = b.maxHeight
		}
	}
	b.cachedHeight = total
	b.cachedWidth = cell.Width
	return total
}

// Render paints the background+border once, then renders each child row offset by padding.
// The pen is reset before painting the background AND before each child row so cellwriter
// chain nodes that rely on GetXY draw at the right position.
func (b *blockContainer) Render(provider core.Provider, cell *entity.Cell) {
	height := b.GetHeight(provider, cell)
	if cell.Height > 0 {
		height = cell.Height
	}
	pp, _ := provider.(core.PositionProvider)
	if b.style != nil {
		if pp != nil {
			pp.SetCursor(cell.X, cell.Y)
		}
		provider.CreateCol(cell.Width, height, b.config, b.style)
	}
	innerCell := cell.Copy()
	innerCell.X += b.paddingLeft
	innerCell.Y += b.paddingTop
	innerCell.Width = cell.Width - b.paddingLeft - b.paddingRight
	if innerCell.Width < 0 {
		innerCell.Width = 0
	}
	for _, r := range b.rows {
		h := r.GetHeight(provider, &innerCell)
		innerCell.Height = h
		if pp != nil {
			pp.SetCursor(innerCell.X, innerCell.Y)
		}
		r.Render(provider, innerCell)
		innerCell.Y += h
	}
	if pp != nil {
		pp.SetCursor(cell.X, cell.Y)
	}
}

// shouldUseContainer reports whether a container's computed style has anything
// worth painting around its children: background, border on any side, or padding.
func shouldUseContainer(style *css.ComputedStyle) bool {
	if style == nil {
		return false
	}
	if style.BackgroundColor != nil {
		return true
	}
	if style.BackgroundGradient != nil {
		return true
	}
	if style.BackgroundImageURL != "" {
		return true
	}
	if len(style.BoxShadow) > 0 || style.OutlineWidth > 0 {
		return true
	}
	if visibleBorderSide(style.BorderTopWidth, style.BorderTopStyle) ||
		visibleBorderSide(style.BorderRightWidth, style.BorderRightStyle) ||
		visibleBorderSide(style.BorderBottomWidth, style.BorderBottomStyle) ||
		visibleBorderSide(style.BorderLeftWidth, style.BorderLeftStyle) {
		return true
	}
	if style.PaddingTop > 0 || style.PaddingRight > 0 ||
		style.PaddingBottom > 0 || style.PaddingLeft > 0 {
		return true
	}
	if style.Height > 0 || style.MinHeight > 0 || style.MaxHeight > 0 {
		return true
	}
	// break-inside: avoid needs a container to act on — it groups the children
	// into one splittable unit that SplitAt can push to the next page whole.
	if style.BreakInside == breakInsideAvoid {
		return true
	}
	return false
}

func blockContainerPadding(style *css.ComputedStyle) (float64, float64, float64, float64) {
	if style == nil {
		return 0, 0, 0, 0
	}
	return style.PaddingTop,
		style.PaddingRight + visibleBorderWidth(style.BorderRightWidth, style.BorderRightStyle),
		style.PaddingBottom,
		style.PaddingLeft + visibleBorderWidth(style.BorderLeftWidth, style.BorderLeftStyle)
}

// newBlockContainer wraps rows in the box model derived from style: padding
// (including visible border widths), height clamps and break-inside.
func (tr *translator) newBlockContainer(style *css.ComputedStyle, rows []core.Row) *blockContainer {
	paddingTop, paddingRight, paddingBottom, paddingLeft := blockContainerPadding(style)
	return &blockContainer{
		rows:          rows,
		style:         tr.blockCellStyle(style),
		paddingTop:    paddingTop,
		paddingRight:  paddingRight,
		paddingBottom: paddingBottom,
		paddingLeft:   paddingLeft,
		height:        style.Height,
		minHeight:     style.MinHeight,
		maxHeight:     style.MaxHeight,
		breakInside:   style.BreakInside == breakInsideAvoid,
	}
}

// buildContainerRow wraps the given child rows into a single splittableContainerRow.
func (tr *translator) buildContainerRow(style *css.ComputedStyle, childRows []core.Row) core.Row {
	return newSplittableContainerRow(tr.newBlockContainer(style, childRows))
}

// splittableContainerRow wraps a blockContainer in a real row.Row (so it
// renders identically to the pre-Plan-B output) and also exposes SplitAt so
// paper.addRow() can split it across pages.
type splittableContainerRow struct {
	inner     core.Row        // row.New().Add(col.New().Add(container)) — handles Render/GetHeight
	container *blockContainer // direct reference so SplitAt can introspect children
}

func newSplittableContainerRow(c *blockContainer) *splittableContainerRow {
	inner := row.New().Add(col.New().Add(c))
	return &splittableContainerRow{inner: inner, container: c}
}

func (s *splittableContainerRow) SetConfig(cfg *entity.Config) { s.inner.SetConfig(cfg) }
func (s *splittableContainerRow) GetStructure() *node.Node[core.Structure] {
	return s.inner.GetStructure()
}

func (s *splittableContainerRow) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	return s.inner.GetHeight(provider, cell)
}

func (s *splittableContainerRow) Render(provider core.Provider, cell entity.Cell) {
	s.inner.Render(provider, cell)
}
func (s *splittableContainerRow) Add(cols ...core.Col) core.Row    { return s.inner.Add(cols...) }
func (s *splittableContainerRow) WithStyle(p *props.Cell) core.Row { return s.inner.WithStyle(p) }
func (s *splittableContainerRow) GetColumns() []core.Col           { return s.inner.GetColumns() }

// SplitAt implements core.Splittable. It splits the container's child rows at
// the point where cumulative row heights would exceed remainingHeight.
// Returns (nil, self, true) when no child rows fit (push whole container to
// next page). Returns (self, nil, false) when the container fits entirely.
func (s *splittableContainerRow) SplitAt(provider core.Provider, remainingHeight float64, width float64) (core.Row, core.Row, bool) {
	if s.container == nil {
		return nil, nil, false
	}

	// Measure children at the REAL render width handed in by the page builder so
	// the split sizes wrapped content exactly as it will render. Fall back to the
	// cached build-phase width, then a huge width, only if no render width is
	// supplied — the fallbacks underestimate wrapped text and can overflow.
	if width <= 0 {
		width = s.container.cachedWidth
	}
	if width <= 0 {
		width = 10000
	}
	dummyCell := &entity.Cell{Width: width, Height: 10000}

	totalHeight := s.container.GetHeight(provider, dummyCell)
	if totalHeight <= remainingHeight {
		return nil, nil, false // fits — no split needed
	}

	// break-inside: avoid — never divide the box. Push it whole to the next
	// page (atomic mode). If it is taller than a full page the page builder
	// places it oversized once it reaches the top of a fresh page, so this
	// cannot loop.
	if s.container.breakInside {
		return nil, s, true
	}

	// Greedy split: accumulate rows until they no longer fit.
	padding := s.container.paddingTop + s.container.paddingBottom
	available := remainingHeight - padding
	if available < 0 {
		available = 0
	}

	var firstRows, restRows []core.Row
	cumHeight := 0.0
	splitDone := false
	for _, r := range s.container.rows {
		rh := r.GetHeight(provider, dummyCell)
		if !splitDone && cumHeight+rh <= available+0.001 {
			firstRows = append(firstRows, r)
			cumHeight += rh
		} else {
			restRows = append(restRows, r)
			splitDone = true
		}
	}

	if len(firstRows) == 0 {
		// Nothing fits on the current page — push the whole container to next page.
		return nil, s, true
	}

	firstContainer := &blockContainer{
		rows:          firstRows,
		style:         s.container.style,
		paddingTop:    s.container.paddingTop,
		paddingRight:  s.container.paddingRight,
		paddingBottom: 0, // flat bottom at split point
		paddingLeft:   s.container.paddingLeft,
		minHeight:     s.container.minHeight,
		maxHeight:     s.container.maxHeight,
		config:        s.container.config,
	}

	var restRow core.Row
	if len(restRows) > 0 {
		restContainer := &blockContainer{
			rows:          restRows,
			style:         s.container.style,
			paddingTop:    0, // flat top at split point
			paddingRight:  s.container.paddingRight,
			paddingBottom: s.container.paddingBottom,
			paddingLeft:   s.container.paddingLeft,
			minHeight:     s.container.minHeight,
			maxHeight:     s.container.maxHeight,
			config:        s.container.config,
		}
		restRow = newSplittableContainerRow(restContainer)
	}

	return newSplittableContainerRow(firstContainer), restRow, true
}
