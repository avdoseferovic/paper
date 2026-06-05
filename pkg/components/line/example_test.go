package line_test

import (
	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/line"
)

// ExampleNew demonstrates how create a line component.
func ExampleNew() {
	m := paper.New()

	line := line.New()
	col := col.New(12).Add(line)
	m.AddRow(10, col)

	// generate document
}

// ExampleNewCol demonstrates how to crete a line wrapped into a column.
func ExampleNewCol() {
	m := paper.New()

	lineCol := line.NewCol(12)
	m.AddRow(10, lineCol)

	// generate document
}

// ExampleNewRow demonstrates how to crete a line wrapped into a row.
func ExampleNewRow() {
	m := paper.New()

	lineRow := line.NewRow(10)
	m.AddRows(lineRow)

	// generate document
}
