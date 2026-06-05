# HTML to PDF Support

Paper can convert a documented subset of HTML/CSS directly into PDF documents. No browser, no Node, no external binary — pure Go.

## Quick start

```go
import paper "github.com/johnfercher/paper/v2"

doc, err := paper.FromHTML(`<h1>Hello</h1><p>World</p>`)
if err != nil {
    log.Fatal(err)
}
_ = doc.Save("out.pdf")
```

Use `paper.New()` when you need to mix HTML with headers, footers, manual rows, or other Paper components:

```go
m := paper.New()
if err := m.AddHTML(`<h1>Hello</h1><p>World</p>`); err != nil {
    log.Fatal(err)
}
doc, _ := m.Generate()
```

Or build rows directly for advanced resolver options:

```go
import "github.com/johnfercher/paper/v2/pkg/html"

rows, err := html.FromString(htmlString)
// rows is []core.Row — add to any Paper document
```

## Supported HTML tags

**Block:** `html, head, body, div, p, h1, h2, h3, h4, h5, h6, hr, pre, blockquote, header, footer, section, article, aside, main, nav, figure, figcaption`

**Inline:** `span, a, strong, b, em, i, u, s, strike, sub, sup, br, mark, small, abbr, code, kbd, samp, var, cite, q, time`

**Tables:** `table, thead, tbody, tfoot, tr, th, td, caption` — with `colspan` and `rowspan` attributes. `colgroup`/`col` are recognised; explicit column widths are not honoured in v1 (logged via `unsupportedHandler`).

**Lists:** `ul, ol, li, dl, dt, dd` — `ol` supports `type="a|A|i|I"` for alpha/roman markers. Nested lists supported. `<dl>`/`<dt>`/`<dd>` render as definition lists with bold term and indented definition.

**Disclosure:** `details, summary` — always rendered expanded with a bold summary above the body (PDFs have no toggle).

**Anchors:** `id="…"` on any element registers a PDF named destination; `<a href="#id">` produces an internal PDF link that jumps to it. Forward references (link before target) are supported via a pre-pass.

