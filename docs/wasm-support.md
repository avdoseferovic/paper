# WebAssembly (Browser) Support

Paper is pure Go and compiles for `GOOS=js GOARCH=wasm`, so its HTML→PDF
conversion can run **entirely in the browser** — no server round-trip, no
upload. With the default configuration, generation touches no filesystem: the
standard fonts and CMaps are embedded in the binary, and PDF output is returned
as bytes/base64 rather than written to disk.

The **Paper Playground** — a live, in-browser editor with an HTML mode and a
component-grid mode — and the `syscall/js` bindings live in
[`examples/cmd/wasm`](https://github.com/avdoseferovic/paper/tree/main/examples/cmd/wasm).

## How it works

The wasm entry point (`examples/cmd/wasm/main.go`, built behind a
`//go:build js && wasm` tag) registers two global functions and then blocks to
keep them alive:

```js
// HTML → PDF (thin wrapper around paper.FromHTML)
paperGeneratePDF(html)               // → { pdf: "<base64>" } | { error: "<message>" }

// Component grid → PDF (builds a real Paper component tree from a JSON spec)
paperGenerateFromSpec(json, pageSize) // → { pdf: "<base64>" } | { error: "<message>" }
```

- On success each returns `{ pdf }`, a base64-encoded PDF.
- On invalid input, a generation failure, or a recovered render panic, it returns
  `{ error }` with a message — the wasm goroutine always survives.

The conversion logic lives in `examples/internal/wasmconvert`:
`HTMLToBase64` wraps `paper.FromHTML`; `SpecToBase64` parses the JSON layout and
builds a real Paper document (`paper.New` + rows/cols + components).

## Component-grid JSON spec

`paperGenerateFromSpec` accepts a layout document and `pageSize` (`"A4"` or
`"Letter"`, default `A4`):

```jsonc
{
  "rows": [
    { "cols": [
      { "span": 8, "type": "text", "style": "h1", "value": "Invoice" },
      { "span": 4, "type": "text", "style": "label", "align": "right", "value": "NO. INV-0481" }
    ]},
    { "cols": [ { "span": 12, "type": "line" } ] },
    { "cols": [ { "span": 12, "type": "table",
      "head": ["Description", "Qty", "Amount"], "colAlign": ["", "c", "r"],
      "rows": [ ["Team plan", "1", "$480.00"] ] } ]},
    { "cols": [
      { "span": 4, "type": "qrcode", "value": "pay.example.dev/inv-0481" },
      { "span": 4, "type": "barcode", "value": "INV0481", "label": "INV 0481" },
      { "span": 4, "type": "signature", "value": "A. S.", "label": "AUTHORIZED" }
    ]}
  ]
}
```

- **Rows** map to Paper rows; **cols** carry a `span` (1–12). Each col is one
  component.
- **Component `type`s:** `text` (with `style`: `h1`/`h2`/`num`/`label`/`body`),
  `line` (`soft` for a lighter rule), `table` (`head` + `rows` + per-column
  `colAlign` of `""`/`c`/`r`), `qrcode`, `barcode` (optional `label` caption),
  `signature` (optional `label`), `checkbox` (`checked`), `pagenumber`, `footer`.
- `align` (`left`/`center`/`right`) applies to text. Unknown/`image` types render
  as a muted placeholder. Inline HTML in values is reduced to plain text
  (`<br>` → newline; other tags stripped) — Paper text is plain-text only, so
  emphasis like `<strong>` is not preserved in component values.

## Build and serve

```bash
# From the repo root:
make wasm
# …or directly:
cd examples/cmd/wasm && ./build.sh
```

`build.sh` compiles `web/paper.wasm` and copies the matching `wasm_exec.js` from
your active Go toolchain (`$(go env GOROOT)/lib/wasm/wasm_exec.js`). Both files
are generated and git-ignored.

WebAssembly must be loaded over HTTP (not `file://`):

```bash
cd examples/cmd/wasm/web
python3 -m http.server 8080
```

Open <http://localhost:8080/>. Toggle between **HTML** and **Component grid**
modes, pick an example preset, and watch the preview — which shows the **real
generated PDF** (in the browser's native PDF viewer), re-rendered live as you
type (debounced). Click **Generate PDF** (or ⌘/Ctrl+Enter) to download it. The
demo uses non-streaming `WebAssembly.instantiate`, so it works on any static
file server regardless of the `application/wasm` MIME type.

## Loading the module in your own page

```html
<script src="wasm_exec.js"></script>
<script>
  const go = new Go();
  const buf = await (await fetch("paper.wasm")).arrayBuffer();
  const { instance } = await WebAssembly.instantiate(buf, go.importObject);
  // Do NOT await go.run(): Go registers paperGeneratePDF synchronously before
  // its select{} blocks; awaiting would yield before registration.
  go.run(instance);

  const res = paperGeneratePDF("<h1>Hi</h1>");
  if (res.error) throw new Error(res.error);
  // res.pdf is a base64 string — turn it into a Blob, download, etc.
</script>
```

## Browser limitations

There is no filesystem in the browser, so anything that would read or write
local files is unavailable:

| Feature | In the browser |
| ------- | -------------- |
| File-path `<img src="logo.png">` | Refused by the default resolver (`ErrImageResolverRefused`); the `<img>` falls back to its alt text. Use a `data:` URI instead. |
| File-path `<link href="style.css">` | Refused by the default stylesheet resolver. Use a `data:` URI or inline `<style>`. |
| Custom fonts from a file path (`AddUTF8Font(..., file)`) | Unavailable. Use `AddUTF8FontFromBytes` with the font bytes. |
| `Pdf.Save(file)` | Unsupported (no filesystem). Use `GetBase64` / `GetBytes` and hand the result to JavaScript. |

The bundled default fonts and inline (`data:`) assets work normally.

## Notes

- The wasm build is guarded in CI (`GOOS=js GOARCH=wasm go build ./...` for both
  the root and `examples` modules), so changes that break wasm compatibility
  fail the build.
- Generation runs synchronously on the page's main thread. For large documents,
  consider offloading to a Web Worker (not done in the demo).
