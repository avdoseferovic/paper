package core_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/metrics"
)

func TestNewPDF(t *testing.T) {
	t.Parallel()
	// Act
	sut := core.NewPDF(nil, nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*core.Pdf", fmt.Sprintf("%T", sut))
}

func TestPdf_GetBase64(t *testing.T) {
	t.Parallel()
	// Arrange
	sut := core.NewPDF([]byte{1, 2, 3}, nil)

	// Act
	b64 := sut.GetBase64()

	// Assert
	assert.Equal(t, "AQID", b64)
}

func TestPdf_GetBytes(t *testing.T) {
	t.Parallel()
	// Arrange
	sut := core.NewPDF([]byte{1, 2, 3}, nil)

	// Act
	bytes := sut.GetBytes()

	// Assert
	assert.Equal(t, []byte{1, 2, 3}, bytes)
}

func TestPdf_GetReport(t *testing.T) {
	t.Parallel()
	// Arrange
	sut := core.NewPDF(nil, &metrics.Report{SizeMetric: metrics.SizeMetric{
		Key: "key",
		Size: metrics.Size{
			Value: 10.0,
			Scale: metrics.Byte,
		},
	}})

	// Act
	report := sut.GetReport()

	// Assert
	assert.Equal(t, "key", report.SizeMetric.Key)
}

func TestPdf_Save(t *testing.T) {
	t.Parallel()
	t.Run("when cannot save, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := core.NewPDF(nil, nil)

		// Act
		err := sut.Save("")

		// Assert
		assert.NotNil(t, err)
	})
	t.Run("when can save, should not return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		bytes := []byte{1, 2, 3}
		file := filepath.Join(t.TempDir(), "test.txt")
		sut := core.NewPDF(bytes, nil)

		// Act
		err := sut.Save(file)

		// Assert
		assert.Nil(t, err)
		savedBytes, _ := os.ReadFile(file)
		assert.Equal(t, bytes, savedBytes)
	})
}

func TestPdf_Merge(t *testing.T) {
	t.Parallel()
	t.Run("when merge fails due to invalid bytes, should return wrapped error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := core.NewPDF([]byte("not a valid pdf"), nil)

		// Act
		err := sut.Merge([]byte("also not a valid pdf"))

		// Assert
		assert.ErrorIs(t, err, core.ErrCannotMergeBytes)
	})
	t.Run("when merge succeeds and report is nil, should update bytes and return nil", func(t *testing.T) {
		t.Parallel()
		// Arrange
		m := paper.New()
		m.AddRows(text.NewRow(10, "page1"))
		doc, _ := m.Generate()
		pdfBytes := doc.GetBytes()

		sut := core.NewPDF(pdfBytes, nil)

		// Act
		err := sut.Merge(pdfBytes)

		// Assert
		assert.Nil(t, err)
		assert.Greater(t, len(sut.GetBytes()), len(pdfBytes))
	})
	t.Run("when merge succeeds and report is not nil, should update bytes and append metric", func(t *testing.T) {
		t.Parallel()
		// Arrange
		m := paper.New()
		m.AddRows(text.NewRow(10, "page1"))
		doc, _ := m.Generate()
		pdfBytes := doc.GetBytes()

		report := &metrics.Report{}
		sut := core.NewPDF(pdfBytes, report)

		// Act
		err := sut.Merge(pdfBytes)

		// Assert
		assert.Nil(t, err)
		assert.Greater(t, len(sut.GetBytes()), len(pdfBytes))
		assert.NotEmpty(t, sut.GetReport().TimeMetrics)
		assert.Equal(t, "merge_pdf", sut.GetReport().TimeMetrics[0].Key)
		assert.Equal(t, "file_size", sut.GetReport().SizeMetric.Key)
	})
}
