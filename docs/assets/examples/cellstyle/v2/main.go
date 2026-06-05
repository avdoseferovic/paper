package main

import (
	"log"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper/pkg/consts/linestyle"

	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/border"
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

	err = document.Save("docs/assets/pdf/cellstylev2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/cellstylev2.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithDebug(false).
		Build()

	colStyle := &props.Cell{
		BackgroundColor: &props.Color{Red: 80, Green: 80, Blue: 80},
		BorderType:      border.Full,
		BorderColor:     &props.Color{Red: 200},
		LineStyle:       linestyle.Dashed,
		BorderThickness: 0.5,
	}

	rowStyles := []*props.Cell{
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.None,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Full,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Left,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Right,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Top,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Bottom,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Left | border.Top,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Left | border.Right,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Top | border.Bottom,
			BorderColor:     &props.Color{Blue: 200},
		},
		{
			BackgroundColor: &props.Color{Red: 220, Green: 220, Blue: 220},
			BorderType:      border.Left | border.Right | border.Top,
			BorderColor:     &props.Color{Blue: 200},
		},
	}

	whiteText := props.Text{
		Color: &props.Color{Red: 255, Green: 255, Blue: 255},
		Style: fontstyle.Bold,
		Size:  12,
		Align: align.Center,
		Top:   2,
	}

	blackText := props.Text{
		Style: fontstyle.Bold,
		Size:  12,
		Align: align.Center,
		Top:   2,
	}

	mrt := paper.New(cfg)
	m := paper.NewMetricsDecorator(mrt)

	count := 0
	for i := 0; i < 15; i++ {
		m.AddRows(
			row.New(10).Add(
				text.NewCol(4, "string", whiteText).WithStyle(colStyle),
				text.NewCol(4, "string", whiteText).WithStyle(colStyle),
				text.NewCol(4, "string", whiteText).WithStyle(colStyle),
			),
		)

		m.AddRows(row.New(10))

		m.AddRows(
			row.New(10).WithStyle(rowStyles[count]).Add(
				text.NewCol(4, "string", blackText),
				text.NewCol(4, "string", blackText),
				text.NewCol(4, "string", blackText),
			),
		)

		m.AddRows(row.New(10))
		count++
		if count >= len(rowStyles) {
			count = 0
		}
	}
	return m
}
