package main

import (
	"log"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"

	"github.com/avdoseferovic/paper/pkg/components/text"

	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/footer.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/footer.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	err := m.RegisterFooter(text.NewRow(20, "Footer", props.Text{
		Size:  10,
		Style: fontstyle.Bold,
		Align: align.Center,
	}))
	if err != nil {
		log.Fatal(err)
	}

	for range 50 {
		m.AddRows(
			text.NewRow(10, "Dummy text", props.Text{
				Size: 8,
			}),
		)
	}

	return m
}
