package main

import (
	"log"

	"github.com/avdoseferovic/paper/v2/pkg/core"

	"github.com/avdoseferovic/paper/v2"

	"github.com/avdoseferovic/paper/v2/pkg/components/code"

	"github.com/avdoseferovic/paper/v2/pkg/config"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/datamatrixgridv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/datamatrixgridv2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

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
