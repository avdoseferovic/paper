# Tagged PDF

Tagged PDF output can be enabled for generated documents:

```go
doc := paper.New(config.NewBuilder().
	WithTaggedPDF(true).
	Build())
```

or at runtime:

```go
doc.SetTagged(true)
```

When enabled, Paper emits the tagged-PDF foundation required by accessibility
tools: catalog `/MarkInfo`, a `/StructTreeRoot`, a parent tree, per-page
`/StructParents`, and page-level BDC/EMC marked content with MCIDs.

Current scope: Paper tags each generated page as one paragraph structure
element. Per-component semantic tags such as headings, tables, figures, and
links are not inferred yet.
