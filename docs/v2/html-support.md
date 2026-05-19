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

**Layout:** `display: block|inline|inline-block|none`, `width`, `height`

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
- CSS `flexbox`, `grid`, `float`, `position`, `transform`
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
