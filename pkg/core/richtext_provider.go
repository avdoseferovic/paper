package core

import (
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

// RichTextProvider is a narrow capability interface for components that need
// inline-run measurement and placement (RichText, Table, HTMLList).
// It is separate from Provider to avoid triggering mock regeneration across
// the 13+ test files that mock Provider with strict EXPECT() assertions.
// The gofpdf *provider satisfies both Provider and RichTextProvider.
type RichTextProvider interface {
	MeasureString(text string, prop *props.Text) float64
	AddTextAt(x, y float64, text string, prop *props.Text)
	AddRichText(runs []props.RichRun, cell *entity.Cell, prop *props.RichText)
}
