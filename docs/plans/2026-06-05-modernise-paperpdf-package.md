# Modernise Paperpdf Package Implementation Plan

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
> **Type:** `Feature` or `Bugfix` — set at planning time, used by dispatcher for routing.

## Summary

**Goal:** Modernise `internal/paperpdf` and make it more Go idiomatic while preserving PDF generation behavior.

**Architecture:** Treat `internal/paperpdf` as Paper's third-party-derived PDF runtime and make bounded internal refactors only. Preserve the `Fpdf` behavior and the `internal/providers/paper/gofpdfwrapper` adapter boundary, while improving helpers, error flow, parser internals, static initialization, and regression tests.

**Tech Stack:** Go 1.26.1, standard library only for production code, existing Go test suite, `go vet`, and `deadcode`.

## Scope

### In Scope

- Complete and keep the current pending `paperpdf` cleanup baseline: unused inherited interface removal, custom sort shim removal, `io/ioutil` replacement, bad-font error handling, and new helper tests.
- Replace the PHP-style `untypedKeyMap` helper with typed Go data structures for CID width range generation.
- Normalize localized `paperpdf` error paths to the existing `SetError` / `SetErrorf` first-error policy.
- Harden malformed TrueType-like font input so bad bytes return `Fpdf.Error()` instead of panicking.
- Modernize UTF-8 font parser internals by unexporting fields on the unexported `utf8FontFile` type and moving binary reader/packing helpers into focused files.
- Add non-brittle semantic PDF assertions for custom-font CID objects and template usage.
- Simplify static initialization for page sizes and core fonts while preserving constructor behavior.
- Add focused package-local tests for every behavior-sensitive helper touched.
- Update `internal/paperpdf/NOTICE` to record local backend modernization patches.

### Out of Scope

- Removing reachable PDF features such as templates, layers, attachments, protection, SVG, HTML basic rendering, transformations, or spot colors.
- Renaming exported `paperpdf` API types such as `Fpdf`, `SizeType`, `PointType`, `ImageInfoType`, `ImageOptions`, or `FontDescType`.
- Shrinking or redesigning `internal/providers/paper/gofpdfwrapper.Fpdf`.
- Replacing the PDF backend with a new engine.
- Broad style-only churn across all copied backend files.
- Golden byte-for-byte PDF output snapshots; PDF object order and compression make them too brittle for this plan.

## Prerequisites

- Work from the current checkout, not a new worktree. The working tree already contains the prior `internal/paperpdf` cleanup edits, and Task 1 intentionally includes them.
- Preserve `internal/paperpdf/LICENSE` and existing upstream attribution.
- Production code must not add dependencies.
- `gofmt`, `go test`, `go vet`, and `deadcode` must be available in the local environment.

## Context for Implementer

- **Patterns to follow:** `Fpdf.SetError` and `Fpdf.SetErrorf` at `internal/paperpdf/fpdf.go:239` document the first-error policy. Use those helpers instead of direct `f.err = fmt.Errorf(...)` when changing localized paths.
- **Conventions:** `internal/paperpdf` is derived backend code; `internal/paperpdf/OWNERSHIP.md:7` says to avoid broad style-only rewrites and keep behavior changes small and test-backed.
- **Key files:** `internal/paperpdf/fpdf.go` contains the core state machine, page lifecycle, rendering, image, font, and output logic. `internal/paperpdf/utf8fontfile.go` contains TrueType parsing and subsetting. `internal/paperpdf/util.go` contains compression, translation, geometry, and current CID width map helpers. `internal/providers/paper/gofpdfwrapper/fpdf.go:13` is the adapter interface used by Paper providers.
- **Gotchas:** Running `deadcode -test ./internal/paperpdf` alone is misleading because most backend methods are reached from other packages through the provider layer. Use `deadcode -test ./...`.
- **Coverage gotcha:** Do not assume every docs example test renders PDF bytes. Use the generated-PDF tests in `paper_test.go`, `pkg/merge`, `pkg/html`, and the new `internal/paperpdf` semantic PDF tests as rendering evidence.
- **Domain context:** This package writes PDF object streams directly. Small formatting changes can affect output, so behavior-sensitive refactors need targeted unit tests plus full example-generation regression.

