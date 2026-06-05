# Analyse Unused Dead Code Implementation Plan

Created: 2026-06-05
Status: PENDING
Approved: Yes
Iterations: 0
Worktree: No
Type: Feature

> **Status Lifecycle:** PENDING -> COMPLETE -> VERIFIED
> **Iterations:** Tracks implement->verify cycles (incremented by verify phase)
>
> - PENDING: Initial state, awaiting implementation
> - COMPLETE: All tasks implemented
> - VERIFIED: All checks passed
>
> **Approval Gate:** Implementation CANNOT proceed until `Approved: Yes`
> **Worktree:** Set at plan creation. `No` works directly on the current branch.
> **Type:** `Feature` or `Bugfix` - set at planning time, used by dispatcher for routing

## Summary

**Goal:** Analyse unused and dead code, remove code that is mechanically proven dead, and make the project unversioned so the module, docs, examples, generated assets, and active references use `github.com/avdoseferovic/paper` with no `v1` or `v2` surface.

**Architecture:** Treat this as a breaking cleanup because the user confirmed nobody depends on the project yet. First remove the semantic-import-version suffix and versioned docs/example layout so all packages build under `github.com/avdoseferovic/paper`. Then remove dead code identified by `deadcode -test ./...`, separating active source from doc-only examples and third-party-derived backend code. Finally add guardrails that fail on reintroduced `/v2`, `docs/v1`, stale versioned output names, or active dead-code candidates.

**Tech Stack:** Go 1.26.1, `deadcode`, `go list`, `go test`, `go build`, `go vet`, `golangci-lint`, `go mod tidy`, existing mockery-generated mocks, existing docsify markdown/docs layout.

## Scope

### In Scope

- Change `go.mod` from `module github.com/avdoseferovic/paper/v2` to `module github.com/avdoseferovic/paper`.
- Rewrite all active Go imports, generated mocks, command packages, examples, tests, `.mockery.yaml`, `.github/skills/generate-unit-tests.md`, README/docs install snippets, and docs snippets from `/v2` to the unversioned module path.
- Remove legacy `docs/v1/**`.
- Move active docs from `docs/v2/**` to unversioned paths such as `docs/features/**`, `docs/examples/**`, and `docs/html-support.md`, updating docsify navigation and markdown links.
- Move docs examples from `docs/assets/examples/*/v2` to `docs/assets/examples/*` and update `Makefile` and docs includes.
- Rename tracked generated assets whose filenames contain `v2` to unversioned names, and update examples so future runs write the unversioned names.
- Remove active-source dead code reported by `deadcode -test ./...` where it is not a needed feature: stale row-level anchor source wrappers, superseded flex/hr helpers, `props.Green`, `PaperTest.Save`, and doc-only aggregate compatibility packages.
- Remove unexecuted `Example*` test files that deadcode reports as unreachable and that duplicate the maintained `docs/assets/examples/**` examples.
- Prune `internal/paperpdf` functions/files that `deadcode -test ./...` reports unreachable, but only in compile-guided batches with backend tests after each batch.
- Update dependency/source guardrails to enforce unversioned paths and absence of the removed dead-code/versioned artifacts.

### Out of Scope

- Preserving `github.com/avdoseferovic/paper/v2` import compatibility.
- Preserving `docs/v1/**`, v1 branch references, or v2-specific documentation labels.
- Renaming the local repository directory `/Users/avdo/maroto` or the GitHub remote.
- Removing live HTML/CSS rendering features, barcode/SVG/CSS/HTML parser dependencies, or the live `internal/paperpdf` rendering backend.
- Rewriting `internal/paperpdf` for style or Go idioms beyond deleting analyzer-proven unreachable code.
- Replacing generated mocks with hand-written mocks; import rewrites are acceptable, full regeneration is only needed if mockery output is easier and deterministic.
- Re-generating PDF bytes for visual changes. Asset filename/path cleanup should preserve rendering behavior.

## Prerequisites

- Run from repository root `/Users/avdo/maroto`.
- Work directly on the current branch because `Worktree: No`.
- Preserve unrelated untracked `.worktrees/`.
- `deadcode` is installed at `/Users/avdo/go/bin/deadcode`.
- Network should not be required unless `go mod tidy` or mockery attempts to refresh missing module metadata.

## Context for Implementer

