package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
)

func main() {
	m := GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/orientation.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/orientation.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithOrientation(consts.OrientationHorizontal).
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddRows(
		text.NewRow(30, "content"),
	)

	return m
}
