# HTML → PDF in Maroto Implementation Plan

Created: 2026-05-19
Status: COMPLETE
Approved: Yes
Iterations: 0
Worktree: No
Type: Feature

> **Status Lifecycle:** PENDING → COMPLETE → VERIFIED
> **Iterations:** Tracks implement→verify cycles (incremented by verify phase)
>
> - PENDING: Initial state, awaiting implementation
> - COMPLETE: All tasks implemented
> - VERIFIED: All checks passed
>
> **Approval Gate:** Implementation CANNOT proceed until `Approved: Yes`
> **Worktree:** Set at plan creation (from dispatcher). `Yes` uses git worktree isolation; `No` works directly on current branch (default)
> **Type:** `Feature` or `Bugfix` — set at planning time, used by dispatcher for routing

## Summary

**Goal:** Allow users to generate PDFs from a documented subset of HTML/CSS in Maroto, using Maroto's existing gofpdf-based renderer (no Chrome, no Node, no shelling out to external binaries).

**Architecture:** HTML/CSS string → DOM (`golang.org/x/net/html`) + computed styles (`aymerick/douceur` + `andybalholm/cascadia`) → translator that walks the styled DOM and emits Maroto's existing AST (Row/Col/RichText/Table/List/Image). Maroto handles layout, pagination, headers/footers as it does today. New first-class components (`RichText`, `Table` with spans, `HTMLList`) are also useful outside of HTML — they ship as public packages.

**Tech Stack:** Pure Go. New deps: `golang.org/x/net` (html parser), `github.com/aymerick/douceur` (CSS parser), `github.com/andybalholm/cascadia` (CSS selectors). Existing: `gofpdf` for rendering.

**Realistic scope:** "Best-effort arbitrary HTML" with `Inline + <style> selectors` CSS, but **not** flexbox, grid, floats, `position:absolute`, transforms, JS, `@media`, or `@font-face`. Documented supported subset is the contract — unsupported tags/props render as best-effort fallback (text content preserved, structure approximated) rather than failing.

## Scope

### In Scope

**Phase 1 — Component foundation (prerequisites)**

- New `pkg/components/richtext`: paragraph with inline runs (`<span>`, `<b>`, `<i>`, `<u>`, `<s>`, `<a>`, mixed sizes/colors). Real run-based line wrapping.
- New `pkg/components/table`: `<table>` model with rows, cells, `colspan`, `rowspan`, per-cell padding, per-side borders, background.
- New `pkg/components/htmllist`: `<ul>` / `<ol>` with bullets/numbers, nesting, custom markers.
- Extensions to `pkg/props/cell.go`: per-side padding, per-side border thickness/color, optional width override.
- Provider additions to `pkg/core/provider.go`: `MeasureString(text, prop) float64`, `AddTextAt(x, y, text, prop)`, `AddRichText(runs, cell, prop)`.

**Phase 2 — HTML pipeline**

- New `pkg/html/dom`: thin wrapper over `golang.org/x/net/html` that produces a Maroto-friendly DOM walker, including `<head><style>` extraction and inline `style=""` collection.
- New `pkg/html/css`: CSS parser + cascade + computed-style resolver. Inline + `<style>` block selectors via cascadia. Computes a `ComputedStyle` per element with the supported property subset.
- New `pkg/html/translate`: DOM walker that consumes the styled tree and emits `[]core.Row`. Block elements become Rows; inline elements become RichText runs; tables/lists use the new components.

**Phase 3 — Public API**

- New `pkg/html` package: `html.FromString(htmlStr string, opts ...Option) ([]core.Row, error)` + `html.FromReader(r io.Reader, opts ...Option) ([]core.Row, error)`.
- New method `Maroto.AddHTML(htmlStr string, opts ...html.Option) error` on `core.Maroto` (sugar over `FromString` + `AddRows`).
- `Option` includes: base font override, debug mode, on-unsupported-tag callback, custom image resolver (for relative `<img src>`).
- Documented supported HTML subset, supported CSS subset, examples in `docs/`.

**Supported HTML tags (v1):**
`html, head, title, style, body, h1-h6, p, br, hr, div, span, a, img, strong, b, em, i, u, s, strike, sub, sup, ul, ol, li, table, thead, tbody, tfoot, tr, th, td, blockquote, pre, code, header, footer, section, article, aside, main, nav, figure, figcaption`

**Supported CSS properties (v1):**
`color, background-color, font-family, font-size, font-weight, font-style, text-align, text-decoration, line-height, padding, padding-{top,right,bottom,left}, margin, margin-{top,right,bottom,left}, border, border-{top,right,bottom,left}, border-color, border-width, border-style, width, height, display: block|inline|inline-block|none, vertical-align (table cells)`

### Out of Scope

- Chrome/Chromium/headless browser of any kind.
- Shelling out to external binaries (wkhtmltopdf, weasyprint, prince, etc).
- JavaScript execution.
- CSS flexbox, grid, floats, `position`, `transform`, `@media`, `@keyframes`, pseudo-elements (`::before`, `::after`), pseudo-classes (`:hover`, `:nth-child`).
- `@font-face` web fonts. Users must register custom fonts via Maroto's existing `CustomFonts` config.
- External stylesheet fetching (`<link rel="stylesheet" href="https://...">`).
- SVG rendering (PNG/JPG `<img>` only; SVG `<img src>` falls back to alt text).
- Form elements (`<input>`, `<button>`, `<select>`, `<form>`) — rendered as plain text content.
- Video, audio, canvas, iframe — rendered as alt text or skipped.
- Backward-compatibility concerns: this is purely additive. Existing API unchanged.

## Prerequisites

- Go 1.26.1 (already in `go.mod`)
- `go get golang.org/x/net/html github.com/aymerick/douceur github.com/andybalholm/cascadia`
- Run `go mod tidy` after `go get` to reconcile transitive version conflicts (e.g. if `golang.org/x/net` was already pulled in as an indirect dep by another dependency)
- Existing test infrastructure (`pkg/test`, `internal/fixture`) and JSON snapshot fixtures.

## Context for Implementer

- **Patterns to follow:**
  - Component pattern: see `pkg/components/text/text.go:14` — components have `New(...)`, `NewCol(...)`, `NewRow(...)`, `NewAutoRow(...)`. They implement `core.Component`: `GetStructure()`, `GetHeight()`, `SetConfig()`, `Render()`.
  - Provider feature pattern: see `pkg/core/provider.go:12` and `internal/providers/gofpdf/text.go:34` — Provider exposes high-level operations, gofpdf implementation lives in a focused file.
  - Test pattern: snapshot tests via `test.New(t).Assert(...).Equals("components/.../foo.json")` (see `pkg/components/text/text_test.go:22`). JSON fixtures live in `test/`.
