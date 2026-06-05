# Slim Go Mod Implementation Plan

Created: 2026-06-05
Status: VERIFIED
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

**Goal:** Make `go.mod` slimmer by removing source-package dependencies that are cheap to refactor or inline, while keeping dependencies whose replacement would add meaningful maintenance risk.

**Architecture:** Remove `github.com/f-amaral/go-async` from the module graph and remove the `.paper.yml` fixture config workflow so `pkg/test` no longer needs a direct `gopkg.in/yaml.v3` parser import. Keep domain libraries for barcode generation, HTML parsing, CSS parsing/selector matching, SVG rasterization, generated mocks/tests, and `github.com/google/uuid` because replacing those would be larger, riskier, or public-API breaking compared with the dependency cost.

**Tech Stack:** Go standard library concurrency (`sync`), module-root discovery via `go.mod`, existing Go tests, `go mod tidy`, and `golangci-lint`.

## Scope

### In Scope

- Replace `github.com/f-amaral/go-async/pool` in `paper.go` with an ordered standard-library worker runner capped by `ChunkWorkers`.
- Remove `.paper.yml` and make `pkg/test` resolve fixtures from the fixed module-root-relative path `test/paper/`.
- Run `go mod tidy` so `go.mod` and `go.sum` drop dependencies that are no longer needed after source imports are removed.
- Extend dependency guard tests so `github.com/f-amaral/go-async` cannot re-enter the module graph, `.paper.yml` cannot re-enter active docs/source, and `gopkg.in/yaml.v3` cannot re-enter source imports. YAML may remain as an indirect transitive dependency of retained `testify`.

### Out of Scope

- Replacing `github.com/boombuler/barcode`: barcode, QR, and Data Matrix encoding are non-trivial domain algorithms.
- Replacing `golang.org/x/net/html`: tolerant HTML5 parsing is not worth reimplementing.
- Replacing `github.com/andybalholm/cascadia`: selector parsing, matching, and specificity are non-trivial.
- Replacing `github.com/aymerick/douceur`: stylesheet and `@font-face` parsing are non-trivial.
- Replacing `github.com/srwiley/oksvg` or `github.com/srwiley/rasterx`: SVG parsing/rasterization is non-trivial.
- Replacing `github.com/google/uuid`: it appears in the exported `core.Image.GetImageDimensions` contract, so removing it would be a public API break that is not worth the small module savings.
- Rewriting the test suite and generated mocks to remove `github.com/stretchr/testify`: this is broad churn and not a focused source-package simplification with good payoff. Because `testify` is retained, its transitive `gopkg.in/yaml.v3` module may remain indirect in `go.mod`, `go.sum`, and `go list -m all`.
- Changing user-facing PDF rendering semantics, HTML feature support, barcode behavior, or snapshot fixture paths.

## Prerequisites

- Run from repository root `/Users/avdo/maroto`.
- Existing module path remains `github.com/avdoseferovic/paper/v2`.
- No network access should be required for implementation unless `go mod tidy` needs to refresh missing module metadata.

## Context for Implementer

- **Patterns to follow:** Dependency bans live in `internal/dependency/dependency_test.go`; follow its existing `forbiddenModulePaths` and `forbiddenTextPatterns` structure.
- **Conventions:** Keep dependency-removal helpers small and package-local unless used from multiple files. Prefer clear standard-library code over new abstractions.
- **Key files:** `go.mod` lists direct dependencies; `paper.go` contains concurrent generation; `pkg/test/test.go` resolves the module root and fixture path; `internal/dependency/dependency_test.go` enforces removed-dependency guardrails.
- **Gotchas:** `github.com/google/uuid` may look removable, but it is part of the exported `core.Image` interface in `pkg/core/components.go`. Keep it unless a future major-version API cleanup explicitly approves the break.
- **Domain context:** `pkg/test` is a reusable helper package, not a test file. Its YAML import existed only to read one `test_path` value from `.paper.yml`; the helper can instead derive the module root from `go.mod` and use `test/paper/` directly. Removing that import does not guarantee YAML disappears from the module graph while `testify` is retained.

