package main

import (
	"log"

	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"

	"github.com/avdoseferovic/paper/pkg/consts/breakline"

	"github.com/avdoseferovic/paper/pkg/components/text"

	"github.com/avdoseferovic/paper/pkg/consts/align"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/textgridv2.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/textgridv2.txt")
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

	longText := "This is a longer sentence that will be broken into multiple lines " +
		"as it does not fit into the column otherwise."

	m.AddRow(40,
		text.NewCol(2, "Red text", props.Text{Color: &props.RedColor}),
		text.NewCol(6, "Green text", props.Text{Color: &props.GreenColor}),
		text.NewCol(4, "Blue text", props.Text{Color: &props.BlueColor}),
	)

	m.AddRow(40,
		text.NewCol(2, "Left-aligned text"),
		text.NewCol(4, "Centered text", props.Text{Align: align.Center}),
		text.NewCol(6, "Right-aligned text", props.Text{Align: align.Right, Style: fontstyle.Strikethrough}),
	)

	m.AddRows(text.NewRow(10, "Aligned unindented text"))

	m.AddRow(40,
		text.NewCol(2, "Left-aligned text", props.Text{Top: 3, Left: 3, Align: align.Left}),
		text.NewCol(4, "Centered text", props.Text{Top: 3, Align: align.Center}),
		text.NewCol(6, "Right-aligned text", props.Text{Top: 3, Right: 3, Align: align.Right}),
	)

	m.AddRows(text.NewRow(10, "Aligned text with indentation"))

	m.AddRow(40,
		text.NewCol(2, longText, props.Text{Align: align.Left}),
		text.NewCol(4, longText, props.Text{Align: align.Center}),
		text.NewCol(6, longText, props.Text{Align: align.Right}),
	)

	m.AddRows(text.NewRow(10, "Multiline text"))

	m.AddRow(40,
		text.NewCol(2, longText, props.Text{Top: 3, Left: 3, Right: 3, Align: align.Left, BreakLineStrategy: breakline.DashStrategy}),
		text.NewCol(4, longText, props.Text{Top: 3, Left: 3, Right: 3, Align: align.Center}),
		text.NewCol(6, longText, props.Text{Top: 3, Left: 3, Right: 3, Align: align.Right}),
	)

	m.AddRows(text.NewRow(10, "Multiline text with indentation"))

	google := "https://google.com"

	m.AddRows(text.NewRow(10, "text with hyperlink", props.Text{Hyperlink: &google}))

	m.AddRow(45,
		text.NewCol(2, longText, props.Text{Top: 3, Left: 3, Right: 3, Align: align.Justify, BreakLineStrategy: breakline.DashStrategy}),
		text.NewCol(4, longText+" "+longText, props.Text{Top: 10, Left: 3, Right: 3, Align: align.Justify}),
		text.NewCol(6, longText+" "+longText, props.Text{Hyperlink: &google, Top: 10, Left: 10, Right: 10, Align: align.Justify}),
	)
	m.AddRows(text.NewRow(10, "Justify-aligned text", props.Text{Align: align.Justify}))

	m.AddAutoRow(
		text.NewCol(2, longText, props.Text{Top: 0, Left: 3, Right: 3, Align: align.Justify, BreakLineStrategy: breakline.DashStrategy}),
		text.NewCol(4, longText+" "+longText, props.Text{Top: 0, Left: 3, Right: 3, Align: align.Justify}),
		text.NewCol(6, longText+" "+longText+" "+longText, props.Text{Hyperlink: &google, Top: 0, Left: 10, Right: 10, Align: align.Justify}),
	)

	m.AddAutoRow(
		text.NewCol(12, longText+" "+longText+" "+longText,
			props.Text{
				Left:              3,
				Right:             3,
				Align:             align.Justify,
				BreakLineStrategy: breakline.EmptySpaceStrategy,
			},
		),
	)

	m.AddAutoRow(
		text.NewCol(12, longText+" "+longText+" "+longText,
			props.Text{
				VerticalPadding:   10,
				Left:              3,
				Right:             3,
				Align:             align.Justify,
				BreakLineStrategy: breakline.EmptySpaceStrategy,
			},
		),
	)
	return m
}
