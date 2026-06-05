package main

import (
	"log"

	"github.com/avdoseferovic/paper/v2/pkg/core"

	"github.com/avdoseferovic/paper/v2"

	"github.com/avdoseferovic/paper/v2/pkg/components/text"

	"github.com/avdoseferovic/paper/v2/pkg/config"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/lowmemoryv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/lowmemoryv2.txt")
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

	for i := 0; i < 50; i++ {
		m.AddRows(
			text.NewRow(10, "Dummy text", props.Text{
				Size: 8,
			}),
		)
	}

	return m
}
