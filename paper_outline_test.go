package paper_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestGenerate_WhenTextHasOutlineProp_ShouldEmitPDFOutline(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()
	m := paper.New(cfg)
	m.AddAutoRow(col.New(12).Add(text.New("Chapter 1", props.Text{
		Outline: &props.Outline{Level: 0},
	})))
	m.AddAutoRow(col.New(12).Add(text.New("Section 1.1", props.Text{
		Outline: &props.Outline{Level: 1, Title: "First Section"},
	})))

	doc, err := m.Generate(context.Background())

	require.NoError(t, err)
	pdfBytes := doc.GetBytes()
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Outlines")), "catalog should reference /Outlines")
	assert.True(t, bytes.Contains(pdfBytes, []byte("Chapter 1")), "outline title from text content")
	assert.True(t, bytes.Contains(pdfBytes, []byte("First Section")), "explicit outline title")
}

func TestFromHTML_WhenOutlineFromHeadingsEnabled_ShouldEmitOutlineFromHeadings(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithOutlineFromHeadings(true).
		Build()

	doc, err := paper.FromHTML(context.Background(),
		"<h1>Alpha</h1><p>prose</p><h2>Beta</h2><h1 hidden>Skipped</h1><h2><strong>Bold</strong> Title</h2>", cfg)

	require.NoError(t, err)
	pdfBytes := doc.GetBytes()
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Outlines")), "headings should produce an outline")
	assert.True(t, bytes.Contains(pdfBytes, []byte("Alpha")), "h1 title present")
	assert.True(t, bytes.Contains(pdfBytes, []byte("Beta")), "h2 title present")
	assert.False(t, outlineTitleCount(pdfBytes, "Skipped") > 0, "hidden heading must not create an outline entry")
	assert.True(t, outlineTitleCount(pdfBytes, "Bold Title") > 0, "multi-run heading title must concatenate runs")
}

func TestFromHTML_WhenOutlineFromHeadingsDisabled_ShouldNotEmitOutline(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()

	doc, err := paper.FromHTML(context.Background(), "<h1>Alpha</h1><h2>Beta</h2>", cfg)

	require.NoError(t, err)
	assert.False(t, bytes.Contains(doc.GetBytes(), []byte("/Outlines")), "no outline without the option")
}

func TestGenerate_WhenConcurrentModeWithOutlines_ShouldPreserveOutline(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithConcurrentMode(4).
		Build()
	assertOutlineSurvivesGeneration(t, cfg)
}

func TestGenerate_WhenLowMemoryModeWithOutlines_ShouldPreserveOutline(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithSequentialLowMemoryMode(4).
		Build()
	assertOutlineSurvivesGeneration(t, cfg)
}

func assertOutlineSurvivesGeneration(t *testing.T, cfg *entity.Config) {
	t.Helper()

	m := paper.New(cfg)
	for page := 1; page <= 12; page++ {
		m.AddAutoRow(col.New(12).Add(text.New(fmt.Sprintf("Chapter %d", page), props.Text{
			Outline: &props.Outline{Level: 0},
		})))
		m.AddRow(250, col.New(12).Add(text.New("filler", props.Text{})))
	}

	doc, err := m.Generate(context.Background())

	require.NoError(t, err)
	pdfBytes := doc.GetBytes()
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Outlines")), "outline must survive chunked generation + merge")
	assert.True(t, bytes.Contains(pdfBytes, []byte("/PageMode /UseOutlines")))
	for page := 1; page <= 12; page++ {
		title := fmt.Sprintf("Chapter %d", page)
		assert.True(t, outlineTitleCount(pdfBytes, title) > 0, "missing outline title "+title)
	}
}

// outlineTitleCount counts outline objects whose /Title contains title text.
// Engine outline titles for non-UTF8 docs are written as literal strings.
func outlineTitleCount(pdfBytes []byte, title string) int {
	count := 0
	for _, chunk := range bytes.Split(pdfBytes, []byte("obj")) {
		if bytes.Contains(chunk, []byte("/Title")) && bytes.Contains(chunk, []byte(title)) {
			count++
		}
	}
	return count
}

func TestGenerate_WhenNoOutlineProps_ShouldNotEmitPDFOutline(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()
	m := paper.New(cfg)
	m.AddAutoRow(col.New(12).Add(text.New("Plain text", props.Text{})))

	doc, err := m.Generate(context.Background())

	require.NoError(t, err)
	assert.False(t, bytes.Contains(doc.GetBytes(), []byte("/Outlines")), "no outline without props")
}