**Images:** `img` — block-level `<img src="…" width="…" height="…" alt="…">` renders PNG, JPG, and SVG (rasterised via oksvg+rasterx). Inline `<img>` inside paragraphs still renders as alt text. See [Images](#images) below.

## Supported CSS properties

**Text:** `color, font-family, font-size, font-weight, font-style, text-align, text-decoration, line-height, letter-spacing, text-transform, text-indent, white-space`

`text-align` supports `left`, `center`, and `right` for RichText paragraphs. `text-indent` indents only the first rendered line of a paragraph. `white-space` supports `normal`, `nowrap`, `pre`, `pre-wrap`, and `pre-line`; these modes control whitespace collapsing, explicit line breaks, and automatic wrapping in RichText paragraphs.

**Opacity:** `opacity` (0–1 or 0%–100%), multiplies into every descendant colour's alpha during the cascade.

**Box model:** `padding, padding-{top,right,bottom,left}, margin, margin-{top,right,bottom,left}`

**Borders:** `border, border-{top,right,bottom,left}, border-color, border-width, border-style`

**Border-radius:** `border-radius` (1–4 values, CSS spec), `border-{top-left,top-right,bottom-left,bottom-right}-radius`. When combined with non-uniform per-side border widths, the renderer uses a single averaged stroke thickness (v1 limitation).

**Background:** `background-color`, `background-image: linear-gradient(...)` / `radial-gradient(...)` (rasterised to PNG and embedded; up to 5 colour stops, angle keywords + `Ndeg`, positions, named radial centres).

**Effects:** `box-shadow` (1–4 shadows, comma-separated; `<x> <y> [blur] [spread] [color] [inset]`; blur approximated by 3 overlaid translucent rects), `text-shadow` (per-run, first shadow only), `outline` + `outline-{width,style,color,offset}` (drawn outside the cell box, does not affect layout).

**Layout:** `display: block|inline|inline-block|none|flex|inline-flex`, `width`, `height`

**Flex:** `flex-direction` (incl. `row-reverse`/`column-reverse`), `flex-wrap` (`nowrap`/`wrap`/`wrap-reverse`), `flex` (shorthand), `flex-grow`, `flex-shrink`, `flex-basis`, `order`, `align-self`, `justify-content`, `align-items`, `gap`, `row-gap`, `column-gap` — see [CSS Flex](#css-flex) below.

**Page breaks:** `page-break-before: always`, `page-break-after: always`, `break-before`, `break-after`, `break-inside: avoid`. Hard breaks honoured at `addRow()` build phase; `blockContainer` is splittable across page boundaries with background + border repaint on each slice.

**Length units:** `px` (1px = 0.264583mm), `pt` (1pt = 0.352778mm), `mm`, `cm`, `em` (relative to parent font-size), `rem`, `%` (inside `calc()` resolved against context width).

**CSS variables:** `--name: value` declarations on any element, `var(--name [, fallback])` in any value. Inherited through the cascade.

**`calc()` expressions:** `+`, `-`, `*`, `/`, one level of parentheses, lenient whitespace (`calc(100%-20mm)` accepted). Mixed units convert via `ParseLength` per token. `%` requires a known context width.

**Colour formats:** named colours (full CSS Color Level 4, ~147 entries), `#rgb` / `#rgba` / `#rrggbb` / `#rrggbbaa`, `rgb()`, `rgba()`, `hsl()`, `hsla()`. Alpha is tracked through to the internal PDF backend.

**Selectors:** Cascadia provides full CSS selector support: tag, class, id, attribute (`[attr]`, `[attr=val]`, `[attr^=val]`, `[attr$=val]`, `[attr*=val]`, `[attr~=val]`, `[attr|=val]`), `:nth-child(n)`, `:first-child`, `:last-child`, `:nth-of-type`, `:first-of-type`, `:last-of-type`, `:not(...)`. State-dependent pseudo-classes (`:hover`, `:focus`, `:active`, `:visited`) silently never match in static PDF output.

### Limitations

These remain partially supported or deferred — most are visual-quality trade-offs of the pure-Go pipeline.

- `letter-spacing` is consumed by `AddRichText` via per-character draw with manual x-advancement. Performance scales with character count, not word count.
- `text-align: justify` currently degrades to left alignment for RichText paragraphs.
- `align-self` is parsed and stored on `ComputedStyle` but the visual cross-axis offset requires knowing the row's max child height at render time. Currently a structural no-op; full visual alignment is deferred (blocked on explicit container height support).
- `align-content` is intentionally out of scope — it requires explicit container height which `blockContainer` does not yet honour. Use spacer rows instead.
- `box-shadow` blur is approximated by 3 overlaid translucent rects (constant-time). True Gaussian blur is deferred.
- `text-shadow` renders only the first shadow when comma-separated multi-shadows are provided.
- `outline` is drawn LAST in the cellwriter chain — in dense flex rows with multiple outlined items the right outline edge of each item except the rightmost is overdrawn by the next item's fill. Workaround: use borders instead, or full-row outlined containers.
- Conic gradients (`conic-gradient`) are not implemented. `filter: drop-shadow(...)` is not implemented.
- Inset `box-shadow` with `border-radius`: the inset shadow does NOT clip to the rounded corners (rectangular). Round-corner inset clipping is deferred.

## Documented v1 limitations

These are intentional v1 limitations — most can be worked around. They are not bugs.

### Container backgrounds spanning page breaks

Now **supported** via `core.Splittable` on `blockContainer`. When a styled `<div>` is too tall for the remaining page space, `paper.addRow()` calls `SplitAt(remainingHeight)` to slice the container; the first slice renders on the current page with the original top corners rounded and a flat bottom, the second slice renders on the next page with a flat top and the original bottom corners. Background and border are repainted on each slice. Set `break-inside: avoid` on the container to push the whole thing to the next page instead.

### Inline `<img>` splits the surrounding paragraph

Block-level `<img>` is fully supported (see [Images](#images)). The translator does not yet flow text around an inline `<img>` inside a paragraph; v1 renders the inline form as alt text. Image-in-paragraph flow is deferred to v2.

### Rounded outer corners on `<table>`

`border-radius` applies to `<div>` containers and table cells, but the outer corners of `<table>` itself are not clipped. Wrap a `<table>` in a `<div style="border-radius:…; padding:…">` for a rounded outer look.

### Out of scope (will not be supported in v1)

- JavaScript (the parser is HTML-only; no JS engine)
- CSS `grid`, `float`, `position`, `transform` (basic `flex` is supported — see [CSS Flex](#css-flex))
- `@media`, `@keyframes`, `@import` (note: `@font-face` IS supported — see [@font-face](#font-face) section)
- Pseudo-elements (`::before`, `::after`) and state-dependent pseudo-classes (`:hover`, `:focus`, `:active`, `:visited`). Non-state pseudo-classes including `:nth-child`, `:first-child`, `:last-child`, `:not(...)` ARE supported via Cascadia (see the Selectors entry above).
- (External stylesheets via `<link rel="stylesheet" href="…">` ARE supported with a configured resolver — see [Resolver options](#resolver-options-image-stylesheet-font))
- Form elements (`<input>`, `<button>`, `<form>`)
- `<video>`, `<audio>`, `<canvas>`, `<iframe>`

Unsupported tags fall through to their children's content. Unsupported CSS properties are silently ignored (you can register `WithUnsupportedHandler` to log them).

## Options

```go
html.FromString(input,
    html.WithUnsupportedHandler(func(prop, val string) {
        log.Printf("unsupported: %s=%s", prop, val)
    }),
)
```

### Resolver options (image, stylesheet, font)

Three resolver families share the same safety model: **by default, only `data:` URIs are accepted**. Local file reads must be explicitly opted in via a base directory or a custom resolver.

| Resolver        | Default-only-data | Base dir option            | Custom callback option          |
| --------------- | ----------------- | -------------------------- | ------------------------------- |
| `<img src>`     | ✓                 | `WithImageBaseDir(dir)`    | `WithImageResolver(fn)`         |
| `<link href>`   | ✓                 | `WithStylesheetBaseDir(dir)` | `WithStylesheetResolver(fn)`  |
| `@font-face`    | ✓ (shared with `<link>`) | shared `WithStylesheetBaseDir` | shared `WithStylesheetResolver` |

The base-dir resolvers use `filepath.Clean` + a prefix check to reject `..` traversal and absolute paths. Resolver errors and panics are wrapped in `defer recover()` and logged via `unsupportedHandler` — they never crash the caller.

### @font-face

Web fonts are loaded at translate time and registered with the internal PDF provider via the `core.LateFontProvider` capability:

```css
@font-face {
    font-family: "MyFont";
    src: url("./assets/MyFont.ttf") format("truetype");
}
p { font-family: "MyFont" }
```

- Only TTF (`format("truetype")`) and OTF (`format("opentype")`) URLs are loaded. `local()` entries and WOFF/WOFF2 are skipped because the internal backend cannot decode them, and they are logged via `unsupportedHandler`.
- Failures (resolver refused, malformed font bytes) log via `unsupportedHandler` and fall back to default fonts — never a panic.

### Internal anchors

```html
<h2 id="summary">Summary</h2>
…
<a href="#summary">jump</a>
```

`id="…"` reserves a PDF named destination during a pre-pass walk of the DOM; `<a href="#id">` makes the link's bounding box clickable. Forward references (link before target) work correctly thanks to the pre-pass.

## CSS Flex

Basic flexbox layout is supported, with some limitations driven by Paper's grid model.

```html
<div style="display:flex; gap:6mm">
  <div style="flex:1">Left</div>
  <div style="flex:2">Middle</div>
  <div style="flex:1">Right</div>
</div>
```

### Supported properties

| Property                      | Notes                                                                 |
| ----------------------------- | --------------------------------------------------------------------- |
| `display: flex`               | Switches the container to flex layout                                 |
| `display: inline-flex`        | Treated as `display: flex`                                            |
| `flex-direction: row`         | Default — children render as columns of a single row                  |
| `flex-direction: column`      | Children stack as separate rows                                       |
| `flex-direction: row-reverse` | Accepted but **does not** reverse order (limitation)                  |
| `flex-direction: column-reverse` | Accepted but **does not** reverse order (limitation)              |
| `flex: <grow> <shrink> <basis>` | Shorthand. Also accepts `auto`, `none`, `initial`, single number    |
| `flex-grow`                   | Proportional growth weight                                            |
| `flex-shrink`                 | Parsed but no independent effect (quantizer prevents overflow)        |
| `flex-basis: <length>`        | Used as item weight                                                   |
| `flex-basis: <percent>`       | Converted to fraction of grid (e.g. `25%` → 3 cols at gridSize=12)    |
| `flex-basis: auto`            | Item participates as a default grow item                              |
| `justify-content`             | `flex-start`, `flex-end`, `center`, `space-between`, `space-around`   |
| `align-items`                 | Accepted but cross-axis alignment is limited (see below)              |
| `gap`, `column-gap`           | Reserves integer spacer cols between items                            |
| `row-gap`                     | Reserves spacer rows between items in `flex-direction:column`         |

### Grid quantization

Paper's grid is **12 columns wide** by default (configurable via `config.WithMaxGridSize(n)`). Flex item sizes are quantized to integer col widths using Hamilton's largest-remainder method (provably the fairest integer split). This means:

- `flex:1 1 1` over 3 items → `[4,4,4]`
- `flex:1 1 1 1` over 4 items → `[3,3,3,3]`
- `flex:1 1 1 1 1` over 5 items → `[3,3,2,2,2]` (Hamilton redistributes the remainder)

`flex-basis:25%` at gridSize=12 → 3 cols. At gridSize=20 → 5 cols.

### Gap approximation

`gap`/`column-gap` is measured in mm but Paper needs integer cols. The translator converts using `mm_per_col = contentWidth / gridSize`, defaulting to 170mm/12 ≈ 14.17mm/col at A4 with 20mm L+R margins. For other page sizes, pass `html.WithContentWidth(mm)`. Gap is clamped to ≤ gridSize/2 cols.

### Limitations (intentional)

- **`flex-wrap`** — not supported. Flex items always stay on a single row.
- **`order`** — items render in DOM order regardless.
- **`align-self`** — per-item cross-axis override not supported (use container-level `align-items`).
- **`align-content`** — N/A without wrap.
- **`flex-shrink`** — parsed but no independent effect. Hamilton's quantizer always sums exactly to the grid total, so overflow is impossible.
- **`flex-direction: *-reverse`** — accepted as valid CSS but children render in source order.
- **`space-between` with no slack** — silently degrades to `flex-start` when item widths sum to the grid. Workaround: use non-equal flex weights, or ensure `gap` reserves spacer cols.
- **`align-items: center`/`flex-end`** — best-effort within Paper's row model. Cross-axis alignment in PDF is bounded by the row's auto-height behaviour.

### Configurable grid + content width

When constructing the paper document with a custom grid:

```go
cfg := config.NewBuilder().WithMaxGridSize(20).Build()
m := paper.New(cfg)
m.AddHTML(htmlStr) // flex quantization automatically uses gridSize=20
```

For non-A4 page sizes, pass content width when using `html.FromString` directly:

```go
rows, _ := html.FromString(input, html.WithGridSize(20), html.WithContentWidth(250.0))
```

## How it works

```
HTML string
   ↓ golang.org/x/net/html
DOM tree + extracted <style> blocks
   ↓ pkg/html/css (douceur + cascadia)
ComputedStyle per element (cascade + specificity + em tree-walk)
   ↓ pkg/html/translate
[]core.Row (uses RichText, Table, HTMLList components)
   ↓ Paper layout + pagination
PDF
```

The conversion is purely additive — your existing Paper code continues to work unchanged.

## Images

Block-level `<img src="…" width="…" height="…" alt="…">` produces a row containing the image. PNG and JPG are passed through directly; SVG sources are rasterised to PNG at 150 DPI via `github.com/srwiley/oksvg` + `rasterx` (both pure-Go, no CGO).

```html
<img src="logo.svg" width="20mm" height="20mm" alt="company logo">
```

### Safe-by-default resolver

The default resolver only accepts `data:` URIs (`data:image/png;base64,…`, `data:image/svg+xml;base64,…`). Local-file reads are refused to prevent path traversal attacks on user-controlled HTML.

To load local files, scope the resolver explicitly:

```go
rows, _ := html.FromString(input,
    html.WithImageBaseDir("./assets"), // <img src="…"> resolves inside ./assets only
)
m.AddRows(rows...)
```

The `WithImageBaseDir` resolver uses `filepath.Clean` + prefix check to reject `..` traversal and absolute paths.

For custom loading (HTTP, CDN, database), pass a `WithImageResolver`:

```go
rows, _ := html.FromString(input,
    html.WithImageResolver(func(src string) ([]byte, string, error) {
        bytes, err := fetchFromCDN(src)
        return bytes, "png", err
    }),
)
```

### Supported `<img>` units

`width` and `height` accept `px`, `pt`, `mm`, `cm`. Bare numbers (`width="20"`) are treated as pixels. `em` and `%` are not supported at image resolution because the translator has no font context at that point.

If only one of width/height is given for an SVG, the intrinsic aspect ratio from the `viewBox` fills the other.

### Image failure handling

When a resolver returns an error, the SVG parser rejects the input, or the PNG encoder fails, the translator falls back to the `<img>`'s `alt` attribute rendered as text. Register `WithUnsupportedHandler` to log these failures for diagnostics.

## Built-in CSS classes

Paper ships a small built-in stylesheet that applies before any user `<style>` block. The cascade precedence is **built-in < user CSS < inline `style=""`** — user styles always win.

### `.title-band`

A heading band with a dark navy background, white text, padding, and rounded corners:

```html
<h2 class="title-band">SUMMARY</h2>
```

To override the colors or padding, declare your own `.title-band` rule in your `<style>` block.

### `.circle-numbers`

Render an ordered list with each marker as a filled circle containing the number:

```html
<ol class="circle-numbers">
  <li>Wire transfer to Account 0123-4567</li>
  <li>Reference your invoice number in the memo</li>
</ol>
```

Marker colors default to navy fill with white text. Configure per-list via `htmllist.Prop.MarkerBackground` / `MarkerTextColor` when constructing the list programmatically.
