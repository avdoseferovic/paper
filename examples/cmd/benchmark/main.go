package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/examples/internal/examplepath"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/list"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/metrics"

	"github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/signature"
	"github.com/avdoseferovic/paper/pkg/components/text"

	"github.com/avdoseferovic/paper/pkg/consts/extension"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

var dummyText = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec ac condimentum sem."

var background = &props.Color{
	Red:   200,
	Green: 200,
	Blue:  200,
}

func main() {
	var builder strings.Builder
	for i := range 100 {
		fmt.Println(i)
		builder.WriteString(fmt.Sprintf("%f", run().Value) + "\n")
	}

	out := examplepath.Repo("docs/assets/text/benchmark.txt")
	if err := examplepath.EnsureParent(out); err != nil {
		log.Fatal(err.Error())
	}

	err := os.WriteFile(out, []byte(builder.String()), os.ModePerm)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func run() *metrics.Time {
	cfg := config.NewBuilder().
		WithPageNumber().
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	err := m.RegisterHeader(buildHeader()...)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = m.RegisterFooter(buildFooter()...)
	if err != nil {
		log.Fatal(err.Error())
	}

	m.AddRows(
		text.NewRow(20, "Main features", props.Text{Size: 15, Top: 6.5}),
	)

	objects := getObjects(1158)
	rows, err := list.Build[Object](objects)
	if err != nil {
		log.Fatal(err.Error())
	}

	m.AddRows(rows...)

	for range 1158 {
		m.AddRows(buildCodesRow()...)
		m.AddRows(buildImagesRow()...)
		m.AddRows(buildTextsRow()...)
	}

	m.AddRows(
		text.NewRow(15, "Dummy Data", props.Text{Size: 12, Top: 5, Align: consts.AlignCenter}),
	)

	for range 1158 {
		m.AddRows(text.NewRow(20, dummyText+dummyText+dummyText+dummyText+dummyText))
	}

	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	return document.GetReport().TimeMetrics[0].Avg
}

func buildCodesRow() []core.Row {
	return []core.Row{
		row.New(20).Add(
			text.NewCol(4, "Barcode:", props.Text{Size: 15, Top: 6, Align: consts.AlignCenter}),
			code.NewBarCol(8, "barcode", props.Barcode{Center: true, Percent: 70}),
		),
		row.New(20).Add(
			text.NewCol(4, "QrCode:", props.Text{Size: 15, Top: 6, Align: consts.AlignCenter}),
			code.NewQrCol(8, "qrcode", props.Rect{Center: true, Percent: 70}),
		),
		row.New(20).Add(
			text.NewCol(4, "MatrixCode:", props.Text{Size: 15, Top: 6, Align: consts.AlignCenter}),
			code.NewMatrixCol(8, "matrixcode", props.Rect{Center: true, Percent: 70}),
		),
	}
}

func buildImagesRow() []core.Row {
	bytes, err := os.ReadFile(examplepath.Repo("docs/assets/images/frontpage.png"))
	if err != nil {
		fmt.Println("Got error while opening file:", err)
		os.Exit(1)
	}

	return []core.Row{
		row.New(20).Add(
			text.NewCol(4, "Image From File:", props.Text{Size: 15, Top: 6, Align: consts.AlignCenter}),
			image.NewFromFileCol(8, examplepath.Repo("docs/assets/images/biplane.jpg"), props.Rect{Center: true, Percent: 90}),
		),
		row.New(20).Add(
			text.NewCol(4, "Image From Bytes:", props.Text{Size: 15, Top: 6, Align: consts.AlignCenter}),
			image.NewFromBytesCol(8, bytes, extension.Png, props.Rect{Center: true, Percent: 90}),
		),
	}
}

func buildTextsRow() []core.Row {
	colText := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec ac condimentum sem."

	return []core.Row{
		row.New(20).Add(
			text.NewCol(4, "Text:", props.Text{Size: 15, Top: 6, Align: consts.AlignCenter}),
			text.NewCol(8, colText, props.Text{Size: 12, Top: 5, Align: consts.AlignCenter}),
		),
		row.New(40).Add(
			text.NewCol(4, "Signature:", props.Text{Size: 15, Top: 17, Align: consts.AlignCenter}),
			signature.NewCol(8, "Name", props.Signature{FontSize: 10}),
		),
	}
}

func buildHeader() []core.Row {
	r1 := row.New(30).Add(
		col.New(12).Add(
			text.New("Paper Config", props.Text{
				Top:   5,
				Size:  15,
				Align: consts.AlignCenter,
			}),
			text.New("Grid system, fast generation, embedded metrics and testable.", props.Text{
				Top:   13,
				Size:  13,
				Align: consts.AlignCenter,
			}),
		),
	)

	return []core.Row{r1}
}

func buildFooter() []core.Row {
	return []core.Row{
		row.New(10).Add(
			text.NewCol(2, "Site: https://paper.io/"),
			text.NewCol(5, "Discussions: https://github.com/avdoseferovic/paper/issues/257"),
			text.NewCol(5, "Repo: https://github.com/avdoseferovic/paper"),
		),
	}
}

type Object struct {
	Key   string
	Value string
}

func (o Object) GetHeader() core.Row {
	return row.New(10).Add(
		text.NewCol(4, "Key", props.Text{Style: fontstyle.Bold}),
		text.NewCol(8, "Bytes", props.Text{Style: fontstyle.Bold}),
	)
}

func (o Object) GetContent(i int) core.Row {
	r := row.New(5).Add(
		text.NewCol(4, o.Key),
		text.NewCol(8, o.Value),
	)

	if i%2 == 0 {
		r.WithStyle(&props.Cell{
			BackgroundColor: background,
		})
	}

	return r
}

func getObjects(maxObjects int) []Object {
	var objects []Object
	for i := range maxObjects {
		objects = append(objects, Object{
			Key:   fmt.Sprintf("Key: %d", i),
			Value: fmt.Sprintf("Bytes: %d", i),
		})
	}
	return objects
}
