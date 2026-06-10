# internal/

Private packages of the Paper module. Two conventions in here are deliberate
and worth knowing before contributing.

## Vendored test infrastructure — why not testify?

The root module must stay free of third-party **test** dependencies so that
consumers of `github.com/avdoseferovic/paper` never inherit them through
`go.mod`. That is the entire reason these four packages exist:

| Package | Replaces |
| --- | --- |
| `internal/assert` | `testify/assert` (non-fatal assertions) |
| `internal/require` | `testify/require` (fatal assertions) |
| `internal/mocktest` | `testify/mock` (runtime for generated mocks) |
| `internal/goleak` | `go.uber.org/goleak` (goroutine leak detection) |

**Do not introduce testify (or any other test framework) into the root
module.** Generated mocks in `internal/mocks/` are rewritten from testify to
`internal/mocktest` by `make mocks` (see `internal/cmd/mockfix`).

## Test helpers

`internal/test` is the canonical golden-structure test helper
(`PaperTest.Equals` against `test/paper/*.json` snapshots). The public
`pkg/test` package is a thin re-export of it for library consumers.

## The embedded PDF engine

`internal/pdf` is a gofpdf-derived PDF writer owned by this repository.
It is intentionally self-contained; application code must depend on Paper's
public packages, never on `internal/pdf` directly.
