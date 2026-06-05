# Paper V2

[![GoDoc](https://godoc.org/github.com/avdoseferovic/paper?status.svg)](https://pkg.go.dev/github.com/avdoseferovic/paper/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/avdoseferovic/paper)](https://goreportcard.com/report/github.com/avdoseferovic/paper)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#template-engines)
[![CI](https://github.com/avdoseferovic/paper/actions/workflows/goci.yml/badge.svg)](https://github.com/avdoseferovic/paper/actions/workflows/goci.yml)
[![Lint](https://github.com/avdoseferovic/paper/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/avdoseferovic/paper/actions/workflows/golangci-lint.yml)
[![Codecov](https://img.shields.io/codecov/c/github/avdoseferovic/paper)](https://codecov.io/gh/avdoseferovic/paper)
[![Visits Badge](https://badges.pufler.dev/visits/avdoseferovic/paper)](https://badges.pufler.dev)
[![Stars Badge](https://img.shields.io/github/stars/avdoseferovic/paper.svg?style=social&label=Stars)](https://github.com/avdoseferovic/paper/stargazers)


A Paper way to create PDFs. Paper can convert HTML directly to PDF, or compose documents with a Bootstrap-inspired row/column API. Fast and simple.

![Paper logo](docs/assets/images/logosmall.png)
> Paper generates PDF documents from structured Go components and HTML.
> [Art by **@marinabankr**](https://www.instagram.com/marinabankr/)

For custom layouts, you can still write PDFs with Paper's row/column API. A row may have many cols, and a col may have many components.
Pages are added automatically when content exceeds the useful area. You can also define headers and footers for documents that need manual composition.

## HTML to PDF

```go
doc, err := paper.FromHTML(`<h1>Hello</h1><p>World</p>`)
if err != nil {
    log.Fatal(err)
}

_ = doc.Save("out.pdf")
```

Use `paper.New()` only when you need to mix HTML with manually composed rows, headers, footers, or other Paper components.

#### Paper `v2.4.0` is here! Try out:

* Installation with`go get`:

```bash
go get github.com/avdoseferovic/paper/v2@v2.4.0
```

* You can see the full `v2` documentation [here](https://paper.tech/#/README?id=paper-v2).
* The `v1` still exists in [this branch](https://github.com/avdoseferovic/paper/tree/v1), and you can see the doc [here]([https://paper.tech/#/v1/README?id=deprecated](https://paper.tech/#/v1/README?id=deprecated)).

![result](docs/assets/images/result.png)

## Contributing

| Command         | Description                                       | Dependencies                                                  |
|-----------------|---------------------------------------------------|---------------------------------------------------------------|
| `make build`    | Build project                                     | `go`                                                          |
| `make test`     | Run unit tests                                    | `go`                                                          |
| `make fmt`      | Format files                                      | `gofmt`, `gofumpt` and `goimports`                            |
| `make lint`     | Check files                                       | `golangci-lint`                                               |
| `make dod`      | (Definition of Done) Format files and check files | Same as `make build`, `make test`, `make fmt` and `make lint` |
| `make install`  | Install all dependencies                          | `go`, `curl` and `git`                                        |
| `make examples` | Run all examples                                  | `go`                                                          |
| `make mocks`    | Generate mocks                                    | `go` and `mockery`                                            |
| `make docs`     | Run docsify docs server local                     | `docsify`                                                     |
| `make godoc`    | Run godoc server local                            | `godoc`                                                       |

## Stargazers over time
[![Stargazers over time](https://starchart.cc/avdoseferovic/paper.svg?variant=adaptive)](https://starchart.cc/avdoseferovic/paper)
