package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/code"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/barcodegrid.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/barcodegrid.txt")
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
			Type:   consts.BarcodeEAN,
		}),
		code.NewBarCol(4, "123456789123", props.Barcode{
			Center: true,
			Type:   consts.BarcodeEAN,
		}),
		code.NewBarCol(6, "123456789123", props.Barcode{
			Center: true,
			Type:   consts.BarcodeEAN,
		}),
	)

	m.AddAutoRow(
		code.NewBarCol(2, "123456789123", props.Barcode{
			Center: true,
			Type:   consts.BarcodeEAN,
		}),
		code.NewBarCol(4, "123456789123", props.Barcode{
			Center: true,
			Type:   consts.BarcodeEAN,
		}),
		code.NewBarCol(6, "123456789123", props.Barcode{
			Center: true,
			Type:   consts.BarcodeEAN,
		}),
	)

	return m
}