- **Patterns to follow:** Dependency/source guardrails live in `internal/dependency/dependency_test.go`; extend that file rather than adding a separate scanning framework. Keep `internal/paperpdf/OWNERSHIP.md` accurate if backend pruning changes maintenance guidance.
- **Conventions:** Use `git mv` for moved docs/examples/assets so history is easier to review. Use mechanical import rewriting followed by `gofmt`/`goimports`. Keep commits by coherent boundary.
- **Key files:** `go.mod` defines the module path; `.mockery.yaml` pins mockery's package root; `Makefile` runs versioned examples; `README.md`, `docs/README.md`, `docs/_sidebar.md`, and `docs/_coverpage.md` contain versioned user-facing text; `pkg/html/translate/*` contains active dead-code candidates; `internal/paperpdf/*` contains the large backend dead-code group.
- **Gotchas:** `deadcode` reports exported functions in libraries when tests do not call them. For this plan, documented but intentionally retained lower-level features must be justified by tests or excluded from the removal list. `internal/paperpdf` is third-party-derived code, so delete only unreachable code and rerun backend tests; do not opportunistically refactor live functions.
- **Domain context:** Paper is a Go PDF library, not a server. The runtime verification is building and running package/example tests plus generated-output smoke checks; no localhost service or browser flow exists.

## Feature Inventory - Refactoring Coverage

### Versioned Surfaces Being Removed

| Current Surface | Current Form | New Form | Mapped to Task |
| --- | --- | --- | --- |
| Go module path | `github.com/avdoseferovic/paper/v2` | `github.com/avdoseferovic/paper` | Task 1 |
| Go imports and generated mocks | `github.com/avdoseferovic/paper/v2/...` | `github.com/avdoseferovic/paper/...` | Task 1 |
| Mockery package root | `.mockery.yaml` package key ending in `/v2` | unversioned package key | Task 1 |
| Test-generation skill instructions | `.github/skills/generate-unit-tests.md` imports ending in `/v2` | unversioned imports | Task 1 |
| Active docs | `docs/v2/**` | `docs/**` unversioned docs paths | Task 2 |
| Legacy docs | `docs/v1/**` | removed | Task 2 |
| Example packages | `docs/assets/examples/*/v2` | `docs/assets/examples/*` | Task 2 |
| Generated assets | `docs/assets/pdf/*v2.pdf`, `docs/assets/text/*v2.txt`, `v2.pdf`, `v2.txt` | unversioned filenames | Task 2 |
| README/docs labels | `Paper V2`, `v2.4.0`, v1 branch/docs references | unversioned Paper documentation | Task 2 |

### Dead-Code Candidates

| Current Location | Dead-Code Finding | Planned Action | Mapped to Task |
| --- | --- | --- | --- |
| `pkg/html/html.go` | `WithImageResolver`, `WithStylesheetResolver` wrappers are unreachable in repo tests | Remove root-package wrappers and docs references; keep lower-level `translate` resolver support that is tested | Task 3 |
| `pkg/html/translate/anchor.go` | `wrapRowAnchorSource`, `rowComponent`, `anchorSource` methods unreachable | Remove stale row-level anchor source path; per-run/local-anchor path remains | Task 3 |
| `pkg/html/translate/flex.go` | `translator.flexRow` unreachable after `flexRows` replacement | Remove superseded helper | Task 3 |
| `pkg/html/translate/translate.go` | `hrRow` unreachable after `styledHrRow` replacement | Remove superseded helper | Task 3 |
| `pkg/props/color.go` | `Green` helper unreachable | Remove helper; keep `GreenColor` var unless compiler/tests prove it can go too | Task 3 |
| `pkg/test/test.go` | `PaperTest.Save` unreachable | Remove fixture-writing helper and docs mention | Task 3 |
| `pkg/pkg.go`, `pkg/components/components.go`, `pkg/consts/consts.go` | doc-only v2 compatibility anchors | Remove packages now that v2 compatibility is not required | Task 3 |
| `pkg/components/*/example_test.go`, `pkg/config/example_test.go` | doc-only examples without `Output:` are reported unreachable | Remove redundant example tests; docs/assets examples remain the maintained examples | Task 3 |
| `internal/paperpdf/{compare,font,fpdf,grid,label,svgbasic,template,ttfparser,utf8fontfile,util}.go` | 77 unreachable backend functions | Prune in compile-guided batches; keep anything needed by live provider/tests | Task 4 |

### Feature Mapping Verification

