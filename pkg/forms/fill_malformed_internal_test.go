package forms

import (
	"strings"
	"testing"
)

// TestAppendFormPageContent_RejectsUnterminatedArray covers a page dictionary
// whose /Contents array is never closed. pdfscan.SkipArray then stops at the end
// of the content, and slicing the array interior panicked with slice bounds out
// of range while flattening a malformed PDF.
func TestAppendFormPageContent_RejectsUnterminatedArray(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"open bracket at end":  "<</Type /Page /Contents [",
		"reference not closed": "<</Type /Page /Contents [3 0 R",
		"nested not closed":    "<</Type /Page /Contents [[3 0 R]",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := appendFormPageContent([]byte(content), 9)

			if err == nil {
				t.Fatal("appendFormPageContent() error = nil, want a malformed dictionary error")
			}
		})
	}
}

func TestAppendFormPageContent_AppendsToClosedArray(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		content string
		want    string
	}{
		"empty array":    {"<</Type /Page /Contents []>>", "[9 0 R]"},
		"existing entry": {"<</Type /Page /Contents [3 0 R]>>", "[3 0 R 9 0 R]"},
		"single ref":     {"<</Type /Page /Contents 3 0 R>>", "[3 0 R 9 0 R]"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out, err := appendFormPageContent([]byte(tc.content), 9)
			if err != nil {
				t.Fatalf("appendFormPageContent() error = %v", err)
			}
			if !strings.Contains(string(out), tc.want) {
				t.Fatalf("appendFormPageContent() = %q, want it to contain %q", out, tc.want)
			}
		})
	}
}

// TestFormRect_RejectsUnterminatedArray covers the same unterminated-array slice
// in the widget rectangle parser.
func TestFormRect_RejectsUnterminatedArray(t *testing.T) {
	t.Parallel()

	_, ok := formRectForKey([]byte("<</Subtype /Widget /Rect ["), "Rect")

	if ok {
		t.Fatal("formRectForKey() ok = true, want false for an unterminated array")
	}
}
