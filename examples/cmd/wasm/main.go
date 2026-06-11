//go:build js && wasm

// Command wasm exposes Paper's PDF generation to JavaScript running in a web
// browser via WebAssembly. It registers two global functions —
// paperGeneratePDF(html) and paperGenerateFromSpec(json, pageSize) — and then
// blocks so they stay callable for the lifetime of the page.
//
// Build it with:
//
//	GOOS=js GOARCH=wasm go build -o web/paper.wasm .
//
// See README.md for the full build/serve instructions.
package main

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/avdoseferovic/paper/examples/internal/wasmconvert"
)

// safeResult runs fn and packages the outcome as a JS object: { pdf: "<base64>" }
// on success or { error: "<message>" } on failure. A deferred recover() ensures
// a panic deep in the render tree becomes an error instead of unwinding past the
// js.FuncOf boundary — which would abort the program and permanently disable
// every registered callback for the page.
func safeResult(fn func() (string, error)) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = js.ValueOf(map[string]any{"error": fmt.Sprintf("paper: %v", r)})
		}
	}()

	b64, err := fn()
	if err != nil {
		return js.ValueOf(map[string]any{"error": err.Error()})
	}
	return js.ValueOf(map[string]any{"pdf": b64})
}

// generate is registered as globalThis.paperGeneratePDF(html).
func generate(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return js.ValueOf(map[string]any{
			"error": "paperGeneratePDF(html) requires a single string argument",
		})
	}
	html := args[0].String()
	return safeResult(func() (string, error) {
		return wasmconvert.HTMLToBase64(context.Background(), html)
	})
}

// generateFromSpec is registered as globalThis.paperGenerateFromSpec(json, pageSize).
// pageSize is optional and defaults to "A4".
func generateFromSpec(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return js.ValueOf(map[string]any{
			"error": "paperGenerateFromSpec(json, pageSize) requires a JSON string argument",
		})
	}
	spec := args[0].String()
	pageSize := "A4"
	if len(args) > 1 && args[1].Type() == js.TypeString {
		pageSize = args[1].String()
	}
	return safeResult(func() (string, error) {
		return wasmconvert.SpecToBase64(context.Background(), spec, pageSize)
	})
}

func main() {
	js.Global().Set("paperGeneratePDF", js.FuncOf(generate))
	js.Global().Set("paperGenerateFromSpec", js.FuncOf(generateFromSpec))
	// Block forever so the registered callbacks stay alive for the page.
	select {}
}
