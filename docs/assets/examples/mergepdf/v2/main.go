package main

import (
	"log"
	"os"

	"github.com/avdoseferovic/paper/v2/pkg/core"

	"github.com/avdoseferovic/paper/v2"

	"github.com/avdoseferovic/paper/v2/pkg/components/text"
	"github.com/avdoseferovic/paper/v2/pkg/config"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	savedPdf, err := os.ReadFile("docs/assets/pdf/v2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Merge(savedPdf)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/mergepdfv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/mergepdfv2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithPageNumber().
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	for i := 0; i < 50; i++ {
		m.AddRows(text.NewRow(20, "content"))
	}

	return m
}
