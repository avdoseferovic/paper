# Paper Playground (WebAssembly)

A live, in-browser playground that generates **real PDFs** with Paper compiled to
WebAssembly — entirely client-side, no server. A CodeMirror editor on the left
(HTML mode or Component-grid JSON mode, with example presets and an A4/Letter
selector) drives a preview on the right that re-renders the actual generated PDF
as you type.

The Go code (`main.go`, built with `//go:build js && wasm`) registers two globals:

```js
paperGeneratePDF(html)                // HTML → PDF
paperGenerateFromSpec(json, pageSize) // component-grid JSON → PDF
// each → { pdf: "<base64>" } | { error: "<message>" }
```

See [`docs/wasm-support.md`](../../../docs/wasm-support.md) for the JSON spec
schema.

## Build

```bash
./build.sh
```

This compiles `web/paper.wasm` and copies the matching `wasm_exec.js` from your
active Go toolchain into `web/`. Both files are git-ignored (they are generated
and toolchain-version specific). You can also run `make wasm` from the repo root.

## Serve

`WebAssembly` must be loaded over HTTP (not `file://`):

```bash
cd web
python3 -m http.server 8080
```

Then open <http://localhost:8080/>, edit the source (HTML or component JSON), and
watch the live PDF preview; click **Generate PDF** (or ⌘/Ctrl+Enter) to download.

## Browser limitations

There is no filesystem in the browser, so:

- File-path `<img src="…">` / `<link href="…">` are refused by the default
  resolver — use `data:` URIs for inline assets.
- Custom file-path fonts are unavailable — use `AddUTF8FontFromBytes`.
- `Pdf.Save` is unsupported — use `GetBase64` / `GetBytes`.

See [`docs/wasm-support.md`](../../../docs/wasm-support.md) for details.
