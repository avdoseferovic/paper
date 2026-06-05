// Package code implements creation of Barcode, MatrixCode and QrCode.
// nolint:dupl
package code

import (
	"github.com/avdoseferovic/paper/pkg/tree/node"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type QrCode struct {
	code   string
	prop   props.Rect
	config *entity.Config
}

// NewQr is responsible to create an instance of a QrCode.
func NewQr(code string, barcodeProps ...props.Rect) core.Component {
	prop := props.Rect{}
	if len(barcodeProps) > 0 {
		prop = barcodeProps[0]
	}
	prop.MakeValid()

	return &QrCode{
		code: code,
		prop: prop,
	}
}

// NewQrCol is responsible to create an instance of a QrCode wrapped in a Col.
func NewQrCol(size int, code string, ps ...props.Rect) core.Col {
	qrCode := NewQr(code, ps...)
	return col.New(size).Add(qrCode)
}

// NewAutoMatrixRow is responsible to create an instance of a qrcode wrapped in a Row with automatic height.
//   - code: The value that must be placed in the qrcode
//   - ps: A set of settings that must be applied to the qrcode
func NewAutoQrRow(code string, ps ...props.Rect) core.Row {
	qrCode := NewQr(code, ps...)
	c := col.New().Add(qrCode)
	return row.New().Add(c)
}

// NewQrRow is responsible to create an instance of a QrCode wrapped in a Row.
func NewQrRow(height float64, code string, ps ...props.Rect) core.Row {
	qrCode := NewQr(code, ps...)
	c := col.New().Add(qrCode)
	return row.New(height).Add(c)
}

// Render renders a QrCode into a PDF context.
func (q *QrCode) Render(provider core.Provider, cell *entity.Cell) {
	provider.AddQrCode(q.code, cell, &q.prop)
}

// GetStructure returns the Structure of a QrCode.
func (q *QrCode) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:    "qrcode",
		Value:   q.code,
		Details: q.prop.ToMap(),
	}

	return node.New(str)
}

// GetHeight returns the height that the QrCode will have in the PDF
func (q *QrCode) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	dimensions, err := provider.GetDimensionsByQrCode(q.code)
	if err != nil {
		return 0
	}
	proportion := dimensions.Height / dimensions.Width
	width := (q.prop.Percent / 100) * cell.Width
	return proportion * width
}

// SetConfig set the config for the component.
func (q *QrCode) SetConfig(config *entity.Config) {
	q.config = config
}
