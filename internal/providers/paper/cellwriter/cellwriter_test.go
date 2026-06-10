package cellwriter_test

import (
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestNewCellCreator(t *testing.T) {
	t.Parallel()
	// Act
	sut := cellwriter.NewCellWriter(nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*cellwriter.cellWriter", fmt.Sprintf("%T", sut))
}

func TestCellWriter_Apply(t *testing.T) {
	t.Parallel()
	t.Run("when prop is nil without debug, should call cellformat correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		config := &entity.Config{}
		width := 100.0
		height := 200.0
		fpdf := newPDF(t)
		fpdf.EXPECT().CellFormat(width, height, "", "", 0, "C", false, 0, "").Once()

		sut := cellwriter.NewCellWriter(fpdf)

		// Act
		sut.Apply(width, height, config, nil)
	})
	t.Run("when prop is nil with debug, should call cellformat correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		config := &entity.Config{
			Debug: true,
		}
		width := 100.0
		height := 200.0
		fpdf := newPDF(t)
		fpdf.EXPECT().CellFormat(width, height, "", "LTRB", 0, "C", false, 0, "").Once()

		sut := cellwriter.NewCellWriter(fpdf)

		// Act
		sut.Apply(width, height, config, nil)
	})
	t.Run("when has prop without debug, should call cellformat correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		config := &entity.Config{}
		prop := fixture.CellProp()
		width := 100.0
		height := 200.0
		fpdf := newPDF(t)
		fpdf.EXPECT().CellFormat(width, height, "", "L", 0, "C", true, 0, "").Once()

		sut := cellwriter.NewCellWriter(fpdf)

		// Act
		sut.Apply(width, height, config, &prop)
	})
	t.Run("when has prop with debug, should call cellformat correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		config := &entity.Config{
			Debug: true,
		}
		prop := fixture.CellProp()
		width := 100.0
		height := 200.0
		fpdf := newPDF(t)
		fpdf.EXPECT().CellFormat(width, height, "", "LTRB", 0, "C", true, 0, "").Once()

		sut := cellwriter.NewCellWriter(fpdf)

		// Act
		sut.Apply(width, height, config, &prop)
	})
}
