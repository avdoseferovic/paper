# Remove Pdfcpu Gofpdf And Rename To Paper Implementation Plan

Created: 2026-06-05
Status: PENDING
Approved: Yes
Iterations: 0
Worktree: No
Type: Feature

> **Status Lifecycle:** PENDING -> COMPLETE -> VERIFIED
> **Iterations:** Tracks implement->verify cycles.
>
> - PENDING: Initial state, awaiting implementation
> - COMPLETE: All tasks implemented
> - VERIFIED: All checks passed
>
> **Approval Gate:** Implementation CANNOT proceed until `Approved: Yes`
> **Worktree:** No. Work happens in the current workspace.
> **Type:** Feature

## Summary

**Goal:** Remove `github.com/pdfcpu/pdfcpu` and `github.com/phpdave11/gofpdf` as module dependencies, rename the library from Maroto to Paper, and preserve PDF generation behavior.

**Architecture:** Treat dependency removal as true module removal, not a `replace` directive. The current GoFPDF runtime behavior will be preserved by internalizing a pruned, licensed copy of the currently used GoFPDF root package under a project-owned Paper backend package, then wiring the provider to that local backend. `pdfcpu` will be replaced with a focused in-repo PDF merger that supports the standard unencrypted xref-table PDFs produced by Paper and returns the existing wrapped error for unsupported or invalid inputs.

**Tech Stack:** Go 1.26.1, standard library PDF parsing/writing for merge, existing public component/config/core packages, generated mocks through mockery.

## Scope

### In Scope

- Remove direct imports and `go.mod`/`go.sum` entries for `github.com/pdfcpu/pdfcpu` and `github.com/phpdave11/gofpdf`.
- Replace GoFPDF type leaks in core interfaces with Paper-owned types.
- Rename the concrete provider from `gofpdf` to `paper` and update provider constants/default config.
- Internalize only the GoFPDF runtime source and assets needed by the existing provider, preserving upstream license/attribution.
- Replace `pkg/merge.Bytes` with an in-repo merger that preserves current public error behavior and supports generated Paper PDFs.
- Rename the module, root package, exported builder interface/type names, tests, examples, docs, and mockery config to Paper.
- Regenerate mocks after interface/module changes.
- Add dependency guard tests or verification checks so `pdfcpu`, `phpdave11/gofpdf`, and the old module path cannot silently reappear.

### Out of Scope

- Publishing or renaming the GitHub remote/repository outside this workspace.
- Regenerating static PDFs under `docs/assets/pdf/**` unless a test explicitly depends on it.
- Rewriting deprecated/historical docs such as `docs/v1/**` or prior files under `docs/plans/**`, except for adding an explicit migration note if needed.
- Preserving pdfcpu's full arbitrary-PDF merge compatibility for encrypted PDFs, object streams, malformed files, or advanced PDF features. Unsupported inputs must fail with `merge.ErrCannotMergePDFs`.
- Adding a new third-party PDF engine dependency.
- Keeping the old `github.com/johnfercher/maroto/v2` module path working after the rename.

## Prerequisites

- Baseline status: `go test ./...` passed on 2026-06-05 before implementation.
- The Go module cache contains the currently used GoFPDF source at `/Users/avdo/go/pkg/mod/github.com/phpdave11/gofpdf@v1.4.3`.
- Any internalized third-party source must keep its license text and clear attribution.
- Mock generation uses `.mockery.yaml`; update it before regenerating mocks.

## Context for Implementer

