# Watermark

`config.WithWatermark` stamps every page with translucent diagonal text,
drawn **under** the page content (right after the background) so it never
obscures what the reader needs to see.

Defaults: 48pt, 12% opacity, rotated 45° around the page center. When the text
would exceed the page diagonal, the font size scales down automatically.
Customize via `props.Watermark`:

```go
cfg := config.NewBuilder().
    WithWatermark("CONFIDENTIAL", props.Watermark{
        Size:  64,
        Alpha: 0.08,
        Angle: 30,
        Color: &props.Color{Red: 200, Green: 30, Blue: 30},
    }).
    Build()
```

For image-based page backgrounds, see
[Background](background.md?id=add-background).

[filename](../assets/examples/watermark/main.go ':include :type=code')

[watermark.pdf](../assets/pdf/watermark.pdf ':include :type=pdf')
