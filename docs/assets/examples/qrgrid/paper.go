// Package qrgrid demonstrates placing QR codes across a grid of columns.
package qrgrid

import (
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/code"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GetPaper builds the qrgrid example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddRow(40,
		code.NewQrCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 50,
		}),
		code.NewQrCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 75,
		}),
		code.NewQrCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewQrCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		code.NewQrCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 75,
		}),
		code.NewQrCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewQrCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 50,
		}),
		code.NewQrCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 75,
		}),
		code.NewQrCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewQrCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		code.NewQrCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 75,
		}),
		code.NewQrCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 100,
		}),
	)

	m.AddAutoRow(
		code.NewQrCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:             true,
			Percent:            30,
			JustReferenceWidth: true,
		}),
		code.NewQrCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:             true,
			Percent:            75,
			JustReferenceWidth: true,
		}),
		code.NewQrCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:             true,
			Percent:            100,
			JustReferenceWidth: true,
		}),
	)
	return m
}
