package paper_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
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
	doc, err := paper.FromHTML(multiPageHeaderHTML(), cfg)

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
	doc, err := paper.FromHTML("<article><header><p>INLINE-HEADER</p></header><p>body</p></article>", cfg)

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

	_, err := paper.FromHTML(sb.String())

	require.ErrorIs(t, err, paper.ErrHeaderHeightIsGreaterThanUsefulArea)
}

func TestAddHTML_WhenDocumentAlreadyHasRows_ShouldRejectTopLevelHeader(t *testing.T) {
	t.Parallel()

	m := paper.New()
	m.AddAutoRow(col.New(12).Add(text.New("existing content", props.Text{})))

	err := m.AddHTML("<header><p>late header</p></header><p>body</p>")

	require.ErrorIs(t, err, paper.ErrHTMLHeaderAfterContent)
}

func TestHTMLFromString_WhenTopLevelHeader_ShouldKeepLegacyInlineBehavior(t *testing.T) {
	t.Parallel()

	rows, err := html.FromString("<header><p>legacy inline</p></header><p>body</p>")

	require.NoError(t, err)
	assert.Len(t, rows, 2, "rows-only API keeps header inline as a normal row")
}