- [x] All versioned surfaces being removed are listed above
- [x] All active-source dead-code candidates from filtered `deadcode -test ./...` output are mapped
- [x] `internal/paperpdf` dead-code candidates are grouped by source file
- [x] Every feature has a task number
- [x] No live rendering feature is intentionally removed without a test/guard update

## Progress Tracking

**MANDATORY: Update this checklist as tasks complete. Change `[ ]` to `[x]`.**

- [x] Task 1: Remove `/v2` from module and Go import surface
- [x] Task 2: Remove v1/v2 docs, examples, and generated asset names
- [x] Task 3: Remove active-source dead code and redundant examples
- [ ] Task 4: Prune unreachable `internal/paperpdf` backend code
- [ ] Task 5: Add guardrails for unversioned paths and dead-code cleanup
- [ ] Task 6: Tidy module metadata and run full verification

**Total Tasks:** 6 | **Completed:** 3 | **Remaining:** 3

## Implementation Tasks

### Task 1: Remove `/v2` From Module and Go Imports

**Objective:** Make the Go module and every active Go import unversioned, using `github.com/avdoseferovic/paper` everywhere.

**Dependencies:** None

**Commit:** "refactor(module): remove v2 module suffix"

**Files:**

- Modify: `go.mod`
- Modify: `.mockery.yaml`
- Modify: `.github/skills/generate-unit-tests.md`
- Modify: all active `.go` files importing `github.com/avdoseferovic/paper/v2`
- Modify: generated mocks under `mocks/*.go`

**Key Decisions / Notes:**

- Use a mechanical path rewrite from `github.com/avdoseferovic/paper/v2` to `github.com/avdoseferovic/paper`, then run `gofmt`/`goimports`.
- Update generated mocks by mechanical import rewrite unless mockery regeneration is needed; if regenerated, ensure no unrelated mock API churn.
- Do not change package names (`paper`, `config`, `core`, etc.) in this task.

**Definition of Done:**

- [x] `go.mod` module line is `module github.com/avdoseferovic/paper`.
- [x] `rg -n "github\\.com/avdoseferovic/paper/v2" --glob '!docs/plans/**'` returns no active matches.
- [x] `.mockery.yaml` points at the unversioned module path.
- [x] `go list ./...` lists unversioned import paths and succeeds.

**Verify:**

- `go list ./...`
- `go test ./pkg/core ./pkg/config ./mocks`
- `rg -n "github\\.com/avdoseferovic/paper/v2" --glob '!docs/plans/**'`

### Task 2: Remove Versioned Docs, Examples, and Asset Names

**Objective:** Remove all user-facing v1/v2 documentation structure and make docs/examples/assets unversioned.

**Dependencies:** Task 1

**Commit:** "docs: remove versioned docs and examples"

**Files:**

- Delete: `docs/v1/README.md`
- Delete: `docs/v1/documentation.md`
- Move: `docs/v2/html-support.md` -> `docs/html-support.md`
- Move: `docs/v2/features/**` -> `docs/features/**`
- Move: `docs/v2/examples/**` -> `docs/examples/**`
- Move: `docs/assets/examples/*/v2/*` -> `docs/assets/examples/*/*`
- Rename: tracked `docs/assets/pdf/*v2.pdf`, `docs/assets/pdf/v2.pdf`, `docs/assets/text/*v2.txt`, `docs/assets/text/v2.txt` to unversioned filenames
- Modify: `README.md`
- Modify: `docs/README.md`
- Modify: `docs/_sidebar.md`
- Modify: `docs/_coverpage.md`
- Modify: `Makefile`
- Modify: docs markdown links/includes under `docs/**/*.md`
- Modify: docs example `main.go` output paths under `docs/assets/examples/**`
- Modify: command demo output paths and text labels in `cmd/dev/pdf/main.go`, `cmd/benchmark/main.go` as needed

**Key Decisions / Notes:**

- Use `git mv` for docs/example/asset moves.
- Keep current active docs content, but remove version labels and update links from `v2/features/...` to `features/...`.
- Keep `docs/plans/**` historical references untouched except this plan; guardrails should skip plan history.
- Keep generated PDFs/text if they are tracked, but rename them and update examples so future runs overwrite the unversioned names.

**Definition of Done:**

