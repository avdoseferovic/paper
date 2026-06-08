package main

import (
	"log"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"

	"github.com/avdoseferovic/paper/pkg/components/text"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/lowmemory.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/lowmemory.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithSequentialLowMemoryMode(7).
		WithDebug(true).
		WithPageNumber().
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	for range 50 {
		m.AddRows(
			text.NewRow(10, "Dummy text", props.Text{
				Size: 8,
			}),
		)
	}

	return m
}
