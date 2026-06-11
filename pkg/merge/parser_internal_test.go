package merge

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
)

type rawObject struct {
	num     int
	content string
}

// assemblePDF builds a classic-xref PDF from sequentially numbered objects so
// tests can target individual parser branches with valid surrounding bytes.
func assemblePDF(objects []rawObject) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make(map[int]int, len(objects))
	for _, o := range objects {
		offsets[o.num] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", o.num, o.content)
	}
	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return buf.Bytes()
}

func minimalObjects() []rawObject {
	return []rawObject{
		{1, "<< /Type /Catalog /Pages 2 0 R >>"},
		{2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>"},
		{3, "<< /Type /Page /Parent 2 0 R >>"},
	}
}

func TestParsePDF_MinimalDocument(t *testing.T) {
	t.Parallel()

	document, err := parsePDF(assemblePDF(minimalObjects()))

	require.NoError(t, err)
	assert.Equal(t, "1.4", document.version)
	assert.Equal(t, 1, document.rootID)
	assert.Equal(t, 2, document.pagesRootID)
	assert.Equal(t, []int{3}, document.pageIDs)
}

func TestParsePDF_Errors(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		data    []byte
		wantErr string
	}{
		"missing header": {
			data:    []byte("not a pdf"),
			wantErr: "missing PDF header",
		},
		"encrypted": {
			data:    []byte("%PDF-1.4\n/Encrypt 5 0 R"),
			wantErr: "encrypted PDFs are not supported",
		},
		"startxref missing": {
			data:    []byte("%PDF-1.4\nno tail"),
			wantErr: "startxref not found",
		},
		"startxref offset missing": {
			data:    []byte("%PDF-1.4\nstartxref"),
			wantErr: "startxref offset not found",
		},
		"startxref offset not a number": {
			data:    []byte("%PDF-1.4\nstartxref\nabc"),
			wantErr: "invalid startxref offset",
		},
		"startxref offset out of bounds": {
			data:    []byte("%PDF-1.4\nstartxref\n9999"),
			wantErr: "startxref offset out of bounds",
		},
		"xref stream instead of table": {
			data:    []byte("%PDF-1.5\n<< /Type /XRef >>\nstartxref\n9\n"),
			wantErr: "xref streams are not supported",
		},
		"xref subsection header malformed": {
			data:    withXref("0 1 extra\n"),
			wantErr: "invalid xref subsection header",
		},
		"xref subsection start not a number": {
			data:    withXref("a 1\n0000000099 00000 n \n"),
			wantErr: "invalid xref subsection start",
		},
		"xref subsection count not a number": {
			data:    withXref("0 b\n0000000099 00000 n \n"),
			wantErr: "invalid xref subsection count",
		},
		"xref entry too short": {
			data:    withXref("0 1\n0000000099\n"),
			wantErr: "invalid xref entry",
		},
		"xref entry offset not a number": {
			data:    withXref("0 1\nxxxxxxxxxx 00000 n \n"),
			wantErr: "invalid xref object offset",
		},
		"xref entry generation not a number": {
			data:    withXref("0 1\n0000000099 yyyyy n \n"),
			wantErr: "invalid xref object generation",
		},
		"xref entry offset out of bounds": {
			data:    withXref("0 1\n0000009999 00000 n \n"),
			wantErr: "xref object offset out of bounds",
		},
		"xref without in-use objects": {
			data:    withXref("0 1\n0000000000 65535 f \ntrailer\n<< /Root 1 0 R >>\n"),
			wantErr: "xref table has no in-use objects",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			document, err := parsePDF(tc.data)

			assert.Nil(t, document)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// withXref builds a document whose xref table body is the given string, with
// startxref pointing at the table. A dummy object provides in-bounds offsets.
func withXref(xrefBody string) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString(strings.Repeat(" ", 90))
	buf.WriteString("1 0 obj\n<< >>\nendobj\n")
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(xrefBody)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)
	return buf.Bytes()
}

func TestParsePDF_XrefSubsectionEndedEarly(t *testing.T) {
	t.Parallel()

	// startxref appears before the table so the data can end mid-subsection.
	prefix := "%PDF-1.4\nstartxref\n"
	offsetText := fmt.Sprintf("%d", 0)
	// Compute the real offset of "xref" after the startxref block.
	for {
		candidate := prefix + offsetText + "\n"
		realOffset := len(candidate)
		if offsetText == fmt.Sprintf("%d", realOffset) {
			data := []byte(candidate + "xref\n0 2\n0000000000 65535 f \n")

			document, err := parsePDF(data)

			assert.Nil(t, document)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "xref subsection ended early")
			return
		}
		offsetText = fmt.Sprintf("%d", len(prefix+offsetText+"\n"))
	}
}