- **Conventions:**
  - Public package goes in `pkg/`, private in `internal/`. New `pkg/components/...` for components, new `pkg/html/...` for HTML.
  - Error variables use `Err*` prefix at package level (see `pkg/components/list/list.go:11`).
  - Each `.go` file kept under 300 lines (user rule).
- **Key files to read first:**
  - `pkg/core/provider.go` — Provider interface (adding methods here)
  - `pkg/components/text/text.go` — model for RichText component
  - `internal/providers/gofpdf/text.go` — model for inline run measurement & placement
  - `pkg/components/row/row.go` + `pkg/components/col/col.go` — layout model
  - `maroto.go:107` (`AddRow`/`AddAutoRow`) — how rows are added (extension point for `AddHTML`)
- **Gotchas:**
  - `Provider` is marked `nolint:interfacebloat` — adding methods is OK but every gofpdf integration test that asserts mock call counts will need updating.
  - `pkg/components/list` is the legacy "tablelist" row-repeater, NOT an HTML list. New component must be named differently (chose `htmllist`) to avoid confusion.
  - Maroto rows are TOP-LEVEL only — there is no row-inside-col concept. Block elements that need to contain other block elements (`<div><p>...</p></div>`) require flattening into sibling rows OR a new Col-with-rows abstraction. Plan flattens.
  - gofpdf's `Text(x,y,s)` is baseline-positioned; existing code in `text.go:68` does `y += fontHeight` to convert to baseline. RichText must mirror this.
  - Font measurement requires `SetFont` first (`text.go:35`). Mixing styles in a line means re-`SetFont` per run.
- **Domain context:**
  - "Best-effort arbitrary HTML" means: documented subset works correctly, unsupported tags fall through to children's content, unsupported CSS properties are silently ignored. Never panic on unknown input.
  - CSS cascade order: user-agent defaults → `<style>` block (specificity) → inline `style=""` (always wins among same source). douceur gives parsed declarations; we compute specificity ourselves (standard a-b-c-d formula).

## Runtime Environment

Not applicable — Maroto is a library, not a service.

## Progress Tracking

- [x] Task 1: Extend Provider interface with measurement & positioning primitives
- [x] Task 2: Extend props.Cell with padding and per-side borders
- [x] Task 3: RichText component with inline run wrapping
- [x] Task 4: Table component with rowspan/colspan/borders
- [x] Task 5: HTMLList component (ul/ol with markers and nesting)
- [x] Task 6: HTML DOM wrapper with style extraction
- [x] Task 7: CSS parser + cascade + computed styles
- [x] Task 8: DOM-to-Maroto-AST translator (block elements)
- [x] Task 9: Inline element translation into RichText runs
- [x] Task 10: Table/list translation
- [x] Task 11: Public API (pkg/html + Maroto.AddHTML) + documented subset + examples

**Total Tasks:** 11 | **Completed:** 11 | **Remaining:** 0

## Implementation Tasks

### Task 1: Add RichTextProvider interface with measurement & positioning primitives

**Objective:** Add a NEW narrow interface `core.RichTextProvider` (containing `MeasureString`, `AddTextAt`, `AddRichText`) and implement it in the gofpdf provider. **Do NOT extend `core.Provider` or `core.Text`** — those interfaces are mocked extensively across 13+ test files; extending them would force mock regeneration that breaks 42+ existing call-count assertions. The new interface is consumed only by RichText/Table/HTMLList components in subsequent tasks.

**Dependencies:** None

**Files:**

- Create: `pkg/core/richtext_provider.go` (new `RichTextProvider` interface — 3 methods)
- Modify: `internal/providers/gofpdf/provider.go` (`*provider` implements `RichTextProvider`; the type already satisfies `core.Provider`, now also satisfies the new interface)
- Modify: `internal/providers/gofpdf/text.go` (add `MeasureString`, `AddTextAt`, `AddRichText` methods directly to `*Text` struct; provider delegates to these)
- Create: `mocks/RichTextProvider.go` (NEW mock only — existing `Provider.go` and `Text.go` mocks are untouched)
- Test: `internal/providers/gofpdf/text_test.go` (extend)
- Create: `pkg/props/richtext.go` (RichRun struct + helpers)
- Test: `pkg/props/richtext_test.go`

**Key Decisions / Notes:**

- **Why a separate interface (architectural):** `core.Provider` has 24 methods, is mocked in 13 test files with 42 `EXPECT()` / `AssertExpectations` call sites. Adding methods triggers a mockery regeneration that breaks every existing test that uses strict mock matching. The new `RichTextProvider` is consumed only by new components (RichText, Table, HTMLList), so no existing test mocks it. The `*provider` struct in `internal/providers/gofpdf/provider.go` implements BOTH interfaces — Maroto components can hold either one depending on need.
- `core.RichTextProvider` shape:
  ```go
  type RichTextProvider interface {
      MeasureString(text string, prop *props.Text) float64
      AddTextAt(x, y float64, text string, prop *props.Text)
      AddRichText(runs []props.RichRun, cell *entity.Cell, prop *props.RichText)
  }
  ```
- `MeasureString` — sets font, calls `pdf.GetStringWidth`, returns width in mm.
- `AddTextAt` — absolute placement, baseline-positioned (caller pre-computes baseline).
- **AddRichText font-state contract (must be implemented exactly):**
  - At entry: capture current font state via `pdf.GetFontDesc()` (or equivalent) into local variables: `origFamily, origStyle, origSize, origColor`.
  - `defer` a restore of those values so any early return or panic does not leak font state to subsequent rendering.
  - The line-break measurement loop calls `SetFont` only on run-change (track `currentRun int = -1`; call `SetFont` only when iterating to a different run index). Worst-case calls = number of distinct runs per line, not per word.
  - **ALL intermediate state (current line buffer, x-cursor, current-run-index) is method-local — never stored on the `*Text` struct.** This guarantees concurrent safety for `generateConcurrently` mode where each goroutine has its own `*provider` but shares no other state.
- `props.RichRun` shape:
  ```go
  type RichRun struct {
      Text          string
      Family        string
      Style         fontstyle.Type
      Size          float64
      Color         *Color
      Underline     bool
      Strikethrough bool
      Hyperlink     *string
      VerticalAlign string // "baseline" | "sub" | "super"
  }
  ```
  Note: `RichRun` deliberately has NO image field. Inline `<img>` in HTML is split into separate before/image/after rows by the translator (Task 9 update), not embedded in a run.
- `props.RichText` is paragraph-level (align, line-height, top/bottom/left/right padding, break strategy).
- Existing `Text.Add` and `AddText` stay unchanged — fully backward compatible.

**Definition of Done:**

