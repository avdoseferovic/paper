package main

import (
	"log"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/decorator"
)

func main() {
	m := GetPaper()

	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/addpage.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/addpage.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithPageNumber().
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddPages(
		page.New().Add(
			text.NewRow(30, "page1 row1"),
			text.NewRow(30, "page1 row2"),
			text.NewRow(30, "page1 row3"),
			text.NewRow(30, "page1 row4"),
			text.NewRow(30, "page1 row5"),
			text.NewRow(30, "page1 row6"),
			text.NewRow(30, "page1 row7"),
			text.NewRow(30, "page1 row8"),
			text.NewRow(30, "page1 row9"),
		),
		page.New().Add(
			text.NewRow(10, "page2 row1"),
		),
		page.New().Add(
			text.NewRow(10, "page3 row1"),
		),
	)

	return m
}