func TestParsePDF_ObjectErrors(t *testing.T) {
	t.Parallel()

	t.Run("object header not found", func(t *testing.T) {
		t.Parallel()

		data := bytes.Replace(assemblePDF(minimalObjects()), []byte("1 0 obj"), []byte("garbage"), 1)

		_, err := parsePDF(data)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "header not found")
	})

	t.Run("object number mismatch", func(t *testing.T) {
		t.Parallel()

		data := bytes.Replace(assemblePDF(minimalObjects()), []byte("1 0 obj"), []byte("9 0 obj"), 1)

		_, err := parsePDF(data)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "points to object")
	})

	t.Run("object end marker not found", func(t *testing.T) {
		t.Parallel()

		// Only the last object's endobj can be cut without shifting offsets.
		data := assemblePDF(minimalObjects())
		idx := bytes.LastIndex(data, []byte("endobj"))
		mutated := append([]byte{}, data...)
		copy(mutated[idx:], "endXXX")

		_, err := parsePDF(mutated)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "end marker not found")
	})

	t.Run("trailer root reference not found", func(t *testing.T) {
		t.Parallel()

		data := bytes.Replace(assemblePDF(minimalObjects()), []byte("/Root 1 0 R"), []byte("/NoRoot 0 0 X"), 1)

		_, err := parsePDF(data)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "trailer root reference not found")
	})

	t.Run("root object missing from xref", func(t *testing.T) {
		t.Parallel()

		data := bytes.Replace(assemblePDF(minimalObjects()), []byte("/Root 1 0 R"), []byte("/Root 9 0 R"), 1)

		_, err := parsePDF(data)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "root object 9 not found")
	})

	t.Run("catalog pages reference not found", func(t *testing.T) {
		t.Parallel()

		objects := minimalObjects()
		objects[0].content = "<< /Type /Catalog >>"

		_, err := parsePDF(assemblePDF(objects))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "catalog pages reference not found")
	})
}

func TestParsePDF_PageTreeErrors(t *testing.T) {
	t.Parallel()

	t.Run("page tree object not found", func(t *testing.T) {
		t.Parallel()

		objects := minimalObjects()
		objects[1].content = "<< /Type /Pages /Kids [9 0 R] /Count 1 >>"

		_, err := parsePDF(assemblePDF(objects))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "page tree object 9 not found")
	})

	t.Run("page tree object is not /Pages", func(t *testing.T) {
		t.Parallel()

		objects := minimalObjects()
		objects[2].content = "<< /Type /Font >>"

		_, err := parsePDF(assemblePDF(objects))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not /Pages")
	})

	t.Run("pages object without kids", func(t *testing.T) {
		t.Parallel()

		objects := minimalObjects()
		objects[1].content = "<< /Type /Pages /Count 0 >>"

		_, err := parsePDF(assemblePDF(objects))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no kids")
	})

	t.Run("page tree cycle", func(t *testing.T) {
		t.Parallel()

		objects := minimalObjects()
		objects[2].content = "<< /Type /Pages /Kids [2 0 R] /Count 1 >>"

		_, err := parsePDF(assemblePDF(objects))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "page tree cycle")
	})
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "1.7", parseVersion([]byte("%PDF-1.7\nrest")))
	assert.Equal(t, "1.3", parseVersion([]byte("no header at all")))
}

func TestTrailerBytes(t *testing.T) {
	t.Parallel()

	t.Run("returns nil without trailer keyword", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, trailerBytes([]byte("xref\n0 0\n"), 0))
	})

	t.Run("returns tail when startxref is missing", func(t *testing.T) {
		t.Parallel()

		got := trailerBytes([]byte("xref\ntrailer<< /Root 1 0 R >>"), 0)

		assert.Equal(t, "<< /Root 1 0 R >>", string(got))
	})
}

func TestMaxPDFVersion(t *testing.T) {
	t.Parallel()

	documents := []*pdfDocument{
		{version: "1.4"},
		{version: "1.7"},
		{version: "1.5"},
	}

	assert.Equal(t, "1.7", maxPDFVersion(documents))
}

func TestParseVersionParts(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		version   string
		wantMajor int
		wantMinor int
	}{
		"valid":           {"1.6", 1, 6},
		"missing dot":     {"16", 1, 3},
		"non-numeric":     {"a.b", 1, 3},
		"non-numeric min": {"1.b", 1, 3},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			major, minor := parseVersionParts(tc.version)

			assert.Equal(t, tc.wantMajor, major)
			assert.Equal(t, tc.wantMinor, minor)
		})
	}
}

func TestCountSource(t *testing.T) {
	t.Parallel()

	tops := []mergedTopItem{
		{source: 0}, {source: 1}, {source: 1}, {source: 2},
	}

	assert.Equal(t, 2, countSource(tops, 1))
	assert.Equal(t, 0, countSource(tops, 9))
}

func TestCollectMergedOutlineTops_DropsSourceOnMissingMapping(t *testing.T) {
	t.Parallel()

	documents := []*pdfDocument{
		{outlineRootID: 10, outlineTopIDs: []int{11, 12}},
		{outlineRootID: 20, outlineTopIDs: []int{21}},
	}
	objectMap := map[objectKey]int{
		{source: 0, number: 11}: 101,
		// {source: 0, number: 12} intentionally missing: source 0 is dropped.
		{source: 1, number: 21}: 201,
	}

	tops := collectMergedOutlineTops(documents, objectMap)

	require.Len(t, tops, 1)
	assert.Equal(t, 201, tops[0].mergedID)
	assert.Equal(t, 1, tops[0].source)
}

func TestBytes_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	merged, err := Bytes(ctx, assemblePDF(minimalObjects()), assemblePDF(minimalObjects()))

	assert.Nil(t, merged)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "merge: canceled")
}

func TestBytes_MergesMinimalDocuments(t *testing.T) {
	t.Parallel()

	merged, err := Bytes(context.Background(), assemblePDF(minimalObjects()), assemblePDF(minimalObjects()))

	require.NoError(t, err)
	document, err := parsePDF(merged)
	require.NoError(t, err)
	assert.Len(t, document.pageIDs, 2)
}
