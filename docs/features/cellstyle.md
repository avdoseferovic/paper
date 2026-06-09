# Cell Style

Cell Style applies visual decoration — background fill, borders, and line styling — to any `Col` or `Row` via `.WithStyle(*props.Cell)`. Styling is independent of the content inside the cell, so the same style can be reused across many rows and columns.

## Props (`props.Cell`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `BackgroundColor` | `*props.Color` | `nil` (transparent) | Cell fill color |
| `BorderColor` | `*props.Color` | `nil` (black) | Border line color |
| `BorderType` | `border.Type` | `border.None` | Which sides to draw — combinable with `\|` |
| `BorderThickness` | `float64` | `0.2` | Border line thickness in mm |
| `LineStyle` | `linestyle.Type` | `linestyle.Solid` | `linestyle.Solid` or `linestyle.Dashed` |

## Border combinations

`border.Type` values can be combined with the `|` operator:

| Expression | Result |
|------------|--------|
| `border.Full` | All four sides |
| `border.Left \| border.Right` | Left and right sides only |
| `border.Top \| border.Bottom` | Top and bottom sides only |
| `border.Left \| border.Top` | Top-left corner |
| `border.Left \| border.Right \| border.Top` | Three sides, open bottom |

## GoDoc
* [component : Col : WithStyle](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/components/col#Col.WithStyle)
* [component : Row : WithStyle](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/components/row#Row.WithStyle)
* [props : Cell](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/props#Cell)

## Code Example
[filename](../assets/examples/cellstyle/main.go ':include :type=code')

## PDF Generated
```pdf
	assets/pdf/cellstyle.pdf
```

## Time Execution
[filename](../assets/text/cellstyle.txt  ':include :type=code')

## Test File
[filename](https://raw.githubusercontent.com/avdoseferovic/paper/master/test/paper/examples/cellstyle.json  ':include :type=code')