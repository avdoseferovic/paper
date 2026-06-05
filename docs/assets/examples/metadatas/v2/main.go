package main

import (
	"log"
	"time"

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

	err = document.Save("docs/assets/pdf/metadatasv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/metadatasv2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithAuthor("author", false).
		WithCreator("creator", false).
		WithSubject("subject", false).
		WithTitle("title", false).
		WithKeywords("keyword", false).
		WithCreationDate(time.Now()).
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	m.AddRows(
		text.NewRow(30, "metadatas"),
	)

	return m
}
