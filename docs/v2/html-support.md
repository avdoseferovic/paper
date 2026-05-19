# HTML → PDF Support

Maroto can convert a documented subset of HTML/CSS into PDF rows. No browser, no Node, no external binary — pure Go.

## Quick start

```go
import maroto "github.com/johnfercher/maroto/v2"

m := maroto.New()
if err := m.AddHTML(`<h1>Hello</h1><p>World</p>`); err != nil {
    log.Fatal(err)
}
doc, _ := m.Generate()
_ = doc.Save("out.pdf")
```

Or build rows directly:

```go
import "github.com/johnfercher/maroto/v2/pkg/html"

rows, err := html.FromString(htmlString)
// rows is []core.Row — add to any Maroto document
```

## Supported HTML tags

**Block:** `html, head, body, div, p, h1, h2, h3, h4, h5, h6, hr, pre, blockquote, header, footer, section, article, aside, main, nav, figure, figcaption`

**Inline:** `span, a, strong, b, em, i, u, s, strike, sub, sup, br`

**Tables:** `table, thead, tbody, tfoot, tr, th, td` — with `colspan` and `rowspan` attributes.

**Lists:** `ul, ol, li` — `ol` supports `type="a|A|i|I"` for alpha/roman markers. Nested lists supported.

**Images:** `img` — inline `<img>` renders as alt text in v1; block-level full-row rendering deferred to v2.

## Supported CSS properties

**Text:** `color, font-family, font-size, font-weight, font-style, text-align, text-decoration, line-height`

**Box model:** `padding, padding-{top,right,bottom,left}, margin, margin-{top,right,bottom,left}`

**Borders:** `border, border-{top,right,bottom,left}, border-color, border-width, border-style`

**Background:** `background-color`

**Layout:** `display: block|inline|inline-block|none|flex|inline-flex`, `width`, `height`

**Flex:** `flex-direction`, `flex` (shorthand), `flex-grow`, `flex-shrink`, `flex-basis`, `justify-content`, `align-items`, `gap`, `row-gap`, `column-gap` — see [CSS Flex](#css-flex) below.

**Length units:** `px` (1px = 0.264583mm), `pt` (1pt = 0.352778mm), `mm`, `cm`, `em` (relative to parent font-size), `rem`.

## Documented v1 limitations

These are intentional v1 limitations — most can be worked around. They are not bugs.

### Container backgrounds/borders do not span children

```html
<div style="border: 1px solid black">
  <p>A</p>
  <p>B</p>
</div>
```

The `<div>` border is approximated by rendering it per contained row, not as a single rectangle around both. Each `<p>` gets its own bordered row. For a true single-rectangle container, wrap content in a `<table>` instead (table grid borders are supported natively).

### Inline `<img>` splits the surrounding paragraph

The translator does not yet flow text around inline images. v1 renders inline `<img>` as its `alt` text inline; full image-in-paragraph splitting is deferred to v2.

### Out of scope (will not be supported in v1)

- JavaScript (the parser is HTML-only; no JS engine)
- CSS `grid`, `float`, `position`, `transform` (basic `flex` is supported — see [CSS Flex](#css-flex))
- `@media`, `@keyframes`, `@font-face`
- Pseudo-elements (`::before`, `::after`), pseudo-classes (`:hover`, `:nth-child`)
- External stylesheets (`<link rel="stylesheet" href="…">`)
- SVG (raster `<img>` only — PNG/JPG via `WithImageResolver` if you need it)
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

## CSS Flex

Basic flexbox layout is supported, with some limitations driven by Maroto's grid model.

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

Maroto's grid is **12 columns wide** by default (configurable via `config.WithMaxGridSize(n)`). Flex item sizes are quantized to integer col widths using Hamilton's largest-remainder method (provably the fairest integer split). This means:

- `flex:1 1 1` over 3 items → `[4,4,4]`
- `flex:1 1 1 1` over 4 items → `[3,3,3,3]`
- `flex:1 1 1 1 1` over 5 items → `[3,3,2,2,2]` (Hamilton redistributes the remainder)

`flex-basis:25%` at gridSize=12 → 3 cols. At gridSize=20 → 5 cols.

### Gap approximation

`gap`/`column-gap` is measured in mm but Maroto needs integer cols. The translator converts using `mm_per_col = contentWidth / gridSize`, defaulting to 170mm/12 ≈ 14.17mm/col at A4 with 20mm L+R margins. For other page sizes, pass `html.WithContentWidth(mm)`. Gap is clamped to ≤ gridSize/2 cols.

### Limitations (intentional)

- **`flex-wrap`** — not supported. Flex items always stay on a single row.
- **`order`** — items render in DOM order regardless.
- **`align-self`** — per-item cross-axis override not supported (use container-level `align-items`).
- **`align-content`** — N/A without wrap.
- **`flex-shrink`** — parsed but no independent effect. Hamilton's quantizer always sums exactly to the grid total, so overflow is impossible.
- **`flex-direction: *-reverse`** — accepted as valid CSS but children render in source order.
- **`space-between` with no slack** — silently degrades to `flex-start` when item widths sum to the grid. Workaround: use non-equal flex weights, or ensure `gap` reserves spacer cols.
- **`align-items: center`/`flex-end`** — best-effort within Maroto's row model. Cross-axis alignment in PDF is bounded by the row's auto-height behaviour.

### Configurable grid + content width

When constructing the maroto document with a custom grid:

```go
cfg := config.NewBuilder().WithMaxGridSize(20).Build()
m := maroto.New(cfg)
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
   ↓ Maroto layout + pagination
PDF
```

The conversion is purely additive — your existing Maroto code continues to work unchanged.
