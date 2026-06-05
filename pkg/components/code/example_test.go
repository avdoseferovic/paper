package code_test

import (
	"github.com/avdoseferovic/paper/v2/pkg/components/col"
	"github.com/avdoseferovic/paper/v2/pkg/props"

	"github.com/avdoseferovic/paper/v2"
	"github.com/avdoseferovic/paper/v2/pkg/components/code"
)

// ExampleNewBar demonstrates how to generate a barcode and add it to paper.
func ExampleNewBar() {
	m := paper.New()

	barCode := code.NewBar("123456789", props.Barcode{Percent: 70.5})
	col := col.New(6).Add(barCode)
	m.AddRow(10, col)

	// generate document
}

// ExampleNewBarCol demonstrates how to generate a column with a barcode and add it to paper.
func ExampleNewBarCol() {
	m := paper.New()

	barCodeCol := code.NewBarCol(6, "123456", props.Barcode{Percent: 70.5})
	m.AddRow(10, barCodeCol)

	// generate document
}

// ExampleNewBarRow demonstrates how to generate a row with a barcode and add it to paper.
func ExampleNewBarRow() {
	m := paper.New()

	barCodeRow := code.NewBarRow(10, "123456789", props.Barcode{Percent: 70.5})
	m.AddRows(barCodeRow)

	// generate document
}

// ExampleNewQr demonstrates how to generate a qrcode and add it to paper.
func ExampleNewQr() {
	m := paper.New()

	qrCode := code.NewQr("123456789", props.Rect{Percent: 70.5})
	col := col.New(6).Add(qrCode)
	m.AddRow(10, col)

	// generate document
}

// ExampleNewQrCol demonstrates how to generate a column with a qrcode and add it to paper.
func ExampleNewQrCol() {
	m := paper.New()

	qrCodeCol := code.NewQrCol(12, "123456789", props.Rect{Percent: 70.5})
	m.AddRow(10, qrCodeCol)

	// generate document
}

// ExampleNewQrRow demonstrates how to generate a row with a qrcode and add it to paper.
func ExampleNewQrRow() {
	m := paper.New()

	qrCodeRow := code.NewQrRow(10, "123456789", props.Rect{Percent: 70.5})
	m.AddRows(qrCodeRow)

	// generate document
}

// ExampleNewMatrix demonstrates how to generate a matrixcode and add it to paper.
func ExampleNewMatrix() {
	m := paper.New()

	matrixCode := code.NewMatrix("123456789", props.Rect{Percent: 70.5})
	col := col.New(6).Add(matrixCode)
	m.AddRow(10, col)

	// generate document
}

// ExampleNewMatrixCol demonstrates how to generate a column with a matrixcode and add it to paper.
func ExampleNewMatrixCol() {
	m := paper.New()

	matrixCodeCol := code.NewMatrixCol(12, "123456789", props.Rect{Percent: 70.5})
	m.AddRow(10, matrixCodeCol)

	// generate document
}

// ExampleNewMatrixRow demonstrates how to generate a row with a matrixcode and add it to paper.
func ExampleNewMatrixRow() {
	m := paper.New()

	matrixCodeRow := code.NewMatrixRow(10, "123456789", props.Rect{Percent: 70.5})
	m.AddRows(matrixCodeRow)

	// generate document
}
