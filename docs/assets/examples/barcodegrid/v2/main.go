package main

import (
	"log"

	"github.com/avdoseferovic/paper/v2/pkg/consts/barcode"

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

	err = document.Save("docs/assets/pdf/barcodegridv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/barcodegridv2.txt")
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
		code.NewBarCol(2, "https://github.com/avdoseferovic/paper", props.Barcode{
			Percent: 50,
		}),
		code.NewBarCol(4, "https://github.com/avdoseferovic/paper", props.Barcode{
			Percent: 75,
		}),
		code.NewBarCol(6, "https://github.com/avdoseferovic/paper", props.Barcode{
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewBarCol(2, "https://github.com/avdoseferovic/paper", props.Barcode{
			Center:  true,
			Percent: 50,
		}),
		code.NewBarCol(4, "https://github.com/avdoseferovic/paper", props.Barcode{
			Center:  true,
			Percent: 75,
		}),
		code.NewBarCol(6, "https://github.com/avdoseferovic/paper", props.Barcode{
			Center:  true,
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewBarCol(6, "https://github.com/avdoseferovic/paper", props.Barcode{
			Percent: 50,
		}),
		code.NewBarCol(4, "https://github.com/avdoseferovic/paper", props.Barcode{
			Percent: 75,
		}),
		code.NewBarCol(2, "https://github.com/avdoseferovic/paper", props.Barcode{
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewBarCol(6, "https://github.com/avdoseferovic/paper", props.Barcode{
			Center:  true,
			Percent: 50,
		}),
		code.NewBarCol(4, "https://github.com/avdoseferovic/paper", props.Barcode{
			Center:  true,
			Percent: 75,
		}),
		code.NewBarCol(2, "https://github.com/avdoseferovic/paper", props.Barcode{
			Center:  true,
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewBarCol(2, "123456789123", props.Barcode{
			Center: true,
			Type:   barcode.EAN,
		}),
		code.NewBarCol(4, "123456789123", props.Barcode{
			Center: true,
			Type:   barcode.EAN,
		}),
		code.NewBarCol(6, "123456789123", props.Barcode{
			Center: true,
			Type:   barcode.EAN,
		}),
	)

	m.AddAutoRow(
		code.NewBarCol(2, "123456789123", props.Barcode{
			Center: true,
			Type:   barcode.EAN,
		}),
		code.NewBarCol(4, "123456789123", props.Barcode{
			Center: true,
			Type:   barcode.EAN,
		}),
		code.NewBarCol(6, "123456789123", props.Barcode{
			Center: true,
			Type:   barcode.EAN,
		}),
	)

	return m
}
