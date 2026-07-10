package paper_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
)

func TestGenerate_WithTaggedPDF_ShouldEmitStructureTreeAndMarkedContent(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithTaggedPDF(true).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("Tagged content")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)
	pdfBytes := pdf.GetBytes()

	assert.True(t, bytes.Contains(pdfBytes, []byte("/MarkInfo << /Marked true >>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/StructTreeRoot")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/S /Document")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/S /P")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/StructParents 0")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Nums [")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/P << /MCID 0 >> BDC")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("EMC")))
}

func TestGenerate_TaggedPDFDisabledByDefault(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("Untagged content")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)
	pdfBytes := pdf.GetBytes()

	assert.False(t, bytes.Contains(pdfBytes, []byte("/StructTreeRoot")))
	assert.False(t, bytes.Contains(pdfBytes, []byte("BDC")))
}

func TestGenerate_WithRuntimeSetTagged_ShouldEmitTaggedPDF(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()
	doc := paper.New(cfg)
	doc.SetTagged(true)
	doc.AddAutoRow(col.New(12).Add(text.New("Runtime tagged content")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)

	assert.True(t, bytes.Contains(pdf.GetBytes(), []byte("/Marked true")))
}

func TestGenerate_WithTaggedPDFAndConcurrentMode_ShouldKeepCatalogEntries(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithConcurrentMode(2).
		WithTaggedPDF(true).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("Tagged content")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)

	assert.True(t, bytes.Contains(pdf.GetBytes(), []byte("/StructTreeRoot")))
}
