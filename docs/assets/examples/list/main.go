package main

import (
	"fmt"
	"log"

	"github.com/avdoseferovic/paper"

	"github.com/avdoseferovic/paper/pkg/components/list"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"

	"github.com/avdoseferovic/paper/pkg/components/text"
)

var background = &props.Color{
	Red:   200,
	Green: 200,
	Blue:  200,
}

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/list.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/list.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper() core.Paper {
	mrt := paper.New()
	m := paper.NewMetricsDecorator(mrt)

	objects := getObjects(100)
	rows, err := list.Build[Object](objects)
	if err != nil {
		log.Fatal(err.Error())
	}

	m.AddRows(rows...)
	return m
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

func getObjects(max int) []Object {
	var objects []Object
	for i := range max {
		objects = append(objects, Object{
			Key:   fmt.Sprintf("Key: %d", i),
			Value: fmt.Sprintf("Bytes: %d", i),
		})
	}
	return objects
}
