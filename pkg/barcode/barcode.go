// Package barcode provides named barcode constructors over Paper's component
// model.
package barcode

import (
	codecomp "github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

// Type is the barcode symbology to encode with.
type Type = consts.BarcodeType

// The linear symbologies this package can encode.
const (
	Code128 Type = consts.BarcodeCode128
	EAN13   Type = consts.BarcodeEAN
)

// NewCode128 encodes value as a Code 128 barcode component.
func NewCode128(value string, barcodeProps ...props.Barcode) core.Component {
	return codecomp.NewBar(value, withType(Code128, barcodeProps)...)
}

// NewCode128Col encodes value as a Code 128 barcode in a column spanning size grid units.
func NewCode128Col(size int, value string, barcodeProps ...props.Barcode) core.Col {
	return codecomp.NewBarCol(size, value, withType(Code128, barcodeProps)...)
}

// NewCode128Row encodes value as a Code 128 barcode in a row of the given height in
// millimeters.
func NewCode128Row(height float64, value string, barcodeProps ...props.Barcode) core.Row {
	return codecomp.NewBarRow(height, value, withType(Code128, barcodeProps)...)
}

// NewAutoCode128Row encodes value as a Code 128 barcode in a row whose height follows the
// code itself.
func NewAutoCode128Row(value string, barcodeProps ...props.Barcode) core.Row {
	return codecomp.NewAutoBarRow(value, withType(Code128, barcodeProps)...)
}

// NewEAN13 encodes value as a EAN-13 barcode component.
func NewEAN13(value string, barcodeProps ...props.Barcode) core.Component {
	return codecomp.NewBar(value, withType(EAN13, barcodeProps)...)
}

// NewEAN13Col encodes value as a EAN-13 barcode in a column spanning size grid units.
func NewEAN13Col(size int, value string, barcodeProps ...props.Barcode) core.Col {
	return codecomp.NewBarCol(size, value, withType(EAN13, barcodeProps)...)
}

// NewEAN13Row encodes value as a EAN-13 barcode in a row of the given height in
// millimeters.
func NewEAN13Row(height float64, value string, barcodeProps ...props.Barcode) core.Row {
	return codecomp.NewBarRow(height, value, withType(EAN13, barcodeProps)...)
}

// NewAutoEAN13Row encodes value as a EAN-13 barcode in a row whose height follows the
// code itself.
func NewAutoEAN13Row(value string, barcodeProps ...props.Barcode) core.Row {
	return codecomp.NewAutoBarRow(value, withType(EAN13, barcodeProps)...)
}

// NewQR encodes value as a QR code component.
func NewQR(value string, rectProps ...props.Rect) core.Component {
	return codecomp.NewQr(value, rectProps...)
}

// NewQRCol encodes value as a QR code in a column spanning size grid units.
func NewQRCol(size int, value string, rectProps ...props.Rect) core.Col {
	return codecomp.NewQrCol(size, value, rectProps...)
}

// NewQRRow encodes value as a QR code in a row of the given height in
// millimeters.
func NewQRRow(height float64, value string, rectProps ...props.Rect) core.Row {
	return codecomp.NewQrRow(height, value, rectProps...)
}

// NewAutoQRRow encodes value as a QR code in a row whose height follows the
// code itself.
func NewAutoQRRow(value string, rectProps ...props.Rect) core.Row {
	return codecomp.NewAutoQrRow(value, rectProps...)
}

// NewDataMatrix encodes value as a Data Matrix code component.
func NewDataMatrix(value string, rectProps ...props.Rect) core.Component {
	return codecomp.NewMatrix(value, rectProps...)
}

// NewDataMatrixCol encodes value as a Data Matrix code in a column spanning size grid units.
func NewDataMatrixCol(size int, value string, rectProps ...props.Rect) core.Col {
	return codecomp.NewMatrixCol(size, value, rectProps...)
}

// NewDataMatrixRow encodes value as a Data Matrix code in a row of the given height in
// millimeters.
func NewDataMatrixRow(height float64, value string, rectProps ...props.Rect) core.Row {
	return codecomp.NewMatrixRow(height, value, rectProps...)
}

// NewAutoDataMatrixRow encodes value as a Data Matrix code in a row whose height follows the
// code itself.
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
