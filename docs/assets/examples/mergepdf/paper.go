// Package mergepdf demonstrates merging a generated document with another PDF.
package mergepdf

import (
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
)

// GetPaper builds the mergepdf example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithPageNumber().
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	for range 50 {
		m.AddRows(text.NewRow(20, "content"))
	}

	return m
}
