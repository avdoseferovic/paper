// Package html implements a component wrapper for Paper's HTML translator.
package html

import (
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	paperhtml "github.com/avdoseferovic/paper/pkg/html"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// Option configures HTML component translation.
type Option = paperhtml.Option

// HTML is a core.Component backed by rows produced from an HTML fragment.
type HTML struct {
	source string
	rows   []core.Row
}

// WithUnsupportedHandler registers a callback invoked for unsupported HTML
// tags or CSS properties.
func WithUnsupportedHandler(fn func(thing, value string)) Option {
	return paperhtml.WithUnsupportedHandler(fn)
}

// WithGridSize overrides the default grid size used for flex quantization.
func WithGridSize(n int) Option {
	return paperhtml.WithGridSize(n)
}

// WithContentWidth sets the content width in mm for width-aware CSS features.
func WithContentWidth(mm float64) Option {
	return paperhtml.WithContentWidth(mm)
}

// WithImageBaseDir scopes local image reads to a single directory.
func WithImageBaseDir(dir string) Option {
	return paperhtml.WithImageBaseDir(dir)
}

// WithStylesheetBaseDir scopes local stylesheet reads to a single directory.
func WithStylesheetBaseDir(dir string) Option {
	return paperhtml.WithStylesheetBaseDir(dir)
}

// New converts an HTML string into a component that can be placed inside a
// column like any other Paper component.
func New(htmlStr string, opts ...Option) (*HTML, error) {
	rows, err := paperhtml.FromString(htmlStr, opts...)
	if err != nil {
		return nil, err
	}
	return &HTML{source: htmlStr, rows: rows}, nil
}

// NewCol wraps an HTML component in a column of the given grid size.
func NewCol(size int, htmlStr string, opts ...Option) (core.Col, error) {
	component, err := New(htmlStr, opts...)
	if err != nil {
		return nil, err
	}
	return col.New(size).Add(component), nil
}

// NewRow wraps an HTML component in a fixed-height row.
func NewRow(height float64, htmlStr string, opts ...Option) (core.Row, error) {
	component, err := New(htmlStr, opts...)
	if err != nil {
		return nil, err
	}
	return row.New(height).Add(col.New().Add(component)), nil
}

// NewAutoRow wraps an HTML component in an auto-height row.
func NewAutoRow(htmlStr string, opts ...Option) (core.Row, error) {
	component, err := New(htmlStr, opts...)
	if err != nil {
		return nil, err
	}
	return row.New().Add(col.New().Add(component)), nil
}

// SetConfig propagates the Paper configuration to translated rows.
func (h *HTML) SetConfig(config *entity.Config) {
	for _, r := range h.rows {
		r.SetConfig(config)
	}
}

// GetStructure returns the component tree node for snapshots/debugging.
func (h *HTML) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:  "html",
		Value: h.source,
		Details: map[string]any{
			"rows": len(h.rows),
		},
	}
	n := node.New(str)
	for _, r := range h.rows {
		n.AddNext(r.GetStructure())
	}
	return n
}

// GetHeight returns the sum of the translated row heights inside the target
// cell width.
func (h *HTML) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	inner := cell.Copy()
	total := 0.0
	for _, r := range h.rows {
		total += r.GetHeight(provider, &inner)
	}
	return total
}

// Render draws each translated row sequentially inside the component cell.
func (h *HTML) Render(provider core.Provider, cell *entity.Cell) {
	inner := cell.Copy()
	positioner, _ := provider.(core.PositionProvider)
	for _, r := range h.rows {
		height := r.GetHeight(provider, &inner)
		inner.Height = height
		if positioner != nil {
			positioner.SetCursor(inner.X, inner.Y)
		}
		r.Render(provider, inner)
		inner.Y += height
	}
	if positioner != nil {
		positioner.SetCursor(cell.X+cell.Width, cell.Y)
	}
}
