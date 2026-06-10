package main

import (
	"log"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/code"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/datamatrixgrid.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/datamatrixgrid.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddRow(40,
		code.NewMatrixCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 50,
		}),
		code.NewMatrixCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 75,
		}),
		code.NewMatrixCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewMatrixCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		code.NewMatrixCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 75,
		}),
		code.NewMatrixCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewMatrixCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 50,
		}),
		code.NewMatrixCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 75,
		}),
		code.NewMatrixCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewMatrixCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		code.NewMatrixCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 75,
		}),
		code.NewMatrixCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:  true,
			Percent: 100,
		}),
	)

	m.AddAutoRow(
		code.NewMatrixCol(6, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:             true,
			Percent:            20,
			JustReferenceWidth: true,
		}),
		code.NewMatrixCol(4, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:             true,
			Percent:            75,
			JustReferenceWidth: true,
		}),
		code.NewMatrixCol(2, "https://github.com/avdoseferovic/paper", props.Rect{
			Center:             true,
			Percent:            100,
			JustReferenceWidth: true,
		}),
	)
	return m
}
