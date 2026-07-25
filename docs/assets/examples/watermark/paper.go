// Package watermark demonstrates the per-page text watermark: translucent
// diagonal text drawn under the content of every page.
package watermark

import (
	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GetPaper builds the watermark example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithWatermark("DRAFT").
		Build()

	m := paper.New(cfg)
	for range 2 {
		m.AddRow(250, col.New(12).Add(text.New("Body content flows over the watermark.", props.Text{Top: 5})))
	}

	return m
}