- **Patterns to follow:** Existing provider capability interfaces in `pkg/core/*_provider.go` use narrow interfaces and safe type assertions. Keep that pattern.
- **Conventions:** Standard Go tests with testify; generated mocks live in `mocks/`; run `gofmt`; avoid hand-editing generated mocks when mockery can regenerate them.
- **Key files:** `maroto.go` is the root builder implementation; `pkg/core/core.go` defines the public builder/document interfaces; `internal/providers/gofpdf/**` is the concrete backend; `pkg/merge/merge.go` is the only `pdfcpu` call site; `.mockery.yaml` pins the old module path; `.golangci.yml` and `.github/skills/generate-unit-tests.md` also reference old module/provider names.
- **Gotchas:** `pkg/core/components.go` currently exposes `*gofpdf.ImageInfoType`; remove that first so no public-ish core package depends on backend implementation types. `merge.Bytes` is used by concurrent generation, low-memory generation, and `core.Pdf.Merge`, so it cannot be dropped. The current default provider value is `provider.Gofpdf`, and config snapshots include `config_provider_type`. Protected chunked generation should not merge encrypted chunk PDFs; when protection is configured, concurrent/low-memory modes must fall back to sequential protected generation or otherwise produce a documented wrapped error. This plan chooses sequential fallback to preserve successful PDF generation.
- **Domain context:** Paper builds a logical component tree first, then renders pages through a provider. The provider's PDF page creation must remain consistent with `maroto.go`/future `paper.go` pagination logic, especially for sequential, concurrent, and low-memory generation.

## Feature Inventory

### Files Being Replaced

| Old File | Functions/Types/Constants | Mapped to Task |
| --- | --- | --- |
| `pkg/merge/merge.go` | `ErrCannotMergePDFs`, `Bytes`, `mergePdfs` | Task 3 |
| `internal/providers/gofpdf/builder.go` | `Dependencies`, `Builder`, `builder`, `NewBuilder`, `Build` | Task 2 |
| `internal/providers/gofpdf/provider.go` | `ErrCannotReadImageOptions`, provider compile-time assertions, `provider`, all provider methods (`AddText`, `AddImageFromBytes`, `GenerateBytes`, etc.) | Task 2 |
| `internal/providers/gofpdf/gofpdfwrapper/fpdf.go` | `Fpdf`, `NewCustom` | Task 2 |
| `internal/providers/gofpdf/image.go` | `ErrCouldNotRegisterImageOptions`, `Image`, `NewImage`, `GetImageInfo`, `Add` | Task 1, Task 2 |
| `internal/providers/gofpdf/text.go` and `richtext.go` | `Text`, `NewText`, `Add`, `GetLinesQuantity`, rich text measurement/render helpers | Task 2 |
| `internal/providers/gofpdf/line.go`, `checkbox.go`, `font.go`, `gradient.go`, `parseimage.go` | line, checkbox, font, gradient, and image parsing runtime behavior | Task 2 |
| `internal/providers/gofpdf/cellwriter/*.go` | cell fill, border, outline, radius, shadow, gradient writer chain | Task 2 |
| `pkg/core/components.go` | `core.Image` backend type dependency | Task 1 |
| `pkg/consts/provider/type.go`, `pkg/config/builder.go`, `pkg/core/entity/config.go` tests/snapshots | default provider name `gofpdf` | Task 4 |
| `maroto.go`, `metricsdecorator.go`, `pkg/core/core.go`, root tests/examples | root package/type/interface names | Task 4 |
| `go.mod`, `go.sum`, `.mockery.yaml`, `.golangci.yml`, `.github/skills/generate-unit-tests.md`, `mocks/*.go` | module path, lint/test-generation config, dependencies, generated mock imports | Task 4, Task 5 |
| `README.md`, `docs/**`, `cmd/**`, `docs/assets/examples/**` | user-facing Maroto import/name examples | Task 5 |

### Feature Mapping Verification

- [x] All old files being replaced are listed above
- [x] All functions/classes/constants are identified by file or file group
- [x] Every replaced feature maps to Task 1, 2, 3, 4, or 5
- [x] No features are intentionally removed without an explicit scope note

## Progress Tracking

**MANDATORY: Update this checklist as tasks complete. Change `[ ]` to `[x]`.**

- [x] Task 1: Remove backend type leaks from core interfaces
- [x] Task 2: Internalize and rename the PDF provider backend
- [x] Task 3: Replace pdfcpu merge with an in-repo merger
- [x] Task 4: Rename the Go module and public API to Paper
- [ ] Task 5: Regenerate mocks, update docs, and add final dependency guards

