// Package margins demonstrates setting the page margins.
package margins

import (
	"log"

	"github.com/avdoseferovic/paper/pkg/components/row"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/decorator"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GetPaper builds the margins example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithTopMargin(20).
		WithLeftMargin(20).
		WithRightMargin(20).
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	err := m.RegisterHeader(
		row.New(40).Add(
			image.NewFromFileCol(4, "docs/assets/images/gopherbw.png", props.Rect{
				Center:  true,
				Percent: 50,
			}),
			text.NewCol(4, "Margins Test", props.Text{
				Top:  12,
				Size: 12,
			}),
			col.New(4),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	for range 10 {
		m.AddRows(text.NewRow(30, "any text"))
	}

	return m
}
