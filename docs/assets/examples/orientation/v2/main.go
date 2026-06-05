package main

import (
	"log"

	"github.com/johnfercher/paper/v2/pkg/core"

	"github.com/johnfercher/paper/v2"

	"github.com/johnfercher/paper/v2/pkg/consts/orientation"

	"github.com/johnfercher/paper/v2/pkg/components/text"
	"github.com/johnfercher/paper/v2/pkg/config"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/orientationv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/orientationv2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithOrientation(orientation.Horizontal).
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	m.AddRows(
		text.NewRow(30, "content"),
	)

	return m
}
