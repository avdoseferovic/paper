# Bookmark (PDF Outline)

Text and richtext components can register themselves in the PDF document
outline — the bookmark sidebar shown by PDF viewers — via the
`props.Outline` property.

- `Level` controls nesting: `0` is the top level, `1` nests under the previous
  level-0 entry, and so on. Negative values are treated as `0`.
- `Title` overrides the sidebar text; when empty, the component's own text
  content is used.

For HTML documents, see [HTML support](../html-support.md) for the
`WithOutlineFromHeadings` option that generates outline entries from
`h1`–`h6` headings automatically.

Outline entries are preserved in all generation modes, including concurrent
and low-memory generation.

```go
m := paper.New()

m.AddAutoRow(col.New(12).Add(text.New("Getting Started", props.Text{
    Size:    16,
    Outline: &props.Outline{Level: 0},
})))
m.AddAutoRow(col.New(12).Add(text.New("Installation", props.Text{
    Size:    12,
    Outline: &props.Outline{Level: 1, Title: "Installing Paper"},
})))
```

[filename](../assets/examples/bookmark/main.go ':include :type=code')

[bookmark.pdf](../assets/pdf/bookmark.pdf ':include :type=pdf')