## Feature Inventory

### Files Being Replaced

| Old File | Functions/Classes | Mapped to Task |
| --- | --- | --- |
| `internal/paperpdf/sort.go` | `sortType`, `gensort` | Task 1 |
| `internal/paperpdf/def.go` | inherited unused `Pdf` interface | Task 1 |
| `internal/paperpdf/template.go` | `templateKeyList` sorting path | Task 1 |
| `internal/paperpdf/fpdf.go` | `addFontFromBytes`, `loadFontFile`, `generateCIDFontMap`, variadic formatting helpers, custom-font output, constructor static data | Tasks 1, 2, 3, 4, 6 |
| `internal/paperpdf/util.go` | `sliceUncompress`, `removeInt`, `untypedKeyMap`, `arrayMerge`, `sprintf`, `fmtBuffer` helper signatures | Tasks 1, 2, 3 |
| `internal/paperpdf/png.go` | `pngColorSpace`, `parsepngstream` error paths | Task 3 |
| `internal/paperpdf/fpdftrans.go` | transform error paths | Task 3 |
| `internal/paperpdf/spotcolor.go` | spot-color error paths | Task 3 |
| `internal/paperpdf/utf8fontfile.go` | `utf8FontFile` exported fields, `GenerateCutFont`, malformed font reading, `fileReader`, binary packing/unpacking helpers | Tasks 4, 5 |
| `internal/paperpdf/NOTICE` | local patch record | Task 7 |

### Feature Mapping Verification

- [x] All old files listed above
- [x] All functions/classes identified
- [x] Every feature has a task number
- [x] No features accidentally omitted

**Intentional removals:** `internal/paperpdf/sort.go` and the inherited unused `Pdf` interface are removed because full-module `deadcode -test ./...` is clean and no callers use them.

## Progress Tracking

**MANDATORY: Update this checklist as tasks complete. Change `[ ]` to `[x]`.**

- [x] Task 1: Preserve current cleanup baseline
- [x] Task 2: Replace CID width map helper with typed Go structures
- [x] Task 3: Normalize localized error handling and `any` usage
- [x] Task 4: Harden and modernize UTF-8 font parser internals
- [ ] Task 5: Extract font binary helpers
- [ ] Task 6: Simplify static initialization data
- [ ] Task 7: Record local backend patches and run final verification

**Total Tasks:** 7 | **Completed:** 4 | **Remaining:** 3

## Implementation Tasks

### Task 1: Preserve Current Cleanup Baseline

**Objective:** Keep the already-pending cleanup work as the first committed modernization unit. This makes the current working-tree changes intentional under this spec instead of leaving them as orphan edits.

**Dependencies:** None

**Commit:** `refactor(paperpdf): preserve backend cleanup baseline`

**Files:**

- Modify: `internal/paperpdf/def.go`
- Modify: `internal/paperpdf/fpdf.go`
- Modify: `internal/paperpdf/png.go`
- Modify: `internal/paperpdf/template.go`
- Modify: `internal/paperpdf/utf8fontfile.go`
- Modify: `internal/paperpdf/util.go`
- Delete: `internal/paperpdf/sort.go`
- Test: `internal/paperpdf/util_test.go`
- Test: `internal/paperpdf/template_test.go`

**Key Decisions / Notes:**

- Preserve the current removal of the inherited unused `Pdf` interface from `def.go`.
- Preserve the replacement of `gensort` with standard library sorting in `template.go`.
- Preserve `io/ioutil` replacement, bad UTF-8 font bytes recording through `Fpdf.SetError`, and zlib invalid-data handling.
- Keep this task behavior-preserving except for the intended fix that non-TrueType bad font bytes no longer write to stdout and invalid zlib data returns an error.
- Add template integration coverage because `template.go` sorting is touched here and templates must remain a reachable backend feature.

**Definition of Done:**

