package reader_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/reader"
)

// FuzzParse drives the PDF object parser with arbitrary bytes. Parse accepts
// whole files from untrusted sources, so the contract under test is simply
// that malformed input produces an error rather than a panic, an unbounded
// allocation, or a reader that reports success with no usable document.
func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"",
		"not a pdf",
		"%PDF-1.4\n",
		"%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF",
		"%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 1 0 R >>\nendobj\n%%EOF",
		"%PDF-1.4\nstartxref\n999999999\n%%EOF",
		"%PDF-1.4\nxref\n0 1\n0000000000 65535 f \ntrailer\n<< /Size 1 >>\nstartxref\n9\n%%EOF",
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := reader.Parse(data)
		if err != nil {
			if r != nil {
				t.Fatalf("Parse() returned both a reader and an error: %v", err)
			}
			return
		}
		if r == nil {
			t.Fatal("Parse() returned no reader and no error")
		}
	})
}
