package main

import (
	"log"
	"os"

	"github.com/avdoseferovic/paper/pkg/consts/extension"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/orientation"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/core"
)

func main() {
	backgroundImage := "docs/assets/images/certificate.png"
	m := GetPaper(backgroundImage)
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/disablepagebreak.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/disablepagebreak.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper(image string) core.Paper {
	bytes, err := os.ReadFile(image)
	if err != nil {
		log.Fatal(err)
	}
	b := config.NewBuilder().
		WithTopMargin(0).
		WithRightMargin(0).
		WithLeftMargin(0).
		WithDimensions(361.8, 203.2).
		WithDisableAutoPageBreak(true).
		WithOrientation(orientation.Horizontal).
		WithMaxGridSize(20).
		WithBackgroundImage(bytes, extension.Png).
		Build()

	b.Margins.Bottom = 0

	mrt := paper.New(b)
	m := decorator.NewMetrics(mrt)

	m.AddPages(page.New())
	return m
}