- [x] `docs/v1` and `docs/v2` no longer exist.
- [x] `find docs/assets/examples -type d -name v2` returns no directories.
- [x] `README.md`, `docs/README.md`, docs nav, and docs snippets no longer market the project as V1/V2.
- [x] `make examples` paths point at unversioned example locations.
- [x] Versioned generated asset names are removed or renamed.

**Verify:**

- `test ! -d docs/v1`
- `test ! -d docs/v2`
- `find docs/assets/examples -type d -name v2`
- `rg -n "\\bv1\\b|\\bv2\\b|/v2\\b|docs/v1|docs/v2" README.md docs Makefile cmd --glob '!docs/plans/**'`
- `go test ./docs/assets/examples/...`

### Task 3: Remove Active-Source Dead Code and Redundant Examples

**Objective:** Remove dead code from active source packages outside `internal/paperpdf`, plus redundant doc-only example tests and compatibility anchor packages that no longer serve a user.

**Dependencies:** Task 1, Task 2

**Commit:** "refactor: remove active dead code"

**Files:**

- Modify: `pkg/html/html.go`
- Modify: `pkg/html/translate/anchor.go`
- Modify: `pkg/html/translate/flex.go`
- Modify: `pkg/html/translate/translate.go`
- Modify: `pkg/props/color.go`
- Modify: `pkg/test/test.go`
- Delete: `pkg/pkg.go`
- Delete: `pkg/components/components.go`
- Delete: `pkg/consts/consts.go`
- Delete: `pkg/components/checkbox/example_test.go`
- Delete: `pkg/components/code/example_test.go`
- Delete: `pkg/components/col/example_test.go`
- Delete: `pkg/components/image/example_test.go`
- Delete: `pkg/components/line/example_test.go`
- Delete: `pkg/components/list/example_test.go`
- Delete: `pkg/components/row/example_test.go`
- Delete: `pkg/components/signature/example_test.go`
- Delete: `pkg/components/text/example_test.go`
- Delete: `pkg/config/example_test.go`
- Modify: docs that mention `PaperTest.Save` or removed resolver wrappers
- Test: relevant HTML, props, pkg/test, docs example tests

**Key Decisions / Notes:**

- Remove root `html.WithImageResolver` and `html.WithStylesheetResolver` wrappers because they are uncalled in active code; keep `html.WithImageBaseDir`, `html.WithStylesheetBaseDir`, and tested lower-level image resolver support.
- Remove only row-level anchor source wrappers; preserve target registration and rich text/local anchor rendering.
- Remove `flexRow` because `flexRows` is the active dispatch path.
- Remove `hrRow` because `styledHrRow` is the active dispatch path.
- Remove unexecuted `Example*` tests rather than adding empty `// Output:` blocks; maintained runnable examples live under `docs/assets/examples/**`.

**Definition of Done:**

- [x] Filtered deadcode output outside `internal/paperpdf`, `*_test.go`, and docs plans no longer reports active source functions.
- [x] Removed APIs are also removed from docs.
- [x] Aggregate doc-only packages no longer appear in `go list ./...`.
- [x] HTML anchor, flex, and hr tests still pass.
- [x] `pkg/test` no longer exposes a fixture-writing `Save` helper.

**Verify:**

- `go test ./pkg/html/... ./pkg/props ./pkg/test`
- `go list ./... | rg 'github.com/avdoseferovic/paper/(pkg|pkg/components|pkg/consts)$'`
- `deadcode -test ./... | rg -v 'internal/paperpdf|_test\\.go|docs/plans'`
- `rg -n "WithImageResolver|WithStylesheetResolver" docs README.md pkg/html/html.go --glob '!docs/plans/**'`
- `rg -n "PaperTest\\.Save|\\.Assert\\([^\\n]+\\)\\.Save" docs README.md pkg/test --glob '!docs/plans/**'`

### Task 4: Prune Unreachable `internal/paperpdf` Backend Code

**Objective:** Remove dead backend code from `internal/paperpdf` while preserving Paper's live PDF generation behavior.

**Dependencies:** Task 3

**Commit:** "refactor(paperpdf): prune unreachable backend code"

**Files:**

