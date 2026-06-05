package main

import (
	"log"

	"github.com/johnfercher/paper/v2/pkg/core"

	"github.com/johnfercher/paper/v2"

	"github.com/johnfercher/paper/v2/pkg/components/text"
	"github.com/johnfercher/paper/v2/pkg/consts/protection"

	"github.com/johnfercher/paper/v2/pkg/config"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/protectionv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/protectionv2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithProtection(protection.None, "user", "owner").
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	m.AddRows(
		text.NewRow(30, "supersecret content"),
	)

	return m
}
