package list_test

import (
	"strconv"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/list"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/core"
)

// Obj implements Listable interface
type Obj struct {
	ID   int
	Name string
}

// GetHeader is a method from Listable interface to create a header row
// based in an array element.
func (o Obj) GetHeader() core.Row {
	idCol := text.NewCol(6, "ID")
	nameCol := text.NewCol(6, "Name")
	return row.New(5).Add(idCol, nameCol)
}

// GetContent is a method from Listable interface to create a row
// based in an array element.
// i is the current index of the object list to be added into a row
// this can be used to customize pair/odd rows.
func (o Obj) GetContent(_ int) core.Row {
	idCol := text.NewCol(6, strconv.Itoa(o.ID))
	nameCol := text.NewCol(6, o.Name)
	return row.New(5).Add(idCol, nameCol)
}

// ExampleBuild demonstrates how to create a list of rows based into an array of objects.
func ExampleBuild() {
	/*
		// Obj implements Listable interface
		type Obj struct {
			ID   int
			Name string
		}

		// GetHeader is a method from Listable interface to create a header row
		// based in an array element.
		func (o Obj) GetHeader() core.Row {
			idCol := text.NewCol(6, "ID")
			nameCol := text.NewCol(6, "Name")
			return row.New(5).Add(idCol, nameCol)
		}

		// GetContent is a method from Listable interface to create a row
		// based in an array element.
		// i is the current index of the object list to be added into a row
		// this can be used to customize pair/odd rows.
		func (o Obj) GetContent(_ int) core.Row {
			idCol := text.NewCol(6, fmt.Sprintf("%d", o.ID))
			nameCol := text.NewCol(6, o.Name)
			return row.New(5).Add(idCol, nameCol)
		}
	*/

	objs := []Obj{
		{
			ID:   0,
			Name: "obj name 0",
		},
		{
			ID:   1,
			Name: "obj name 1",
		},
	}

	rows, _ := list.Build[Obj](objs)

	m := paper.New()
	m.AddRows(rows...)

	// generate document
}

// ExampleBuild demonstrates how to create a list of rows based into an array of pointer objects.
func ExampleBuildFromPointer() {
	/*
		// Obj implements Listable interface
		type Obj struct {
			ID   int
			Name string
		}

		// GetHeader is a method from Listable interface to create a header row
		// based in an array element.
		func (o Obj) GetHeader() core.Row {
			idCol := text.NewCol(6, "ID")
			nameCol := text.NewCol(6, "Name")
			return row.New(5).Add(idCol, nameCol)
		}

		// GetContent is a method from Listable interface to create a row
		// based in an array element.
		// i is the current index of the object list to be added into a row
		// this can be used to customize pair/odd rows.
		func (o Obj) GetContent(_ int) core.Row {
			idCol := text.NewCol(6, fmt.Sprintf("%d", o.ID))
			nameCol := text.NewCol(6, o.Name)
			return row.New(5).Add(idCol, nameCol)
		}
	*/

	objs := []*Obj{
		{
			ID:   0,
			Name: "obj name 0",
		},
		{
			ID:   1,
			Name: "obj name 1",
		},
	}

	rows, _ := list.BuildFromPointer[Obj](objs)

	m := paper.New()
	m.AddRows(rows...)

	// generate document
}
