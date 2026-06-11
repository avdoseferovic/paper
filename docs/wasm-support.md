# WebAssembly (Browser) Support

Paper is pure Go and compiles for `GOOS=js GOARCH=wasm`, so its HTML→PDF
conversion can run **entirely in the browser** — no server round-trip, no
upload. With the default configuration, generation touches no filesystem: the
standard fonts and CMaps are embedded in the binary, and PDF output is returned
as bytes/base64 rather than written to disk.

A runnable demo and the `syscall/js` bindings live in
[`examples/cmd/wasm`](https://github.com/avdoseferovic/paper/tree/main/examples/cmd/wasm).

## How it works

The wasm entry point (`examples/cmd/wasm/main.go`, built behind a
`//go:build js && wasm` tag) registers a single global function and then blocks
to keep it alive:

```js
paperGeneratePDF(html) // → { pdf: "<base64>" } | { error: "<message>" }
```

- On success it returns `{ pdf }`, a base64-encoded PDF.
- On invalid input or a generation failure it returns `{ error }` with a message.

The conversion itself is a thin wrapper around `paper.FromHTML` followed by
`Pdf.GetBase64`; the testable logic lives in
`examples/internal/wasmconvert`.

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

Open <http://localhost:8080/>, edit the HTML, and click **Generate PDF**. The
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