- [ ] `internal/paperpdf/sort.go` no longer exists.
- [ ] `internal/paperpdf/def.go` has no `type Pdf interface`.
- [ ] `internal/paperpdf` has no live `fmt.Printf`, `io/ioutil`, `SeekTable`, `gensort`, `sortType`, or `func remove(` occurrences.
- [ ] `TestAddUTF8FontFromBytesRecordsParseErrorWithoutStdout` proves bad font bytes set `Fpdf.Error()` and write nothing to stdout.
- [ ] `TestSliceUncompressInvalidDataReturnsError` proves invalid compressed data returns an error.
- [ ] Template tests create/use a template, serialize/deserialize it, generate a PDF, and assert XObject/template resource references are present.

**Verify:**

- `gofmt -w internal/paperpdf/*.go`
- `rg "fmt\\.Printf|io/ioutil|SeekTable|gensort|sortType|func remove\\(|type Pdf interface" internal/paperpdf -n` — no matches
- `go test ./internal/paperpdf -run 'Test.*Template|TestSliceUncompress|TestAddUTF8FontFromBytes'`
- `go test ./internal/paperpdf` — package tests pass
- `go test ./...` — full module regression passes

### Task 2: Replace CID Width Map Helper With Typed Go Structures

**Objective:** Replace the PHP-style `untypedKeyMap` / `arrayMerge` helper with typed Go structures dedicated to CID width range generation. Preserve the `/W [...]` output generated by `Fpdf.generateCIDFontMap`.

**Dependencies:** Task 1

**Commit:** `refactor(paperpdf): type CID width map generation`

**Files:**

- Create: `internal/paperpdf/cid_width.go`
- Create: `internal/paperpdf/cid_width_test.go`
- Modify: `internal/paperpdf/fpdf.go`
- Modify: `internal/paperpdf/util.go`

**Key Decisions / Notes:**

- `untypedKeyMap` currently starts at `internal/paperpdf/util.go:291` and uses `[]interface{}` plus a magic `"interval"` key.
- `generateCIDFontMap` currently starts at `internal/paperpdf/fpdf.go:4122`; keep it as the `Fpdf` integration point but delegate range building/formatting to typed helpers.
- Suggested shape: typed `cidWidthRun` / `cidWidthRuns` structs with explicit `start`, `widths`, and `interval` fields. The helper should return the exact string currently appended inside `/W [ ... ]`.
- Do not change glyph width selection rules, `font.usedRunes` filtering, or the `65535 -> 0` mapping.
- Add tests for those edge rules before replacing the helper so the typed rewrite has clear behavioral locks.

**Definition of Done:**

- [ ] `untypedKeyMap` and `arrayMerge` are removed.
- [ ] CID width generation no longer uses `interface{}` or a string sentinel key.
- [ ] Unit tests cover contiguous equal-width interval output and mixed-width array output.
- [ ] A regression test covers the current merge behavior for adjacent non-interval runs.
- [ ] Unit tests cover CID values `<=255` being included, CID values `>255` being excluded unless `usedRunes[cid] > 0`, `65535` width mapping to `0`, sparse ranges, and `LastRune` boundary behavior.
- [ ] `Fpdf.generateCIDFontMap` still emits `/W [` through `f.out(...)`.

**Verify:**

- `gofmt -w internal/paperpdf/*.go`
- `rg "untypedKeyMap|arrayMerge|\\[\\]interface\\{}|interface\\{}" internal/paperpdf -n` — no CID helper matches; any remaining matches must be unrelated gob/formatting before Task 3
- `go test ./internal/paperpdf -run 'TestCID|TestGenerateCID'`
- `go test ./internal/paperpdf`
- `go test ./...`

### Task 3: Normalize Localized Error Handling And `any` Usage

**Objective:** Make localized error handling follow the existing `SetError` / `SetErrorf` policy and modernize variadic `interface{}` signatures to `any` where signatures are otherwise unchanged.

**Dependencies:** Task 2

**Commit:** `refactor(paperpdf): normalize backend error flow`

