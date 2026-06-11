package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/image"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/imagegrid.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/imagegrid.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddRow(40,
		image.NewFromFileCol(2, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  true,
			Percent: 80,
		}),
		image.NewFromFileCol(4, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  true,
			Percent: 80,
		}),
		image.NewFromFileCol(6, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  true,
			Percent: 80,
		}),
	)

	m.AddRow(40,
		image.NewFromFileCol(2, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  false,
			Percent: 50,
			Left:    10,
		}),
		image.NewFromFileCol(4, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  false,
			Percent: 50,
			Top:     10,
		}),
		image.NewFromFileCol(6, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  false,
			Percent: 50,
			Left:    15,
			Top:     15,
		}),
	)

	m.AddRow(40,
		image.NewFromFileCol(8, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  true,
			Percent: 80,
		}),
		image.NewFromFileCol(4, "docs/assets/images/biplane.jpg", props.Rect{
			Center:  true,
			Percent: 80,
		}),
	)

	m.AddRow(40,
		image.NewFromFileCol(6, "docs/assets/images/frontpage.png", props.Rect{
			Center:  false,
			Percent: 80,
			Top:     5,
			Left:    10,
		}),
		image.NewFromFileCol(4, "docs/assets/images/frontpage.png", props.Rect{
			Center:  false,
			Percent: 80,
			Top:     5,
		}),
		image.NewFromFileCol(2, "docs/assets/images/frontpage.png", props.Rect{
			Center:  false,
			Percent: 80,
			Left:    5,
		}),
	)

	m.AddRow(40,
		image.NewFromFileCol(6, "docs/assets/images/frontpage.png", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		image.NewFromFileCol(4, "docs/assets/images/frontpage.png", props.Rect{
			Center:  true,
			Percent: 50,
		}),
		image.NewFromFileCol(2, "docs/assets/images/frontpage.png", props.Rect{
			Center:  true,
			Percent: 50,
		}),
	)

	m.AddRow(40,
		image.NewFromFileCol(4, "docs/assets/images/frontpage.png", props.Rect{
			Center:  true,
			Percent: 80,
		}),
		image.NewFromFileCol(8, "docs/assets/images/frontpage.png", props.Rect{
			Center:  true,
			Percent: 80,
		}),
	)
	m.AddAutoRow(
		image.NewFromFileCol(4, "docs/assets/images/frontpage.png", props.Rect{
			Center:             true,
			Percent:            20,
			JustReferenceWidth: true,
		}),
		image.NewFromFileCol(8, "docs/assets/images/frontpage.png", props.Rect{
			Center:             true,
			Percent:            30,
			JustReferenceWidth: true,
		}),
	)

	m.AddAutoRow(
		image.NewFromFileCol(2, "docs/assets/images/biplane.jpg", props.Rect{
			Center:             false,
			Percent:            50,
			Left:               10,
			JustReferenceWidth: true,
		}),
		image.NewFromFileCol(4, "docs/assets/images/biplane.jpg", props.Rect{
			Center:             false,
			Percent:            50,
			Top:                10,
			JustReferenceWidth: true,
		}),
		image.NewFromFileCol(6, "docs/assets/images/biplane.jpg", props.Rect{
			Center:             false,
			Percent:            50,
			Left:               15,
			Top:                15,
			JustReferenceWidth: true,
		}),
	)
	return m
}
