package html_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html"
)

func TestDefaultLimits_ReturnsSafeNonZeroValues(t *testing.T) {
	t.Parallel()
	l := html.DefaultLimits()
	assert.Greater(t, l.MaxImagePixels, int64(0))
	assert.Greater(t, l.MaxImageBytes, int64(0))
	assert.Greater(t, l.MaxDOMDepth, 0)
	assert.Greater(t, l.MaxDOMNodes, 0)
	assert.Greater(t, l.MaxSVGPixels, int64(0))
	assert.Greater(t, l.MaxStyleRules, 0)
}

func TestWithUnsafeNoLimits_AllowsInputBeyondDefaultCaps(t *testing.T) {
	t.Parallel()

	// The same input is rejected with a small explicit limit
	// (see TestFromString_RejectsLargeDOM) but accepted without limits.
	var b strings.Builder
	b.WriteString("<html><body>")
	for range 20 {
		b.WriteString("<span>x</span>")
	}
	b.WriteString("</body></html>")

	rows, err := html.FromString(context.Background(), b.String(), html.WithUnsafeNoLimits())

	require.NoError(t, err)
	assert.NotEmpty(t, rows)
}

func TestWithOutlineFromHeadings_TranslatesAllHeadingLevels(t *testing.T) {
	t.Parallel()

	input := "<h1>a</h1><h2>b</h2><h3>c</h3><h4>d</h4><h5>e</h5><h6>f</h6><p>body</p>"
	rows, err := html.FromString(context.Background(), input, html.WithOutlineFromHeadings())

	require.NoError(t, err)
	assert.Len(t, rows, 7)
}

func TestWithImageBaseDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pngBytes := pngHeaderWithDimensions(2, 2)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pic.png"), pngBytes, 0o600))

	t.Run("loads image inside base dir", func(t *testing.T) {
		t.Parallel()
		rows, err := html.FromString(context.Background(),
			`<html><body><img src="pic.png" alt="fallback"></body></html>`,
			html.WithImageBaseDir(dir),
		)
		require.NoError(t, err)
		assert.NotEmpty(t, rows)
	})

	t.Run("escaping path falls back to alt text", func(t *testing.T) {
		t.Parallel()
		var unsupported []string
		rows, err := html.FromString(context.Background(),
			`<html><body><img src="../pic.png" alt="fallback"></body></html>`,
			html.WithImageBaseDir(dir),
			html.WithUnsupportedHandler(func(thing, _ string) {
				unsupported = append(unsupported, thing)
			}),
		)
		require.NoError(t, err)
		assert.NotEmpty(t, rows, "alt text row should remain")
		assert.Contains(t, unsupported, "img.src")
	})
}

func TestWithStylesheetBaseDir_LoadsLinkedCSS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "style.css"), []byte("p{color:#ff0000}"), 0o600))

	rows, err := html.FromString(context.Background(),
		`<html><head><link rel="stylesheet" href="style.css"></head><body><p>x</p></body></html>`,
		html.WithStylesheetBaseDir(dir),
	)

	require.NoError(t, err)
	assert.NotEmpty(t, rows)
}

func TestDocumentFromString_EmptyInputReturnsEmptyDocument(t *testing.T) {
	t.Parallel()
	doc, err := html.DocumentFromString(context.Background(), "")
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Empty(t, doc.Rows)
	assert.Nil(t, doc.Page)
}

func TestDocumentFromString_ReturnsContextError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	doc, err := html.DocumentFromString(ctx, "<p>hi</p>")

	assert.Nil(t, doc)
	require.ErrorIs(t, err, context.Canceled)
}

func TestDocumentFromReader_ReturnsContextError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	doc, err := html.DocumentFromReader(ctx, strings.NewReader("<p>hi</p>"))

	assert.Nil(t, doc)
	require.ErrorIs(t, err, context.Canceled)
}

// failingReader always errors to exercise the io.ReadAll failure path.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failure")
}

func TestFromReader_ReadErrorIsWrapped(t *testing.T) {
	t.Parallel()
	rows, err := html.FromReader(context.Background(), failingReader{})
	assert.Nil(t, rows)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading input")
}

func TestDocumentFromReader_ReadErrorIsWrapped(t *testing.T) {
	t.Parallel()
	doc, err := html.DocumentFromReader(context.Background(), failingReader{})
	assert.Nil(t, doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading input")
}