- [ ] `core.RichTextProvider` interface defined with 3 methods; `*gofpdf.provider` satisfies it (compile-time assertion `var _ core.RichTextProvider = (*provider)(nil)`)
- [ ] `core.Provider` interface, `mocks/Provider.go`, and `mocks/Text.go` are UNCHANGED (verified via `git diff` after Task 1 commit shows zero changes in those files)
- [ ] `mocks/RichTextProvider.go` exists as a new file with mockery-generated mock for the new interface
- [ ] `MeasureString("hello", props.Text{Family:"Helvetica", Style:"", Size:12})` returns a positive float matching `pdf.GetStringWidth`
- [ ] `AddRichText` correctly wraps a 3-run paragraph (e.g. `["hello ", "bold ", "world"]` with mixed styles) within a 50mm cell — verified by snapshot or pixel-position assertion
- [ ] **Font state restore:** test calls `AddRichText` with runs that switch fonts (Helvetica 12pt → Courier 14pt → Times 10pt), then immediately calls `provider.AddText` with a plain Text using default 12pt Helvetica. Assert that the plain text renders at the expected width — proves font state was restored by `AddRichText`.
- [ ] **Race safety:** `go test -race -run RichText ./internal/providers/gofpdf/...` passes with a test that spawns 4 goroutines each calling `AddRichText` on separate `*Text` instances.
- [ ] `go test ./...` green — no existing tests broken because no existing mock was regenerated
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/core/... ./pkg/props/... ./internal/providers/gofpdf/... -run RichText -v`
- `go test ./...` — full suite green
- `go vet ./... && gofmt -l .` clean

---

### Task 2: Extend props.Cell with padding and per-side borders

**Objective:** Add the box-model primitives RichText, Table, and HTMLList need (padding per side; border per side with color/thickness/style).

**Dependencies:** None

**Files:**

- Modify: `pkg/props/cell.go` (extend `Cell` struct with `PaddingTop/Right/Bottom/Left`, `BorderTopColor/RightColor/...`, `BorderTopThickness/...`)
- Test: `pkg/props/cell_test.go`
- Modify: `internal/providers/gofpdf/cellwriter/cellwriter.go` (honor new fields when drawing borders/background)
- Test: `internal/providers/gofpdf/cellwriter/cellwriter_test.go`

**Key Decisions / Notes:**

- Backward compatibility: existing `BorderColor`/`BorderThickness`/`BorderType` continue to set all four sides via the existing `pdf.CellFormat`-based code path. New per-side fields require a SEPARATE code path because `gofpdf.CellFormat(...)` takes one border string (`"LRTB"`/`"1"`) and uses a single global draw color and line width — it cannot express different colors/thicknesses per side.
- **Implementation approach:** add a new `PerSideBorderStyler` to the cellwriter chain. When any per-side field is set on `props.Cell`, this styler is selected and bypasses `CellFormat` borders entirely:
  1. Compute cell rect (X, Y, W, H).
  2. For each side that has `BorderXThickness > 0`: call `pdf.SetDrawColor(...)` + `pdf.SetLineWidth(...)` + `pdf.Line(x1,y1,x2,y2)` with the per-side values.
  3. If background color is set, fill rect via `pdf.Rect(..., "F")` BEFORE drawing per-side lines.
- When no per-side fields are set, the existing `cellwriter` styler chain continues to use `CellFormat` (zero regression for current users).
- Padding affects content placement only — does NOT affect outer cell dimensions (CSS `box-sizing: border-box` semantics by default).
- Padding is applied by components when they render content (RichText, Table cells, List items). The cellwriter does not need to know about padding directly.

**Definition of Done:**

- [ ] `props.Cell` has 4 per-side padding fields and 4 per-side border fields (color + thickness)
- [ ] A snapshot or mock-call test asserts that a `props.Cell` with only `BorderTopThickness: 1.0` set results in exactly one `pdf.Line` call (top side) and zero `pdf.Line` calls on the other three sides
- [ ] A test renders a cell with `BorderTopThickness:1.0`, `BorderTopColor: red`, `BorderLeftThickness:0.2`, `BorderLeftColor: black` and verifies the gofpdf mock received exactly 2 `pdf.SetDrawColor` + `pdf.SetLineWidth` + `pdf.Line` triples with different values — confirms different colors/thicknesses per side are achievable
- [ ] Regression: a `props.Cell` with legacy `BorderType: border.Full` still renders four sides via existing `pdf.CellFormat` (assert same mock-call sequence as today; PerSideBorderStyler is NOT selected when no per-side fields are set)
- [ ] Setting `PaddingLeft: 5` on a cell containing RichText shifts text 5mm right inside the cell — verified by integration test
- [ ] Existing cell tests still pass (no regressions)
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/props/... ./internal/providers/gofpdf/cellwriter/... -v`
- Visual: open `test/fixture` outputs and confirm border + padding render in `example_test.go` cell snapshot

---

### Task 3: RichText component with inline run wrapping

**Objective:** Public `pkg/components/richtext` component that renders a paragraph of mixed inline runs (bold/italic/colored/sized text, hyperlinks, sub/super). Wraps text on word boundaries across runs.

**Commit:** `feat(richtext): inline runs, word-wrap and box model`

**Dependencies:** Task 1, Task 2

**Files:**

- Create: `pkg/components/richtext/richtext.go`
- Create: `pkg/components/richtext/richtext_test.go`
- Create: `pkg/components/richtext/example_test.go`
- Create: `test/components/richtext/` (snapshot fixtures)
- Modify: `internal/fixture/fixture.go` (add `RichTextProp()`, `RichRuns()`)

**Key Decisions / Notes:**

