//go:build js && wasm

// Command wasm exposes Paper's HTML→PDF conversion to JavaScript running in a
// web browser via WebAssembly. It registers a single global function,
// paperGeneratePDF(html), and then blocks so that function stays callable for
// the lifetime of the page.
//
// Build it with:
//
//	GOOS=js GOARCH=wasm go build -o web/paper.wasm .
//
// See README.md for the full build/serve instructions.
package main

import (
	"context"
	"syscall/js"

	"github.com/avdoseferovic/paper/examples/internal/wasmconvert"
)

// generate is the JS-facing entry point registered as globalThis.paperGeneratePDF.
// It accepts a single HTML string argument and returns a JS object:
//
//	{ pdf: "<base64>" }   on success
//	{ error: "<message>" } on invalid input or generation failure
func generate(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return js.ValueOf(map[string]any{
			"error": "paperGeneratePDF(html) requires a single string argument",
		})
	}

	b64, err := wasmconvert.HTMLToBase64(context.Background(), args[0].String())
	if err != nil {
		return js.ValueOf(map[string]any{"error": err.Error()})
	}

	return js.ValueOf(map[string]any{"pdf": b64})
}

func main() {
	js.Global().Set("paperGeneratePDF", js.FuncOf(generate))
	// Block forever so the registered callback stays alive for the page.
	select {}
}
