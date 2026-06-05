package main

import (
	"log"

	"github.com/johnfercher/paper/v2/pkg/core"

	"github.com/johnfercher/paper/v2"

	"github.com/johnfercher/paper/v2/pkg/components/code"

	"github.com/johnfercher/paper/v2/pkg/config"
	"github.com/johnfercher/paper/v2/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/qrgridv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/qrgridv2.txt")
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
		code.NewQrCol(2, "https://github.com/johnfercher/paper", props.Rect{
			Percent: 50,
		}),
		code.NewQrCol(4, "https://github.com/johnfercher/paper", props.Rect{
			Percent: 75,
		}),
		code.NewQrCol(6, "https://github.com/johnfercher/paper", props.Rect{
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewQrCol(2, "https://github.com/johnfercher/paper", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		code.NewQrCol(4, "https://github.com/johnfercher/paper", props.Rect{
			Center:  true,
			Percent: 75,
		}),
		code.NewQrCol(6, "https://github.com/johnfercher/paper", props.Rect{
			Center:  true,
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewQrCol(6, "https://github.com/johnfercher/paper", props.Rect{
			Percent: 50,
		}),
		code.NewQrCol(4, "https://github.com/johnfercher/paper", props.Rect{
			Percent: 75,
		}),
		code.NewQrCol(2, "https://github.com/johnfercher/paper", props.Rect{
			Percent: 100,
		}),
	)

	m.AddRow(40,
		code.NewQrCol(6, "https://github.com/johnfercher/paper", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		code.NewQrCol(4, "https://github.com/johnfercher/paper", props.Rect{
			Center:  true,
			Percent: 75,
		}),
		code.NewQrCol(2, "https://github.com/johnfercher/paper", props.Rect{
			Center:  true,
			Percent: 100,
		}),
	)

	m.AddAutoRow(
		code.NewQrCol(6, "https://github.com/johnfercher/paper", props.Rect{
			Center:             true,
			Percent:            30,
			JustReferenceWidth: true,
		}),
		code.NewQrCol(4, "https://github.com/johnfercher/paper", props.Rect{
			Center:             true,
			Percent:            75,
			JustReferenceWidth: true,
		}),
		code.NewQrCol(2, "https://github.com/johnfercher/paper", props.Rect{
			Center:             true,
			Percent:            100,
			JustReferenceWidth: true,
		}),
	)
	return m
}
