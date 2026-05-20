package translate

import (
	"sync"

	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// wrapRowAnchorTarget wraps r so the rendered Y position registers as a PDF
// named destination for the given anchor name. We achieve this by replacing
// the row's first column with one that contains an anchorTarget wrapping the
// original column's contents — but simpler and safe: prepend a zero-height
// row whose anchorTarget records the location. We use a single-column row
// containing only the anchorTarget that, at render time, reads cell.Y.
func wrapRowAnchorTarget(r core.Row, name string, reg *anchorRegistry) core.Row {
	// Wrap the entire row into a single-col row containing an anchorTarget
	// whose child is a tiny no-op component. The original row precedes the
	// target marker in the same vertical block so we synthesize a
	// composite row: append a 0-height row containing the target marker
	// after the original row. Simpler: return a "stack" wrapper.
	target := newAnchorTarget(nil, name, reg)
	markerRow := row.New(0).Add(col.New().Add(target))
	return &combinedRow{rows: []core.Row{markerRow, r}}
}

// wrapRowAnchorSource wraps a row so its rendered bounding box becomes a
// clickable link area pointing to the first anchor in names.
func wrapRowAnchorSource(r core.Row, names []string, reg *anchorRegistry) core.Row {
	src := newAnchorSource(rowComponent{r: r}, names, reg)
	// Replace the row's col with one containing the wrapper.
	return row.New().Add(col.New().Add(src))
}

// rowComponent adapts a core.Row to core.Component for embedding inside
// anchorSource. SetConfig/GetStructure/GetHeight/Render delegate to the row.
type rowComponent struct{ r core.Row }

func (r rowComponent) SetConfig(c *entity.Config) { r.r.SetConfig(c) }
func (r rowComponent) GetStructure() *node.Node[core.Structure] {
	return r.r.GetStructure()
}
func (r rowComponent) GetHeight(p core.Provider, cell *entity.Cell) float64 {
	return r.r.GetHeight(p, cell)
}
func (r rowComponent) Render(p core.Provider, cell *entity.Cell) {
	r.r.Render(p, *cell)
}

// combinedRow stacks multiple rows into a single Row interface.
type combinedRow struct {
	rows   []core.Row
	config *entity.Config
}

func (c *combinedRow) SetConfig(cfg *entity.Config) {
	c.config = cfg
	for _, r := range c.rows {
		r.SetConfig(cfg)
	}
}

func (c *combinedRow) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{Type: "combined", Details: map[string]any{"rows": len(c.rows)}}
	n := node.New(str)
	for _, r := range c.rows {
		n.AddNext(r.GetStructure())
	}
	return n
}

func (c *combinedRow) GetHeight(p core.Provider, cell *entity.Cell) float64 {
	total := 0.0
	for _, r := range c.rows {
		total += r.GetHeight(p, cell)
	}
	return total
}

func (c *combinedRow) Render(p core.Provider, cell entity.Cell) {
	inner := cell
	for _, r := range c.rows {
		h := r.GetHeight(p, &inner)
		inner.Height = h
		r.Render(p, inner)
		inner.Y += h
	}
}

// Add and other Row methods aren't needed for our internal use; combinedRow
// is treated as a black-box Row in the rendering pipeline.
func (c *combinedRow) Add(cols ...core.Col) core.Row  { return c }
func (c *combinedRow) WithStyle(*props.Cell) core.Row { return c }
func (c *combinedRow) GetColumns() []core.Col         { return nil }

// anchorRegistry stores the id→linkID mapping shared across all anchor target
// and source components produced by a single Translate call. Lookups are
// concurrent-safe so the renderer's row-by-row Render is safe to parallelise.
type anchorRegistry struct {
	mu     sync.Mutex
	idToLP map[string]int // anchor name → linkID returned by provider.AddLink
}

func newAnchorRegistry() *anchorRegistry {
	return &anchorRegistry{idToLP: map[string]int{}}
}

