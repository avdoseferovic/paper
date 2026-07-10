# Named Destinations

Named destinations add stable destination names to the generated PDF catalog. Other PDF content or external tools can target these names without relying on physical page labels.

## Builder API

```go
cfg := config.NewBuilder().
	WithNamedDestinations(
		entity.NamedDestination{Name: "top", PageIndex: 0},
		entity.NamedDestination{
			Name:      "section",
			PageIndex: 2,
			FitType:   entity.DestinationFitH,
			Top:       700,
		},
	).
	Build()
```

## Runtime API

```go
m := paper.New(config.NewBuilder().Build())
m.AddNamedDest(entity.NamedDestination{
	Name:      "detail",
	PageIndex: 0,
	FitType:   entity.DestinationXYZ,
	Left:      10,
	Top:       20,
	Zoom:      1.5,
})
```

`PageIndex` is zero-based. `FitType` defaults to `entity.DestinationFit`; supported values are `DestinationFit`, `DestinationFitH`, and `DestinationXYZ`.

## Notes

- Named destinations are document-level catalog entries, so Paper keeps generation on the whole-document path even if concurrent generation was requested.
- Invalid page indexes and empty destination names are skipped.