**Total Tasks:** 5 | **Completed:** 4 | **Remaining:** 1

## Implementation Tasks

### Task 1: Remove Backend Type Leaks From Core Interfaces

**Objective:** Replace the `gofpdf.ImageInfoType` dependency in `pkg/core/components.go` with a Paper-owned image dimension contract. This makes the core/component layer independent from the concrete PDF backend before removing the module.

**Dependencies:** None

**Files:**

- Modify: `pkg/core/components.go`
- Modify: `internal/providers/gofpdf/image.go`
- Modify: `internal/providers/gofpdf/provider.go`
- Modify: `internal/providers/gofpdf/image_test.go`
- Modify: `internal/providers/gofpdf/provider_test.go`
- Regenerate: `mocks/Image.go`

**Key Decisions / Notes:**

- Prefer changing `core.Image.GetImageInfo` to a project-owned method such as `GetImageDimensions(img *entity.Image, extension extension.Type) (*entity.Dimensions, uuid.UUID)` rather than introducing a public alias of backend internals.
- Keep provider behavior the same: failed image registration still returns `ErrCannotReadImageOptions`; successful registration returns width/height in millimeters.
- Update tests to assert dimensions, not backend-specific `ImageInfoType`.
- Regenerate mocks instead of hand-editing generated code.

**Definition of Done:**

- [x] `pkg/core/components.go` imports no GoFPDF package.
- [x] `core.Image` exposes only project-owned types.
- [x] Image dimension tests still cover file bytes, invalid bytes, QR code, matrix code, and image file paths.
- [x] Generated mocks compile with the new image interface.

**Verify:**

- `go test ./internal/providers/gofpdf ./pkg/core ./mocks`
- `rg -n "github.com/phpdave11/gofpdf|gofpdf\\.ImageInfoType" pkg/core mocks/Image.go`

### Task 2: Internalize And Rename The PDF Provider Backend

**Objective:** Remove the external GoFPDF module by copying/pruning the runtime backend source into a Paper-owned internal package and renaming the provider package from `internal/providers/gofpdf` to `internal/providers/paper`.

**Dependencies:** Task 1

**Commit:** `refactor(pdf): internalize paper backend`

**Files:**

- Create: `internal/paperpdf/` or `internal/pdf/backend/` with pruned runtime source from `/Users/avdo/go/pkg/mod/github.com/phpdave11/gofpdf@v1.4.3`
- Create: `internal/paperpdf/LICENSE` or `internal/paperpdf/NOTICE` with upstream attribution
- Move/Modify: `internal/providers/gofpdf/**` -> `internal/providers/paper/**`
- Modify: `maroto.go` (later renamed in Task 4) provider import and `getProvider`
- Modify: `pkg/consts/provider/type.go`
- Modify: `pkg/config/builder.go`
- Modify: `pkg/config/builder_test.go`
- Modify: `pkg/core/entity/config_test.go`
- Regenerate: `mocks/*.go`

**Key Decisions / Notes:**

- Copy the full root GoFPDF runtime package first (`*.go`, embedded font data, font maps, and root runtime assets), excluding upstream tests, reference PDFs, contrib packages, examples, and docs. Only prune after targeted rendering tests prove fonts, images, SVG/basic helpers, templates, custom fonts, gradients, and image registration still work.
- Keep copied package imports on the current module path during this task so Task 2 can compile before the module rename. Task 4 performs the global module path change to `github.com/johnfercher/paper/v2`.
- Preserve the wrapper API shape so provider code changes stay mostly mechanical.
- Rename provider constants from `provider.Gofpdf`/`"gofpdf"` to `provider.Paper`/`"paper"`. Do not keep a `provider.Gofpdf` alias; the final guard should be able to reject it.
- Keep behavior for metadata, protection, compression, pages, images, gradients, alpha, links, rich text, line styles, fonts, and cell styling.

**Definition of Done:**

