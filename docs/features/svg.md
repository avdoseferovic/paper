# SVG

`pkg/svg` parses SVG markup and exposes the document dimensions, root node,
`viewBox`, reusable `defs`, and `preserveAspectRatio` setting:

```go
doc, err := svg.Parse(`<svg width="120" height="60" viewBox="0 0 240 120">
	<rect width="240" height="120" fill="red"/>
</svg>`)
if err != nil {
	return err
}

fmt.Println(doc.Width(), doc.Height(), doc.ViewBox())
```

The parsed SVG can be rasterized to PNG bytes:

```go
pngBytes, pxW, pxH, err := doc.Rasterize(40, 0)
```

It can also be adapted directly to Paper image components. Paper keeps the
original SVG bytes and rasterizes them during PDF image embedding:

```go
row := doc.AutoRow(props.Rect{Percent: 80, Center: true})
pdf.AddRows(row)
```

Current scope:

- `Parse`, `ParseBytes`, and `ParseReader`
- root node and child tree access
- `Defs` indexed by element `id`
- `Width`, `Height`, `ViewBox`, `AspectRatio`, and `PreserveAspectRatio`
- package-level and document-level `Rasterize`
- `Component`, `Col`, `Row`, and `AutoRow` adapters

Limitation: SVG output is currently rasterized to PNG before PDF embedding.
Folio-style native vector SVG drawing is still future work.
