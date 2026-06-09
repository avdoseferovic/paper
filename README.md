# Paper

[![pkg.go.dev](https://pkg.go.dev/badge/github.com/avdoseferovic/paper.svg)](https://pkg.go.dev/github.com/avdoseferovic/paper)
[![Go Report Card](https://goreportcard.com/badge/github.com/avdoseferovic/paper)](https://goreportcard.com/report/github.com/avdoseferovic/paper)
[![CI](https://github.com/avdoseferovic/paper/actions/workflows/goci.yml/badge.svg)](https://github.com/avdoseferovic/paper/actions/workflows/goci.yml)
[![Lint](https://github.com/avdoseferovic/paper/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/avdoseferovic/paper/actions/workflows/golangci-lint.yml)
[![Codecov](https://img.shields.io/codecov/c/github/avdoseferovic/paper)](https://codecov.io/gh/avdoseferovic/paper)
[![License](https://img.shields.io/github/license/avdoseferovic/paper)](LICENSE)

Paper is a Go library for creating PDF documents from HTML and structured
components.

Use it when you want a direct HTML-to-PDF path, or when you need programmatic
layout with rows, columns, headers, footers, metrics, tests, and explicit
control over document generation.

![Paper logo](docs/assets/images/paper-icon.svg)

## Installation

```bash
go get github.com/avdoseferovic/paper
```

## HTML to PDF

For HTML-only documents, use `paper.FromHTML`.

```go
package main

import (
	"log"

	"github.com/avdoseferovic/paper"
)

func main() {
	doc, err := paper.FromHTML(`<h1>Hello</h1><p>World</p>`)
	if err != nil {
		log.Fatal(err)
	}

	if err := doc.Save("out.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

Use `paper.FromHTMLReader` when the source HTML is already available as an
`io.Reader`. For advanced HTML options such as asset base directories, call
`pkg/html` directly and add the returned rows to a `paper.New(...)` document.

For mixed layouts, HTML fragments can also be used as regular components via
`github.com/avdoseferovic/paper/pkg/components/html`:

```go
htmlBlock, err := htmlcomponent.New(`<h2>Terms</h2><p>Rendered from HTML.</p>`)
if err != nil {
	log.Fatal(err)
}

doc := paper.New()
doc.AddAutoRow(
	col.New(6).Add(text.New("Direct Paper component")),
	col.New(6).Add(htmlBlock),
)
```

## Component Layout

Use the row and column API when a document needs manual layout, repeated
headers or footers, generated pages, metrics, or a testable component tree.

```go
package main

import (
	"log"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	htmlcomponent "github.com/avdoseferovic/paper/pkg/components/html"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	cfg := config.NewBuilder().
		WithLeftMargin(15).
		WithTopMargin(15).
		Build()

	doc := paper.New(cfg)
	htmlBlock, err := htmlcomponent.New(`<p>HTML fragment inside the grid</p>`)
	if err != nil {
		log.Fatal(err)
	}

	doc.AddRows(
		row.New(12).Add(
			col.New(12).Add(text.New("Invoice", props.Text{Size: 18})),
		),
		row.New().Add(
			col.New(6).Add(text.New("Programmatic content")),
			col.New(6).Add(htmlBlock),
		),
		text.NewRow(8, "Generated with Paper"),
	)

	pdf, err := doc.Generate()
	if err != nil {
		log.Fatal(err)
	}

	if err := pdf.Save("invoice.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

Paper uses a grid-based layout model. Rows stack vertically, columns split a row
across the configured grid width (12 columns by default, configurable with
`WithMaxGridSize`), and new pages are added automatically when content exceeds
the useful page area after margins, headers, and footers are reserved.

## Key Features

- HTML-to-PDF conversion through `paper.FromHTML`, `paper.FromHTMLReader`, and
  `pkg/html`.
- Programmatic PDF layout with rows, columns, text, images, codes, tables,
  signatures, page numbers, headers, and footers.
- Document output as bytes, base64, saved files, or merged PDFs.
- Component-tree inspection through `GetStructure`, designed for deterministic
  unit tests.
- Optional generation metrics through `paper.NewMetricsDecorator`.
- Internal PDF backend ownership, so application code depends on Paper's public
  packages rather than a third-party renderer API.

## Performance

Generation is benchmarked in [`paper_benchmark_test.go`](paper_benchmark_test.go).
Two benchmarks live there:

- `BenchmarkPDFGeneration` — representative documents (text-heavy, mixed
  components, full HTML demo).
- `BenchmarkPDFScaling` — a text document swept across 10–1000 rows to expose
  the per-row / per-page cost curve.

Run them with:

```bash
# Representative scenarios
go test -run='^$' -bench=BenchmarkPDFGeneration -benchmem -count=6 .

# Size scaling curve
go test -run='^$' -bench=BenchmarkPDFScaling -benchmem -count=6 .
```

The numbers below are the median of 6 runs on an Apple M1 Pro (Go 1.26),
single-threaded. They are representative of the bundled fixtures, not a
universal guarantee — actual time scales with page count, image size, and
component mix.

### Representative documents (`BenchmarkPDFGeneration`)

| Scenario           | Document                                                | Time / doc | Mem / doc | Allocs / doc |
|--------------------|---------------------------------------------------------|-----------:|----------:|-------------:|
| `TextHeavy`        | 180 text rows (~6 pages)                                |    1.05 ms |  1.16 MiB |        9,511 |
| `MixedComponents`  | 40× (barcode + QR + image + signature + text)           |    4.85 ms |  3.49 MiB |       12,457 |
| `HTMLDemoFull`     | HTML → PDF: header + styled body + embedded PNG         |    7.83 ms | 13.47 MiB |       43,629 |

### Size scaling (`BenchmarkPDFScaling`)

| Rows | ~Pages (A4) | Time / doc | Mem / doc | Allocs / doc |
|-----:|------------:|-----------:|----------:|-------------:|
|   10 |           1 |    0.30 ms |   153 KiB |        2,283 |
|   50 |           2 |    0.48 ms |   377 KiB |        3,969 |
|  100 |           4 |    0.69 ms |   671 KiB |        6,103 |
|  500 |          17 |    2.48 ms |  2.87 MiB |       23,108 |
| 1000 |          34 |    4.79 ms |  5.65 MiB |       44,307 |

The curve is linear, giving a simple cost model for text content:

```
time(N rows) ≈ 0.25 ms (fixed setup) + 4.5 µs × N
```

That is roughly **~140 µs per A4 page** and **~42 allocations per row**. Generation
is single-threaded and the internal compression writers are pooled in a
concurrency-safe way, so throughput scales ~linearly across cores when
generating documents in parallel.

## Documentation

- [API reference](https://pkg.go.dev/github.com/avdoseferovic/paper)
- [Project docs](docs/README.md)
- [HTML support](docs/html-support.md)
- [Examples](docs/assets/examples)

## Development

| Command         | Description                                       |
|-----------------|---------------------------------------------------|
| `make build`    | Build the project                                 |
| `make test`     | Run unit tests                                    |
| `make fmt`      | Format Go files                                   |
| `make lint`     | Run lint checks                                   |
| `make dod`      | Run the local definition-of-done checks           |
| `make examples` | Run documentation examples                        |
| `make docs`     | Start the local docs server                       |

## Credits

Paper is derived from and inspired by
[Maroto](https://github.com/johnfercher/maroto), created by Johnathan Fercher
da Rosa and contributors. The original project established the Bootstrap-style
row and column PDF authoring model that Paper continues to evolve.

Logo art credit remains with
[@marinabankr](https://www.instagram.com/marinabankr/).

## License

Paper is released under the [MIT License](LICENSE).
