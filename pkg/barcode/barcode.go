// Package barcode provides named barcode constructors over Paper's component
// model.
package barcode

import (
	codecomp "github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

type Type = consts.BarcodeType

const (
	Code128 Type = consts.BarcodeCode128
	EAN13   Type = consts.BarcodeEAN
)

func NewCode128(value string, barcodeProps ...props.Barcode) core.Component {
	return codecomp.NewBar(value, withType(Code128, barcodeProps)...)
}

func NewCode128Col(size int, value string, barcodeProps ...props.Barcode) core.Col {
	return codecomp.NewBarCol(size, value, withType(Code128, barcodeProps)...)
}

func NewCode128Row(height float64, value string, barcodeProps ...props.Barcode) core.Row {
	return codecomp.NewBarRow(height, value, withType(Code128, barcodeProps)...)
}

func NewAutoCode128Row(value string, barcodeProps ...props.Barcode) core.Row {
	return codecomp.NewAutoBarRow(value, withType(Code128, barcodeProps)...)
}

func NewEAN13(value string, barcodeProps ...props.Barcode) core.Component {
	return codecomp.NewBar(value, withType(EAN13, barcodeProps)...)
}

func NewEAN13Col(size int, value string, barcodeProps ...props.Barcode) core.Col {
	return codecomp.NewBarCol(size, value, withType(EAN13, barcodeProps)...)
}

func NewEAN13Row(height float64, value string, barcodeProps ...props.Barcode) core.Row {
	return codecomp.NewBarRow(height, value, withType(EAN13, barcodeProps)...)
}

func NewAutoEAN13Row(value string, barcodeProps ...props.Barcode) core.Row {
	return codecomp.NewAutoBarRow(value, withType(EAN13, barcodeProps)...)
}

func NewQR(value string, rectProps ...props.Rect) core.Component {
	return codecomp.NewQr(value, rectProps...)
}

func NewQRCol(size int, value string, rectProps ...props.Rect) core.Col {
	return codecomp.NewQrCol(size, value, rectProps...)
}

func NewQRRow(height float64, value string, rectProps ...props.Rect) core.Row {
	return codecomp.NewQrRow(height, value, rectProps...)
}

func NewAutoQRRow(value string, rectProps ...props.Rect) core.Row {
	return codecomp.NewAutoQrRow(value, rectProps...)
}

func NewDataMatrix(value string, rectProps ...props.Rect) core.Component {
	return codecomp.NewMatrix(value, rectProps...)
}

func NewDataMatrixCol(size int, value string, rectProps ...props.Rect) core.Col {
	return codecomp.NewMatrixCol(size, value, rectProps...)
}

func NewDataMatrixRow(height float64, value string, rectProps ...props.Rect) core.Row {
	return codecomp.NewMatrixRow(height, value, rectProps...)
}

func NewAutoDataMatrixRow(value string, rectProps ...props.Rect) core.Row {
	return codecomp.NewAutoMatrixRow(value, rectProps...)
}

func withType(kind Type, barcodeProps []props.Barcode) []props.Barcode {
	prop := props.Barcode{}
	if len(barcodeProps) > 0 {
		prop = barcodeProps[0]
	}
	prop.Type = kind
	return []props.Barcode{prop}
}
