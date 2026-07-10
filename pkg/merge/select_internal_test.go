package merge

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
)

func a4Objects(label string) []rawObject {
	return []rawObject{
		{1, "<< /Type /Catalog /Pages 2 0 R >>"},
		{2, "<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 595.28 841.89] /Rotate 90 >>"},
		{3, "<< /Type /Page /Parent 2 0 R /Contents 4 0 R >>"},
		{4, "<< /Length " + lenStr(label) + " >>\nstream\n" + label + "\nendstream"},
	}
}

func lenStr(s string) string {
	return itoa(len(s) + 1)
}

func itoa(n int) string {
	digits := "0123456789"
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{digits[n%10]}, out...)
		n /= 10
	}
	return string(out)
}

func twoPageObjects() []rawObject {
	return []rawObject{
		{1, "<< /Type /Catalog /Pages 2 0 R >>"},
		{2, "<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 /MediaBox [0 0 595.28 841.89] >>"},
		{3, "<< /Type /Page /Parent 2 0 R >>"},
		{4, "<< /Type /Page /Parent 2 0 R >>"},
	}
}

func mergedPageContents(t *testing.T, merged []byte) []string {
	t.Helper()
	re := regexp.MustCompile(`(?s)<<[^<>]*?/Type /Page\b.*?>>`)
	var pages []string
	for _, obj := range strings.Split(string(merged), "endobj") {
		if idx := strings.Index(obj, "/Type /Page"); idx >= 0 && !strings.Contains(obj, "/Type /Pages") {
			match := re.FindString(obj)
			if match == "" {
				match = obj
			}
			pages = append(pages, match)
		}
	}
	return pages
}

func TestBytes_PreservesInheritedPageAttributes(t *testing.T) {
	t.Parallel()

	merged, err := Bytes(context.Background(), assemblePDF(a4Objects("A")), assemblePDF(a4Objects("B")))

	require.NoError(t, err)
	pages := mergedPageContents(t, merged)
	require.Equal(t, 2, len(pages))
	for _, page := range pages {
		assert.Contains(t, page, "/MediaBox [0 0 595.28 841.89]")
		assert.Contains(t, page, "/Rotate 90")
	}
}

func TestBytes_KeepsPageOwnMediaBox(t *testing.T) {
	t.Parallel()

	objects := []rawObject{
		{1, "<< /Type /Catalog /Pages 2 0 R >>"},
		{2, "<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 595.28 841.89] >>"},
		{3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"},
	}

	merged, err := Bytes(context.Background(), assemblePDF(objects))

	require.NoError(t, err)
	pages := mergedPageContents(t, merged)
	require.Equal(t, 1, len(pages))
	assert.Contains(t, pages[0], "/MediaBox [0 0 612 792]")
	assert.NotContains(t, pages[0], "595.28")
}

func TestBytes_InheritedResourcesReferenceIsRewritten(t *testing.T) {
	t.Parallel()

	objects := []rawObject{
		{1, "<< /Type /Catalog /Pages 2 0 R >>"},
		{2, "<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 100 100] /Resources 4 0 R >>"},
		{3, "<< /Type /Page /Parent 2 0 R >>"},
		{4, "<< /ProcSet [/PDF] >>"},
	}

	merged, err := Bytes(context.Background(), assemblePDF(objects))

	require.NoError(t, err)
	pages := mergedPageContents(t, merged)
	require.Equal(t, 1, len(pages))
	// Object 4 is the first copied non-page object; whatever its new number,
	// the injected /Resources must point at an object that exists.
	m := regexp.MustCompile(`/Resources (\d+) 0 R`).FindStringSubmatch(pages[0])
	require.Equal(t, 2, len(m))
	assert.Contains(t, string(merged), m[1]+" 0 obj\n<< /ProcSet [/PDF] >>")
}

func TestBytesSelected_SubsetAndOrder(t *testing.T) {
	t.Parallel()

	source := assemblePDF(twoPageObjects())

	reversed, err := BytesSelected(context.Background(), PageSelection{PDF: source, Pages: []int{1, 0}})
	require.NoError(t, err)
	document, err := parsePDF(reversed)
	require.NoError(t, err)
	assert.Equal(t, 2, len(document.pageIDs))

	subset, err := BytesSelected(context.Background(), PageSelection{PDF: source, Pages: []int{1}})
	require.NoError(t, err)
	document, err = parsePDF(subset)
	require.NoError(t, err)
	require.Equal(t, 1, len(document.pageIDs))
	selected := string(document.objects[document.pageIDs[0]].content)
	assert.Contains(t, selected, "/MediaBox [0 0 595.28 841.89]")
}

func TestBytesSelected_Validation(t *testing.T) {
	t.Parallel()

	source := assemblePDF(twoPageObjects())

	_, err := BytesSelected(context.Background(), PageSelection{PDF: source, Pages: []int{2}})
	assert.ErrorIs(t, err, ErrCannotMergePDFs)

	_, err = BytesSelected(context.Background(), PageSelection{PDF: source, Pages: []int{-1}})
	assert.ErrorIs(t, err, ErrCannotMergePDFs)

	_, err = BytesSelected(context.Background(), PageSelection{PDF: source, Pages: nil})
	assert.ErrorIs(t, err, ErrCannotMergePDFs)

	_, err = BytesSelected(context.Background())
	assert.ErrorIs(t, err, ErrCannotMergePDFs)
}

func TestBytesSelected_MultipleSelections(t *testing.T) {
	t.Parallel()

	a := assemblePDF(a4Objects("A"))
	b := assemblePDF(twoPageObjects())

	merged, err := BytesSelected(context.Background(),
		PageSelection{PDF: a, Pages: []int{0}},
		PageSelection{PDF: b, Pages: []int{1}},
	)

	require.NoError(t, err)
	document, err := parsePDF(merged)
	require.NoError(t, err)
	assert.Equal(t, 2, len(document.pageIDs))
}
