package paper_test

import (
	"bytes"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestGenerate_WhenWatermarkConfigured_ShouldStampEveryPage(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithWatermark("DRAFT").
		Build()
	m := paper.New(cfg)
	for range 3 {
		m.AddRow(250, col.New(12).Add(text.New("page filler", props.Text{})))
	}

	doc, err := m.Generate()

	require.NoError(t, err)
	pdfBytes := doc.GetBytes()
	assert.Equal(t, 3, bytes.Count(pdfBytes, []byte("DRAFT")), "watermark text once per page")
	assert.True(t, bytes.Contains(pdfBytes, []byte(" cm\n")), "rotation transform ops present")
}

func TestGenerate_WhenNoWatermark_ShouldNotStamp(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()
	m := paper.New(cfg)
	m.AddRow(20, col.New(12).Add(text.New("plain", props.Text{})))

	doc, err := m.Generate()

	require.NoError(t, err)
	assert.False(t, bytes.Contains(doc.GetBytes(), []byte("DRAFT")))
}

func TestGenerate_WhenWatermarkOnSmallCustomPage_ShouldScaleDownAndRender(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithDimensions(100, 80).
		WithWatermark("A VERY LONG CONFIDENTIAL WATERMARK STRING").
		Build()
	m := paper.New(cfg)
	m.AddRow(20, col.New(12).Add(text.New("content", props.Text{})))

	doc, err := m.Generate()

	require.NoError(t, err)
	pdfBytes := doc.GetBytes()
	assert.True(t, bytes.Contains(pdfBytes, []byte("A VERY LONG CONFIDENTIAL WATERMARK STRING")), "scaled watermark still renders")
}