- API mirrors text component: `richtext.New(runs []props.RichRun, prop ...props.RichText)`, `NewCol(size, runs, prop)`, `NewRow(height, runs, prop)`, `NewAutoRow(runs, prop)`.
- `GetHeight(provider, cell)` calls `provider.MeasureString` per run to plan line breaks, returns total height.
- **GetHeight memoization (mandatory):** Maroto calls `GetHeight` twice for auto-height rows (once in `addRow` for page-fit check at `maroto.go:214`, once during `Render` at `row.go:89`). RichText caches the computed height keyed by `(cellWidth, configFingerprint)` — fingerprint covers `config.DefaultFont` family/style/size/color and `config.MaxGridSize`. If the cache key matches, return cached height without re-running the word-wrap pass. This avoids height drift across the two passes if `SetConfig` was applied between them, and prevents O(N²) work on large documents.
- `SetConfig` MUST be called before the first `GetHeight` — already true in Maroto's `addRow` flow (`maroto.go:213` calls `r.SetConfig` before `r.GetHeight`), but document this in the component's contract.
- `Render` calls `provider.AddRichText(runs, cell, prop)` — provider does the actual glyph placement. The RichText component takes a `core.RichTextProvider` (the narrow interface from Task 1), accessible via type assertion on the `core.Provider` passed to `Render`. If the assertion fails (some future non-gofpdf provider doesn't implement it), `Render` falls back to plain `provider.AddText` with the first run's style and logs a warning.
- Empty runs collapse to single space; consecutive whitespace collapses to one (HTML behavior).

**Definition of Done:**

- [ ] `richtext.New([]props.RichRun{{Text:"hello ", Style:""}, {Text:"world", Style:"B"}})` returns a `core.Component`
- [ ] `GetHeight` computed value matches actual rendered height ±0.5mm for a 3-run paragraph
- [ ] Word wrap across run boundaries: input `[{Text:"the quick brown "}, {Text:"fox", Style:"B"}, {Text:" jumps"}]` in a 30mm column wraps at the correct word and continues on the next line with the right style applied
- [ ] Hyperlink run renders as blue underlined text with the `LinkString` annotation
- [ ] **Memoization test:** `GetHeight` called twice in a row on the same RichText component with identical cell width returns the same value AND the second call invokes `provider.MeasureString` zero times (verified via mock-call count = 0 on second invocation)
- [ ] Structure snapshot fixture matches
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/components/richtext/... -v`
- `go test ./internal/providers/gofpdf/... -run RichText -v`

---

### Task 4: Table component with rowspan/colspan/borders

**Objective:** Public `pkg/components/table` component modelling `<table>` semantics: cells with row/col spans, per-cell padding/borders/background, header rows, column widths.

**Commit:** `feat(table): table with rowspan, colspan, per-cell styling`

**Dependencies:** Task 2

**Files:**

- Create: `pkg/components/table/table.go`
- Create: `pkg/components/table/cell.go`
- Create: `pkg/components/table/table_test.go`
- Create: `pkg/components/table/cell_test.go`
- Create: `pkg/components/table/example_test.go`
- Create: `test/components/table/` snapshot fixtures
- Create: `pkg/props/table.go` (TableProp, TableCell, ColumnWidth)
- Create: `pkg/props/table_test.go`

**Key Decisions / Notes:**

- Table is a `core.Component` that participates in row/col layout. `NewAutoRow(table.New(...))` is the typical usage.
- Internally: 2D grid normalized for spans before rendering. Algorithm: walk declared cells, mark grid slots occupied by spans, refuse overlap (error).
- **Column count derivation:** column count is computed from the NORMALIZED grid (not from the first row). It equals the maximum effective column index reached by any row after accounting for colspans. This handles cases where the first row is a single colspan-N header cell — the table still has N columns derived from later rows.
- Column widths: explicit (`[20, 50, 30]` percentages or mm), or "auto" (equal split among the derived column count). Stretches to parent cell width.
- **Row height algorithm (two-pass, handles rowspan):**
  1. **Pass 1 (single-row cells):** compute `rowHeight[i] = max(content height of all non-spanning cells in row i)`.
  2. **Pass 2 (spanning cells):** for each cell C with `rowspan = k` starting at row `i`, compute `contentH = height of C`. Let `currentSum = rowHeight[i] + ... + rowHeight[i+k-1]`. If `contentH > currentSum`, distribute the deficit `delta = contentH - currentSum` across the spanned rows proportionally to their current heights (so empty rows still grow). Each spanned row's height increases by `delta * rowHeight[r] / currentSum` (or `delta / k` if `currentSum == 0`).
  3. **Pass 2 iteration:** if multiple rowspan cells affect the same rows, run pass 2 until heights are stable (typically 1-2 iterations; cap at 5 to avoid pathological inputs).
- Inside a `<td>` we render exactly one component (typically RichText or another Table for nested tables) — multiple components per cell handled by stacking richtexts in subsequent versions.
- Cell rendering uses Task 2's per-side border props for clean grid borders (no double-thickness on shared edges — first-pass: each cell draws its own borders; collisions documented as known v1 quirk).

**Definition of Done:**

- [ ] `table.New(cells [][]table.Cell)` builds a table from a row-major matrix; cell can declare `Colspan`/`Rowspan` > 1
- [ ] Overlapping spans return `ErrTableSpanOverlap` from `New`
- [ ] Grid borders render around each cell when `BorderType != None`
- [ ] Column widths: explicit slice + "auto" both work
- [ ] Snapshot test for 3×3 table with one 2-col span and one 2-row span
- [ ] **Rowspan height test:** 3-row table with cell (0,0) `rowspan=3` containing a 5-paragraph RichText that is taller than the other rows. After construction, the sum of `rowHeight[0..2]` equals the spanned cell's content height (no clipping, no gap). Verified by `table.New(...).GetHeight(provider, &cell)` returning the spanned cell's required height.
- [ ] **Column-count from grid test:** table whose first row is a single colspan=3 header and subsequent rows have 3 cells each — the table correctly derives 3 columns from the normalized grid (not 1 from the first row)
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/components/table/... -v`
- `go test ./pkg/props/table_test.go -v`

---

### Task 5: HTMLList component (ul/ol with markers and nesting)

**Objective:** Public `pkg/components/htmllist` for HTML-style lists. Bullets (`•`, `◦`, `▪`) or numbered (1., 2., a., A., i., I.). Supports nesting with indentation.

**Commit:** `feat(htmllist): bullet and numbered lists with nesting`

**Dependencies:** Task 3 (uses RichText for item content)

**Files:**

- Create: `pkg/components/htmllist/htmllist.go`
- Create: `pkg/components/htmllist/marker.go` (number formatting: decimal, lower-alpha, upper-alpha, lower-roman, upper-roman)
- Create: `pkg/components/htmllist/htmllist_test.go`
- Create: `pkg/components/htmllist/marker_test.go`
- Create: `pkg/components/htmllist/example_test.go`
- Create: `test/components/htmllist/` snapshot fixtures
- Create: `pkg/props/htmllist.go`

**Key Decisions / Notes:**

- `htmllist.New(items []htmllist.Item, prop ...props.HTMLList)`. Item: `{Content core.Component, SubList *List}` so any component (typically RichText) plus optional nested list.
- Marker: prop carries `Style: bullet|decimal|lower-alpha|upper-alpha|lower-roman|upper-roman` and `Indent` (mm per level).
- **Gutter sizing (computed lazily):** the gutter width is computed at `GetHeight` time (when a provider is available), NOT at `New()` time. Algorithm: for each item, render its marker string (e.g. `"100."`, `"viii."`) and call `provider.MeasureString` on it. Gutter = max marker width + `props.HTMLList.MarkerPadding` (default 1mm). Cache the result in the component (same memoization pattern as Task 3's RichText). Users can override with `props.HTMLList.GutterWidth` (mm); if set, that value is used and no measurement happens.
- Layout: each item is a "mini-row" — marker in the (computed or fixed) gutter, content in the remainder. Multi-line content correctly aligns to the marker's first line.
- Roman/alpha number conversion is a small pure function in marker.go.

**Definition of Done:**

- [ ] `htmllist.New([]htmllist.Item{...})` with 3 items renders 3 bullet lines
- [ ] Decimal numbering: items 1-3 produce "1.", "2.", "3."; lower-roman produces "i.", "ii.", "iii."
- [ ] Nested list (item with `SubList`) renders one indent level deeper
- [ ] Wrapping inside an item keeps subsequent lines aligned with the content, not the marker
- [ ] **Gutter sizing test:** 150-item decimal list with default 10pt Helvetica — `"150."` marker does not overflow the gutter (measured gutter width ≥ `MeasureString("150.")` + MarkerPadding)
- [ ] Snapshot fixture for mixed nesting
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/components/htmllist/... -v`

---

### Task 6: HTML DOM wrapper with style extraction

**Objective:** Internal package that parses HTML via `golang.org/x/net/html` and produces a Maroto-friendly DOM walker. Extracts `<style>` block CSS text and per-element inline `style` attributes.

**Commit:** `feat(html): DOM wrapper and style extraction`

**Dependencies:** None

**Files:**

- Create: `pkg/html/dom/dom.go`
- Create: `pkg/html/dom/dom_test.go`
- Create: `pkg/html/dom/walker.go`
- Create: `pkg/html/dom/walker_test.go`
- Modify: `go.mod` / `go.sum` (`go get golang.org/x/net`)

**Key Decisions / Notes:**

- `dom.Parse(html string) (*Document, error)` wraps `html.Parse` from `golang.org/x/net/html`.
- `Document` exposes: `Root() *Node`, `StyleText() string` (concatenated `<style>` contents from `<head>` and inline), `Title() string`.
- `Node` is a thin wrapper exposing: `Tag()`, `Attr(name) string`, `Children()`, `TextContent()`, `InlineStyle() string`, `IsBlock()`, `IsInline()`.
- Block/inline classification per HTML5 default UA stylesheet — a static lookup table in `walker.go`.
- Whitespace handling: collapse runs of whitespace between inline nodes; preserve in `<pre>` and `<code>` block descendants.

**Definition of Done:**

- [ ] `dom.Parse("<html><head><style>p{color:red}</style></head><body><p style='font-weight:bold'>hi</p></body></html>")` returns a Document
- [ ] `doc.StyleText()` returns `"p{color:red}"`
- [ ] Walking finds the `<p>` node, `.InlineStyle()` returns `"font-weight:bold"`, `.TextContent()` returns `"hi"`
- [ ] Block/inline classification: `<div>` is block, `<span>` is inline, `<a>` is inline
- [ ] Whitespace collapsing: `"<p>hello   world</p>"` → text content `"hello world"`; `<pre>hello   world</pre>` preserves spaces
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/html/dom/... -v`

---

### Task 7: CSS parser + cascade + computed styles

**Objective:** Internal CSS parsing layer. Parse `<style>` blocks + inline styles, run cascade (user-agent defaults → stylesheet → inline) with specificity, produce a `ComputedStyle` for each DOM node.

**Commit:** `feat(html): CSS parser, cascade, and computed styles`

**Dependencies:** Task 6

**Files:**

- Create: `pkg/html/css/parser.go` (wraps `aymerick/douceur/parser`)
- Create: `pkg/html/css/shorthand.go` (expand `border`, `padding`, `margin`, `font` shorthands)
- Create: `pkg/html/css/cascade.go`
- Create: `pkg/html/css/specificity.go`
- Create: `pkg/html/css/computed.go` (`ComputedStyle` struct + property setters)
- Create: `pkg/html/css/defaults.go` (user-agent default stylesheet)
- Create: `pkg/html/css/parser_test.go`
- Create: `pkg/html/css/shorthand_test.go`
- Create: `pkg/html/css/cascade_test.go`
- Create: `pkg/html/css/computed_test.go`
- Create: `pkg/html/css/specificity_test.go`
- Modify: `go.mod` / `go.sum` (`go get github.com/aymerick/douceur github.com/andybalholm/cascadia`)

**Key Decisions / Notes:**

- `ComputedStyle` fields are the subset listed in Summary: color, bg color, font-family/size/weight/style, text-align/decoration, line-height, padding-X, margin-X, border-X, width, height, display, vertical-align.
- Specificity: standard a-b-c-d formula (`a` = inline, `b` = #id, `c` = .class/attr/pseudo-class, `d` = element/pseudo-element).
- Selector matching uses `cascadia.MustCompile` — pass `*html.Node` from x/net/html.
- Defaults: minimal UA stylesheet hardcoded as a string parsed at init. Covers `h1-6` sizing, `p` margins, `strong/b` bold, `em/i` italic, `u` underline, `s/strike` strikethrough, `a` blue+underline, `code` monospace, `pre` monospace+preserve-ws, list defaults, table defaults.
- Unsupported CSS properties are skipped silently. Unparseable CSS values (e.g. `font-size: 1.2rem` if rem isn't supported) skip just that property and log via the on-unsupported callback (Task 11).
- Length units supported: `px` (1px=0.264583mm), `pt` (1pt=0.352778mm), `mm`, `cm`, `em` (relative to current font-size), `%` (context-dependent), unitless line-height.
- **em-resolution algorithm (tree walk, mandatory):** The cascade walker is recursive and threads the parent's computed font-size (in absolute mm) down to each child:
  1. Root: `currentFontSize = config.DefaultFont.Size` (converted from pt to mm).
  2. For each child node: first resolve THIS node's `font-size` declaration (if any) against the parent's `currentFontSize` (so `1.5em` becomes `1.5 * parentSize`).
  3. After resolving font-size, resolve all OTHER em-based lengths (padding, margin, border, etc.) against THIS node's now-computed font-size.
  4. Pass THIS node's computed font-size down as `currentFontSize` for its children.
- **CSS shorthand expansion (mandatory):** douceur returns declarations as-written, not expanded. `pkg/html/css/parser.go` includes a `expandShorthands(decls []css.Declaration) []css.Declaration` step that runs AFTER douceur parsing and BEFORE cascade:
  - `border: 1px solid red` → `border-{top,right,bottom,left}-width:1px` + `-style:solid` + `-color:red` (all 4 sides)
  - `padding: 5px 10px` → `padding-top:5px; padding-right:10px; padding-bottom:5px; padding-left:10px` (2-value form)
  - `padding: 5px 10px 15px 20px` → 4-value form
  - `margin: ...` → same as padding
  - `font: 12pt Helvetica` → `font-size:12pt; font-family:Helvetica`
  - Per-side shorthand: `border-top: 1px solid black` → `border-top-width:1px; border-top-style:solid; border-top-color:black`

**Definition of Done:**

- [ ] `css.Parse("p { color: red; font-size: 12pt } .x { color: blue }")` returns parsed rules
- [ ] `css.Compute(node, stylesheet, inlineStyle)` returns ComputedStyle with merged properties
- [ ] Specificity: `.x` beats `p`, inline `style=""` beats both
- [ ] Length parsing: `"12pt"` → 12 × 0.352778 mm, `"5mm"` → 5, `"16px"` → 16 × 0.264583 mm
- [ ] em resolution with font-size inheritance: node with `font-size: 16px` and `padding: 1em` computes padding ≈ 4.23mm; a child with `font-size: 0.5em` resolves its own font-size to 8px first, then resolves subsequent ems against that. Covered by a table-driven test in `css/computed_test.go` with at least 3 nested levels.
- [ ] Defaults loaded: `<h1>` ComputedStyle has bold weight and ~24pt size
- [ ] Unsupported property (`box-shadow`) does not error, returns ComputedStyle with that field unset
- [ ] **Shorthand expansion test:** `border: 1px solid red` expands to all 8 longhand properties (4 sides × {width, style, color}); `padding: 5 10` expands to top:5/right:10/bottom:5/left:10; `font: 12pt Helvetica` expands to font-size and font-family. Verified by direct unit tests in `shorthand_test.go` BEFORE cascade is applied.
- [ ] **End-to-end shorthand integration:** `<table style='border: 1px solid black'>` produces a ComputedStyle with all four `border-X-width=1px`, `border-X-style=solid`, `border-X-color=black` set — proving expansion + cascade work together.
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/html/css/... -v`

---

### Task 8: DOM-to-Maroto-AST translator (block elements)

**Objective:** Walk styled DOM, emit `[]core.Row` for block-level elements (`div`, `p`, `h1-6`, `hr`, `pre`, `blockquote`, semantic sectioning tags). Inline content collected into RichText runs (full inline handling in Task 9). Tables/lists are placeholders (full impl Task 10).

> **Note:** Tasks 8, 9, and 10 are a single continuous implementation unit committed together at the end of Task 10. Mark Tasks 8 and 9 complete in progress tracking only after Task 10 commits. If implementation stalls mid-Task 9, run `git stash` to preserve in-progress work rather than committing partial state.

**Commit:** none (intermediate; commits in Task 10)

**Dependencies:** Task 3, Task 6, Task 7

**Files:**

- Create: `pkg/html/translate/translate.go` (main walker)
- Create: `pkg/html/translate/block.go` (block element handlers)
- Create: `pkg/html/translate/context.go` (style stack, current parent row builder)
- Create: `pkg/html/translate/translate_test.go`
- Create: `pkg/html/translate/block_test.go`

**Key Decisions / Notes:**

- Walker is recursive, depth-first.
- Each block element either produces a row (paragraph-like) or recurses (container-like).
- `<div>` and other containers: no row of their own; descend with style context pushed onto a stack. **Documented v1 limitation:** CSS `background-color` on a container repeats per contained row (visually equivalent if rows are adjacent). CSS `border` on a container is rendered per row (appears as multiple stacked rectangles, NOT a single spanning rectangle around all children). CSS `padding` is applied to each contained row's content (effectively duplicating padding instead of applying to the outer bounds). Margin collapsing between adjacent blocks is silently lost. This is the most important known limitation in v1 and MUST be documented prominently in `docs/v2/html-support.md`. Users needing a true single-rectangle container border should wrap content in a `<table>` (which has full grid borders) — documented workaround.
- `<p>` and headings: produce a single auto-height row with RichText.
- `<hr>`: produces a row with a Line component using the existing `pkg/components/line`.
- `<pre>` / `<code>` block: preserve whitespace, monospace font, no wrap (clip overflow). Wrap with `breakline.DashStrategy` as fallback.
- Page-break handling: respect `page-break-before: always` and `page-break-after: always` by inserting an empty filler row sized to remaining page height (caller post-processes).
- `display: none` → skip subtree entirely.

**Definition of Done:**

- [ ] `translate.Translate(domDoc, styleSheet)` returns `[]core.Row`
- [ ] `<h1>Hello</h1><p>World</p>` produces 2 rows; first contains RichText with H1 default styles
- [ ] `<div style="background-color:red"><p>x</p></div>` produces a row containing the `<p>`, where the row's WithStyle has BackgroundColor red
- [ ] **Documented div-flattening behavior test:** `<div style="border:1px solid black"><p>a</p><p>b</p></div>` produces TWO rows each with their own black border (NOT a single border surrounding both). This is the documented v1 behavior; the test exists to lock in the approximation and prevent silent regression. The PDF visual is captured in `docs/v2/html-support.md` as a known limitation example.
- [ ] `<hr>` produces a row with a Line component
- [ ] `<p style="display:none">hidden</p>` produces no row
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/html/translate/... -v`

---

### Task 9: Inline element translation into RichText runs

**Objective:** Within block contexts, walk inline children (`span`, `b`, `i`, `u`, `s`, `a`, `strong`, `em`, `sub`, `sup`, `code` inline, `img` inline-block, `br`) and emit `[]props.RichRun` carrying the cascaded styles. Used by Task 8 to fill paragraph rows.

**Commit:** none (intermediate; commit in Task 10)

**Dependencies:** Task 8

**Files:**

- Create: `pkg/html/translate/inline.go`
- Create: `pkg/html/translate/inline_test.go`

**Key Decisions / Notes:**

- Inline walker returns `[]InlineToken` where `InlineToken` is `{ Run *props.RichRun, Image *InlineImage, BreakParagraph bool }`. Most tokens are runs; some are images that force a paragraph split.
- For each text node child, emit a token with `Run = &props.RichRun{...}` carrying the parent's cascaded ComputedStyle.
- `<br>` → run with `Text: "\n"` (RichText renderer treats `\n` as forced line break).
- `<a href="...">` → run with `Hyperlink = &href`.
- **Inline `<img>` handling (v1 contract — NO leaking image fields into props.RichRun):** when an inline `<img>` is encountered mid-paragraph, the inline walker emits a token with `Image: &InlineImage{Src, Alt, Width, Height}`. The block walker (Task 8) splits the paragraph at every image token into a sequence of sub-rows:
  1. Rows of preceding inline runs (as RichText)
  2. A separate row containing just the image (as `pkg/components/image`)
  3. Rows of subsequent inline runs (as RichText)
  This is a deliberate v1 simplification — true text-flow-around-image requires reworking RichText to consume an Image primitive, which is deferred to v2. Documented in `docs/v2/html-support.md`.
- Style merging: per-run style is built by walking from the run's text-node ancestor up to the block parent, merging ComputedStyles.

**Definition of Done:**

- [ ] `<p>hello <b>bold <i>both</i></b> end</p>` produces one row whose RichText has 4 runs: `"hello "` (normal), `"bold "` (bold), `"both"` (bold+italic), `" end"` (normal)
- [ ] `<a href="https://x">link</a>` produces a run with hyperlink set
- [ ] `<br>` produces a run with `\n` text
- [ ] **Inline image split test:** `<p>See figure <img src="chart.png"> for details.</p>` produces exactly 3 rows: (1) RichText "See figure ", (2) image row, (3) RichText " for details." — no `InlineImage` field appears on `props.RichRun` (verified by `git grep "InlineImage" pkg/props/` returning no matches)
- [ ] Snapshot test for a moderately complex `<p>` with mixed inline children
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/html/translate/... -v`
- `go test ./pkg/components/richtext/... -v` (regression — inline runs from translator render correctly)

---

### Task 10: Table/list translation

**Objective:** Translate `<table>` / `<thead>` / `<tbody>` / `<tr>` / `<th>` / `<td>` into Table component, and `<ul>` / `<ol>` / `<li>` into HTMLList component. Handle nesting (table-in-cell, list-in-list).

**Commit:** `feat(html): translate DOM to Maroto AST (block, inline, table, list)`

**Dependencies:** Task 4, Task 5, Task 8, Task 9

**Files:**

- Create: `pkg/html/translate/table.go`
- Create: `pkg/html/translate/list.go`
- Create: `pkg/html/translate/table_test.go`
- Create: `pkg/html/translate/list_test.go`

**Key Decisions / Notes:**

- Table: parse `<table>` → 2D matrix of `table.Cell`. Read `colspan`/`rowspan` attrs (default 1). `<th>` cells get bold default + center align unless overridden by CSS. Column count is derived from the Table component's normalized grid (Task 4's grid math), NOT from the first row — this handles `<tr><th colspan=3>Header</th></tr>` followed by 3-column rows correctly. Column widths from `<colgroup>`/`<col width>` if present; otherwise auto-equal across the derived column count. Nested `<table>` allowed — recursively translate.
- List: `<ul>` → `htmllist.HTMLList{Style: bullet}`, `<ol>` → `Style: decimal` (or `type` attr → lower-alpha/lower-roman/etc). `<li>` produces an `htmllist.Item`. Nested `<ul>`/`<ol>` inside `<li>` becomes the item's `SubList`.
- `<li>` with mixed inline + block content: inline content becomes the item content (RichText), block children become separate rows after the list (documented quirk in v1).

**Definition of Done:**

- [ ] `<table><tr><td>a</td><td>b</td></tr><tr><td colspan=2>c</td></tr></table>` translates to a Table with the right structure (2×2 grid with col-spanning cell)
- [ ] `<ul><li>x</li><li>y<ul><li>z</li></ul></li></ul>` translates to an HTMLList with two items, second has SubList with one item
- [ ] `<ol type="i">` produces lower-roman markers
- [ ] All tests pass, no diagnostics

**Verify:**

- `go test ./pkg/html/translate/... -v`

---

### Task 11: Public API (pkg/html + Maroto.AddHTML) + documented subset + examples

**Objective:** Expose the HTML pipeline as a clean public API. Document supported subset. Add at least 2 worked examples that produce real PDFs in `pkg/html/example_test.go`.

**Commit:** `feat(html): public API, examples, and documentation`

**Dependencies:** Task 10

**Files:**

- Create: `pkg/html/html.go` (`FromString`, `FromReader`, `Option`, `WithBaseFont`, `WithImageResolver`, `WithDebug`, `WithUnsupportedHandler`)
- Create: `pkg/html/html_test.go`
- Create: `pkg/html/example_test.go` (invoice + article examples)
- Modify: `maroto.go` (add `func (m *Maroto) AddHTML(html string, opts ...html.Option) error`)
- Modify: `maroto_test.go` (test AddHTML integration)
- Modify: `pkg/core/core.go` (add `AddHTML` to `Maroto` interface)
- Create: `docs/v2/html-support.md` (supported HTML tags, CSS properties, known limitations, examples)
- Create: `docs/v2/html-example-invoice.html` (input fixture for example_test.go)
- Modify: `README.md` (add HTML section)
- Modify: `mocks/` regenerate (Maroto interface mock)

**Key Decisions / Notes:**

- `Option` is a functional option closure pattern (consistent with Go idioms).
- `WithImageResolver(func(src string) ([]byte, string, error))` lets users intercept `<img src>` — important for relative paths and remote URLs (default resolver: file system reads, no HTTP).
- `WithUnsupportedHandler(func(tag, css string))` lets users see what's being ignored — useful for diagnostics.
- `AddHTML` is sugar for `rows, err := html.FromString(htmlStr, opts...); if err != nil { return err }; m.AddRows(rows...)`.
- Example test generates a PDF and writes it to `test/output/html-invoice.pdf` — verified to be a valid PDF (file size > 1KB, starts with `%PDF-`).
- Documentation explicitly lists supported tags, supported CSS, and known v1 limitations (no flexbox/grid/floats/JS/etc).

**Definition of Done:**

- [ ] `html.FromString(htmlStr)` works without options
- [ ] `m.AddHTML(htmlStr)` integrates with existing Maroto flow: integration test in `maroto_test.go` registers a 10mm header row, calls `m.AddHTML(htmlStr)` with HTML content sized to force at least 2 pages, then asserts `m.GetStructure()` returns ≥ 2 pages and each page contains the header row structure
- [ ] Pagination test: with content forcing 3 pages, page-number footer renders `1/3`, `2/3`, `3/3`
- [ ] Example test generates `test/output/html-invoice.pdf`; file starts with `%PDF-` and is > 1KB
- [ ] `docs/v2/html-support.md` lists every supported tag and CSS property; readers can determine support without reading code
- [ ] README has a 5-line "HTML to PDF" snippet showing the basic flow
- [ ] `go test ./...` green after Maroto interface mock regeneration (including tests in `maroto_test.go` and `metricsdecorator_test.go`)
- [ ] All tests pass, no diagnostics, lint clean
- [ ] `go vet ./... && gofmt -l . && golangci-lint run` clean (or matches existing baseline)

**Verify:**

- `go test ./pkg/html/... -v`
- `go test ./... -count=1` — full suite green
- `golangci-lint run` — clean (or no new warnings)
- Manual: open `test/output/html-invoice.pdf` in a viewer, confirm visual fidelity

---

## Testing Strategy

- **Unit tests:** Each new component (richtext, table, htmllist) gets its own `*_test.go` mirroring existing patterns. CSS parser, cascade, computed-style each unit-tested with table-driven cases.
- **Integration tests:** Translator tests exercise the full DOM→AST pipeline against representative HTML fixtures (paragraph, heading hierarchy, table with spans, nested lists, mixed inline).
- **Snapshot tests:** Use existing `pkg/test` snapshot infrastructure for component structure JSON (consistent with `pkg/components/text/text_test.go`).
- **PDF smoke tests:** `pkg/html/example_test.go` produces real PDF bytes; assert non-empty and well-formed (`%PDF-` magic).
- **Regression:** After Task 1 (provider interface change) and Task 11 (Maroto interface change), `go test ./...` must remain green.
- **Coverage:** Aim ≥ 80% on new packages, matching project baseline.

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Provider interface change breaks downstream users | High | Med | All new methods are additive; no existing signatures change. Document as `v2.5.0` minor bump per semver. |
| RichText line wrapping has off-by-one or measurement drift across runs | Med | High | Reuse `pdf.GetStringWidth` (same primitive existing Text uses, well-tested). Test against existing single-style line wrapping for parity when all runs share style. |
| Table rowspan/colspan grid math is buggy with overlap | Med | High | `table.New` validates non-overlap at construction; returns `ErrTableSpanOverlap`. Table tests include 3 known-tricky shapes (T-shape, L-shape, full-row span). |
| CSS specificity/cascade incorrect (e.g. inline doesn't beat #id) | Med | Med | Dedicated `specificity_test.go` with cases derived from CSS spec examples. Cascade tested with table-driven cases including ties (later rule wins). |
| Unsupported CSS silently produces wrong output | Med | Med | **Structural guard:** all CSS property setters in `pkg/html/css/computed.go` use an explicit allow-list switch/case keyed on property name. Any property name not in the case statement calls `WithUnsupportedHandler` and leaves `ComputedStyle` at its default — there is no fallback parser. Integration test verifies that a stylesheet of 100% unknown properties (`box-shadow`, `animation`, `transform`, `filter`) produces byte-identical output to an empty stylesheet. `WithUnsupportedHandler` callback exposes ignored properties/tags for user diagnostics. |
| Whitespace handling diverges from browsers (extra/missing spaces around inline elements) | High | Low | Test fixtures derived from real HTML (invoices, articles). Document v1 quirks where they differ; iterate. |
| Pure-Go HTML parser misses edge cases (malformed HTML) | Low | Low | `golang.org/x/net/html` is the de-facto std library; passes html5lib conformance suite. |
| `pkg/components/list` confusion with new HTMLList | Med | Low | New package named `htmllist`, documented prominently in README that they are different things. Possible v3 rename of legacy `list` to `tablelist`. |
| Image resolution breaks on remote `<img src="https://...">` | High | Low | Default resolver only handles local files; users opt in via `WithImageResolver`. Documented behavior. |
| Scope optimism — 11 tasks understate effort | High | Med | The plan is intentionally structured so Tasks 1-5 (RichText, Table, HTMLList + provider/cell extensions) form a self-contained shippable foundation: those components are useful on their own without HTML. After Task 5, a release candidate `v2.5.0-rc1` can ship with the new components even if Tasks 6-11 are still in progress. Tasks 6-11 then ship as `v2.6.0`. This caps blast radius and gives users value earlier. Commit at Task 5 marks the foundation milestone. |

## Goal Verification

> Derived from the plan's goal using goal-backward methodology. The spec-reviewer-goal agent verifies these criteria during verification.

### Truths (what must be TRUE for the goal to be achieved)

- Users can call `m.AddHTML(htmlString)` on a Maroto instance and produce a valid PDF
- Users can call `html.FromString(htmlString)` directly and receive `[]core.Row`
- HTML headings (`<h1>` through `<h6>`), paragraphs, and basic inline styling (bold/italic/underline/color) render with visible style differentiation in the PDF
- HTML tables with `colspan`/`rowspan` render as actual PDF tables with grid borders
- HTML `<ul>` and `<ol>` render with appropriate markers (bullets / numbers / letters / roman numerals)
- CSS supplied via `<style>` blocks AND inline `style=""` attributes both affect the output, with inline winning on conflict
- No external binaries or browsers are spawned at runtime — `go test ./...` passes in an offline, headless CI environment

### Artifacts (what must EXIST to support those truths)

- `pkg/components/richtext/richtext.go` — RichText component with run-based wrapping
- `pkg/components/table/table.go` — Table component supporting spans
- `pkg/components/htmllist/htmllist.go` — HTMLList component with markers and nesting
- `pkg/html/dom/dom.go` — HTML parser wrapper
- `pkg/html/css/computed.go` — ComputedStyle resolver with cascade + specificity
- `pkg/html/translate/translate.go` — DOM-to-AST translator
- `pkg/html/html.go` — Public `FromString` / `FromReader` / `Option`
- `maroto.go` — `AddHTML` method on Maroto
- `docs/v2/html-support.md` — Supported-subset documentation
- `pkg/html/example_test.go` — Working examples that produce real PDFs

### Key Links (critical connections that must be WIRED)

- `m.AddHTML(...)` → `html.FromString(...)` → `dom.Parse + css.Compute + translate.Translate` → `m.AddRows(...)`
- `pkg/components/richtext` → `provider.AddRichText` → `internal/providers/gofpdf/text.AddRichText` → `pdf.Text`/`pdf.GetStringWidth`
- `pkg/html/translate/inline.go` → emits `[]props.RichRun` → consumed by `pkg/components/richtext`
- `pkg/html/translate/table.go` → emits `table.Cell` matrix → consumed by `pkg/components/table`
- `pkg/html/css/cascade.go` → applies stylesheet + inline → emits `ComputedStyle` → consumed by `pkg/html/translate`

## Open Questions

- Are nested tables inside `<td>` truly needed in v1? (Current plan: yes, recursive translation. Could defer to v2.)
- Should `WithImageResolver` allow HTTP by default with a `WithRemoteImages(true)` opt-in? (Current plan: file-only by default; HTTP requires user-supplied resolver.)
- Should we ship a CLI `maroto-html2pdf input.html output.pdf` as a thin wrapper? (Out of scope for this plan; track separately if desired.)

### Deferred Ideas

- CSS variables (`--my-color: red`).
- `@page` rules for page size / margins driven by HTML (currently config-only).
- `<svg>` rendering (would need a separate SVG-to-PDF translator).
- Form fields rendered as fillable PDF widgets (`<input>`, `<textarea>`).
- `@media print` selector branch.
- A v3 rename of legacy `pkg/components/list` to `pkg/components/tablelist` to free up the `list` name.
