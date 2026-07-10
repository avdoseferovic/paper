# File ID

Paper can write an explicit PDF trailer file identifier through `/ID`. This is useful when generated output needs a stable document identifier for downstream workflows.

## Builder API

```go
cfg := config.NewBuilder().
	WithFileID([]byte{0xab, 0xcd, 0xef}).
	Build()
```

## Runtime API

```go
m := paper.New(config.NewBuilder().Build())
m.SetFileID([]byte{0xab, 0xcd, 0xef})
```

The identifier bytes are written as the first and second entries of the trailer `/ID` array.

## Deterministic Output

```go
cfg := config.NewBuilder().
	WithDeterministic(true).
	Build()

m := paper.New(cfg)
m.SetDeterministic(true)
```

Deterministic mode derives a stable trailer `/ID` for generated PDFs that do not already have an explicit file ID. It also replaces implicit wall-clock creation, modification, and attachment dates with the zero time so identical inputs produce identical output.

## Notes

- Paper clones the identifier bytes at builder and runtime boundaries.
- When AES-128 protection is enabled after setting a file ID, the identifier is also used by the protection key setup.
- Protected PDFs keep the security handler's encryption-specific identifier behavior.
