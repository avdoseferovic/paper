# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added interactive AcroForm generation (`config.WithAcroForm`, `Paper.SetAcroForm`,
  `pkg/forms` field builders) with fill, flatten, and read-back via `pkg/forms.FormFiller`.
- Added PDF/A conformance output (`config.WithPdfA`, levels 1B-4E) with XMP
  metadata, extension schemas, and sRGB or custom output intents; Level-A
  profiles enable tagged output automatically.
- Added tagged (accessible) PDF output (`config.WithTaggedPDF`, `Paper.SetTagged`)
  with structure tree, marked content, and `/Lang` (`config.WithLanguage`).
- Added document-catalog features: viewer preferences (`config.WithViewerPreferences`),
  page labels, file attachments, named destinations, page annotations
  (`Paper.AddAnnotation` and friends), per-page geometry boxes/rotation,
  explicit file IDs (`config.WithFileID`), and deterministic byte-stable builds
  (`config.WithDeterministic`).
- Added `pkg/reader` (parse, page remove/reorder/extract, rotate, crop, redact,
  `reader.MergeFiles`) backed by new `merge.BytesSelected`/`merge.PageSelection`.
- Added `pkg/sign` (CMS/PAdES digital signatures, timestamps, DSS/LTV), `pkg/svg`,
  `pkg/barcode`, `pkg/image` (GIF/WebP/TIFF via codec normalization), `pkg/tmpl`,
  and CMYK color support (`props.Color.CMYK`).
- Added HTML pipeline features: CSS multi-column, grid, and position/transform
  support, form controls, automatic fallback fonts for non-WinAnsi text
  (`html.WithFallbackFontPath`), opt-in remote assets (`html.WithRemoteAssets`,
  `html.WithURLPolicy`, `html.WithHTTPClient`), strict asset errors
  (`html.WithStrictAssets`), `!important` cascade handling, and typed
  `html.ParseError`/`html.AssetError`/`html.LimitError` surfaces.

### Fixed

- Fixed merged PDFs rendering blank: `merge.Bytes` now materializes inherited
  page-tree attributes (`/MediaBox`, `/Resources`, `/Rotate`, crop boxes) onto
  each copied page.
- Fixed RC4-protected documents corrupting every Info string after the first
  (per-string keystream), non-ASCII outline titles showing mojibake (UTF-16
  encoding), page-number aliases rendering missing glyphs with UTF-8 fonts,
  `SetAlpha` emitting an invalid blend mode for the empty string, XMP metadata
  never being referenced from the catalog, bookmark destinations on mixed page
  sizes, PDF dates missing timezone offsets, and a potential panic embedding
  uncompressed Type1 fonts.
- Fixed silent text corruption: core-font Unicode translation is now
  case-insensitive on the family name, and characters outside cp1252 are
  reported through `GetReport()` instead of being replaced silently.
- Fixed text layout issues: dash-break line caching, trailing-space alignment
  at wrap points, dotted cell borders rendering dashed, checkbox label
  encoding and centering, CSS-correct justify (last line not stretched), and
  `props.Text` mutation during rendering.
- Fixed HTML/CSS fidelity: `rem`/`in` units, `&nbsp;` preservation,
  deterministic inline shorthand expansion, `line-height` with units or
  percentages, `border` shorthand color functions and width keywords,
  `font-size` percentages/keywords, modern color syntax (`rgb(255 0 0 / .5)`,
  hue units), `<tfoot>` ordering, body-level inline grouping,
  `<ol start="0">`, CSS hex escapes in `content` strings, em-valued heights
  and border widths, and malformed box shorthands being dropped instead of
  zeroed.

### Security

- Upgraded `golang.org/x/image` to v0.43.0 (fixes GO-2026-5032, GO-2026-4961,
  GO-2026-5066, GO-2026-5062 and related decode panics) and pinned the Go
  toolchain to 1.26.5.

- Added richer HTML/CSS rendering support for inline boxes, shadows, pseudo-element
  image content, flex/table layout, page-break controls, and typography details
  including semibold, line-height, letter-spacing, vertical-align, and `nowrap`.
- Added public controls for positioned page images, first-page foreground/final
  foreground overlays, page-number offsets, table width/alignment/border-spacing,
  and pagination directives used by HTML translation.
- Added a public PDF outline (bookmarks) API: `props.Outline` on `props.Text`
  and the richtext paragraph prop adds entries to the viewer's bookmark
  sidebar (`Level` 0-n nesting, optional `Title` override).