- Modify/Delete: `internal/paperpdf/compare.go`
- Modify/Delete: `internal/paperpdf/font.go`
- Modify: `internal/paperpdf/fpdf.go`
- Delete: `internal/paperpdf/grid.go`
- Delete: `internal/paperpdf/label.go`
- Delete: `internal/paperpdf/svgbasic.go`
- Modify/Delete: `internal/paperpdf/template.go`
- Modify/Delete: `internal/paperpdf/template_impl.go`
- Modify/Delete: `internal/paperpdf/ttfparser.go`
- Modify: `internal/paperpdf/utf8fontfile.go`
- Modify: `internal/paperpdf/util.go`
- Modify: `internal/paperpdf/NOTICE`
- Modify: `internal/paperpdf/OWNERSHIP.md` if needed

**Key Decisions / Notes:**

- Use `deadcode -test ./...` as the removal source of truth, but prune in batches and let compiler/tests identify coupled declarations that also need deletion or preservation.
- Do not delete live methods used by `internal/providers/paper` even if their supporting dead siblings are removed.
- After each batch, run `go test ./internal/paperpdf ./internal/providers/paper/... ./docs/assets/examples/customfont/...` to protect backend behavior.
- Keep license/NOTICE attribution intact even when deleting unused upstream feature files.

**Definition of Done:**

- [ ] `deadcode -test ./...` no longer reports `internal/paperpdf` unreachable functions, or any remaining entry is documented in `OWNERSHIP.md` with a concrete reason it must stay.
- [ ] Provider, merge, custom font, SVG image, and docs example tests still pass.
- [ ] `internal/paperpdf/NOTICE` remains accurate.
- [ ] No style-only rewrite is mixed into backend pruning.

**Verify:**

- `go test ./internal/paperpdf ./internal/providers/paper/...`
- `go test ./pkg/merge ./docs/assets/examples/customfont/... ./docs/assets/examples/imagegrid/... ./docs/assets/examples/mergepdf/...`
- `deadcode -test ./... | rg 'internal/paperpdf'`
- `go test ./...`

### Task 5: Add Unversioned and Dead-Code Guardrails

**Objective:** Make the cleanup durable by extending tests that scan dependencies/source/docs for removed versioned paths and dead-code artifacts.

**Dependencies:** Task 1, Task 2, Task 3, Task 4

**Commit:** "test: guard unversioned module and dead-code cleanup"

**Files:**

- Modify: `internal/dependency/dependency_test.go`
- Test: `internal/dependency/dependency_test.go`

**Key Decisions / Notes:**

- Add guardrails for `github.com/avdoseferovic/paper/v2`, `docs/v1`, `docs/v2`, `/v2` example dirs, and `*v2.pdf`/`*v2.txt` active assets.
- Stop skipping `docs/v1` because the directory should be gone.
- Keep skipping `docs/plans/**` so historical plans remain readable.
- Add a lightweight guard for active dead-code symbols removed in Task 3: `wrapRowAnchorSource`, `newAnchorSource`, `flexRow`, `hrRow`, `PaperTest.Save`, and removed aggregate package files.
- Do not shell out to `deadcode` inside unit tests; that tool may not be installed in every consumer environment. Use source-pattern and filesystem guardrails instead.

**Definition of Done:**

- [ ] Dependency tests fail if `/v2` module path returns in active source/docs/config.
- [ ] Dependency tests fail if `docs/v1`, `docs/v2`, or `docs/assets/examples/*/v2` returns.
- [ ] Dependency tests fail if removed active dead-code symbols return.
- [ ] Existing removed-dependency guards still pass.

**Verify:**

- `go test ./internal/dependency -count=1`
- `rg -n "github\\.com/avdoseferovic/paper/v2|docs/v1|docs/v2|/v2\\b|wrapRowAnchorSource|newAnchorSource|func \\(tr \\*translator\\) flexRow|func hrRow|func \\(m \\*PaperTest\\) Save" README.md docs pkg internal cmd .mockery.yaml .github Makefile --glob '!docs/plans/**' --glob '!internal/paperpdf/**'`

### Task 6: Tidy Module Metadata and Run Full Verification

**Objective:** Clean module metadata after removals and prove the unversioned, dead-code-pruned project still builds and behaves.

**Dependencies:** Task 5

**Commit:** "chore: tidy after dead-code cleanup"

**Files:**

- Modify: `go.mod`
- Modify: `go.sum`
- Modify: any files touched by `go mod tidy`

**Key Decisions / Notes:**

- Run `go mod tidy` after all source and import removals.
- If `go.sum` changes only due to removed test/doc packages, keep it.
- Run full verification after tidy, including deadcode.

**Definition of Done:**

