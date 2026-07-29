package code_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/code"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		// Act
		sut := code.New()

		// Assert
		assert.NotNil(t, sut)
		assert.Equal(t, "*code.Code", fmt.Sprintf("%T", sut))
	})
	t.Run("singleton is applied", func(t *testing.T) {
		t.Parallel()
		// Act
		sut1 := code.New()
		sut2 := code.New()

		// Assert
		assert.NotNil(t, sut1)
		assert.NotNil(t, sut2)
	})
}

func TestCode_GenDataMatrix(t *testing.T) {
	t.Parallel()
	t.Run("When cannot generate data matrix, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := code.New()

		data := genStringWithLength(5000)

		// Act
		bytes, err := sut.GenDataMatrix(data)

		// Assert
		assert.NotNil(t, err)
		assert.Nil(t, bytes)
	})
	t.Run("When can generate data matrix, should return bytes", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := code.New()

		data := genStringWithLength(50)

		// Act
		bytes, err := sut.GenDataMatrix(data)

		// Assert
		assert.NotNil(t, bytes)
		assert.Nil(t, err)
	})
}

func TestCode_GenBar(t *testing.T) {
	t.Parallel()
	t.Run("When cannot generate bar code, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := code.New()

		cell := &entity.Cell{
			X:      10,
			Y:      10,
			Width:  100,
			Height: 100,
		}

		prop := &props.Barcode{}
		prop.MakeValid()

		data := genStringWithLength(5000)

		// Act
		bytes, err := sut.GenBar(data, cell, prop)

		// Assert
		assert.NotNil(t, err)
		assert.Nil(t, bytes)
	})
	t.Run("When can generate bar code, should return bytes", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := code.New()

		cell := &entity.Cell{
			X:      10,
			Y:      10,
			Width:  100,
			Height: 100,
		}

		prop := &props.Barcode{}
		prop.MakeValid()

		data := genStringWithLength(60)

		// Act
		bytes, err := sut.GenBar(data, cell, prop)

		// Assert
		assert.NotNil(t, bytes)
		assert.Nil(t, err)
	})
	t.Run("When is ean and can generate bar code, should return bytes", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := code.New()

		cell := &entity.Cell{
			X:      10,
			Y:      10,
			Width:  100,
			Height: 100,
		}

		prop := &props.Barcode{
			Type: consts.BarcodeEAN,
		}
		prop.MakeValid()

		// Act
		bytes, err := sut.GenBar("123456789123", cell, prop)

		// Assert
		assert.NotNil(t, bytes)
		assert.Nil(t, err)
	})
}

func TestCode_GenQr(t *testing.T) {
	t.Parallel()
	t.Run("When cannot generate qr code, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := code.New()

		data := genStringWithLength(5000)

		// Act
		bytes, err := sut.GenQr(data)

		// Assert
		assert.NotNil(t, err)
		assert.Nil(t, bytes)
	})
	t.Run("When can generate qr code, should return bytes", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := code.New()

		data := genStringWithLength(50)

		// Act
		bytes, err := sut.GenQr(data)

		// Assert
		assert.NotNil(t, bytes)
		assert.Nil(t, err)
	})
}

func TestMatrixCodeCurrentDimensions(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		gen   func(*code.Code, string) (*entity.Image, error)
		sizes map[string]float64
	}{
		{name: "QR", gen: (*code.Code).GenQr, sizes: map[string]float64{"HELLO WORLD": 21, strings.Repeat("a", 50): 33}},
		{name: "DataMatrix", gen: (*code.Code).GenDataMatrix, sizes: map[string]float64{"HELLO WORLD": 16, strings.Repeat("a", 50): 32}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for input, size := range tt.sizes {
				image, err := tt.gen(code.New(), input)
				if err != nil {
					t.Fatalf("generate %s: %v", tt.name, err)
				}
				if image.Dimensions.Width != size || image.Dimensions.Height != size {
					t.Fatalf("%s dimensions = %.0fx%.0f, want %.0fx%.0f", tt.name, image.Dimensions.Width, image.Dimensions.Height, size, size)
				}
			}
		})
	}
}

func genStringWithLength(length int) string {
	var builder strings.Builder
	for range length {
		builder.WriteString("a")
	}
	return builder.String()
}