**Files:**

- Modify: `internal/paperpdf/fpdf.go`
- Modify: `internal/paperpdf/fpdftrans.go`
- Modify: `internal/paperpdf/png.go`
- Modify: `internal/paperpdf/spotcolor.go`
- Modify: `internal/paperpdf/util.go`
- Test: `internal/paperpdf/util_test.go`
- Test: `internal/paperpdf/png_test.go`

**Key Decisions / Notes:**

- `Fpdf.SetError` / `SetErrorf` at `internal/paperpdf/fpdf.go:257` are the canonical first-error helpers.
- Convert local direct error writes in `png.go`, `fpdftrans.go`, and `spotcolor.go` to `SetError` / `SetErrorf` when doing so preserves first-error semantics.
- Replace `args ...interface{}` with `args ...any` for formatting helpers and methods. This is source-compatible for callers.
- Fix malformed PNG error strings such as `"'unknown compression method in PNG buffer"` while preserving the condition that triggers them.

**Definition of Done:**

- [ ] `SetErrorf`, `fmtBuffer.printf`, `sprintf`, `Cellf`, `Writef`, `outf`, and font parser `setErrorf` use `any`.
- [ ] `png.go`, `fpdftrans.go`, and `spotcolor.go` use `SetError` / `SetErrorf` for newly touched error paths.
- [ ] Tests prove the first error is not overwritten by a later PNG/transform/spot-color error.
- [ ] Tests prove malformed PNG compression/filter method errors do not include stray leading quotes.

**Verify:**

- `gofmt -w internal/paperpdf/*.go`
- `rg "args \\.\\.\\.interface\\{}|fmtStr string, args \\.\\.\\.interface\\{}" internal/paperpdf -n` — no matches
- `go test ./internal/paperpdf -run 'Test.*Error|TestPNG'`
- `go test ./internal/paperpdf`
- `go test ./...`

### Task 4: Harden And Modernize UTF-8 Font Parser Internals

**Objective:** Make the UTF-8 font parser safer and more idiomatic without changing font registration or subsetting behavior. This task handles parser hardening, compile-checked internal renames, and semantic custom-font PDF assertions before any helper extraction.

**Dependencies:** Task 3

**Commit:** `refactor(paperpdf): harden utf8 font internals`

**Files:**

- Create: `internal/paperpdf/font_pdf_test.go`
- Modify: `internal/paperpdf/utf8fontfile.go`
- Modify: `internal/paperpdf/fpdf.go`
- Test: `internal/paperpdf/util_test.go`

**Key Decisions / Notes:**

- Add a regression for truncated or structurally corrupt TrueType-like bytes, not just non-TrueType magic bytes. The test must call `AddUTF8FontFromBytes`, assert no panic, assert no stdout, and assert `Fpdf.Error()` is non-nil.
- Guard `fileReader.Read`, table description generation, and table seeking so malformed font data returns an error instead of relying on slice bounds panics.
- Rename unexported-type fields like `LastRune`, `Ascent`, `Descent`, `Bbox`, `CapHeight`, `StemV`, `ItalicAngle`, `Flags`, `UnderlinePosition`, `UnderlineThickness`, `CharWidths`, `DefaultWidth`, and `CodeSymbolDictionary` to lower-case names.
- Rename `(*utf8FontFile).GenerateCutFont` to `generateCutFont` and update its only package-local caller in `fpdf.go`.
- Add non-brittle semantic PDF assertions for a generated custom-font PDF. Do not compare full bytes. Assert object-level markers such as `/Subtype /CIDFontType2`, `/W [`, `/FontFile2`, `/ToUnicode`, `/CIDToGIDMap`, and a `%PDF` header are present.
- Use a small known string with non-ASCII runes so UTF-8 subsetting and CID width output are exercised.

**Definition of Done:**

