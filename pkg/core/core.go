// Package core contains all core interfaces and basic implementations.
package core

import (
	"context"
	"io"

	"github.com/avdoseferovic/paper/pkg/tree/node"

	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/metrics"
	"github.com/avdoseferovic/paper/pkg/props"
)

// Paper is the interface that wraps the basic methods of paper.
type Paper interface {
	RegisterHeader(rows ...Row) error
	RegisterFooter(rows ...Row) error
	AddRows(rows ...Row)
	AddRow(rowHeight float64, cols ...Col) Row
	AddAutoRow(cols ...Col) Row
	AddHTML(ctx context.Context, htmlStr string) error
	// FitInCurrentPage reports whether a row of the given height fits in the remaining useful area of the current page.
	FitInCurrentPage(heightNewLine float64) bool
	GetCurrentConfig() *entity.Config
	AddPages(pages ...Page)
	GetStructure() *node.Node[Structure]
	Generate(ctx context.Context) (*Pdf, error)
}

// Document is the interface that wraps the basic methods of a document.
type Document interface {
	GetBytes() []byte
	GetBase64() string
	// Write streams the document to w without an intermediate copy beyond the already-generated buffer.
	Write(w io.Writer) (int64, error)
	Save(file string) error
	GetReport() *metrics.Report
	Merge(ctx context.Context, bytes []byte) error
}

// Node is the interface that wraps the basic methods of a node.
type Node interface {
	SetConfig(config *entity.Config)
	GetStructure() *node.Node[Structure]
}

// Component is the interface that wraps the basic methods of a component.
type Component interface {
	Node
	Render(provider Provider, cell *entity.Cell)
	GetHeight(provider Provider, cell *entity.Cell) float64
}

// Col is the interface that wraps the basic methods of a col.
type Col interface {
	Node
	Add(components ...Component) Col
	GetSize() int
	GetHeight(provider Provider, cell *entity.Cell) float64
	WithStyle(style *props.Cell) Col
	Render(provider Provider, cell entity.Cell, createCell bool)
}

// Row is the interface that wraps the basic methods of a row.
type Row interface {
	Node
	Add(cols ...Col) Row
	GetHeight(provider Provider, cell *entity.Cell) float64
	GetColumns() []Col
	WithStyle(style *props.Cell) Row
	Render(provider Provider, cell entity.Cell)
}

// Page is the interface that wraps the basic methods of a page.
type Page interface {
	Node
	Add(rows ...Row) Page
	GetRows() []Row
	GetNumber() int
	SetNumber(number int, total int)
	Render(provider Provider, cell entity.Cell)
}
