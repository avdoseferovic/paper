package main

import (
	"log"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/line"
	"github.com/avdoseferovic/paper/pkg/consts/linestyle"
	"github.com/avdoseferovic/paper/pkg/consts/orientation"
	"github.com/avdoseferovic/paper/pkg/props"

	"github.com/avdoseferovic/paper/pkg/config"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/linegrid.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/linegrid.txt")
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
		line.NewCol(2),
		line.NewCol(4),
		line.NewCol(6),
	)

	m.AddRow(40,
		line.NewCol(6),
		line.NewCol(4),
		line.NewCol(2),
	)

	m.AddRow(40,
		line.NewCol(2, props.Line{Thickness: 0.5}),
		line.NewCol(4, props.Line{Color: &props.RedColor}),
		line.NewCol(6, props.Line{Orientation: orientation.Vertical}),
	)

	m.AddRow(40,
		line.NewCol(6, props.Line{OffsetPercent: 50}),
		line.NewCol(4, props.Line{OffsetPercent: 50, Orientation: orientation.Vertical}),
		line.NewCol(2, props.Line{SizePercent: 50}),
	)

	m.AddRow(40,
		line.NewCol(2, props.Line{Style: linestyle.Dashed}),
		line.NewCol(4,
			props.Line{
				Color:         &props.RedColor,
				Style:         linestyle.Dashed,
				Thickness:     0.8,
				Orientation:   orientation.Vertical,
				OffsetPercent: 70,
				SizePercent:   70,
			},
		),
		line.NewCol(6,
			props.Line{
				Color:         &props.RedColor,
				Style:         linestyle.Dashed,
				Thickness:     0.8,
				Orientation:   orientation.Horizontal,
				OffsetPercent: 40,
				SizePercent:   40,
			},
		),
	)

	m.AddAutoRow(
		line.NewCol(2, props.Line{Style: linestyle.Dashed}),
		line.NewCol(4,
			props.Line{
				Color:     &props.RedColor,
				Style:     linestyle.Dashed,
				Thickness: 0.8, Orientation: orientation.Vertical,
				OffsetPercent: 70,
				SizePercent:   70,
			},
		),
		line.NewCol(6,
			props.Line{
				Color:         &props.RedColor,
				Style:         linestyle.Dashed,
				Thickness:     0.8,
				Orientation:   orientation.Horizontal,
				OffsetPercent: 40,
				SizePercent:   40,
			},
		),
	)
	return m
}
