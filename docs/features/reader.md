# Reader

`pkg/reader` provides a foundation for working with existing PDFs:

```go
r, err := reader.Load("document.pdf")
if err != nil {
	return err
}

fmt.Println("pages:", r.PageCount())

page, err := r.Page(0)
if err != nil {
	return err
}

text, err := page.ExtractText()
if err != nil {
	return err
}
fmt.Println(text)
```

The package can also merge parsed readers through Paper's existing PDF merge
engine and extract selected pages into a new PDF:

```go
first, _ := reader.Load("a.pdf")
second, _ := reader.Load("b.pdf")

merged, err := reader.Merge(first, second)
if err != nil {
	return err
}

err = merged.SaveTo("merged.pdf")

subset, err := reader.ExtractPages(first, 2, 0)
if err != nil {
	return err
}
err = subset.SaveTo("subset.pdf")

err = merged.RemovePage(0)
if err != nil {
	return err
}
err = merged.RotatePage(0, 90)
if err != nil {
	return err
}
err = merged.CropPage(0, [4]float64{36, 36, 576, 756})
if err != nil {
	return err
}
err = merged.AddBlankPage(612, 792)
```

## Redaction Foundation

`RedactText` and `RedactPattern` permanently remove matching bytes from
uncompressed page content streams while preserving stream length and xref
offsets:

```go
r, _ := reader.Load("document.pdf")
redacted, err := reader.RedactText(r, []string{"John Doe", "555-12-3456"}, nil)
if err != nil {
	return err
}
err = redacted.SaveTo("redacted.pdf")
```

Regex redaction is also available:

```go
ssn := regexp.MustCompile(`\d{3}-\d{2}-\d{4}`)
redacted, err := reader.RedactPattern(r, ssn, nil)
```

`RedactOptions.StripMetadata` additionally empties the document information
dictionary and any uncompressed XMP `/Metadata` stream, in place and without
changing the file length:

```go
redacted, err := reader.RedactText(r, targets, &reader.RedactOptions{StripMetadata: true})
```

A compressed `/Metadata` stream cannot be rewritten byte-preservingly, so it is
reported as an error rather than left in the output. The remaining
`RedactOptions` fields describe visual overlays, which this foundation does not
draw.

Current scope:

- `Load`, `Parse`, and `ParseWithOptions`
- `RawBytes`, `Version`, `PageCount`, and zero-based `Page`
- page `MediaBox`, `CropBox`, `BleedBox`, `TrimBox`, `ArtBox`, rotation,
  width, and height metadata
- decoded page `ContentStream` for unfiltered and `/FlateDecode` streams
- basic text extraction from literal and hex strings used with `Tj`, `TJ`,
  `'`, and `"` operators
- `Merge` and `MergeFiles` convenience wrappers
- `ExtractPages` for zero-based page selection/reordering backed by the merge
  engine
- `Modifier` page operations: `RemovePage`, `ReorderPages`, `RotatePage`,
  `CropPage`, and `AddBlankPage`
- byte-preserving `RedactText` and `RedactPattern` for uncompressed/simple
  content streams

Limitations:

- encrypted PDFs are rejected
- xref streams and object streams are not yet parsed
- full font CMap Unicode extraction is not implemented
- compressed content streams are rejected by the redaction foundation
- redaction overlays and glyph-bounding-box matching are not implemented yet
- Folio-style object/resource page import and modifier-level form flattening
  are not part of this reader foundation slice; use `forms.FormFiller.Flatten`
  for supported simple AcroForm flattening
- `RotatePage` and `CropPage` rewrite classic-xref PDFs with updated page
  dictionaries; full xref-stream/object-stream modifier support is future work
