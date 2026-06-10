package cellwriter_test

import (
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/internal/mocks"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"

	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestNewBorderColorStyler(t *testing.T) {
	t.Parallel()
	// Act
	sut := cellwriter.NewBorderColorStyler(nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*cellwriter.BorderColorStyler", fmt.Sprintf("%T", sut))
}

func TestBorderColorStyler_Apply(t *testing.T) {
	t.Parallel()
	t.Run("When prop is nil and next is nil, should skip calls", func(t *testing.T) {
		t.Parallel()
		// Arrange
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("code unexpectedly panicked: %v", r)
			}
		}()

		sut := cellwriter.NewBorderColorStyler(nil)

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

		sut := cellwriter.NewBorderColorStyler(nil)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, nilCellProp)
	})
	t.Run("When has prop but border color is nil, should skip current and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		width := 100.0
		height := 100.0
		cfg := &entity.Config{}
		prop := &props.Cell{}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(width, height, cfg, prop).Once()

		sut := cellwriter.NewBorderColorStyler(nil)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, prop)
	})
	t.Run("When has prop and border color is defined, should apply current and call next", func(t *testing.T) {
		t.Parallel()
		// Arrange
		width := 100.0
		height := 100.0
		cfg := &entity.Config{}
		prop := &props.Cell{
			BorderColor: &props.Color{Red: 140, Green: 100, Blue: 80},
		}

		inner := mocks.NewCellWriter(t)
		inner.EXPECT().Apply(width, height, cfg, prop).Once()

		fpdf := newPDF(t)
		fpdf.EXPECT().SetDrawColor(prop.BorderColor.Red, prop.BorderColor.Green, prop.BorderColor.Blue).Once()
		fpdf.EXPECT().SetDrawColor(0, 0, 0).Once()

		sut := cellwriter.NewBorderColorStyler(fpdf)
		sut.SetNext(inner)

		// Act
		sut.Apply(width, height, cfg, prop)
	})
}
