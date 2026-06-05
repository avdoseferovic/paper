package text_test

import (
	"github.com/johnfercher/paper/v2"
	"github.com/johnfercher/paper/v2/pkg/components/col"
	"github.com/johnfercher/paper/v2/pkg/components/text"
)

// ExampleNew demonstrates how to create a text component.
func ExampleNew() {
	m := paper.New()

	text := text.New("text")
	col := col.New(12).Add(text)
	m.AddRow(10, col)

	// generate document
}

// ExampleNewCol demonstrates how to create a text component wrapped into a column.
func ExampleNewCol() {
	m := paper.New()

	textCol := text.NewCol(12, "text")
	m.AddRow(10, textCol)

	// generate document
}

// ExampleNewRow demonstrates how to create a text component wrapped into a row.
func ExampleNewRow() {
	m := paper.New()

	textRow := text.NewRow(10, "text")
	m.AddRows(textRow)

	// generate document
}
