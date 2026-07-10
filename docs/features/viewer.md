# Viewer Preferences

Viewer preferences are PDF catalog hints for how a reader should open the document. They do not change page content, and PDF viewers may ignore some hints.

## Page layout and mode

```go
cfg := config.NewBuilder().
	WithViewerPreferences(entity.ViewerPreferences{
		PageLayout:      entity.LayoutTwoPageRight,
		PageMode:        entity.ModeUseThumbs,
		DisplayDocTitle: true,
	}).
	Build()
```

## Window hints

`entity.ViewerPreferences` supports `HideToolbar`, `HideMenubar`, `HideWindowUI`, `FitWindow`, `CenterWindow`, and `DisplayDocTitle`.

## Initial view

Use `OpenPage` and `OpenZoom` to suggest the page and zoom used when the PDF opens.

```go
cfg := config.NewBuilder().
	WithViewerPreferences(entity.ViewerPreferences{
		OpenPage: 0,
		OpenZoom: "FitH",
	}).
	Build()
```

`OpenPage` is zero-based. `OpenZoom` supports `Fit`, `FitH`, `FitV`, `FitB`, or a percentage such as `100` or `125%`. If only unrelated viewer preferences are set, Paper does not emit an `/OpenAction` from the zero-value `OpenPage`.

## Page labels

Page labels define the labels shown by PDF viewers instead of plain physical page numbers.

```go
cfg := config.NewBuilder().
	WithPageLabels(
		entity.PageLabelRange{PageIndex: 0, Style: entity.LabelRomanLower, Prefix: "front-"},
		entity.PageLabelRange{PageIndex: 4, Style: entity.LabelDecimal, Start: 1},
	).
	Build()
```

`PageIndex` is zero-based. `Style` can be decimal, Roman, alphabetic, or none. `Prefix` is prepended to the generated number, and `Start` sets the first number for that range.

## Notes

- Catalog options are document-level settings, so Paper keeps generation on the whole-document path even if concurrent generation was requested.
- These settings are advisory. They rely on PDF viewer support and do not affect the rendered page content.
