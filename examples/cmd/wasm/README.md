# Paper in the browser (WebAssembly)

This demo runs Paper's HTML→PDF conversion entirely client-side in the browser
via WebAssembly. The Go code (`main.go`, built with `//go:build js && wasm`)
registers a single global function:

```js
paperGeneratePDF(html) // → { pdf: "<base64>" } | { error: "<message>" }
```

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

Then open <http://localhost:8080/>, edit the HTML, and click **Generate PDF**.

## Browser limitations

There is no filesystem in the browser, so:

- File-path `<img src="…">` / `<link href="…">` are refused by the default
  resolver — use `data:` URIs for inline assets.
- Custom file-path fonts are unavailable — use `AddUTF8FontFromBytes`.
- `Pdf.Save` is unsupported — use `GetBase64` / `GetBytes`.

See [`docs/wasm-support.md`](../../../docs/wasm-support.md) for details.