- [x] No Go source imports `github.com/phpdave11/gofpdf`.
- [x] `go.mod` no longer directly or indirectly requires `github.com/phpdave11/gofpdf` after `go mod tidy`.
- [x] Provider tests pass under the renamed `internal/providers/paper` package.
- [x] Default config provider type is `provider.Paper` and snapshots/tests expect `"paper"`.
- [x] Internalized backend carries upstream license/attribution.

**Verify:**

- `go test ./internal/providers/paper/... ./pkg/config ./pkg/core/entity`
- `go mod tidy`
- `rg -n "github.com/phpdave11/gofpdf|provider\\.Gofpdf|\\\"gofpdf\\\"" --glob '*.go' go.mod go.sum`

### Task 3: Replace Pdfcpu Merge With An In-Repo Merger

**Objective:** Remove `pdfcpu` while preserving `merge.Bytes`, `core.Pdf.Merge`, concurrent generation, and low-memory generation for Paper-generated PDFs.

**Dependencies:** Task 2

**Commit:** `refactor(merge): replace pdfcpu merger`

**Files:**

- Modify: `pkg/merge/merge.go`
- Modify: `pkg/merge/merge_test.go`
- Modify: `pkg/core/pdf_test.go`
- Modify: `maroto_test.go`
- Create as needed: `pkg/merge/parser.go`, `pkg/merge/writer.go`, `pkg/merge/object.go`

**Key Decisions / Notes:**

- Implement a focused PDF merger using standard library only:
  - Parse unencrypted `%PDF-` inputs with classic xref tables.
  - Read indirect objects, renumber them, rewrite references, and build a new Pages tree.
  - Preserve page content/resources enough for Paper-generated PDFs.
  - Reject unsupported PDFs with `fmt.Errorf("%w: ...", ErrCannotMergePDFs)`.
- Keep the public API and error sentinel unchanged: callers still use `merge.Bytes` and can `errors.Is(err, merge.ErrCannotMergePDFs)`.
- Preserve protected PDF generation by avoiding encrypted chunk merging: when `cfg.Protection != nil` and the selected generation mode is concurrent or sequential-low-memory, route through the sequential generation path. Add tests that `WithProtection(...).WithConcurrentMode(...)` and `WithProtection(...).WithSequentialLowMemoryMode(...)` still return valid `%PDF-` bytes.
- Add stronger tests than the current byte-length delta:
  - merged bytes start with `%PDF-`;
  - a parser-level assertion traverses Catalog -> Pages -> Kids and verifies the page count;
  - every page's `Contents` and `Resources` references resolve after renumbering;
  - merged outputs cover text, images, custom fonts, gradients/backgrounds, links, compression on/off, concurrent generation, and sequential-low-memory generation;
  - invalid bytes return `ErrCannotMergePDFs`;
  - concurrent and sequential-low-memory generation produce valid PDF bytes for multi-page documents;
  - `core.Pdf.Merge` updates document bytes and metrics as before.

**Definition of Done:**

- [x] `pkg/merge` has no `pdfcpu` import.
- [x] Generated one-page PDFs merge into a valid two-page PDF with a traversable Catalog/Pages/Kids graph.
- [x] Every merged page's `Contents` and `Resources` references resolve in parser-level tests.
- [x] Merge tests cover Paper-generated PDFs containing text, images, custom fonts, gradients/backgrounds, links, and compression on/off.
- [x] Invalid input returns an error wrapping `merge.ErrCannotMergePDFs`.
- [x] Concurrent and low-memory generation still succeed and return `%PDF-` bytes.
- [x] Protected concurrent and protected low-memory generation fall back to sequential protected generation and still return `%PDF-` bytes.
- [x] `core.Pdf.Merge` still wraps failures in `core.ErrCannotMergeBytes` and updates bytes on success.

**Verify:**

- `go test ./pkg/merge ./pkg/core .`
- `rg -n "github.com/pdfcpu|pdfcpu\\." --glob '*.go' go.mod go.sum`

### Task 4: Rename The Go Module And Public API To Paper

