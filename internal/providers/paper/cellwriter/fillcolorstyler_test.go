package cellwriter_test

import (
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/internal/mocks"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"
)

func TestNewFillColorStyler(t *testing.T) {
	t.Parallel()
	// Act
	sut := cellwriter.NewFillColorStyler(nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*cellwriter.FillColorStyler", fmt.Sprintf("%T", sut))
}

func TestFillColorStyle_Apply(t *testing.T) {
	t.Parallel()
	t.Run("When prop is nil and next is nil, should skip calls", func(t *testing.T) {
		t.Parallel()
		// Arrange
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("code unexpectedly panicked: %v", r)
			}
		}()

		sut := cellwriter.NewFillColorStyler(nil)

		// Act
		sut.Apply(100, 100, &entity.Config{}, nil)
	})
	t.Run("When prop is nil and next is filled, should skip current and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		width := 100.0
		height := 100.0
		cfg := &entity.Config{}
		var nilCellProp *props.Cell

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(width, height, cfg, nilCellProp).Once()

		sut := cellwriter.NewFillColorStyler(nil)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, nilCellProp)
	})
	t.Run("When has prop but background color is nil, should skip current and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		width := 100.0
		height := 100.0
		cfg := &entity.Config{}
		prop := &props.Cell{}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(width, height, cfg, prop).Once()

		sut := cellwriter.NewFillColorStyler(nil)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, prop)
	})
	t.Run("When has prop and color is filled, should apply current and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		width := 100.0
		height := 100.0
		cfg := &entity.Config{}
		prop := &props.Cell{
			BackgroundColor: &props.Color{Red: 100, Green: 150, Blue: 170},
		}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(width, height, cfg, prop).Once()

		fpdf := newPDF(t)
		fpdf.EXPECT().SetFillColor(prop.BackgroundColor.Red, prop.BackgroundColor.Green, prop.BackgroundColor.Blue).Once()
		fpdf.EXPECT().SetFillColor(255, 255, 255).Once()

		sut := cellwriter.NewFillColorStyler(fpdf)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, prop)
	})
}
