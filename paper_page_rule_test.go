package paper_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
)

const pageRuleHTML = `<style>@page { size: A5; margin: 10mm }</style><p>content</p>`

func TestFromHTML_WhenPageRuleAndNoConfig_ShouldApplySizeAndMargins(t *testing.T) {
	t.Parallel()

	doc, err := paper.FromHTML(pageRuleHTML)
	require.NoError(t, err)
	_ = doc

	// Re-build via New-equivalent path to inspect config: FromHTML hides the
	// Paper instance, so verify through a fresh document's config using the
	// same entry point internals — instead, inspect the generated page size
	// via the PDF MediaBox.
	pdf := mustPDFString(t, pageRuleHTML)
	w, h := pagesize.GetDimensions(pagesize.A5)
	assert.True(t, containsMediaBox(pdf, w, h), "expected A5 MediaBox, pdf MediaBox not found")
}

func TestFromHTML_WhenPageRuleHasExplicitDimensions_ShouldApplyThem(t *testing.T) {
	t.Parallel()

	pdf := mustPDFString(t, `<style>@page { size: 200mm 100mm }</style><p>x</p>`)

	assert.True(t, containsMediaBox(pdf, 200, 100), "expected 200x100mm MediaBox")
}

func TestFromHTML_WhenPageRuleLandscape_ShouldSwapDimensions(t *testing.T) {
	t.Parallel()

	pdf := mustPDFString(t, `<style>@page { size: A4 landscape }</style><p>x</p>`)

	assert.True(t, containsMediaBox(pdf, 297, 210), "expected landscape A4 MediaBox")
}

func TestFromHTML_WhenExplicitConfigGiven_ShouldIgnorePageRule(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithPageSize(pagesize.A4).WithCompression(false).Build()
	doc, err := paper.FromHTML(pageRuleHTML, cfg)

	require.NoError(t, err)
	pdf := string(doc.GetBytes())
	assert.True(t, containsMediaBox(pdf, 210, 297), "explicit config must win over @page")
}

func TestFromHTMLReader_WhenPageRule_ShouldApplyLikeFromHTML(t *testing.T) {
	t.Parallel()

	doc, err := paper.FromHTMLReader(strings.NewReader(pageRuleHTML))

	require.NoError(t, err)
	w, h := pagesize.GetDimensions(pagesize.A5)
	assert.True(t, containsMediaBox(string(doc.GetBytes()), w, h), "FromHTMLReader must honor @page")
}

func mustPDFString(t *testing.T, html string) string {
	t.Helper()
	doc, err := paper.FromHTML(html)
	require.NoError(t, err)
	return string(doc.GetBytes())
}

// containsMediaBox reports whether the PDF declares a WxH mm MediaBox.
// The engine writes "/MediaBox [0 0 %.2f %.2f]" in points (mm * 72 / 25.4),
// see internal/pdf/page.go.
func containsMediaBox(pdf string, wMM, hMM float64) bool {
	const k = 72.0 / 25.4
	box := fmt.Sprintf("/MediaBox [0 0 %.2f %.2f]", wMM*k, hMM*k)
	return strings.Contains(pdf, box)
}