- Added `config.WithOutlineFromHeadings(true)`: HTML conversion turns `h1`-`h6`
  headings into outline entries (also `html.WithOutlineFromHeadings()` for the
  rows-only API). Hidden headings are skipped.
- Added outline preservation to `merge.Bytes`: merged documents now carry a
  rebuilt outline tree, so bookmarks survive concurrent and low-memory
  generation modes.
- Added `config.WithWatermark(text, ...props.Watermark)`: translucent diagonal
  text stamped under the content of every page (auto-scales to the page
  diagonal; defaults 48pt / 12% alpha / 45°).
- Added CSS `@page` support (`size` with named sizes, explicit dimensions, and
  `landscape`; `margin` shorthand and per-side) applied by `paper.FromHTML`
  and `paper.FromHTMLReader` when no explicit config is passed.
- Added repeating HTML page headers/footers: the first top-level `<header>` /
  `<footer>` in `FromHTML`/`AddHTML` registers as the document header/footer.
- Added `html.DocumentFromString`/`DocumentFromReader` returning rows plus
  parsed `@page` options and header/footer rows.
- Added a committed `go.work` workspace, `DEVELOPMENT.md` contributor guide,
  and `RELEASING.md` release checklist.
- Added a benchstat-based informational benchmark comparison workflow for
  pull requests, a Go version matrix and module caching in CI, and a blocking
  dead-code check.
- Added release engineering scaffolding for the first tagged release.
- Added nested module coverage in CI for `pkg/test`, `examples`, and docs examples.
- Added `govulncheck` CI coverage for pull requests, manual runs, and weekly scheduled scans.
- Added `core.Document.Write(io.Writer)` and `(*core.Pdf).Write` for HTTP-handler friendly output.
- Added context-aware generation and HTML translation: generation observes
  ctx between pages and phases, HTML translation at parse/translate
  boundaries, and `merge.Bytes` between documents.
- Added default resource limits for HTML input, including image, SVG, DOM, and CSS rule caps.
- Added opt-in AES-128 PDF protection through `WithProtectionAlgorithm(protection.AES128)`.

### Changed

- **BREAKING:** the public API is now context-first. Every potentially
  long-running operation takes a `context.Context` as its first parameter and
  the transitional `*Ctx` variants were removed:

  | Old | New |
  | --- | --- |
  | `(*Paper).Generate()` / `GenerateCtx(ctx)` | `(*Paper).Generate(ctx)` |
  | `(*Paper).AddHTML(s)` / `AddHTMLCtx(ctx, s)` | `(*Paper).AddHTML(ctx, s)` |
  | `paper.FromHTML(s, …)` / `FromHTMLCtx(ctx, s, …)` | `paper.FromHTML(ctx, s, …)` |
  | `paper.FromHTMLReader(r, …)` / `FromHTMLReaderCtx(ctx, r, …)` | `paper.FromHTMLReader(ctx, r, …)` |
  | `html.FromString(s, …)` / `FromStringCtx(ctx, s, …)` | `html.FromString(ctx, s, …)` |
  | `html.FromReader(r, …)` / `FromReaderCtx(ctx, r, …)` | `html.FromReader(ctx, r, …)` |
  | `html.DocumentFromString(s, …)` / `DocumentFromStringCtx(ctx, s, …)` | `html.DocumentFromString(ctx, s, …)` |
  | `html.DocumentFromReader(r, …)` / `DocumentFromReaderCtx(ctx, r, …)` | `html.DocumentFromReader(ctx, r, …)` |
  | `translate.Translate(doc, …)` / `TranslateCtx(ctx, doc, …)` | `translate.Translate(ctx, doc, …)` |
  | `translate.TranslateDocument(doc, …)` / `TranslateDocumentCtx(ctx, doc, …)` | `translate.TranslateDocument(ctx, doc, …)` |
  | `merge.Bytes(pdfs…)` | `merge.Bytes(ctx, pdfs…)` |
  | `(*core.Pdf).Merge(b)` | `(*core.Pdf).Merge(ctx, b)` |
  | `htmlcomponent.New(s, …)` / `NewCol` / `NewRow` / `NewAutoRow` | same names with `ctx` as first parameter |

  The `core.Paper` and `core.Document` interfaces changed accordingly, as did
  the `decorator.Metrics` wrapper. Callers that do not need cancellation can
  pass `context.Background()`. A nil context is not supported (per staticcheck
  SA1012) and will panic.
