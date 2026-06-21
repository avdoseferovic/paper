package translate

import (
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// pageBreakRow is a zero-content row that implements core.PageBreaker.
// paper.addRow() detects it via type assertion and calls fillPageToAddNew()
// without placing the row on any page. Render() is intentionally a no-op.
type pageBreakRow struct{}

// NewPageBreakRow creates an exported pageBreakRow for use in tests and the
// translate pipeline.
func NewPageBreakRow() core.Row {
	return &pageBreakRow{}
}

// IsPageBreak satisfies core.PageBreaker — always returns true.
func (p *pageBreakRow) IsPageBreak() bool { return true }

// GetHeight returns 0 — the row occupies no space.
func (p *pageBreakRow) GetHeight(_ core.Provider, _ *entity.Cell) float64 { return 0 }

// Render is a no-op — the row is consumed entirely by addRow's page-break logic.
func (p *pageBreakRow) Render(_ core.Provider, _ entity.Cell) {}

// SetConfig is a no-op — no components to configure.
func (p *pageBreakRow) SetConfig(_ *entity.Config) {}

// GetStructure returns a minimal structure node for debugging.
func (p *pageBreakRow) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{Type: "page_break"})
}

// Add is a no-op; pageBreakRow has no columns.
func (p *pageBreakRow) Add(_ ...core.Col) core.Row { return p }

// WithStyle is a no-op.
func (p *pageBreakRow) WithStyle(_ *props.Cell) core.Row { return p }

// GetColumns returns nil — no columns.
func (p *pageBreakRow) GetColumns() []core.Col { return nil }

type pageControlRow struct {
	control core.PageControl
}

// NewPageControlRow creates a zero-content row that changes how the current
// physical page is committed, for example suppressing footer/page-number
// reservation on a title or interstitial page.
func NewPageControlRow(control core.PageControl) core.Row {
	return &pageControlRow{control: control}
}

// NewBlankPageRow creates a zero-content row that inserts one physical blank
// page. The supplied control determines whether that blank page receives
// repeated footer/page-number decorations.
func NewBlankPageRow(control core.PageControl) core.Row {
	control.Blank = true
	return &pageControlRow{control: control}
}

func (p *pageControlRow) PageControl() core.PageControl { return p.control }

func (p *pageControlRow) GetHeight(_ core.Provider, _ *entity.Cell) float64 { return 0 }

func (p *pageControlRow) Render(_ core.Provider, _ entity.Cell) {}

func (p *pageControlRow) SetConfig(_ *entity.Config) {}

func (p *pageControlRow) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{
		Type: "page_control",
		Details: map[string]any{
			"blank":                      p.control.Blank,
			"suppress_footer":            p.control.SuppressFooter,
			"suppress_page_number":       p.control.SuppressPageNumber,
			"count_page_number":          p.control.CountPageNumber,
			"decor_first_page_only":      p.control.DecorFirstPageOnly,
			"top_margin":                 p.control.TopMargin,
			"top_margin_first_page_only": p.control.TopMarginFirstPageOnly,
			"continuation_top_margin":    p.control.ContinuationTopMargin,
		},
	})
}

func (p *pageControlRow) Add(_ ...core.Col) core.Row { return p }

func (p *pageControlRow) WithStyle(_ *props.Cell) core.Row { return p }

func (p *pageControlRow) GetColumns() []core.Col { return nil }
