package signature_test

import (
	"github.com/avdoseferovic/paper/v2"
	"github.com/avdoseferovic/paper/v2/pkg/components/col"
	"github.com/avdoseferovic/paper/v2/pkg/components/signature"
)

// ExampleNew demonstrates how to create a signature component.
func ExampleNew() {
	m := paper.New()

	signature := signature.New("signature label")
	col := col.New(12).Add(signature)
	m.AddRow(10, col)

	// generate document
}

// ExampleNewCol demonstrates how to create a signature component wrapped into a column.
func ExampleNewCol() {
	m := paper.New()

	signatureCol := signature.NewCol(12, "signature label")
	m.AddRow(10, signatureCol)

	// generate document
}

// ExampleNewRow demonstrates how to create a signature component wrapped into a row.
func ExampleNewRow() {
	m := paper.New()

	signatureRow := signature.NewRow(10, "signature label")
	m.AddRows(signatureRow)

	// generate document
}
