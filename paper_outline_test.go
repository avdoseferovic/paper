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

	doc, err := m.Generate()

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

	doc, err := paper.FromHTML("<h1>Alpha</h1><p>prose</p><h2>Beta</h2><h1 hidden>Skipped</h1>", cfg)

	require.NoError(t, err)
	pdfBytes := doc.GetBytes()
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Outlines")), "headings should produce an outline")
	assert.True(t, bytes.Contains(pdfBytes, []byte("Alpha")), "h1 title present")
	assert.True(t, bytes.Contains(pdfBytes, []byte("Beta")), "h2 title present")
	assert.False(t, outlineTitleCount(pdfBytes, "Skipped") > 0, "hidden heading must not create an outline entry")
}

func TestFromHTML_WhenOutlineFromHeadingsDisabled_ShouldNotEmitOutline(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()

	doc, err := paper.FromHTML("<h1>Alpha</h1><h2>Beta</h2>", cfg)

	require.NoError(t, err)
	assert.False(t, bytes.Contains(doc.GetBytes(), []byte("/Outlines")), "no outline without the option")
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

	doc, err := m.Generate()

	require.NoError(t, err)
	assert.False(t, bytes.Contains(doc.GetBytes(), []byte("/Outlines")), "no outline without props")
}
