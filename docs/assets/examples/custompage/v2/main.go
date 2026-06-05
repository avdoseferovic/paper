package main

import (
	"log"

	"github.com/johnfercher/paper/v2/pkg/core"

	"github.com/johnfercher/paper/v2"

	"github.com/johnfercher/paper/v2/pkg/consts/pagesize"

	"github.com/johnfercher/paper/v2/pkg/components/col"
	"github.com/johnfercher/paper/v2/pkg/components/image"
	"github.com/johnfercher/paper/v2/pkg/components/text"

	"github.com/johnfercher/paper/v2/pkg/config"
	"github.com/johnfercher/paper/v2/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/custompagev2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/custompagev2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A2).
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	m.AddRow(40,
		image.NewFromFileCol(4, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		text.NewCol(4, "Gopher International Shipping, Inc.", props.Text{
			Top:  12,
			Size: 12,
		}),
		col.New(4),
	)

	return m
}