**Objective:** Rename the library from Maroto to Paper at the module, root package, exported builder/interface, snapshots, tests, examples, and command import sites.

**Dependencies:** Task 2, Task 3

**Commit:** `refactor: rename maroto to paper`

**Files:**

- Modify: `go.mod`
- Move/Modify: `maroto.go` -> `paper.go`
- Modify: `metricsdecorator.go`
- Modify: `pkg/core/core.go`
- Modify: `pkg/test/test.go`, `pkg/test/config.go`, `.maroto.yml` -> `.paper.yml`
- Modify: all Go imports from `github.com/johnfercher/maroto/v2` to `github.com/johnfercher/paper/v2`
- Modify: package declarations in root tests from `maroto_test` to `paper_test`
- Move/Modify: `test/maroto/**` -> `test/paper/**`
- Modify: `.mockery.yaml`
- Modify: `.golangci.yml`
- Modify: `.github/skills/generate-unit-tests.md`
- Regenerate: `mocks/*.go`

**Key Decisions / Notes:**

- Assumption for approval: module path becomes `github.com/johnfercher/paper/v2`. If the owner path should instead be `github.com/avdoseferovic/paper/v2`, change this task before implementation.
- Rename `type Maroto` to `type Paper` and `core.Maroto` to `core.Paper`.
- Keep `New`, `FromHTML`, `FromHTMLReader`, and `NewMetricsDecorator` as the main constructor API, returning `core.Paper`.
- Do not keep old `Maroto` type/interface/package aliases in active code; final guards should reject `core.Maroto`, `*maroto.Maroto`, `package maroto`, and the old module import path.
- Update comments and error prefixes from `maroto:` to `paper:` where user-visible.
- Rename the public test helper workflow: `.maroto.yml` -> `.paper.yml`, `MarotoTest` -> `PaperTest`, `ErrMarotoYMLNotFound` -> `ErrPaperYMLNotFound`, fixture roots from `test/maroto/**` to `test/paper/**`, and snapshot root structure/config keys from `maroto_*` to `paper_*` where generated by current code.

**Definition of Done:**

- [x] `go.mod` module path is `github.com/johnfercher/paper/v2`.
- [x] Root package declaration is `package paper`.
- [x] `paper.New()` returns a `core.Paper`.
- [x] Root concrete type is `*paper.Paper`; tests no longer expect `*maroto.Maroto`.
- [x] Structure snapshots use root type `"paper"` for new output.
- [x] The test helper looks for `.paper.yml`, exposes `PaperTest`, and fixture paths live under `test/paper/**`.
- [x] No active Go import path points at `github.com/johnfercher/maroto/v2`.
- [x] Mockery config uses the new module path and generated mocks compile.
- [x] Lint config and local skill instructions reference the new Paper module/import names.

**Verify:**

- `go test ./...`
- `rg -n "github.com/johnfercher/maroto/v2|package maroto|\\*maroto\\.Maroto|core\\.Maroto" --glob '*.go' .mockery.yaml .golangci.yml .github/skills`
- `go test ./pkg/test ./pkg/config .`

### Task 5: Regenerate Mocks, Update Docs, And Add Final Dependency Guards

**Objective:** Finish the rename and dependency removal by updating user-facing docs/examples, regenerating generated files, and adding checks that prevent old dependencies or names from returning unnoticed.

**Dependencies:** Task 4

**Commit:** `docs: rename maroto examples to paper`

**Files:**

- Modify: `README.md`
- Modify: `docs/README.md`, `docs/v2/**`, and active docs metadata where examples mention Maroto
- Modify: `docs/assets/examples/**`
- Modify: `cmd/html-demo/**`, `cmd/survey-report/**`, `cmd/dev/pdf/main.go`, `cmd/benchmark/main.go`
- Modify: repo metadata/support files under `.github/**`, `pull_request_template.md`, `docs/index.html`, `docs/_*.md`, and other active non-historical text that advertises the old name or module path
- Modify/Create: dependency guard test or script, for example `internal/dependency/dependency_test.go` or `shell/dependency-check.sh`
- Modify: `Makefile` if guard is added to `dod`
- Regenerate: `mocks/*.go`

