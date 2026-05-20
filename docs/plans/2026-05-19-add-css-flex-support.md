# CSS Flex Layout Support Implementation Plan

Created: 2026-05-19
Status: VERIFIED
Approved: Yes
Iterations: 0
Worktree: No
Type: Feature

> **Status Lifecycle:** PENDING → COMPLETE → VERIFIED
> **Approval Gate:** Implementation CANNOT proceed until `Approved: Yes`

## Summary

**Goal:** Extend Maroto's HTML/CSS pipeline to recognise CSS flexbox layout on container elements and translate it to Maroto's configurable grid (default 12 cols), then showcase it in `cmd/html-demo`.

**Architecture:** Add flex properties to `pkg/html/css.ComputedStyle`, expand the `flex` shorthand, extend `ParseLength` to handle `%`, thread `MaxGridSize` (and optional content width) from the maroto config through `html.FromString` into the translator, then add `pkg/html/translate/flex.go` that detects `display:flex` containers and emits a single `core.Row` with quantized cols sized by flex weights via Hamilton's largest-remainder method. Cross-axis (`align-items`) is best-effort via row-height model. `flex-direction:column` collapses to block stacking with optional row-gap spacers.

**Tech Stack:** Pure Go, existing deps (`golang.org/x/net/html`, `aymerick/douceur`, `andybalholm/cascadia`). No new third-party packages.

## Scope

### In Scope

