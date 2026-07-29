package reader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/reader"
)

func writeTestPDF(t *testing.T, dir, name, content string) string {
	t.Helper()
	doc, err := paper.FromHTML(t.Context(), "<p>"+content+"</p>")
	require.NoError(t, err)
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, doc.GetBytes(), 0o600))
	return path
}

func TestMergeFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first := writeTestPDF(t, dir, "a.pdf", "first document")
	second := writeTestPDF(t, dir, "b.pdf", "second document")

	merged, err := reader.MergeFiles(first, second)
	require.NoError(t, err)

	r, err := reader.Parse(merged.Bytes())
	require.NoError(t, err)
	assert.Equal(t, 2, r.PageCount())

	_, err = reader.MergeFiles(filepath.Join(dir, "missing.pdf"))
	assert.Error(t, err)
}
