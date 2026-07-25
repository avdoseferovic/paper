// Package lowmemory demonstrates low-memory mode, which writes pages out as they are built.
package lowmemory

import (
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/text"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GetPaper builds the lowmemory example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithSequentialLowMemoryMode(7).
		WithDebug(true).
		WithPageNumber().
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	for range 50 {
		m.AddRows(
			text.NewRow(10, "Dummy text", props.Text{
				Size: 8,
			}),
		)
	}

	return m
}