- **BREAKING:** consolidated eight one-type packages under `pkg/consts/*`
  into the single `pkg/consts` package. Constant string values are unchanged;
  only Go identifiers moved:

  | Old | New |
  | --- | --- |
  | `align.Type` / `align.Left`, `Center`, … | `consts.Align` / `consts.AlignLeft`, `consts.AlignCenter`, … |
  | `align.Justify` (untyped) | `consts.AlignJustify` (typed `consts.Align`) |
  | `orientation.Type` / `Vertical`, `Horizontal` | `consts.Orientation` / `consts.OrientationVertical`, `consts.OrientationHorizontal` |
  | `linestyle.Type` / `Solid`, `Dashed`, `Dotted` | `consts.LineStyle` / `consts.LineStyleSolid`, `consts.LineStyleDashed`, `consts.LineStyleDotted` |
  | `linestyle.DefaultLineThickness` | `consts.DefaultLineThickness` |
  | `breakline.Strategy` / `EmptySpaceStrategy`, `DashStrategy` | `consts.BreakLineStrategy` / `consts.BreakLineEmptySpace`, `consts.BreakLineDash` |
  | `fontfamily.Arial`, `Helvetica`, `Symbol`, `ZapBats`, `Courier` | `consts.FontFamilyArial`, …`Helvetica`, …`Symbol`, …`ZapBats`, …`Courier` |
  | `barcode.Type` / `Code128`, `EAN` | `consts.BarcodeType` / `consts.BarcodeCode128`, `consts.BarcodeEAN` |
  | `generation.Mode` / `Sequential`, `Concurrent`, `SequentialLowMemory` | `consts.GenerationMode` / `consts.GenerationSequential`, `consts.GenerationConcurrent`, `consts.GenerationSequentialLowMemory` |
  | `provider.Type` / `Paper` | `consts.ProviderType` / `consts.ProviderPaper` |

  `pkg/consts/{border,extension,fontstyle,pagesize,protection}` are unchanged.
- **BREAKING:** top-level `<header>`/`<footer>` elements (direct children of
  `<body>`) now become the repeating page header/footer in `paper.FromHTML`
  and `paper.AddHTML` instead of rendering inline once. Wrap the element in a
  `<div>` (or use `<section>`) to keep the old inline rendering. The rows-only
  `html.FromString` API is unchanged.
- **BREAKING:** folded the `pkg/test` nested module back into the root module
  (supersedes the earlier nested-module split for `pkg/test` below). It is now
  a dependency-free re-export of the internal golden-structure helper and its
  testify dependency was removed. The import path is unchanged
  (`github.com/avdoseferovic/paper/pkg/test`), no separate `go get` is needed,
  and the historical `pkg/test/v0.1.0` tag remains usable.
- Renamed the current-page fit check method to `FitInCurrentPage`.
- Split `pkg/test`, `examples`, and `docs` into nested Go modules to keep root consumer dependencies and module downloads lean.
- Moved generated mocks under `internal/mocks` and removed the root-level generated `mocks` package.
- Moved the metrics decorator into `pkg/decorator`.
- Changed `New` to return `*paper.Paper`.
- Changed document-generation helpers to return concrete `*core.Pdf` values where appropriate.
- Collapsed the internal gofpdf wrapper interface and pruned unused legacy PDF internals.
- Updated the supported Go toolchain to 1.26.4 and refreshed vulnerable dependency versions used by the HTML/image paths.
- Renamed root package files to drop the redundant `paper_` prefix
  (`generation.go`, `html.go`, `page_builder.go`, and their tests) and
  consolidated small root files: `GetStructure` moved into `paper.go`,
  header/`@page` tests merged into `html_test.go`, outline/watermark tests
  merged into `generation_test.go`, and white-box tests named after what they
  test (`page_groups_test.go`, `page_options_test.go`). No API change.

### Removed

- Removed root-module demo binaries from `cmd`; demos now live in the nested `examples` module.
- Removed the root-module `pkg/test` package; it is now the separate `github.com/avdoseferovic/paper/pkg/test` module.

### Security

- Documented the current PDF protection behavior as RC4-based protection, not confidentiality-grade encryption.
- Kept RC4 as the compatibility default and documented AES-128 as the preferred option for new protected documents.
- Verified AES-128 protected output with `pdfcpu validate` during release preparation.
