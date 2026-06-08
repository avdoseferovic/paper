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

![Paper logo](docs/assets/images/logosmall.png)

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

## Component Layout

Use the row and column API when a document needs manual layout, repeated
headers or footers, generated pages, metrics, or a testable component tree.

```go
package main

import (
	"log"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
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
	doc.AddRows(
		row.New(12).Add(
			col.New(12).Add(text.New("Invoice", props.Text{Size: 18})),
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
across the configured grid width, and new pages are added automatically when
content exceeds the available page area.

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