## Feature Inventory - Refactoring Coverage

### Dependency-Backed Features Being Replaced

| Current Location | Dependency Feature | Mapped to Task |
| --- | --- | --- |
| `paper.go` | `pool.NewPool`, sorted concurrent processing, `Process` results | Task 1 |
| `pkg/test/test.go`, `.paper.yml`, `docs/v2/features/unittests.md` | YAML-backed fixture path config | Task 2 |
| `go.mod`, `go.sum`, `internal/dependency/dependency_test.go` | Module metadata and removed-dependency guardrails | Task 3 |

### Feature Mapping Verification

- [x] All dependency-backed source features being replaced are listed above
- [x] All affected functions/interfaces identified
- [x] Every feature has a task number
- [x] No user-facing features are intentionally removed

## Progress Tracking

**MANDATORY: Update this checklist as tasks complete. Change `[ ]` to `[x]`.**

- [x] Task 1: Replace `go-async` concurrent generation
- [x] Task 2: Remove `.paper.yml` fixture config from `pkg/test`
- [x] Task 3: Tidy module files and add removed-dependency guards

**Total Tasks:** 3 | **Completed:** 3 | **Remaining:** 0

## Implementation Tasks

### Task 1: Replace `go-async` Concurrent Generation

**Objective:** Replace the only `github.com/f-amaral/go-async/pool` usage in `paper.go` with standard-library concurrency that preserves page-group output order, caps active workers at `ChunkWorkers`, and keeps the existing error surface.

**Dependencies:** None

**Files:**

- Modify: `paper.go`
- Create: `paper_generation.go`
- Create: `paper_layout.go`
- Test: `paper_test.go`

**Key Decisions / Notes:**

- Extract a small unexported helper that accepts `workerCount`, `pageGroups`, and a processor function so ordering and worker-limit behavior can be unit tested without generating PDFs.
- Use a result slice sized to `len(pageGroups)` so each completed job writes to its original index and output order is deterministic.
- Use `sync.WaitGroup` plus a worker semaphore or bounded job channel so active workers never exceed `ChunkWorkers`.
- Keep the existing `ErrCannotGenerateInParallelMode` return when any page group fails.
- Preserve the existing chunking behavior in `generateConcurrently` and `generateLowMemory`.

**Definition of Done:**

- [x] `paper.go` no longer imports `github.com/f-amaral/go-async/pool`.
- [x] Concurrent generation still returns merged PDFs in page-group order.
- [x] Active page-group processing is capped by `ChunkWorkers`.
- [x] Existing concurrent generation tests pass, including the goroutine leak test.

**Verify:**

- `go test ./...` - all package tests pass after replacing the concurrent runner.
- `go test . -run 'Test.*Concurrent|Test.*PageGroup'` - focused concurrent runner tests pass.
- `rg -n "github.com/f-amaral/go-async|go-async|pool\\.NewPool" --glob "*.go"` - no source imports/usages remain.

### Task 2: Remove `.paper.yml` Fixture Config From `pkg/test`

**Objective:** Remove the `.paper.yml` file and direct `gopkg.in/yaml.v3` import from the reusable test helper package by deriving the module root from `go.mod` and using `test/paper/` as the fixed fixture root.

**Dependencies:** None

**Files:**

- Delete: `.paper.yml`
- Modify: `pkg/test/test.go`
- Modify: `pkg/test/config.go`
- Modify: `pkg/test/test_test.go`
- Modify: `docs/v2/features/unittests.md`

**Key Decisions / Notes:**

