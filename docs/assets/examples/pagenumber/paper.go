// Package pagenumber demonstrates page numbering in a footer.
package pagenumber

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GetPaper builds the pagenumber example document.
func GetPaper() core.Paper {
	pageNumber := props.PageNumber{
		Pattern: "Page {current} of {total}",
		Place:   props.Bottom,
		Family:  consts.FontFamilyCourier,
		Style:   fontstyle.Bold,
		Size:    9,
		Color: &props.Color{
			Red: 255,
		},
	}

	cfg := config.NewBuilder().
		WithDebug(true).
		WithPageNumber(pageNumber).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	for range 15 {
		m.AddRows(text.NewRow(20, "dummy text"))
	}

	return m
}
