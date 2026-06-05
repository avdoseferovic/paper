package main

import (
	"fmt"
	"log"

	"github.com/avdoseferovic/paper"

	"github.com/avdoseferovic/paper/pkg/components/text"

	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/maxgridsumv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/maxgridsumv2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	gridSum := 14
	cfg := config.NewBuilder().
		WithDebug(true).
		WithMaxGridSize(gridSum).
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	m.AddRows(text.NewRow(10, fmt.Sprintf("Table with %d Columns", gridSum), props.Text{Style: fontstyle.Bold}))

	var headers []core.Col
	var contents []core.Col
	for i := 0; i < gridSum; i++ {
		headers = append(headers, text.NewCol(1, fmt.Sprintf("H %d", i), props.Text{Style: fontstyle.Bold, Top: 1.5, Left: 1.5}))
		contents = append(contents, text.NewCol(1, fmt.Sprintf("C %d", i), props.Text{Top: 1, Left: 1.5, Size: 9}))
	}

	m.AddRow(8, headers...)
	m.AddRow(8, contents...)

	return m
}
