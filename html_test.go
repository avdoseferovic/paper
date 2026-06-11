package paper_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
	"github.com/avdoseferovic/paper/pkg/html"
	"github.com/avdoseferovic/paper/pkg/props"
)

func multiPageHeaderHTML() string {
	var sb strings.Builder
	sb.WriteString("<header><p>REPEATING-HEADER</p></header>")
	sb.WriteString("<footer><p>REPEATING-FOOTER</p></footer>")
	for range 120 {
		sb.WriteString("<p>body paragraph that fills the page with content</p>")
	}
	return sb.String()
}

func TestFromHTML_WhenTopLevelHeaderFooter_ShouldRepeatOnEveryPage(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()
	doc, err := paper.FromHTML(context.Background(), multiPageHeaderHTML(), cfg)

	require.NoError(t, err)
	pdfBytes := doc.GetBytes()
	headerCount := bytes.Count(pdfBytes, []byte("REPEATING-HEADER"))
	footerCount := bytes.Count(pdfBytes, []byte("REPEATING-FOOTER"))
	assert.True(t, headerCount >= 2, "header must repeat on page 2+")
	assert.True(t, footerCount >= 2, "footer must repeat on every page")
	assert.Equal(t, headerCount, footerCount, "header and footer repeat together")
}

func TestFromHTML_WhenHeaderNestedInArticle_ShouldStayInline(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithCompression(false).Build()
	doc, err := paper.FromHTML(context.Background(), "<article><header><p>INLINE-HEADER</p></header><p>body</p></article>", cfg)

	require.NoError(t, err)
	assert.Equal(t, 1, bytes.Count(doc.GetBytes(), []byte("INLINE-HEADER")), "nested header renders inline exactly once")
}

func TestFromHTML_WhenHeaderTallerThanPage_ShouldReturnError(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	sb.WriteString("<header>")
	for range 400 {
		sb.WriteString("<p>very tall header content</p>")
	}
	sb.WriteString("</header><p>body</p>")

	_, err := paper.FromHTML(context.Background(), sb.String())

	require.ErrorIs(t, err, paper.ErrHeaderHeightIsGreaterThanUsefulArea)
}

func TestAddHTML_WhenDocumentAlreadyHasRows_ShouldRejectTopLevelHeader(t *testing.T) {
	t.Parallel()

	m := paper.New()
	m.AddAutoRow(col.New(12).Add(text.New("existing content", props.Text{})))

	err := m.AddHTML(context.Background(), "<header><p>late header</p></header><p>body</p>")

	require.ErrorIs(t, err, paper.ErrHTMLHeaderAfterContent)
}

func TestAddHTML_WhenHeaderAlreadyRegistered_ShouldRejectTopLevelHeader(t *testing.T) {
	t.Parallel()

	m := paper.New()
	require.NoError(t, m.RegisterHeader(row.New(10).Add(col.New(12).Add(text.New("registered", props.Text{})))))

	err := m.AddHTML(context.Background(), "<header><p>late header</p></header><p>body</p>")

	require.ErrorIs(t, err, paper.ErrHTMLHeaderAfterContent)
}

func TestHTMLFromString_WhenTopLevelHeader_ShouldKeepLegacyInlineBehavior(t *testing.T) {
	t.Parallel()

	rows, err := html.FromString(context.Background(), "<header><p>legacy inline</p></header><p>body</p>")

	require.NoError(t, err)
	assert.Len(t, rows, 2, "rows-only API keeps header inline as a normal row")
}

const pageRuleHTML = `<style>@page { size: A5; margin: 10mm }</style><p>content</p>`

func TestFromHTML_WhenPageRuleAndNoConfig_ShouldApplySizeAndMargins(t *testing.T) {
	t.Parallel()

	doc, err := paper.FromHTML(context.Background(), pageRuleHTML)
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
	doc, err := paper.FromHTML(context.Background(), pageRuleHTML, cfg)

	require.NoError(t, err)
	pdf := string(doc.GetBytes())
	assert.True(t, containsMediaBox(pdf, 210, 297), "explicit config must win over @page")
}

func TestFromHTMLReader_WhenPageRule_ShouldApplyLikeFromHTML(t *testing.T) {
	t.Parallel()

	doc, err := paper.FromHTMLReader(context.Background(), strings.NewReader(pageRuleHTML))

	require.NoError(t, err)
	w, h := pagesize.GetDimensions(pagesize.A5)
	assert.True(t, containsMediaBox(string(doc.GetBytes()), w, h), "FromHTMLReader must honor @page")
}

func mustPDFString(t *testing.T, html string) string {
	t.Helper()
	doc, err := paper.FromHTML(context.Background(), html)
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
