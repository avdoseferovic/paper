package translate

import (
	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// pageBreakRow is a zero-content row that implements core.PageBreaker.
// maroto.addRow() detects it via type assertion and calls fillPageToAddNew()
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
