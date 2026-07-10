# Page Geometry

Paper can write page dictionary geometry entries for generated PDFs: rotation plus CropBox, BleedBox, TrimBox, and ArtBox.

## Runtime API

```go
m := paper.New(config.NewBuilder().Build())
m.SetPageRotation(0, 90)
m.SetCropBox(0, [4]float64{10, 20, 210, 290})
m.SetTrimBox(0, [4]float64{15, 25, 205, 285})
```

`PageIndex` is zero-based. Rotation must be a multiple of 90 degrees.

## Builder API

```go
crop := [4]float64{10, 20, 210, 290}
cfg := config.NewBuilder().
	WithPageGeometries(entity.PageGeometry{
		PageIndex: 0,
		Rotate:    90,
		CropBox:   &crop,
	}).
	Build()
```

## Notes

- Box values are written directly to the generated PDF page dictionary.
- Page geometry entries are document-level page dictionary settings, so Paper keeps generation on the whole-document path even if concurrent generation was requested.
- This feature applies to newly generated Paper PDFs. It does not inspect or rewrite page geometry from existing PDFs.