- `pkg/test` should walk upward to `go.mod`, store that directory as `Config.AbsolutePath`, and use `test/paper/` as `Config.TestPath`.
- Delete the YAML parser, YAML-specific errors, and YAML struct tag.
- Update unit-test docs so users no longer create `.paper.yml`.
- Do not add a new parser dependency or a replacement config file.

**Definition of Done:**

- [x] `pkg/test/test.go` no longer imports `gopkg.in/yaml.v3`.
- [x] `.paper.yml` is deleted.
- [x] Unit tests cover resolving the module root from `go.mod` without `.paper.yml`.
- [x] Existing snapshot assertion helpers still resolve fixture paths under `test/paper/`.
- [x] Unit-test docs no longer instruct users to create `.paper.yml`.

**Verify:**

- `go test ./pkg/test` - helper package tests pass.
- `rg -n "\"gopkg.in/yaml.v3\"|\\byaml\\.[A-Z]" --glob "*.go"` - no direct YAML parser import or package usage remains in Go source.
- `rg -n "\\.paper\\.yml|test_path" --hidden --glob "!docs/plans/**" --glob "!.git/**" --glob "!.worktrees/**"` - no active docs/source/config references remain.

### Task 3: Tidy Module Files and Add Removed-Dependency Guards

**Objective:** Run module cleanup and permanently guard against `go-async` re-entering the module graph and direct YAML parser usage re-entering active source text.

**Dependencies:** Task 1, Task 2

**Commit:** "refactor: slim go module dependencies"

**Files:**

- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `internal/dependency/dependency_test.go`
- Modify: `docs/plans/2026-06-05-slim-go-mod.md`

**Key Decisions / Notes:**

- Run `go mod tidy` only after source imports are removed.
- Add `github.com/f-amaral/go-async` to the forbidden module/text patterns using string concatenation where needed to avoid self-matches.
- Add source/import text guards for `gopkg.in/yaml.v3`, `.paper.yml`, and `test_path` where appropriate, without banning YAML from `go.mod`/`go.sum`, because retained `testify` may keep it indirectly.
- Add `go list -deps -test ./...` coverage for `go-async` to catch reintroduction through tests.
- Keep direct requirements for dependencies intentionally retained: barcode, cascadia, douceur, oksvg, rasterx, x/net, uuid, and testify.
- Exclude `docs/plans/**` from ad hoc text scans, matching the guard test's existing historical-plan skip behavior.

**Definition of Done:**

- [x] `go.mod` no longer directly requires `github.com/f-amaral/go-async` or `gopkg.in/yaml.v3`.
- [x] `go.sum` no longer contains checksums only needed by `github.com/f-amaral/go-async`.
- [x] Dependency guard tests fail if `go-async` reappears in the module/dependency graph.
- [x] Dependency guard tests fail if Go source imports `gopkg.in/yaml.v3` directly.
- [x] Dependency guard tests fail if active docs/source/config reintroduce `.paper.yml` or `test_path`.
- [x] `go list -m all`, `go list -deps ./...`, and `go list -deps -test ./...` do not contain `github.com/f-amaral/go-async`.
- [x] The plan progress checklist is updated to completed before verification.

**Verify:**

- `go mod tidy` - module metadata is normalized.
- `go test ./...` - all tests pass.
- `go build ./...` - all packages build.
- `golangci-lint run --config=.golangci.yml --new-from-rev=HEAD ./...` - linter reports no new issues.
- `go list -m all | rg "github.com/f-amaral/go-async"` - no matches.
- `go list -deps ./... | rg "github.com/f-amaral/go-async"` - no matches.
- `go list -deps -test ./... | rg "github.com/f-amaral/go-async"` - no matches.
- `rg -n "github.com/f-amaral/go-async|\"gopkg.in/yaml.v3\"|\\byaml\\.[A-Z]" --hidden --glob "*.go" --glob "!.git/**" --glob "!.worktrees/**"` - no active Go-source matches for removed imports/usages.
- `rg -n "\\.paper\\.yml|test_path" --hidden --glob "!docs/plans/**" --glob "!.git/**" --glob "!.worktrees/**"` - no active docs/source/config matches.

