package main

import (
	"log"

	"github.com/johnfercher/paper/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/paper/v2/pkg/consts/fontstyle"

	"github.com/johnfercher/paper/v2/pkg/core"

	"github.com/johnfercher/paper/v2"

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

	err = document.Save("docs/assets/pdf/pagenumberv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/pagenumberv2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	pageNumber := props.PageNumber{
		Pattern: "Page {current} of {total}",
		Place:   props.Bottom,
		Family:  fontfamily.Courier,
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
	m := paper.NewMetricsDecorator(mrt)

	for i := 0; i < 15; i++ {
		m.AddRows(text.NewRow(20, "dummy text"))
	}

	return m
}
