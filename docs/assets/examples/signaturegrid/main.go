package main

import (
	"log"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/signature"
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

	err = document.Save("docs/assets/pdf/signaturegrid.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/signaturegrid.txt")
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
		signature.NewCol(2, "Signature 1"),
		signature.NewCol(4, "Signature 2", props.Signature{FontFamily: consts.FontFamilyCourier}),
		signature.NewCol(6, "Signature 3", props.Signature{FontStyle: fontstyle.BoldItalic}),
	)

	m.AddRow(40,
		signature.NewCol(6, "Signature 4", props.Signature{FontStyle: fontstyle.Italic}),
		signature.NewCol(4, "Signature 5", props.Signature{FontSize: 12}),
		signature.NewCol(2, "Signature 6", props.Signature{FontColor: &props.RedColor}),
	)

	m.AddRow(40,
		signature.NewCol(4, "Signature 7", props.Signature{LineColor: &props.RedColor}),
		signature.NewCol(4, "Signature 8", props.Signature{LineStyle: consts.LineStyleDashed}),
		signature.NewCol(4, "Signature 9", props.Signature{LineThickness: 0.5}),
	)

	m.AddAutoRow(
		signature.NewCol(4, "Signature 7", props.Signature{LineColor: &props.RedColor}),
		signature.NewCol(4, "Signature 8", props.Signature{LineStyle: consts.LineStyleDashed}),
		signature.NewCol(4, "Signature 9", props.Signature{LineThickness: 0.5}),
	)

	return m
}