- [ ] Truncated TrueType-like font bytes do not panic, do not write stdout, and set `Fpdf.Error()`.
- [ ] No exported fields remain on `utf8FontFile`.
- [ ] `GenerateCutFont` is renamed to `generateCutFont`.
- [ ] A semantic custom-font PDF test asserts CID font, `/W`, `/FontFile2`, `/ToUnicode`, and `/CIDToGIDMap` objects/markers are present.
- [ ] Custom font examples, font repository tests, and merge tests still pass.
- [ ] No production dependency is added.

**Verify:**

- `gofmt -w internal/paperpdf/*.go`
- `rg "utf8File\\.[A-Z]|func \\(utf \\*utf8FontFile\\) GenerateCutFont" internal/paperpdf -n` — no matches
- `go test ./internal/paperpdf -run 'Test.*Font|Test.*UTF8|Test.*CIDFont|TestAddUTF8FontFromBytes'`
- `go test ./pkg/fontrepository ./docs/assets/examples/customfont ./pkg/merge`
- `go test ./...`

### Task 5: Extract Font Binary Helpers

**Objective:** Move binary reader, packing, unpacking, and key-sort helpers out of `utf8fontfile.go` into focused files after Task 4 has locked parser and PDF behavior.

**Dependencies:** Task 4

**Commit:** `refactor(paperpdf): extract font binary helpers`

**Files:**

- Create: `internal/paperpdf/font_binary.go`
- Create: `internal/paperpdf/font_binary_test.go`
- Modify: `internal/paperpdf/utf8fontfile.go`

**Key Decisions / Notes:**

- Move `fileReader`, `pack*`, `unpack*`, and key-sort helpers into `font_binary.go` where possible.
- Preserve byte order, padding, and sorting output exactly.
- Do not mix this extraction with further font parser logic changes.

**Definition of Done:**

- [ ] `utf8fontfile.go` no longer contains `fileReader`, `pack*`, `unpack*`, or key-sort helper implementations.
- [ ] Binary pack/unpack helpers have direct unit tests for representative 16-bit and 32-bit values.
- [ ] Key-sort helper tests cover deterministic ordering for string and int-keyed maps.
- [ ] Task 4 semantic custom-font PDF test still passes.

**Verify:**

- `gofmt -w internal/paperpdf/*.go`
- `go test ./internal/paperpdf -run 'TestPack|TestUnpack|TestKeySort|Test.*CIDFont'`
- `go test ./internal/paperpdf`
- `go test ./...`

### Task 6: Simplify Static Initialization Data

**Objective:** Replace imperative constructor setup for static page sizes and core fonts with explicit package-level data and small copy helpers. Keep each `Fpdf` instance isolated from accidental map mutation.

**Dependencies:** Task 5

**Commit:** `refactor(paperpdf): simplify backend initialization data`

**Files:**

- Create: `internal/paperpdf/initdata.go`
- Create: `internal/paperpdf/initdata_test.go`
- Modify: `internal/paperpdf/fpdf.go`
- Modify: `internal/paperpdf/def.go`

**Key Decisions / Notes:**

- Current standard page size setup is imperative in `fpdfNew` at `internal/paperpdf/fpdf.go:138`.
- Current `coreFonts` is `map[string]bool` at `internal/paperpdf/fpdf.go:115` but is used as a set. Convert it to a typed set representation if it does not ripple beyond `paperpdf`.
- Use package-level literals plus clone/copy helpers so each `Fpdf` receives independent mutable maps.
- Do not change accepted unit strings, orientation strings, page size names, or default values.

**Definition of Done:**

- [ ] `fpdfNew` no longer manually assigns every standard page size one line at a time.
- [ ] `coreFonts` is represented as a set or otherwise documented if left as `map[string]bool`.
- [ ] Tests prove default A4 sizing, custom size override, and instance map isolation.
- [ ] Invalid unit and orientation behavior remains unchanged.

**Verify:**

- `gofmt -w internal/paperpdf/*.go`
- `go test ./internal/paperpdf -run 'TestNewCustom|TestPageSize|TestInitData'`
- `go test ./internal/paperpdf -run 'Test.*Template'`
- `go test ./internal/providers/paper/gofpdfwrapper ./internal/providers/paper`
- `go test ./...`