## Testing Strategy

- Unit tests: Run focused tests for the concurrent page-group runner and `pkg/test` fixture-root resolution.
- Integration tests: Run `go test ./...` to cover docs examples, snapshots, mocks, HTML, merge, provider, and package behavior together.
- Module verification: Use `go mod tidy`, `go list -m all`, `go list -deps ./...`, and `go list -deps -test ./...` to prove `go-async` is absent from the graph and direct YAML usage is absent from source.
- Quality verification: Run `go build ./...`, `golangci-lint`, and `git diff --check`.

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Concurrent generation order changes after removing `go-async` | Medium | High | Write results into index-stable slots and add a focused ordering test for the runner helper. |
| Replacement concurrent runner ignores `ChunkWorkers` | Medium | Medium | Use a bounded worker queue or semaphore and add a test that records max active processors. |
| Goroutine leak or blocked channel in replacement concurrent runner | Medium | Medium | Use `sync.WaitGroup` with channel closing controlled by producer/worker completion and preserve the existing goroutine leak test. |
| Removing `.paper.yml` breaks fixture resolution from package subdirectories | Low | Medium | Resolve the module root by walking upward to `go.mod` and add a unit test for nested package paths without `.paper.yml`. |
| Dependency guard text scan self-matches removed module names | Medium | Low | Build forbidden strings with concatenation in tests and use separate graph-vs-source checks so retained `testify` transitives do not cause false failures. |

## Goal Verification

> Derived from the plan's goal using goal-backward methodology. The spec-reviewer-goal agent verifies these criteria during verification.

### Truths (what must be TRUE for the goal to be achieved)

- `go.mod` has fewer direct requirements because `go-async` is gone and YAML is no longer directly required by source code.
- Source packages no longer import `github.com/f-amaral/go-async` or `gopkg.in/yaml.v3`.
- Concurrent PDF generation still succeeds, preserves page-group order, respects `ChunkWorkers`, and does not leak goroutines.
- Snapshot-based tests still resolve fixtures under `test/paper/` without `.paper.yml`.
- Heavy domain dependencies and public-API-bound dependencies remain in place because replacing them is not worth the risk.

### Artifacts (what must EXIST to support those truths)

- `paper_generation.go` - standard-library concurrent page-group processing.
- `paper_layout.go` - layout internals split out of `paper.go` so changed production files stay focused.
- `pkg/test/test.go` - module-root fixture path resolution without `.paper.yml`.
- `internal/dependency/dependency_test.go` - bans for removed modules, including test dependency graph coverage.
- `go.mod` and `go.sum` - tidied module metadata without `go-async` and without a direct YAML requirement.

### Key Links (critical connections that must be WIRED)

- `Paper.generateConcurrently` -> ordered, bounded page-group runner -> `processPage` -> `merge.Bytes`.
- `pkg/test.New` -> module-root discovery -> fixed `Config.TestPath` -> snapshot fixture resolution.
- `internal/dependency` tests -> `go list -m all` / `go list -deps ./...` / `go list -deps -test ./...` -> `go-async` module absence.
- `internal/dependency` tests -> active Go source scan -> direct YAML import absence.

## Open Questions

- None blocking. The plan interprets "keep only non-worth refactoring/inlining in source packages" as removing narrow dependencies where a small, tested in-repo implementation is clearer, while keeping domain engines, broad test/mocking dependencies, and public-API-bound dependencies.

### Deferred Ideas

- Evaluate replacing generated `testify/mock` mocks with hand-written fakes in a separate test-maintenance pass.
- Consider a future major-version API cleanup that removes `github.com/google/uuid` from `core.Image`.
- Revisit HTML/CSS/SVG dependency boundaries only if the supported HTML subset is intentionally reduced in a future major change.
