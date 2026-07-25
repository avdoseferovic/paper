package translate

import (
	"sync"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
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
func (c *combinedRow) Add(_ ...core.Col) core.Row     { return c }
func (c *combinedRow) WithStyle(*props.Cell) core.Row { return c }
func (c *combinedRow) GetColumns() []core.Col         { return nil }

// anchorRegistry stores the anchor name → linkID mapping shared across all
// anchor target and source components produced by a single Translate call.
// Lookups are concurrent-safe so the renderer's row-by-row Render is safe to
// parallelise.
//
// Link IDs are indices into the provider's own link table, so the mapping is
// kept per provider. Parallel page rendering gives each worker a separate
// provider, and handing worker B an ID that worker A allocated would either
// point at the wrong destination or at no destination at all. Providers that
// implement core.NamedLinkProvider reserve the target under the anchor name
// instead, which is what lets the per-worker reservations be reconciled when
// the rendered pages are spliced into one document.
type anchorRegistry struct {
	mu     sync.Mutex
	idToLP map[core.LinkProvider]map[string]int // provider → anchor name → linkID
}

func newAnchorRegistry() *anchorRegistry {
	return &anchorRegistry{idToLP: map[core.LinkProvider]map[string]int{}}
}

// EnsureLinkID satisfies richtext.anchorResolverIface — exported so the
// richtext package can call it without importing translate.
func (r *anchorRegistry) EnsureLinkID(name string, lp core.LinkProvider) (int, bool) {
	return r.ensureLinkID(name, lp)
}

// ensureLinkID returns lp's linkID for name, reserving one on first access.
// Thread-safe. Reports false when there is no provider to reserve against,
// since a link ID is only meaningful relative to the provider that issued it.
func (r *anchorRegistry) ensureLinkID(name string, lp core.LinkProvider) (int, bool) {
	if lp == nil {
		return 0, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	names, ok := r.idToLP[lp]
	if !ok {
		names = map[string]int{}
		r.idToLP[lp] = names
	}
	if id, ok := names[name]; ok {
		return id, true
	}
	id := reserveLink(name, lp)
	names[name] = id
	return id, true
}

// reserveLink reserves the anchor target, preferring a named reservation so it
// survives page splicing.
func reserveLink(name string, lp core.LinkProvider) int {
	if named, ok := lp.(core.NamedLinkProvider); ok {
		return named.AddNamedLink(name)
	}
	return lp.AddLink()
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