- [ ] `go mod tidy` has been run.
- [ ] Full test suite passes.
- [ ] Build, vet, lint, race subset, and diff checks pass.
- [ ] `deadcode -test ./...` returns no active-source findings except any intentionally documented backend exception.
- [ ] Plan status is moved to COMPLETE after implementation and VERIFIED after verification.

**Verify:**

- `go mod tidy`
- `go test ./...`
- `go test -race . ./pkg/html/... ./internal/providers/paper/...`
- `go build ./...`
- `go vet ./...`
- `golangci-lint run --config=.golangci.yml --new-from-rev=HEAD ./...`
- `deadcode -test ./...`
- `git diff --check`

## Testing Strategy

- Unit tests: run package-specific tests after each removal batch (`pkg/html`, `pkg/props`, `pkg/test`, `internal/paperpdf`, `internal/providers/paper`, `internal/dependency`).
- Integration tests: run `go test ./...` and docs example tests after module path and docs/example moves.
- Static analysis: run `deadcode -test ./...`, `golangci-lint`, `go vet`, and source-pattern guardrails.
- Manual verification: inspect `go list ./...` output to confirm no `/v2` import path remains and inspect docs navigation for unversioned links.

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Mechanical import rewrite misses generated mocks or docs snippets | Medium | High | Guard with `rg "github\\.com/avdoseferovic/paper/v2"` across active files and `go test ./...`. |
| Moving examples breaks package discovery or docs includes | Medium | Medium | Use `git mv`, update `Makefile`, run `go test ./docs/assets/examples/...`, and scan for `docs/assets/examples/.*/v2`. |
| Removing exported but untested APIs drops useful HTML resolver support | Medium | Medium | Remove only root wrappers reported dead; keep lower-level tested `translate` resolver support and base-dir options. Update docs to match. |
| `internal/paperpdf` deadcode deletion removes code needed by a rarely tested PDF path | Medium | High | Prune in batches, run provider/customfont/image/merge/docs tests after each batch, and keep any compile- or test-required declarations. |
| `deadcode` reports doc-only examples that should remain as docs | Low | Low | Remove redundant package example tests because maintained runnable examples live under `docs/assets/examples/**`; ensure docs examples still compile. |
| Historical plan files keep old v1/v2 references and trip guardrails | High | Low | Explicitly skip `docs/plans/**` in source-pattern guardrails while cleaning active docs/source. |

## Goal Verification

### Truths (what must be TRUE for the goal to be achieved)

- The project builds as `github.com/avdoseferovic/paper` with no `/v2` module suffix.
- Active docs, examples, generated asset names, and README content no longer present Paper as v1 or v2.
- Mechanically identified active-source dead code has been removed or explicitly justified by tests.
- `internal/paperpdf` no longer carries unreachable upstream feature code except any documented backend exception.
- Guard tests prevent reintroducing removed versioned paths and removed dead-code symbols.

### Artifacts (what must EXIST to support those truths)

- `go.mod` — unversioned module declaration.
- `.mockery.yaml` and `mocks/*.go` — unversioned generated mock imports.
- `docs/features/**`, `docs/examples/**`, `docs/html-support.md`, `docs/assets/examples/*` — unversioned docs/example layout.
- `internal/dependency/dependency_test.go` — version/dead-code guardrails.
- Updated `pkg/html/translate/*`, `pkg/props/color.go`, `pkg/test/test.go`, and `internal/paperpdf/*` — dead-code removal.

### Key Links (critical connections that must be WIRED)

- `go.mod` module path -> every internal import path -> generated mocks -> `go test ./...`.
- `Makefile examples` -> moved `docs/assets/examples/*/main.go` files -> unversioned PDF/text output paths.
- Docs navigation (`docs/_sidebar.md`) -> moved docs pages -> moved docs example includes.
- `deadcode -test ./...` findings -> removed source symbols -> guard patterns in `internal/dependency/dependency_test.go`.
- `internal/providers/paper` -> pruned `internal/paperpdf` backend -> generated PDF example tests.

## Open Questions

- None. Assumption applied from the user's update: breaking removal of v1/v2 compatibility is allowed because nobody uses the project yet.

### Deferred Ideas

- Regenerate all generated PDF assets from examples after the cleanup if visual docs need byte-fresh outputs. This plan renames tracked assets and updates output paths, but does not require visual changes.
- A future public API pass can further simplify retained exported packages after real usage patterns emerge.