// ensureLinkID returns the linkID for name, registering one via lp on first
// access. Thread-safe.
func (r *anchorRegistry) ensureLinkID(name string, lp core.LinkProvider) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.idToLP[name]; ok {
		return id, true
	}
	if lp == nil {
		return 0, false
	}
	id := lp.AddLink()
	r.idToLP[name] = id
	return id, true
}

// collectAnchorIDs walks the DOM and returns the set of all `id` attribute
// values found on any element. This is the pre-pass that lets forward
// references (link before target) resolve correctly at render time.
func collectAnchorIDs(root *dom.Node) map[string]struct{} {
	out := map[string]struct{}{}
	var walk func(n *dom.Node)
	walk = func(n *dom.Node) {
		if n == nil {
			return
		}
		if id := n.Attr("id"); id != "" {
			out[id] = struct{}{}
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(root)
	return out
}

// anchorTarget wraps a child component, registering its rendered Y position
// as a PDF named destination when the provider implements core.LinkProvider.
type anchorTarget struct {
	child  core.Component
	name   string
	reg    *anchorRegistry
	config *entity.Config
}

func newAnchorTarget(child core.Component, name string, reg *anchorRegistry) *anchorTarget {
	return &anchorTarget{child: child, name: name, reg: reg}
}

func (a *anchorTarget) SetConfig(cfg *entity.Config) {
	a.config = cfg
	if a.child != nil {
		a.child.SetConfig(cfg)
	}
}

func (a *anchorTarget) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:    "anchor_target",
		Details: map[string]any{"name": a.name},
	}
	n := node.New(str)
	if a.child != nil {
		n.AddNext(a.child.GetStructure())
	}
	return n
}

func (a *anchorTarget) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if a.child == nil {
		return 0
	}
	return a.child.GetHeight(provider, cell)
}

func (a *anchorTarget) Render(provider core.Provider, cell *entity.Cell) {
	if lp, ok := provider.(core.LinkProvider); ok {
		if id, ok := a.reg.ensureLinkID(a.name, lp); ok {
			// Page resolution: SetLink(linkID, y, page) with page=-1 means
			// "current page" in gofpdf. cell.Y is margin-relative so we don't
			// add the top margin here — gofpdf interprets Y the same way.
			lp.SetLink(id, cell.Y, -1)
		}
	}
	if a.child != nil {
		a.child.Render(provider, cell)
	}
}

// anchorSource wraps a child component and registers a clickable rectangle
// covering the child's bounding box, jumping to the named anchor. When the
// child contains multiple runs with different anchors, the first anchor wins
// at the row level (limitation: per-run hit testing requires deeper renderer
// integration — full per-run anchor rectangles are deferred).
type anchorSource struct {
	child   core.Component
	anchors []string // anchor names referenced by descendant runs (in DOM order)
	reg     *anchorRegistry
	config  *entity.Config
}

func newAnchorSource(child core.Component, anchors []string, reg *anchorRegistry) *anchorSource {
	return &anchorSource{child: child, anchors: anchors, reg: reg}
}

func (a *anchorSource) SetConfig(cfg *entity.Config) {
	a.config = cfg
	if a.child != nil {
		a.child.SetConfig(cfg)
	}
}

func (a *anchorSource) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:    "anchor_source",
		Details: map[string]any{"anchors": a.anchors},
	}
	n := node.New(str)
	if a.child != nil {
		n.AddNext(a.child.GetStructure())
	}
	return n
}

func (a *anchorSource) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if a.child == nil {
		return 0
	}
	return a.child.GetHeight(provider, cell)
}

func (a *anchorSource) Render(provider core.Provider, cell *entity.Cell) {
	if a.child != nil {
		a.child.Render(provider, cell)
	}
	if lp, ok := provider.(core.LinkProvider); ok && len(a.anchors) > 0 {
		// First anchor wins at the row level.
		if id, ok := a.reg.ensureLinkID(a.anchors[0], lp); ok {
			lp.Link(cell.X, cell.Y, cell.Width, cell.Height, id)
		}
	}
}