**Key Decisions / Notes:**

- Documentation should refer to Paper as the active library name. Mention Maroto only in a migration note if needed.
- Split dependency verification into semantic dependency checks plus scoped text checks:
  - `go list -m all` must not include `github.com/pdfcpu/pdfcpu` or `github.com/phpdave11/gofpdf`;
  - `go list -deps ./...` must not include `github.com/pdfcpu/pdfcpu`, `github.com/phpdave11/gofpdf`, or `github.com/johnfercher/maroto/v2`;
  - scoped text checks scan module files, Go source, README, active docs/examples/cmd, `.mockery.yaml`, `.golangci.yml`, `.github/**`, and repo templates for:
  - `github.com/pdfcpu/pdfcpu`
  - `github.com/phpdave11/gofpdf`
  - `github.com/johnfercher/maroto/v2`
- The guard must explicitly exclude `docs/plans/**`, deprecated legacy docs such as `docs/v1/**`, migration notes that intentionally mention old names, binary/static PDFs under `docs/assets/pdf/**`, and `internal/paperpdf/{LICENSE,NOTICE}` because attribution may cite upstream GoFPDF.
- Keep `docs/assets/pdf/**` binary PDFs unchanged unless tests or docs commands regenerate them.
- Run `gofmt` after mechanical rewrites and mock regeneration.

**Definition of Done:**

- [ ] README and active v2 docs show `paper` imports and terminology.
- [ ] Examples and commands compile with the new module path.
- [ ] Semantic dependency checks fail if `pdfcpu`, `phpdave11/gofpdf`, or the old module path remain in Go dependencies/imports.
- [ ] Scoped text guard fails if forbidden import paths remain in active source/docs/config outside explicit historical/attribution exceptions.
- [ ] `go.mod` and `go.sum` are tidy.
- [ ] Full test suite passes.

**Verify:**

- `go test ./...`
- `go test ./docs/assets/examples/...`
- `go build ./...`
- `go list -m all | rg "github.com/pdfcpu/pdfcpu|github.com/phpdave11/gofpdf"` — should return no matches
- `go list -deps ./... | rg "github.com/pdfcpu/pdfcpu|github.com/phpdave11/gofpdf|github.com/johnfercher/maroto/v2"` — should return no matches
- `rg -n "github.com/pdfcpu/pdfcpu|github.com/phpdave11/gofpdf|github.com/johnfercher/maroto/v2" go.mod go.sum --glob '*.go' README.md docs/README.md docs/v2 docs/assets/examples cmd .mockery.yaml .golangci.yml .github pull_request_template.md` — should return no matches
- `rg -n "\\bMaroto\\b|github.com/johnfercher/maroto|\\bmaroto\\b" README.md docs/README.md docs/v2 docs/assets/examples cmd .github pull_request_template.md` — should return no active user-facing references except explicit migration-note exceptions

## Testing Strategy