### Task 7: Record Local Backend Patches And Run Final Verification

**Objective:** Document the backend-local modernization patches and run the final quality gates. This keeps the ownership boundary honest for future upstream refreshes.

**Dependencies:** Task 6

**Commit:** `docs(paperpdf): record backend modernization patches`

**Files:**

- Modify: `internal/paperpdf/NOTICE`
- Modify: `docs/plans/2026-06-05-modernise-paperpdf-package.md`

**Key Decisions / Notes:**

- `internal/paperpdf/NOTICE` already says to record upstream refreshes or local backend patches.
- Add a short dated note describing local modernization categories, not a long changelog.
- Update this plan's progress checklist as tasks complete.

**Definition of Done:**

- [ ] `NOTICE` records the local modernization patch categories.
- [ ] The plan progress checklist is updated.
- [ ] `gofmt -l internal/paperpdf/*.go` produces no output.
- [ ] Full regression, vet, deadcode, and cleanup scans pass.
- [ ] `git status --short` shows only intended files for this spec.

**Verify:**

- `gofmt -w internal/paperpdf/*.go`
- `gofmt -l internal/paperpdf/*.go` — no output
- `go test ./...`
- `go vet ./...`
- `deadcode -test ./...`
- `rg "fmt\\.Printf|io/ioutil|SeekTable|gensort|sortType|func remove\\(|type Pdf interface" internal/paperpdf -n` — no matches
- `git status --short`

## Testing Strategy

- **Unit tests:** Add package-local tests in `internal/paperpdf` for compression errors, non-TrueType bad font bytes, truncated TrueType-like font bytes, template creation/serialization/use, CID width edge rules, binary font pack/unpack helpers, PNG error messages, first-error behavior, and initialization data.
- **Semantic PDF tests:** Add package-local tests that generate PDFs and assert meaningful object markers instead of full byte-for-byte goldens. Custom-font tests must assert `%PDF`, `/Subtype /CIDFontType2`, `/W [`, `/FontFile2`, `/ToUnicode`, and `/CIDToGIDMap`. Template tests must assert form XObject/resource markers after using templates.
- **Integration tests:** Run provider and wrapper tests where `paperpdf` behavior crosses into `internal/providers/paper`.
- **Generated-PDF regression:** Use `paper_test.go`, `pkg/merge`, `pkg/html`, and the new `internal/paperpdf` semantic tests as rendering evidence. Do not rely on docs example structure tests as proof that PDF bytes rendered.
- **Full regression:** Run `go test ./...` to catch provider, font repository, merge, HTML, and public package regressions.
- **Static checks:** Run `gofmt`, `go vet ./...`, `deadcode -test ./...`, and targeted `rg` scans for removed legacy patterns.

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| CID width formatting changes and breaks UTF-8 font PDFs | Medium | High | Add unit tests around equal-width intervals, mixed-width arrays, adjacent run merging, CID `<=255` inclusion, CID `>255` `usedRunes` filtering, `65535 -> 0`, sparse ranges, and `LastRune` boundaries; run semantic custom-font PDF tests after Task 4. |
| Malformed TrueType-like input panics in parser bounds/table paths | Medium | High | Add a truncated TrueType-like `AddUTF8FontFromBytes` regression that asserts no panic/no stdout/non-nil error; guard reader/table access to return errors. |
| Font parser field renames miss a package-local reference | Medium | Medium | Use `rg "utf8File\\.[A-Z]"` and full `go test ./...` after Task 4. |
| Semantic PDF behavior regresses while helper tests still pass | Medium | High | Add custom-font object marker tests for CIDFontType2, `/W`, FontFile2, ToUnicode, and CIDToGIDMap; add template XObject/resource marker tests. |
| Error handling changes alter first-error semantics | Medium | Medium | Add tests that set an initial error and trigger later PNG/transform/spot-color errors; assert the original error remains. |
| Broad style churn conflicts with `OWNERSHIP.md` | Low | Medium | Restrict changes to files and functions listed in Feature Inventory; update `NOTICE` with local patch categories. |
| Constructor map extraction accidentally shares mutable maps between `Fpdf` instances | Medium | Medium | Add an instance isolation test that mutates one instance's map and verifies a new instance keeps defaults. |
| Existing pending cleanup gets lost or reverted | Low | Medium | Treat current pending `internal/paperpdf` cleanup as Task 1 and verify it before subsequent refactors. |

