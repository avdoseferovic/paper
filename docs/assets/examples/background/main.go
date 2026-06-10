package main

import (
	"log"
	"os"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/extension"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	backgroundImage := "docs/assets/images/certificate.png"
	m := GetPaper(backgroundImage)
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/background.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/background.txt")
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
		WithLeftMargin(0).
		WithRightMargin(0).
		WithOrientation(consts.OrientationHorizontal).
		WithMaxGridSize(20).
		WithBackgroundImage(bytes, extension.Png)

	mrt := paper.New(b.Build())
	m := decorator.NewMetrics(mrt)

	m.AddPages(AddPage(), AddPage(), AddPage(), AddPage(), AddPage())
	return m
}

func AddPage() core.Page {
	return page.New().Add(
		row.New(70),
		row.New(20).Add(
			col.New(4),
			text.NewCol(12, "O GDG-Petrópolis certifica que Fulano de Tal 123 participou do Evento Exemplo 123 no dia 2019-03-30.", props.Text{
				Size: 18,
			}),
			col.New(4),
		),
		row.New(15),
		row.New(30).Add(
			image.NewFromFileCol(20, "docs/assets/images/signature.png", props.Rect{
				Center: true,
			}),
		),
	)
}
