package translate_test

import (
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/html"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStyledRowRendersWithoutPanic confirms the full translate→render pipeline
// for a styled h2 heading (which exercises SetCursor+borderRadius) produces
// a valid non-empty PDF with no panic. This is the integration smoke test for
// the margin-relative coordinate fix: prior to the fix, SetCursor/DrawFilledCircle
// sent the pen to (0,0) instead of (marginLeft, marginTop), producing misaligned
// bgs; the fix adds margin offsets so positioned drawing ops land at the same
// absolute coords as text rendering.
func TestStyledRowRendersWithoutPanic(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithLeftMargin(20).
		WithTopMargin(15).
		WithRightMargin(20).
		WithBottomMargin(15).
		Build()

	m := paper.New(cfg)
	err := m.AddHTML(`<html><head><style>
		h2 { background-color: #1a3e72; color: #ffffff; padding: 3mm 5mm; border-radius: 2mm; font-size: 12pt }
		.numbered { list-style-type: decimal-circle }
	</style></head><body>
		<h2>SUMMARY</h2>
		<p>Content below the band.</p>
		<ol class="numbered"><li>Item one</li><li>Item two</li></ol>
	</body></html>`)
	require.NoError(t, err)

	doc, err := m.Generate()
	require.NoError(t, err)

	pdfBytes := doc.GetBytes()
	assert.Greater(t, len(pdfBytes), 0, "generated PDF must not be empty")
	// A valid PDF begins with the %PDF header.
	assert.Equal(t, "%PDF", string(pdfBytes[:4]))
}

// TestStyledRowWithImageBaseDir exercises the block-level <img> path with the
// margin-corrected SetCursor so the image row height is used correctly.
func TestStyledRowWithImageBaseDir(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithLeftMargin(20).
		WithTopMargin(15).
		Build()

	m := paper.New(cfg)
	// No image resolver — img falls back to alt text path (no panic expected).
	rows, err := html.FromString(`<html><head><style>
		h2 { background-color: #1a3e72; color: #ffffff; padding: 3mm 5mm; border-radius: 2mm }
	</style></head><body>
		<img src="nonexistent.svg" width="14mm" height="14mm" alt="logo">
		<h2>INVOICE</h2>
		<p>Body text.</p>
	</body></html>`)
	require.NoError(t, err)
	m.AddRows(rows...)

	doc, err := m.Generate()
	require.NoError(t, err)
	assert.Greater(t, len(doc.GetBytes()), 0)
}