## Goal Verification

> Derived from the plan's goal using goal-backward methodology. The spec-reviewer-goal agent verifies these criteria during verification.

### Truths (what must be TRUE for the goal to be achieved)

- `internal/paperpdf` has fewer inherited/non-idiomatic helper patterns while keeping the same provider-facing behavior.
- Non-TrueType bad font bytes, truncated TrueType-like font bytes, invalid zlib data, malformed PNG metadata, transform misuse, and spot-color misuse are covered by package-local tests.
- CID font width generation uses typed Go structures rather than `interface{}` keys and PHP-style array emulation.
- UTF-8 font parser internals on the unexported `utf8FontFile` type are unexported and organized into focused files.
- Custom-font generated PDFs expose expected CID font objects, width maps, embedded font data, ToUnicode maps, and CIDToGID maps.
- Template generated PDFs expose expected form XObject/resource markers after create/use/serialize paths.
- Standard page size and core font initialization are explicit, tested, and instance-safe.
- Full module tests, formatting, vet, and deadcode pass after the modernization.

### Artifacts (what must EXIST to support those truths)

- `internal/paperpdf/cid_width.go` — typed CID width range helper.
- `internal/paperpdf/cid_width_test.go` — CID width formatting regression tests.
- `internal/paperpdf/font_pdf_test.go` — semantic custom-font PDF object tests.
- `internal/paperpdf/font_binary.go` — binary reader/packing helpers split from `utf8fontfile.go`.
- `internal/paperpdf/font_binary_test.go` — binary helper tests.
- `internal/paperpdf/png_test.go` — PNG/error behavior tests.
- `internal/paperpdf/template_test.go` — template create/use/serialize integration tests.
- `internal/paperpdf/initdata.go` — static page/core-font initialization data.
- `internal/paperpdf/initdata_test.go` — constructor/default data tests.
- `internal/paperpdf/NOTICE` — local backend patch record.

### Key Links (critical connections that must be WIRED)

- `internal/providers/paper/gofpdfwrapper.NewCustom` -> `paperpdf.NewCustom` still returns `*paperpdf.Fpdf`.
- `Fpdf.putfonts` -> `utf8FontFile.generateCutFont` -> CID map generation still writes valid `/W [...]` font width data.
- `Fpdf.AddUTF8FontFromBytes` -> malformed TrueType-like parser path returns `Fpdf.Error()` instead of panicking.
- `Fpdf.RegisterImageOptionsReader` -> PNG parsing -> `Fpdf.Error()` records malformed image errors without overwriting earlier errors.
- `Fpdf.SetFont` / custom font registration -> lower-case `utf8FontFile` fields still populate `FontDescType`, `fontDefType.Cw`, and `fontFileType`.
- `Fpdf.CreateTemplate` / `UseTemplate` / `Serialize` -> generated PDF still contains expected XObject/resource references.
- `fpdfNew` -> `defaultPageSizes` / `coreFontSet` helpers still produces A4 portrait defaults and supports custom page sizes.

## Open Questions

- None blocking. If implementation uncovers additional dead code, it must be proven with `deadcode -test ./...` and mapped into this plan before removal.

### Deferred Ideas

- Rename legacy exported `*Type` names such as `SizeType` and `ImageInfoType`; this would ripple through the provider wrapper and mocks.
- Split `fpdf.go` by rendering domain more broadly. That is mostly file organization churn unless paired with stronger PDF output tests.
- Shrink `internal/providers/paper/gofpdfwrapper.Fpdf`; this is adapter/API work outside the backend package.
- Add a robust PDF semantic/golden comparison harness for future backend refreshes.