- `display:flex` and `display:inline-flex` (both treated as flex)
- `flex-direction: row | column` (row-reverse / column-reverse accepted but render same as row/column — documented limitation)
- `flex` shorthand (`1`, `auto`, `none`, `initial`, `<grow> <shrink> <basis>`)
- `flex-grow`, `flex-basis` (used in layout). `flex-shrink` parsed and stored but has no independent layout effect — quantizer always sums to grid total so overflow is impossible (documented limitation).
- `flex-basis` with `mm/cm/pt/px/em/rem/auto/%` (% becomes a fractional weight against grid total)
- `justify-content: flex-start | center | flex-end | space-between | space-around`
- `align-items: flex-start | center | flex-end | stretch` (best-effort within Maroto row model — limitations documented)
- `gap`, `column-gap`, `row-gap` (mapped to integer spacer cols)
- Quantization to configured grid size (Hamilton's largest-remainder)
- Class-based `display:none` honoured (currently only inline-style is honoured — fixed during this work)
- Showcase in `cmd/html-demo` with new flex sections
- Updated docs in `docs/v2/html-support.md` (flexbox removed from Out-of-Scope list)

### Out of Scope

- `flex-wrap` (single line only)
- `order` property (children render in source order)
- `align-self` per-item override
- `align-content` (multi-line cross-axis distribution — N/A without wrap)
- CSS Grid (`display:grid`)
- `flex-direction: row-reverse | column-reverse` reordering — accepted but no reversal
- Nested flex inside table cells — works via normal recursion but no special handling

## Prerequisites

- Existing HTML→PDF pipeline (plan `docs/plans/2026-05-19-html-to-pdf.md`, VERIFIED)
- Maroto v2 configurable grid (default 12) with row/col components

## Context for Implementer

- **Patterns to follow:**
  - CSS property parsing: `pkg/html/css/computed.go` — switch in `ComputedStyle.Apply` (starts ~line 88). Add new cases there.
  - CSS shorthand expansion: `pkg/html/css/shorthand.go` — `expandOne` dispatch (line 19). Add `flex` case.
  - Length parsing: `pkg/html/css/length.go:17` — `ParseLength` returns mm; returns 0 for unparseable.
  - Translator block dispatch: `pkg/html/translate/translate.go` — `blockRows` function (~line 57); tag switch (~line 66). Flex containers intercept via style check in default branch.
  - Style resolution: `pkg/html/translate/style.go:14` — `computeNodeStyle` already combines stylesheet + inline.
  - HTML public API: `pkg/html/html.go` — `FromString` and `Option` pattern.
  - Top-level entry: `maroto.go` `AddHTML` (line 126) calls `html.FromString` — needs to thread `cfg.MaxGridSize` (and content width when available).
- **Conventions:**
  - File length < 300 lines; refactor at that boundary.
  - Table-driven tests using `testify/assert` and `t.Parallel()`.
  - All struct fields documented; package-level types have package comments.
- **Key files to read first:**
  - `pkg/html/translate/translate.go` (block walker)
  - `pkg/html/css/computed.go` (style fields + Apply)
  - `pkg/components/row/row.go` and `pkg/components/col/col.go` (Maroto grid)
  - `pkg/config/builder.go:197` (WithMaxGridSize)
- **Gotchas:**
  - `col.New()` with no args = max-width col. For flex items use explicit `col.New(size)`.
  - `props.RichRun.Size` is **pt**, `ComputedStyle.FontSize` is **mm**. Existing code (`translate.go:115`) handles pt conversion.
  - `parseStylesheet` (`stylesheet.go:42`) calls `css.ExpandShorthands` BEFORE compiling rules — the `flex` shorthand expansion flows through automatically.
  - `dom.Node.Children()` (`dom/dom.go:87`) returns BOTH `ElementNode` and `TextNode`. Whitespace-only text nodes between block siblings MUST be filtered before counting flex items.
  - Existing `isDisplayNone` checks only inline style (`translate.go:175`); class-based `display:none` is currently NOT honoured. This work fixes that asymmetry in the default branch.
  - The existing `Translate(doc)` signature does NOT take config. We extend it via `TranslateOption` (variadic) and thread from `html.FromString` and `maroto.AddHTML`.
- **Domain context:**
  - Grid size defaults to 12 but is user-configurable via `WithMaxGridSize(n)`. The flex quantizer MUST honour the configured value, not the literal 12.
  - Quantization is unavoidable. At default A4 with 20mm L+R margins, content width = 170mm and 1 col ≈ 14.17mm (170/12). For custom grid/page configurations the mm/col value differs.
- **Hamilton's largest-remainder method (LRM):**
  1. Compute exact share per item: `(weight_i / total_weight) * gridTotal`
  2. Floor each → integer share; remaining cells = `gridTotal - sum(floors)`
  3. Sort items by fractional part descending; give remaining cells one-at-a-time to highest fractions
  4. **Guards:** empty input returns `[]int{}`; `total_weight == 0` → equal split with remainder distributed left-to-right.

## Runtime Environment

- **Build/run demo:** `go run ./cmd/html-demo/`
- **Output:** `/Users/avdo/maroto/test/output/html-demo.pdf`
- **Visual verify:** `open /Users/avdo/maroto/test/output/html-demo.pdf`

## Progress Tracking

- [x] Task 1: CSS flex property fields, Apply cases, % length support
- [x] Task 2: `flex` shorthand expansion
- [x] Task 3: Hamilton quantizer + flex layout module (leaf items)
- [x] Task 4: Thread MaxGridSize through translator + flex detection in blockRows
- [x] Task 5: Justify-content and gap
- [x] Task 6: Align-items, flex-direction:column, non-leaf flex items
- [x] Task 7: Demo showcase + structural integration test + docs

**Total Tasks:** 7 | **Completed:** 7 | **Remaining:** 0

## Implementation Tasks

### Task 1: CSS flex property fields, Apply cases, % length support

**Objective:** Extend `css.ComputedStyle` with flex container/item fields, recognise the corresponding CSS properties in `Apply`, and extend `ParseLength` to handle `%` units (returning a sentinel-style storage for percentage flex-basis).

**Dependencies:** None

**Files:**

- Modify: `pkg/html/css/computed.go`
- Modify: `pkg/html/css/length.go`
- Modify: `pkg/html/css/css_test.go`

**Key Decisions / Notes:**

- New fields on `ComputedStyle`:
  - `FlexDirection string` (`""`, `"row"`, `"column"`, `"row-reverse"`, `"column-reverse"`)
  - `JustifyContent string`
  - `AlignItems string`
  - `FlexGrow float64`
  - `FlexShrink float64` (parsed/stored only; documented as no-op in v1)
  - `FlexBasis float64` (mm; 0 = auto unless `FlexBasisPct > 0`)
  - `FlexBasisAuto bool` (true when explicit `auto`)
  - `FlexBasisPct float64` (>0 when value was a percentage; mutually exclusive with `FlexBasis`)
  - `RowGap float64`, `ColumnGap float64`
- `display:flex` and `display:inline-flex` both normalise to `Display="flex"` inside the `case "display":` Apply branch.
- New `Apply` cases:
  - `flex-direction` → store value
  - `justify-content` → store value
  - `align-items` → store value
  - `flex-grow`, `flex-shrink` → `strconv.ParseFloat`
  - `flex-basis` → if `"auto"` → `FlexBasisAuto=true, FlexBasis=0, FlexBasisPct=0`; if ends with `"%"` → set `FlexBasisPct`; else `FlexBasis = ParseLength(...)`
  - `gap` → split into 1–2 tokens for row-gap/column-gap
  - `row-gap`, `column-gap` → `ParseLength`
- `ParseLength`: add `%` handling. New behaviour: when value ends with `%`, return a negative sentinel (e.g. `-pct/100`) OR provide a sibling helper `ParsePercentage(value string) (float64, bool)` returning fraction. The latter is cleaner — use it in the `flex-basis` Apply case explicitly. `ParseLength` itself stays unchanged for `%` (still returns 0) — callers requiring `%` MUST use the new helper.

**Definition of Done:**

- [ ] All new fields exist on `ComputedStyle` with documented zero-values
- [ ] `s.Apply("display", "flex", nil)` → `s.Display == "flex"`
- [ ] `s.Apply("display", "inline-flex", nil)` → `s.Display == "flex"` (explicit DoD test, not just notes)
- [ ] `s.Apply("flex-grow", "2", nil)` → `s.FlexGrow == 2.0`
- [ ] `s.Apply("flex-shrink", "0", nil)` → `s.FlexShrink == 0` (stored)
- [ ] `s.Apply("flex-basis", "auto", nil)` → `FlexBasisAuto=true, FlexBasis=0, FlexBasisPct=0`
- [ ] `s.Apply("flex-basis", "50mm", nil)` → `FlexBasis=50.0, FlexBasisPct=0`
- [ ] `s.Apply("flex-basis", "25%", nil)` → `FlexBasisPct=25.0, FlexBasis=0, FlexBasisAuto=false`
- [ ] `s.Apply("gap", "5mm 10mm", nil)` → `RowGap=5, ColumnGap=10`
- [ ] `s.Apply("gap", "5mm", nil)` → `RowGap=5, ColumnGap=5`
- [ ] `ParsePercentage("25%")` returns `(0.25, true)`; `ParsePercentage("50px")` returns `(0, false)`

**Verify:**

- `go test ./pkg/html/css/... -run "TestComputedStyle|TestParse" -race -count=1`

### Task 2: `flex` shorthand expansion

**Objective:** Expand the `flex` shorthand into `flex-grow`/`flex-shrink`/`flex-basis` longhands in both `<style>` blocks and inline `style=""` attributes.

**Dependencies:** Task 1

**Commit:** `feat(html): add CSS flex property parsing`

**Files:**

- Modify: `pkg/html/css/shorthand.go`
- Modify: `pkg/html/css/css_test.go`

**Key Decisions / Notes:**

- Add `case "flex"` in `expandOne` dispatching to new `expandFlex(val string)`.
- Shorthand rules (per MDN):
  - `flex: none` → `flex-grow:0; flex-shrink:0; flex-basis:auto`
  - `flex: auto` → `flex-grow:1; flex-shrink:1; flex-basis:auto`
  - `flex: initial` → `flex-grow:0; flex-shrink:1; flex-basis:auto`
  - `flex: <n>` (single number) → `flex-grow:n; flex-shrink:1; flex-basis:0`
  - `flex: <n> <basis>` (number + length/% / auto) → `flex-grow:n; flex-shrink:1; flex-basis:<basis>`
  - `flex: <n> <m>` (two numbers) → `flex-grow:n; flex-shrink:m; flex-basis:0`
  - `flex: <grow> <shrink> <basis>` → all three explicit
- Token classification: "number" = unitless float; "basis" = ends with length unit, `%`, or equals `auto`.
- Reuse `isLengthValue` (`shorthand.go:138`), extend to consider `%` and `auto` as basis tokens.

**Definition of Done:**

- [ ] `ExpandShorthands({"flex":"1"})` → `{flex-grow:"1", flex-shrink:"1", flex-basis:"0"}`
- [ ] `ExpandShorthands({"flex":"auto"})` → `flex-grow:"1", flex-shrink:"1", flex-basis:"auto"`
- [ ] `ExpandShorthands({"flex":"none"})` → `flex-grow:"0", flex-shrink:"0", flex-basis:"auto"`
- [ ] `ExpandShorthands({"flex":"initial"})` → `flex-grow:"0", flex-shrink:"1", flex-basis:"auto"`
- [ ] `ExpandShorthands({"flex":"2 50mm"})` → `flex-grow:"2", flex-shrink:"1", flex-basis:"50mm"`
- [ ] `ExpandShorthands({"flex":"1 0 100%"})` → `flex-grow:"1", flex-shrink:"0", flex-basis:"100%"`
- [ ] `ExpandShorthands({"flex":"3 2"})` → `flex-grow:"3", flex-shrink:"2", flex-basis:"0"`

**Verify:**

- `go test ./pkg/html/css/... -race -count=1`

### Task 3: Hamilton quantizer + flex layout module (leaf items)

**Objective:** Add `pkg/html/translate/flex.go` with the core algorithm: collect flex item children (filtering whitespace), compute weights from grow/basis(%/mm), quantize to grid via Hamilton's LRM, emit a `core.Row` of `core.Col`s. This task supports LEAF flex items only (those whose descendants are inline). Non-leaf and gap/justify land in later tasks.

**Dependencies:** Task 1, Task 2

**Files:**

- Create: `pkg/html/translate/flex.go`
- Create: `pkg/html/translate/flex_test.go`

**Key Decisions / Notes:**

- Public entry: `(tr *translator) flexRow(n *dom.Node, containerStyle *css.ComputedStyle) core.Row`
- Translator now carries `gridSize int` (default 12, threaded in Task 4).
- Steps inside `flexRow`:
  1. Iterate `n.Children()`; filter out whitespace text nodes with helper `isWhitespaceTextNode(c) bool { return c.Tag()=="" && strings.TrimSpace(c.TextContent())=="" }`. Each remaining child is a flex item.
  2. For each item: compute style via `computeNodeStyle(tr.sheet, child, containerStyle)` — pass `containerStyle` as parent so em units resolve correctly. Raw text nodes use default-style runs.
  3. Weight calculation:
     - If `FlexBasisPct > 0` → weight = `FlexBasisPct/100 * gridSize`
     - Else if `FlexGrow > 0` → weight = `FlexGrow`
     - Else if `FlexBasis > 0` → weight = `FlexBasis` (mm; treated as proportional weight)
     - Else → weight = 1.0 (default)
  4. Quantize: `hamilton(weights, gridSize)` returns per-item integer share.
  5. Build cols: each item → `col.New(share)`. Leaf items (inline content only) → `richtext.New(inlineRuns(child))`. **Non-leaf items in this task: use the existing `inlineRuns` over a synthetic wrap that calls `child.TextContent()`** — wired as a temporary stub to be replaced in Task 6.
- `hamilton(weights []float64, total int) []int`:
  - Returns `[]int{}` for empty input (no panic, no division).
  - If `sum(weights) == 0` → treat all weights as 1 (equal split) then proceed.
  - Single item → `[total]`.
  - Largest-remainder allocation distributing residue by descending fractional part.
- The translator's `gridSize` defaults to 12 if unset (`gridSize == 0` → use 12 in `flexRow`).

**Definition of Done:**

- [ ] `hamilton([]float64{1,1,1}, 12)` → `[4,4,4]`
- [ ] `hamilton([]float64{1,2,1}, 12)` → `[3,6,3]`
- [ ] `hamilton([]float64{1,1,1,1,1}, 12)` → sums to 12, max element 3, min element 2 (Hamilton specific order)
- [ ] `hamilton([]float64{1,3}, 12)` → `[3,9]`
- [ ] `hamilton([]float64{1.0}, 12)` → `[12]` (single-item edge case)
- [ ] `hamilton([]float64{}, 12)` → `[]int{}` (empty input, no panic)
- [ ] `hamilton([]float64{0,0,0}, 12)` returns equal-split (`[4,4,4]`); no NaN, no division by zero
- [ ] `hamilton([]float64{1,1}, 8)` → `[4,4]` (custom grid size honoured)
- [ ] `flexRow` on `<div style="display:flex"><div>a</div><div>b</div></div>` → 1 Row with 2 Cols
- [ ] `flexRow` on the SAME HTML with embedded newlines `"<div style='display:flex'>\n  <div>a</div>\n  <div>b</div>\n</div>"` → 1 Row with 2 Cols (whitespace filtered)
- [ ] `flexRow` on `<div style="display:flex"><div style="flex:1">a</div><div style="flex:2">b</div></div>` → Cols `[4, 8]`
- [ ] `flexRow` on `<div style="display:flex"><div style="flex:0 0 25%">a</div><div style="flex:1">b</div></div>` → Cols `[3, 9]` (%-basis weight resolves correctly)
- [ ] em font-size on a flex item child resolves against `containerStyle.FontSize` (not 0)
- [ ] `flex.go` < 200 lines

**Verify:**

- `go test ./pkg/html/translate/... -run "TestHamilton|TestFlex" -race -count=1 -v`

### Task 4: Thread MaxGridSize through translator + flex detection in blockRows

**Objective:** Make `blockRows` dispatch to `flexRow` when an element has `Display=="flex"`, and thread the maroto config's `MaxGridSize` from `AddHTML` through `html.FromString` into the translator. Also unify class-based `display:none` honouring.

**Dependencies:** Task 3

**Commit:** `feat(html): translate display:flex to Maroto rows (config-aware grid)`

**Files:**

- Modify: `pkg/html/translate/translate.go`
- Modify: `pkg/html/html.go`
- Modify: `maroto.go`
- Modify: `pkg/html/translate/translate_test.go`

**Key Decisions / Notes:**

- `Translate` signature: change to `Translate(doc *dom.Document, opts ...Option) ([]core.Row, error)` with package-local `Option func(*translator)`. Add `WithGridSize(n int) Option`. Default `gridSize = 12`.
- `pkg/html/html.go`: add `WithGridSize(n int) Option` that propagates through to `translate.WithGridSize`. The existing `WithUnsupportedHandler` stays.
- `maroto.go AddHTML`: read `m.config.MaxGridSize` and pass `html.WithGridSize(m.config.MaxGridSize)` when calling `html.FromString`.
- In `blockRows` `default` branch:
  1. Compute style via `computeNodeStyle(tr.sheet, n, nil)`.
  2. If `style.Display == "none"` → return `nil` (unifies inline + class-based display:none).
  3. If `style.Display == "flex"` → return `[]core.Row{tr.flexRow(n, style)}`.
  4. Else → existing flatten-children recursion.
- Empty flex container (`<div style='display:flex'></div>`): return `nil` rows (no empty row emitted).
- Element-named handlers (`p`, `h1`, `table`, etc.) keep their existing behaviour — flex declared on those is ignored. Documented as a limitation.

**Definition of Done:**

- [ ] `<div style="display:flex"><div>a</div><div>b</div></div>` → 1 row (not 2)
- [ ] `<div style="display:inline-flex"><div>a</div><div>b</div></div>` → 1 row with 2 cols (full-path inline-flex test)
- [ ] `<div><div>a</div><div>b</div></div>` (no flex) → 2 rows (existing behaviour preserved)
- [ ] `<div style="display:flex"></div>` → 0 rows
- [ ] Class-based `display:none` via `<style>` block → 0 rows for the hidden element (new behaviour)
- [ ] Inline `style="display:none"` → 0 rows (existing behaviour preserved)
- [ ] With `WithGridSize(8)` and three equal flex items → sum of col sizes == 8 (not 12)
- [ ] `m.config.MaxGridSize=20` and `m.AddHTML(<flex with 3 equal items>)` → renders flex items spanning full row width
- [ ] All existing translator tests still pass

**Verify:**

- `go test ./pkg/html/... -race -count=1`
- `go test ./... -race -count=1`

### Task 5: Justify-content and gap

**Objective:** Implement `justify-content` via offset/spacer cols and `gap`/`column-gap` via integer spacer cols between items. All allocations must sum to ≤ grid size.

**Dependencies:** Task 4

**Files:**

- Modify: `pkg/html/translate/flex.go`
- Modify: `pkg/html/translate/flex_test.go`

**Key Decisions / Notes:**

- **Allocation order:**
  1. Compute requested gap cols (Task 5 below).
  2. Subtract `gap_total = gap_cols * (N-1)` from `gridSize` before Hamilton.
  3. Hamilton over item weights with the reduced total → item shares.
  4. Interleave gap spacers between items.
  5. Apply `justify-content` offsets to remaining slack (if any).
- `justify-content` strategy (let `slack = gridSize - sum(items) - sum(gaps)`):
  - `flex-start` (default): pack left; trailing whitespace implicit.
  - `flex-end`: prepend one offset col of size `slack`.
  - `center`: prepend `floor(slack/2)` and append `ceil(slack/2)` (well-defined for any odd slack).
  - `space-between`: distribute `slack` cells as `N-1` between-spacers via Hamilton over equal weights. If `slack == 0`, falls back silently to `flex-start` — documented limitation.
  - `space-around`: distribute `slack` cells as `N+1` spacers (lead, between, trail) via Hamilton.
- **`gap`/`column-gap`/`row-gap` mapping (gridSize-aware approximation):**
  - We do NOT have content-width access by default. Approximation: `gapCols = round(gap_mm / mm_per_col)` where `mm_per_col = contentWidthMM / gridSize`. Default `contentWidthMM = 170` (A4 with 20mm L+R margins) → 1 col ≈ 14.17mm at gridSize=12.
  - Expose `html.WithContentWidth(mm float64)` and `translate.WithContentWidth(mm float64)` for callers that customise page size; `maroto.AddHTML` computes content width from `cfg.Margins` and `cfg.PageSize` if available, else uses 170mm default.
  - Clamp `gapCols` to `[0, gridSize/2]` so a single absurd gap value can't consume the whole row.

**Definition of Done:**

- [ ] `justify-content:flex-end` with two items `flex:0 0 4` each on gridSize=12 → cols `[4, 4, 4]` (leading offset 4, items 4,4)
- [ ] `justify-content:center` with two items `flex:0 0 4` each on gridSize=12 → cols `[2, 4, 4, 2]` (slack=4 split 2+2)
- [ ] `justify-content:center` with two items `flex:0 0 4` each + odd slack: gridSize=11 case → `[1, 4, 4, 2]` (slack=3 split floor/ceil)
- [ ] `justify-content:space-between` with three items `flex:1` on gridSize=12 → falls back to `flex-start` because items sum to 12, slack=0. Documented; demo uses non-equal weights to avoid this.
- [ ] `justify-content:space-between` with two items `flex:0 0 4` each on gridSize=12 → cols `[4, 4, 4]` (slack=4 distributed as N-1=1 between-spacer of size 4)
- [ ] `gap:10mm` on gridSize=12 with default content width → between-spacers of 1 col (10/14.17 ≈ 0.71 → 1)
- [ ] `column-gap` and `gap` are equivalent on flex-row
- [ ] `gapCols` clamped to ≤ gridSize/2 even for `gap:1000mm`

**Verify:**

- `go test ./pkg/html/translate/... -run TestFlex -race -count=1 -v`

### Task 6: Align-items, flex-direction:column, non-leaf flex items

**Objective:** Honour `align-items` via best-effort props.Cell alignment (or document as limitation if not expressible), handle `flex-direction:column` by stacking children as rows with optional row-gap spacer rows, and implement nested block content inside flex items so the demo's invoice columns can contain headings and paragraphs.

**Dependencies:** Task 5

**Commit:** `feat(html): align-items, flex-direction:column, nested flex items`

**Files:**

- Modify: `pkg/html/translate/flex.go`
- Modify: `pkg/html/translate/translate.go`
- Modify: `pkg/html/translate/flex_test.go`

**Key Decisions / Notes:**

- `flex-direction:column` / `column-reverse`: emit each child as a separate block-flow row via the existing `blockRows` recursion. `row-gap` becomes empty spacer rows of fixed height (e.g. 2mm × gap-multiplier) between children. `column-reverse` is accepted but does NOT reverse order — limitation documented.
- `flex-direction:row-reverse`: accepted but renders same as `row` — limitation documented.
- **Non-leaf flex items (replacement for Task 3 stub):**
  - Introduce a new component-adapter `flexCellContent` that wraps a sequence of child rows. Render path: each child row is rendered sequentially with its own height, accumulated into the parent col's bounds.
  - Concretely: a flex item col contains a single max-height col holding a list of stacked components. For a child like `<div><h2>Bill to</h2><p>Acme...</p></div>`, we call `tr.blockRows(grandchild)` for each grandchild and embed the resulting Rows' components into the col. Use Maroto's existing component pattern (a struct implementing `core.Component` with `GetHeight` summing children, `Render` calling each).
  - Verify by snapshot/structure: a flex container `<div style="display:flex"><div><h2>X</h2><p>Y</p></div></div>` produces 1 Row with 1 Col whose `GetStructure` contains 2 child components (not flattened text).
- `align-items` on `flex-direction:row`:
  - Verify `pkg/props/cell.go` for vertical-alignment field. If absent, accept the property but document as no-op. Do NOT add vertical-alignment to props.Cell in this task — out of scope.
  - `stretch` (default) and `flex-start`: no change.
  - `center`, `flex-end`: best-effort if supported by props.Cell; otherwise no-op + docs note.

**Definition of Done:**

- [ ] `<div style="display:flex;flex-direction:column"><div>a</div><div>b</div><div>c</div></div>` → 3 rows (not 1)
- [ ] `<div style="display:flex;flex-direction:column;row-gap:5mm">3-children</div>` → 5 rows (3 content + 2 gap rows)
- [ ] `flex-direction:row-reverse` produces same col order as `row` (no reversal); limitation noted in docs
- [ ] `flex-direction:column-reverse` produces same row order as `column`; limitation noted in docs
- [ ] `<div style="display:flex"><div><h2>X</h2><p>Y</p></div><div>Z</div></div>` produces 1 Row with 2 Cols; the first Col's structure contains 2 child components (h2 + p), NOT a flattened TextContent string
- [ ] `align-items` is accepted without error; effect documented (best-effort or no-op)

**Verify:**

- `go test ./pkg/html/translate/... -race -count=1 -v`
- `go test ./pkg/html/... -race -count=1`

### Task 7: Demo showcase + structural integration test + docs

**Objective:** Extend `cmd/html-demo` to showcase flex (Bill to / Ship to / Payment 3-col header, totals strip with non-equal weights so `space-between` produces visible slack, optional column-direction stat block). Add a structural integration test that catches "flex didn't dispatch" regressions automatically. Update `docs/v2/html-support.md`: remove flexbox from Out-of-Scope, add the new `## CSS Flex` section enumerating supported properties and documented limitations (no shrink, no row-reverse reordering, no wrap, no order, no align-self, no align-content, gap quantization, space-between collapse when items fill grid).

**Dependencies:** Task 6

**Commit:** `feat(demo): showcase CSS flex in html-demo + docs`

**Files:**

- Modify: `cmd/html-demo/main.go`
- Modify: `docs/v2/html-support.md`
- Modify: `pkg/html/translate/translate_test.go` (add integration test)

**Key Decisions / Notes:**

- Add `<style>` rules:
  - `.cols { display:flex; gap:6mm }` — 3-col layout with visible gap
  - `.totals { display:flex; justify-content:space-between }` — use NON-equal flex weights so slack is non-zero (e.g. left side `flex:2`, right side `flex:1`); otherwise the space-between collapse limitation makes the demo misleading.
- Add HTML body sections that USE the rules (Bill to / Ship to / Payment block; totals row).
- Optional: a flex-column stat block in a sidebar. If real estate is tight, omit and document as future.
- **Structural integration test (new):**
  - In `pkg/html/translate/translate_test.go`: parse a simplified version of the new flex sections; assert `Translate` returns the expected number of rows AND the flex row contains exactly the expected number of cols (e.g., 3 cols for the 3-up section).
  - Use `row.GetColumns()` (returns `[]core.Col`) to count cols on the produced row.
  - This catches "flex didn't dispatch" regressions automatically without requiring PDF visual inspection.
- **Docs update (`docs/v2/html-support.md`):**
  - Remove flexbox from the Out-of-Scope list at line ~76.
  - Add `## CSS Flex` section listing every supported property and limitation.
  - Explicitly document: grid quantization, gap approximation, space-between/center collapse with full grid, flex-shrink no-op, no row-reverse reordering, no wrap, no order, no align-self, no align-content.

**Definition of Done:**

- [ ] `go run ./cmd/html-demo/` produces PDF without errors
- [ ] PDF visibly shows the new flex sections as horizontal layout (not stacked)
- [ ] Structural integration test in translate_test.go passes; asserts the flex section produces 1 row with 3 cols
- [ ] `docs/v2/html-support.md` no longer lists flexbox in Out-of-Scope
- [ ] `docs/v2/html-support.md` lists every supported flex property and limitation (shrink no-op, row-reverse no-reorder, no wrap, no order, no align-self, no align-content, gap quantization, space-between collapse)
- [ ] `cmd/html-demo/main.go` stays under 300 lines
- [ ] Full test suite passes: `go test ./... -race -count=1`
- [ ] `golangci-lint run ./...` clean
- [ ] `gofmt -l .` clean

**Verify:**

- `go test ./pkg/html/translate/... -race -count=1 -v -run TestFlex`
- `go run ./cmd/html-demo/` exits 0
- `go test ./... -race -count=1`
- `golangci-lint run ./...`

## Testing Strategy

- **Unit tests:** Every new CSS property; `flex` shorthand canonical forms; `ParsePercentage`; Hamilton edge cases (empty, single, all-zero, custom grid, non-divisible).
- **Integration tests:** Full `Translate` calls on flex HTML snippets asserting row+col structure; structural assertions for `cmd/html-demo` flex sections.
- **Manual verification:** Generate `cmd/html-demo` PDF; visually verify horizontal layout in flex sections.

## Risks and Mitigations

| Risk                                                                                              | Likelihood | Impact | Mitigation                                                                                                                                                                                       |
| ------------------------------------------------------------------------------------------------- | ---------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Grid quantization causes unexpected visual results for non-integer ideal widths                   | High       | Med    | Document explicitly in `docs/v2/html-support.md`; Hamilton's LRM is provably fairest integer allocation; quantization test matrix.                                                               |
| `gap` mm-to-col approximation wrong for custom page sizes                                         | Med        | Med    | Thread content width via `WithContentWidth`; `maroto.AddHTML` computes from cfg margins+page size; clamp `gapCols ≤ gridSize/2`; document approximation.                                         |
| `MaxGridSize` not honoured by quantizer                                                           | High       | High   | Thread `gridSize` via `WithGridSize` option from `maroto.AddHTML` through `html.FromString` to translator; explicit DoD test with `MaxGridSize=8`.                                              |
| Whitespace text nodes inflate flex item count                                                     | High       | High   | Explicit `isWhitespaceTextNode` helper; explicit DoD test with embedded newlines.                                                                                                                |
| `space-between` collapses silently to `flex-start` when items sum to grid                         | Med        | Med    | Document in `docs/v2/html-support.md`; demo uses non-equal flex weights to avoid the collapse so the showcase is correct.                                                                        |
| em inheritance broken on flex item children                                                       | Med        | Med    | Pass `containerStyle` as parent in `computeNodeStyle` for flex items; explicit DoD test with `font-size:1.5em` child.                                                                            |
| Non-leaf flex items lose rich formatting                                                          | High       | High   | Task 6 implements proper nested rendering via `flexCellContent` component-adapter; structural DoD verifies the col's structure contains child components, not flattened text.                    |
| `display:none` class-based not honoured                                                           | Med        | Low    | Unify check in `blockRows` default branch (Task 4) — `style.Display == "none"` returns nil for any class-based application. Inline behaviour preserved.                                          |
| `flex-shrink` field stored but unused — appears as dead code to reviewers                          | High       | Low    | Documented explicitly as v1 limitation in `docs/v2/html-support.md`; rationale: Hamilton sums exactly to grid total so overflow can't occur. Field kept for forward compatibility.               |
| `align-items: center/flex-end` not expressible in Maroto row model                                | Med        | Low    | Accept without error; document as no-op limitation. Best-effort if `props.Cell` exposes vertical alignment; defer enhancement.                                                                   |
| `flex-direction: *-reverse` not reordering — silent surprise                                      | Low        | Low    | Documented as accepted-but-no-reordering limitation; explicit DoD test that order matches `row`/`column`.                                                                                        |
| Demo verification visually inconclusive — automated checks pass even if flex didn't dispatch     | Med        | High   | Add structural integration test in Task 7 asserting the flex section produces 1 row with N cols; catches dispatch failures automatically.                                                        |

## Goal Verification

### Truths (what must be TRUE for the goal to be achieved)

- A `<div style="display:flex">` with 3 children renders as a single row with 3 cols (not 3 stacked rows)
- `flex:1` and `flex:2` siblings get proportional col widths (e.g. `[4,8]` at gridSize=12)
- `flex-basis:25%` resolves to the correct fractional weight (no silent zero from `%` parsing)
- `justify-content:space-between` places visible horizontal slack between non-equal items
- `gap:Xmm` adds visible horizontal space between flex items
- `flex-direction:column` falls back to stacked block rendering
- `cmd/html-demo` PDF renders the showcase section as a horizontal layout
- `WithMaxGridSize(N)` causes flex cols to sum to `N`, not 12
- Class-based `display:none` suppresses the element (new behaviour)
- Whitespace between flex children does not inflate the item count
- Non-leaf flex items (containing headings/paragraphs) preserve their formatting
- Existing non-flex HTML (paragraphs, tables, lists) renders identically to before this change

### Artifacts (what must EXIST to support those truths)

- `pkg/html/css/computed.go` — flex fields + Apply cases (substantive)
- `pkg/html/css/length.go` — `ParsePercentage` helper (substantive)
- `pkg/html/css/shorthand.go` — `expandFlex` covering every shorthand form
- `pkg/html/translate/flex.go` — `flexRow`, `hamilton`, whitespace filter, non-leaf adapter
- `pkg/html/translate/translate.go` — `blockRows` dispatches to `flexRow`; unified `display:none` handling; `gridSize` field; `WithGridSize`/`WithContentWidth` Options
- `pkg/html/html.go` — `WithGridSize`, `WithContentWidth` options threaded into Translate
- `maroto.go` — `AddHTML` passes `cfg.MaxGridSize` (and content width if available) into `html.FromString`
- `pkg/html/css/css_test.go` — tests for every new property + shorthand + percentage
- `pkg/html/translate/flex_test.go` — Hamilton edge cases + flex translator tests
- `pkg/html/translate/translate_test.go` — structural integration test for demo flex sections
- `cmd/html-demo/main.go` — visible flex sections in body HTML, non-equal weights for space-between
- `docs/v2/html-support.md` — flexbox removed from Out-of-Scope; new `## CSS Flex` section with all properties + limitations

### Key Links (critical connections that must be WIRED)

- `maroto.AddHTML` → `html.FromString(..., html.WithGridSize(cfg.MaxGridSize))` → `translate.Translate(doc, translate.WithGridSize(n))` → `tr.gridSize` propagates into `flexRow`/`hamilton`
- `parseStylesheet` (`stylesheet.go:42`) calls `css.ExpandShorthands` → flex longhands flow into compiled rules
- `blockRows` default branch → `computeNodeStyle` → branches: `Display=="none"` returns nil; `Display=="flex"` returns `flexRow`; else flatten
- `flexRow` filters whitespace → computes per-item style with `containerStyle` as parent → Hamilton allocates → `col.New(size)` per item + justify/gap spacers → `row.New()`
- `cmd/html-demo` body `<div class="cols">` / `<div class="totals">` → translator emits flex rows → PDF shows side-by-side layout
- Integration test parses demo-shaped HTML → asserts row count + col count per row

## Open Questions

- None blocking. All design choices documented in task notes and risk table.

### Deferred Ideas

- `flex-wrap` (multi-line) — wrapping logic over multiple Maroto rows
- `order` per-item — DOM reorder before allocation
- `align-self` per-item — propagate alignment through col render path
- True `flex-shrink` semantics — would require overflow detection and proportional reduction
- `flex-direction: *-reverse` reordering
- `display:grid` — separate effort
- `props.Cell` vertical alignment field (enables proper `align-items: center/flex-end`)
