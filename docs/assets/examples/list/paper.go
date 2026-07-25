// Package list demonstrates rendering a slice of values as a table with a header row.
package list

import (
	"fmt"
	"log"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

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

// GetPaper builds the list example document.
func GetPaper() core.Paper {
	mrt := paper.New()
	m := decorator.NewMetrics(mrt)

	objects := getObjects(100)
	rows, err := list.Build[Object](objects)
	if err != nil {
		log.Fatal(err.Error())
	}

	m.AddRows(rows...)
	return m
}

// Object is one key/value row rendered by the list example. It implements the
// list component's interfaces so a slice of Objects can be built into a table.
type Object struct {
	Key   string
	Value string
}

// GetHeader is part of the list example.
func (o Object) GetHeader() core.Row {
	return row.New(10).Add(
		text.NewCol(4, "Key", props.Text{Style: fontstyle.Bold}),
		text.NewCol(8, "Bytes", props.Text{Style: fontstyle.Bold}),
	)
}

// GetContent is part of the list example.
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

func getObjects(count int) []Object {
	var objects []Object
	for i := range count {
		objects = append(objects, Object{
			Key:   fmt.Sprintf("Key: %d", i),
			Value: fmt.Sprintf("Bytes: %d", i),
		})
	}
	return objects
}
