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

// safeResult runs fn and packages the outcome as a JS object: { <key>: value }
// on success or { error: "<message>" } on failure. A deferred recover() ensures
// a panic deep in the render tree becomes an error instead of unwinding past the
// js.FuncOf boundary — which would abort the program and permanently disable
// every registered callback for the page.
//
// The success key is a parameter because the exports return different shapes:
// the generators answer with a base64 PDF, paperExampleAssets with a list.
func safeResult(key string, fn func() (any, error)) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = js.ValueOf(map[string]any{"error": fmt.Sprintf("paper: %v", r)})
		}
	}()

	value, err := fn()
	if err != nil {
		return js.ValueOf(map[string]any{"error": err.Error()})
	}
	return js.ValueOf(map[string]any{key: value})
}

// generate is registered as globalThis.paperGeneratePDF(html).
func generate(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return js.ValueOf(map[string]any{
			"error": "paperGeneratePDF(html) requires a single string argument",
		})
	}
	html := args[0].String()
	return safeResult("pdf", func() (any, error) {
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
	return safeResult("pdf", func() (any, error) {
		return wasmconvert.SpecToBase64(context.Background(), spec, pageSize)
	})
}

// generateExample is registered as globalThis.paperGenerateExample(name). It
// renders one of the documented examples from docs/assets/examples using the
// same GetPaper builder the docs page displays.
//
// Any files the example reads must already be registered with paperFS; call
// paperExampleAssets(name) first to learn which.
func generateExample(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return js.ValueOf(map[string]any{
			"error": "paperGenerateExample(name) requires a single string argument",
		})
	}
	name := args[0].String()
	return safeResult("pdf", func() (any, error) {
		return wasmconvert.ExampleToBase64(context.Background(), name)
	})
}

// exampleAssets is registered as globalThis.paperExampleAssets(name). It returns
// { assets: [...] }: the repo-relative paths the example reads, which the caller
// fetches and hands to paperFS before generating.
func exampleAssets(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return js.ValueOf(map[string]any{
			"error": "paperExampleAssets(name) requires a single string argument",
		})
	}
	name := args[0].String()
	return safeResult("assets", func() (any, error) {
		assets, err := wasmconvert.ExampleAssets(name)
		if err != nil {
			return nil, err
		}
		// js.ValueOf understands []any, not []string.
		out := make([]any, len(assets))
		for i, a := range assets {
			out[i] = a
		}
		return out, nil
	})
}

func main() {
	js.Global().Set("paperGeneratePDF", js.FuncOf(generate))
	js.Global().Set("paperGenerateFromSpec", js.FuncOf(generateFromSpec))
	js.Global().Set("paperGenerateExample", js.FuncOf(generateExample))
	js.Global().Set("paperExampleAssets", js.FuncOf(exampleAssets))
	// Block forever so the registered callbacks stay alive for the page.
	select {}
}
