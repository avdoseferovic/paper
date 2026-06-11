// Package wasmconvert holds the host-testable HTML→PDF conversion used by the
// browser WebAssembly bindings in ../../cmd/wasm. Keeping the conversion logic
// in a plain (non build-tagged) package lets it be unit tested without a wasm
// runtime, while the syscall/js glue stays a thin wrapper around it.
package wasmconvert

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/avdoseferovic/paper"
)

// ErrEmptyHTML is returned by HTMLToBase64 when the supplied HTML is empty or
// only whitespace.
var ErrEmptyHTML = errors.New("wasmconvert: html is empty")

// errHTMLPanic wraps a recovered panic from the HTML render tree.
var errHTMLPanic = errors.New("wasmconvert: html render panicked")

// HTMLToBase64 renders the given HTML to a PDF and returns it as a base64
// string, suitable for handing back to JavaScript. It returns ErrEmptyHTML when
// html is empty or whitespace-only, propagates any error from generation
// (including context cancellation), and recovers from any panic in the render
// tree so the caller (and the wasm goroutine) always survives.
func HTMLToBase64(ctx context.Context, html string) (string, error) {
	var b64 string
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				b64 = ""
				err = fmt.Errorf("%w: %v", errHTMLPanic, r)
			}
		}()
		b64, err = renderHTML(ctx, html)
	}()

	return b64, err
}

func renderHTML(ctx context.Context, html string) (string, error) {
	if strings.TrimSpace(html) == "" {
		return "", ErrEmptyHTML
	}

	pdf, err := paper.FromHTML(ctx, html)
	if err != nil {
		return "", fmt.Errorf("wasmconvert: generate pdf: %w", err)
	}

	return pdf.GetBase64(), nil
}
