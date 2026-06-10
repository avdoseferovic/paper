# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added release engineering scaffolding for the first tagged release.
- Added nested module coverage in CI for `pkg/test`, `examples`, and docs examples.
- Added `govulncheck` CI coverage for pull requests, manual runs, and weekly scheduled scans.
- Added `core.Document.Write(io.Writer)` and `(*core.Pdf).Write` for HTTP-handler friendly output.
- Added `GenerateCtx`, `AddHTMLCtx`, `FromHTMLCtx`, `FromHTMLReaderCtx`, and context-aware HTML translation entry points.
- Added default resource limits for HTML input, including image, SVG, DOM, and CSS rule caps.
- Added fuzz targets for HTML, CSS, and SVG processing.
- Added opt-in AES-128 PDF protection through `WithProtectionAlgorithm(protection.AES128)`.

### Changed

- Renamed the current-page fit check method to `FitInCurrentPage`.
- Split `pkg/test`, `examples`, and `docs` into nested Go modules to keep root consumer dependencies and module downloads lean.
- Moved generated mocks under `internal/mocks` and removed the root-level generated `mocks` package.
- Moved the metrics decorator into `pkg/decorator`.
- Changed `New` to return `*paper.Paper`.
- Changed document-generation helpers to return concrete `*core.Pdf` values where appropriate.
- Collapsed the internal gofpdf wrapper interface and pruned unused legacy PDF internals.
- Updated the supported Go toolchain to 1.26.4 and refreshed vulnerable dependency versions used by the HTML/image paths.

### Removed

- Removed root-module demo binaries from `cmd`; demos now live in the nested `examples` module.
- Removed the root-module `pkg/test` package; it is now the separate `github.com/avdoseferovic/paper/pkg/test` module.

### Security

- Documented the current PDF protection behavior as RC4-based protection, not confidentiality-grade encryption.
- Kept RC4 as the compatibility default and documented AES-128 as the preferred option for new protected documents.
- Verified AES-128 protected output with `pdfcpu validate` during release preparation.
