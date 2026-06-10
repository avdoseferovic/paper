package code

import (
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type rectCodeConstructor func(code string, ps ...props.Rect) core.Component

type QrCode = rectCode

// NewQr is responsible to create an instance of a QrCode.
func NewQr(code string, barcodeProps ...props.Rect) core.Component {
	return newRectCode("qrcode", code, renderQrCode, qrCodeDimensions, barcodeProps...)
}

// NewQrCol is responsible to create an instance of a QrCode wrapped in a Col.
func NewQrCol(size int, code string, ps ...props.Rect) core.Col {
	return newRectComponentCol(size, NewQr, code, ps...)
}

// NewAutoQrRow is responsible to create an instance of a qrcode wrapped in a Row with automatic height.
//   - code: The value that must be placed in the qrcode
//   - ps: A set of settings that must be applied to the qrcode
func NewAutoQrRow(code string, ps ...props.Rect) core.Row {
	return newAutoRectComponentRow(NewQr, code, ps...)
}

// NewQrRow is responsible to create an instance of a QrCode wrapped in a Row.
func NewQrRow(height float64, code string, ps ...props.Rect) core.Row {
	return newRectComponentRow(height, NewQr, code, ps...)
}

type MatrixCode = rectCode

// NewMatrix is responsible to create an instance of a MatrixCode.
func NewMatrix(code string, barcodeProps ...props.Rect) core.Component {
	return newRectCode("matrixcode", code, renderMatrixCode, matrixCodeDimensions, barcodeProps...)
}

// NewMatrixCol is responsible to create an instance of a MatrixCode wrapped in a Col.
func NewMatrixCol(size int, code string, ps ...props.Rect) core.Col {
	return newRectComponentCol(size, NewMatrix, code, ps...)
}

// NewAutoMatrixRow is responsible to create an instance of a Matrix code wrapped in a Row with automatic height.
//   - code: The value that must be placed in the matrixcode
//   - ps: A set of settings that must be applied to the matrixcode
func NewAutoMatrixRow(code string, ps ...props.Rect) core.Row {
	return newAutoRectComponentRow(NewMatrix, code, ps...)
}

// NewMatrixRow is responsible to create an instance of a MatrixCode wrapped in a Row.
func NewMatrixRow(height float64, code string, ps ...props.Rect) core.Row {
	return newRectComponentRow(height, NewMatrix, code, ps...)
}

func renderQrCode(provider core.Provider, code string, cell *entity.Cell, prop *props.Rect) {
	provider.AddQrCode(code, cell, prop)
}

func qrCodeDimensions(provider core.Provider, code string) (*entity.Dimensions, error) {
	return provider.GetDimensionsByQrCode(code)
}

func renderMatrixCode(provider core.Provider, code string, cell *entity.Cell, prop *props.Rect) {
	provider.AddMatrixCode(code, cell, prop)
}

func matrixCodeDimensions(provider core.Provider, code string) (*entity.Dimensions, error) {
	return provider.GetDimensionsByMatrixCode(code)
}