- Unit tests: provider image dimensions, provider rendering methods, merge parser/writer, config/provider constants, root package constructors.
- Integration tests: `paper.New().Generate()`, `paper.FromHTML`, `core.Pdf.Merge`, concurrent generation, sequential-low-memory generation, docs examples.
- Dependency tests: `go list -m all`, `go list -deps ./...`, and scoped source/docs/config scans for forbidden imports and old module path, with explicit exceptions for historical plans, legacy docs, migration notes, binary PDFs, and upstream license/notice attribution.
- Manual verification: optional run of `go run ./cmd/html-demo` and `go run ./cmd/survey-report` to regenerate sample PDFs after tests pass.

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Internalized backend accidentally omits a file or font asset required at runtime | Medium | High | Copy root runtime source/assets first, prune only after `go test ./...` proves the provider still works, and keep upstream license/notice in the copied package. |
| Focused merger is less capable than pdfcpu | High | Medium | Explicitly support Paper-generated standard xref-table PDFs, add tests for all in-repo merge call sites, and return `ErrCannotMergePDFs` for unsupported inputs. |
| Merged PDF can look valid by header/count but contain broken references | Medium | High | Add parser-level validation that traverses Catalog/Pages/Kids and verifies every page's Contents and Resources references resolve across representative text/image/font/gradient/link/compression outputs. |
| Protected chunked generation would otherwise require merging encrypted PDFs | Medium | Medium | If protection is configured with concurrent or sequential-low-memory mode, route generation through the sequential protected path and test both combinations return valid PDF bytes. |
| Public rename breaks generated mocks and snapshots | High | Medium | Update `.mockery.yaml`, regenerate mocks, and update snapshots in the same task as the module/API rename. |
| Old dependency paths return through generated code or docs examples | Medium | Medium | Add a dependency guard test/script scanning `go.mod`, `go.sum`, Go source, docs, examples, and mockery config. |
| Huge mechanical rename obscures behavior regressions | Medium | High | Keep implementation milestones separate: backend type boundary, backend dependency removal, merge replacement, public rename, docs/guards. Run targeted tests after each milestone and full `go test ./...` at the end. |

## Goal Verification

### Truths

- Users import `github.com/johnfercher/paper/v2`, call `paper.New()`, and generate valid PDF bytes.
- `go.mod`, `go.sum`, and Go source contain no `github.com/pdfcpu/pdfcpu` or `github.com/phpdave11/gofpdf` dependency/import.
- Sequential, concurrent, and sequential-low-memory generation modes still produce valid PDF bytes for multi-page documents.
- `merge.Bytes` and `core.Pdf.Merge` still merge Paper-generated PDFs and wrap invalid input errors with the existing sentinels.
- Active docs, examples, and command packages use Paper naming and compile.

### Artifacts

- `go.mod` - module path and dependency list.
- `internal/paperpdf/**` or `internal/pdf/backend/**` - project-owned internal PDF backend with license attribution.
- `internal/providers/paper/**` - renamed provider implementation wired to the internal backend.
- `pkg/merge/**` - in-repo merger implementation and tests.
- `paper.go`, `metricsdecorator.go`, `pkg/core/core.go` - public Paper API.
- `.mockery.yaml`, `mocks/*.go` - regenerated mocks for the new module path and interfaces.
- `README.md`, `docs/**`, `docs/assets/examples/**`, `cmd/**` - user-facing rename coverage.
- `.golangci.yml`, `.github/skills/generate-unit-tests.md` - local support/config files updated for the new module name.
- `.paper.yml`, `test/paper/**`, `pkg/test/**` - renamed test helper workflow and fixtures.

### Key Links

- `paper.New()` -> `getProvider()` -> `internal/providers/paper.NewBuilder()` -> internal Paper PDF backend.
- `core.Image` -> provider image implementation -> project-owned dimensions type, with no backend type leak.
- `merge.Bytes()` -> in-repo PDF merger -> `Paper.generateConcurrently`, `Paper.generateLowMemory`, and `core.Pdf.Merge`.
- protected config -> sequential fallback -> protected PDF bytes without encrypted chunk merging.
- `.mockery.yaml` -> regenerated `mocks/*.go` -> provider/core tests.
- Dependency guard -> `go list -m all`, `go list -deps ./...`, and scoped source/docs/config scans.

## Open Questions

- Approval of this plan also confirms the assumed new module path `github.com/johnfercher/paper/v2`. If the intended path is `github.com/avdoseferovic/paper/v2` or another owner, update Task 4 before implementation.
- The merge replacement intentionally targets Paper-generated and standard unencrypted xref-table PDFs, not the full pdfcpu feature set. Protected chunked generation remains successful through sequential fallback rather than encrypted chunk merge support.

### Deferred Ideas

- Add a formal migration guide from Maroto to Paper after the code rename lands.
- Add PDF page-count validation through a dedicated test helper if the in-repo merger grows beyond Paper-generated PDFs.
