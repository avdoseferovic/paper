package wasmconvert_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/examples/internal/wasmconvert"
)

func decode(t *testing.T, b64 string) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("result is not valid base64: %v", err)
	}
	return raw
}

func TestHTMLToBase64_ValidHTML_ReturnsPDFDocument(t *testing.T) {
	b64, err := wasmconvert.HTMLToBase64(context.Background(), "<h1>Hello</h1><p>World</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b64 == "" {
		t.Fatal("expected non-empty base64 output")
	}

	raw := decode(t, b64)
	if !strings.HasPrefix(string(raw), "%PDF-") {
		got := raw
		if len(got) > 8 {
			got = got[:8]
		}
		t.Fatalf("decoded output is not a PDF document, prefix=%q", got)
	}
}

func TestHTMLToBase64_EmptyHTML_ReturnsErrEmptyHTML(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t  "} {
		_, err := wasmconvert.HTMLToBase64(context.Background(), in)
		if !errors.Is(err, wasmconvert.ErrEmptyHTML) {
			t.Fatalf("input %q: expected ErrEmptyHTML, got %v", in, err)
		}
	}
}

func TestHTMLToBase64_CanceledContext_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wasmconvert.HTMLToBase64(ctx, "<p>x</p>")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled in error chain, got %v", err)
	}
}

func TestHTMLToBase64_FilePathImage_DoesNotPanic(t *testing.T) {
	// The default image resolver refuses file-path src values (no filesystem
	// access), so an <img> pointing at a file must be handled gracefully:
	// it must never panic. It may either succeed (image dropped / alt text)
	// or return a descriptive error.
	b64, err := wasmconvert.HTMLToBase64(context.Background(), `<img src="logo.png"><p>hi</p>`)
	if err != nil {
		return // a descriptive error is acceptable
	}
	if b64 == "" {
		t.Fatal("expected non-empty base64 output when no error returned")
	}
	if !strings.HasPrefix(string(decode(t, b64)), "%PDF-") {
		t.Fatal("decoded output is not a PDF document")
	}
}
