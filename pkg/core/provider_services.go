package core

import (
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GridProvider is the provider surface used to advance the PDF grid cursor and
// paint row/column cells.
type GridProvider interface {
	CreateRow(height float64)
	CreateCol(width, height float64, config *entity.Config, prop *props.Cell)
}

// LineProvider is the provider surface used by line components.
type LineProvider interface {
	AddLine(cell *entity.Cell, prop *props.Line)
}

// TextProvider is the provider surface used by text-like components.
type TextProvider interface {
	AddText(text string, cell *entity.Cell, prop *props.Text)
	AddCheckbox(label string, cell *entity.Cell, prop *props.Checkbox)
	GetFontHeight(prop *props.Font) float64
	GetLinesQuantity(text string, textProp *props.Text, colWidth float64) int
}

// CodeProvider is the provider surface used by barcode and matrix-code components.
type CodeProvider interface {
	AddMatrixCode(code string, cell *entity.Cell, prop *props.Rect)
	AddQrCode(code string, cell *entity.Cell, rect *props.Rect)
	AddBarCode(code string, cell *entity.Cell, prop *props.Barcode)
	GetDimensionsByMatrixCode(code string) (*entity.Dimensions, error)
	GetDimensionsByQrCode(code string) (*entity.Dimensions, error)
}

// ImageProvider is the provider surface used by image and background-image components.
type ImageProvider interface {
	GetDimensionsByImageByte(bytes []byte, extension extension.Type) (*entity.Dimensions, error)
	GetDimensionsByImage(file string) (*entity.Dimensions, error)
	AddImageFromFile(value string, cell *entity.Cell, prop *props.Rect)
	AddImageFromBytes(bytes []byte, cell *entity.Cell, prop *props.Rect, extension extension.Type)
	AddBackgroundImageFromBytes(bytes []byte, cell *entity.Cell, prop *props.Rect, extension extension.Type)
}

// DocumentProvider is the provider surface that finalizes generated PDF bytes.
type DocumentProvider interface {
	GenerateBytes() ([]byte, error)
}

// DocumentConfigProvider is the provider surface used to apply document-level options.
type DocumentConfigProvider interface {
	SetProtection(protection *entity.Protection)
	SetCompression(compression bool)
	SetMetadata(metadata *entity.Metadata)
}
