package main

import (
	"log"
	"os"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	savedPdf, err := os.ReadFile("docs/assets/pdf/paper.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Merge(savedPdf)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/mergepdf.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/mergepdf.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithPageNumber().
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	for range 50 {
		m.AddRows(text.NewRow(20, "content"))
	}

	return m
}
