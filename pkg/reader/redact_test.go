package reader_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/pkg/reader"
)

func TestRedactTextRemovesLiteralTextFromUncompressedContent(t *testing.T) {
	t.Parallel()

	r, err := reader.Parse(generatedReaderPDF(t, false, "John Doe owes 100 EUR"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	redacted, err := reader.RedactText(r, []string{"John Doe"}, nil)
	if err != nil {
		t.Fatalf("RedactText() error = %v", err)
	}
	if bytes.Contains(redacted.Bytes(), []byte("John Doe")) {
		t.Fatal("redacted PDF bytes still contain target text")
	}

	parsed, err := reader.Parse(redacted.Bytes())
	if err != nil {
		t.Fatalf("Parse(redacted) error = %v", err)
	}
	page, err := parsed.Page(0)
	if err != nil {
		t.Fatalf("Page(0) error = %v", err)
	}
	text, err := page.ExtractText()
	if err != nil {
		t.Fatalf("ExtractText() error = %v", err)
	}
	if strings.Contains(text, "John Doe") {
		t.Fatalf("ExtractText() = %q, target should be removed", text)
	}
	if !strings.Contains(text, "owes 100 EUR") {
		t.Fatalf("ExtractText() = %q, non-target text should remain", text)
	}
}

func TestRedactPatternRemovesRegexMatches(t *testing.T) {
	t.Parallel()

	r, err := reader.Parse(generatedReaderPDF(t, false, "SSN 555-12-3456 remains sensitive"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	redacted, err := reader.RedactPattern(r, regexp.MustCompile(`\d{3}-\d{2}-\d{4}`), nil)
	if err != nil {
		t.Fatalf("RedactPattern() error = %v", err)
	}
	if bytes.Contains(redacted.Bytes(), []byte("555-12-3456")) {
		t.Fatal("redacted PDF bytes still contain regex match")
	}
}

func TestRedactTextRejectsCompressedContent(t *testing.T) {
	t.Parallel()

	r, err := reader.Parse(generatedReaderPDF(t, true, "Compressed Secret"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err = reader.RedactText(r, []string{"Secret"}, nil)
	if err == nil {
		t.Fatal("RedactText() expected error for compressed content stream")
	}
	if !strings.Contains(err.Error(), "compressed") {
		t.Fatalf("RedactText() error = %v, want compressed", err)
	}
}

// TestRedactTextStripsMetadata covers RedactOptions.StripMetadata, which used to
// be accepted and silently ignored: a caller asking for a metadata strip got a
// document with /Info intact.
func TestRedactTextStripsMetadata(t *testing.T) {
	t.Parallel()

	source := generatedReaderPDFWithMetadata(t, "John Doe owes 100 EUR", "Secret Author", "Secret Title")
	if !bytes.Contains(source, []byte("Secret Author")) {
		t.Fatal("fixture PDF does not carry the author metadata")
	}
	r, err := reader.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	redacted, err := reader.RedactText(r, []string{"John Doe"}, &reader.RedactOptions{StripMetadata: true})
	if err != nil {
		t.Fatalf("RedactText() error = %v", err)
	}

	out := redacted.Bytes()
	for _, secret := range []string{"John Doe", "Secret Author", "Secret Title"} {
		if bytes.Contains(out, []byte(secret)) {
			t.Fatalf("redacted PDF still contains %q", secret)
		}
	}
	if len(out) != len(source) {
		t.Fatalf("redaction changed length: got %d, want %d (redaction must preserve xref offsets)", len(out), len(source))
	}
	parsed, err := reader.Parse(out)
	if err != nil {
		t.Fatalf("Parse(redacted) error = %v", err)
	}
	if parsed.PageCount() != r.PageCount() {
		t.Fatalf("PageCount() = %d, want %d", parsed.PageCount(), r.PageCount())
	}
}

func TestRedactTextWithoutStripMetadataKeepsMetadata(t *testing.T) {
	t.Parallel()

	source := generatedReaderPDFWithMetadata(t, "John Doe owes 100 EUR", "Kept Author", "Kept Title")
	r, err := reader.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	redacted, err := reader.RedactText(r, []string{"John Doe"}, nil)
	if err != nil {
		t.Fatalf("RedactText() error = %v", err)
	}

	if !bytes.Contains(redacted.Bytes(), []byte("Kept Author")) {
		t.Fatal("metadata was stripped without StripMetadata being requested")
	}
}
