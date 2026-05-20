# HTML Visual Enhancements Implementation Plan

Created: 2026-05-19
Status: COMPLETE
Approved: Yes
Iterations: 0
Worktree: No
Type: Feature

> **Status Lifecycle:** PENDING → COMPLETE → VERIFIED
>
> - PENDING: Initial state, awaiting implementation
> - COMPLETE: All tasks implemented
> - VERIFIED: All checks passed
>
> **Approval Gate:** Implementation CANNOT proceed until `Approved: Yes`

## Summary

**Goal:** Lift six documented v1 limitations in maroto's HTML→PDF pipeline:
1. Container backgrounds/borders that **span** children (currently rendered per-child)
2. Row-level `<tr style="background-color:…">` backgrounds in tables
3. Block-level `<img>` (PNG/JPG) and SVG icons via a pure-Go rasterizer
4. CSS `border-radius` (uniform + per-corner) on containers and tables
5. Decorative trailing line/band after section titles (fix `border-bottom` on headings + ship a `.title-band` convenience class)
6. Numbered-circle list markers (`<ol class="circle-numbers">`)

**Architecture:** Each feature lands in the same layered stack the v1 flex work used: CSS parser → ComputedStyle → translate → props.Cell → cellwriter chain. Container backgrounds use a new wrapping component (mirroring `flexCellContent` in `pkg/html/translate/flex_cell.go`); `<tr>` styles propagate to `table.Cell.Style`; border-radius adds new fields to `props.Cell` and a new cellwriter node; SVG rasterisation uses `github.com/srwiley/oksvg` + `github.com/srwiley/rasterx` (both pure-Go); circle markers add a new `htmllist.StyleType` rendered via filled circle + centred text.

**Tech Stack:** Go 1.26, gofpdf (`github.com/phpdave11/gofpdf`), oksvg + rasterx (new — pure-Go SVG rasterisers).

## Scope

### In Scope

- New CSS properties: `border-radius`, `border-{top-left,top-right,bottom-left,bottom-right}-radius`
- Container component wrapping children with background + border + radius (used by translator for `<div>`, `<section>`, etc.)
- `<tr>` inline-style → propagate `background-color` and `color` to each `<td>`/`<th>`'s `table.Cell.Style`
- Block-level `<img src="…" width="…" height="…" alt="…">` rendering via existing image component path
- SVG rasterisation: detect `.svg` extension or `image/svg+xml` data URI, rasterise via oksvg+rasterx to PNG bytes at requested dimensions
- `WithImageResolver(func(src string) ([]byte, string, error))` option so callers can plug in custom resolution (default: local filesystem read)
- Fix `perSideBorderStyler` so it uses real cell coordinates (`Fpdf.GetXY()`) instead of margin-relative — required for `border-bottom` on `<h2>` to render correctly
- New `htmllist.DecimalCircle` marker style: filled circle with centred number
- Translator detects `<ol class="circle-numbers">` and switches to `DecimalCircle`
- Built-in `.title-band` class style (white text on dark band) shipped via a Maroto-built-in stylesheet, applied when user writes `<h2 class="title-band">`
- Demo update: `cmd/html-demo/main.go` showcases all six features
- Docs update: `docs/v2/html-support.md` documents the new properties/classes and removes the corresponding v1 limitation entries

### Out of Scope

- Pseudo-elements `::before`/`::after` (still not supported; `.title-band` covers the demo use case)
- Remote `<img src="http://…">` — the default resolver only loads local files. Callers pass `WithImageResolver` for HTTP/data: URIs.
- `<img>` inside paragraphs flowing text around the image (still v2 — block-level only)
- CSS gradients, `box-shadow`, `outline`, `clip-path`
- SVG features beyond oksvg's support (filters, animations, foreignObject)
- Asymmetric border-radius `Xpx / Ypx` (elliptical corners)
- `border-radius` on per-individual-side `<td>` cells (table cells inherit the table's radius via clipping — not per-cell)
- **Container backgrounds spanning page breaks** — `blockContainer` renders atomically per `core.Row`, which the paginator treats as the unit of pagination. A container whose content exceeds the remaining page space will be pushed to the next page; a container taller than a full page logs a warning via `unsupportedHandler` and renders clipped. Splitting backgrounds across pages is deferred to v2.
- Rounded outer corners on `<table>` elements (would need `ClipRoundedRect` around the table's column rendering; deferred to v2). The demo will NOT claim rounded table corners.

## Prerequisites

- Existing CSS-flex feature is merged (it is — last commit `fbcab32`)
- No active uncommitted work in `pkg/html/**`, `pkg/components/**`, `pkg/props/**`, or `internal/providers/gofpdf/**` (will verify before starting)

## Context for Implementer

- **Patterns to follow:**
  - Wrapping rows into a single rendered cell: see `flexCellContent` in `pkg/html/translate/flex_cell.go:11-58`. Use the same `core.Component` shape (SetConfig/GetStructure/GetHeight/Render) for the new container wrapper.
  - CSS parsing extension: add to `pkg/html/css/computed.go:97 Apply()` switch and add shorthand expansion in `pkg/html/css/shorthand.go:20 expandOne()`.
  - cellwriter chain: see `internal/providers/gofpdf/cellwriter/persideborder.go` for the per-side border styler — the chain node template is `stylerTemplate`. New styler nodes are wired in `internal/providers/gofpdf/cellwriter/builder.go`.
  - htmllist marker: see `pkg/components/htmllist/marker.go:9 FormatMarker()` and `pkg/components/htmllist/htmllist.go:108 Render()` — the marker is drawn via `provider.AddText`. For `DecimalCircle` we need a custom render path that draws a filled circle + centred number; do this by extending the `HTMLList.Render` method with a marker-style branch.
- **Conventions:**
  - Tests use `testify` (`assert`/`require`) — see `pkg/html/translate/translate_test.go` for patterns.
  - Snapshot-style tests use `pkg/test`; comparing rendered structures via `core.Structure`. Avoid pixel-diffing PDFs in unit tests.
  - Production files stay under 300 lines (500 hard limit). When a file approaches the limit, split (e.g., `flex.go` + `flex_cell.go` + `flex_layout.go`).
- **Key files:**
  - `pkg/html/translate/translate.go` — main dispatcher; `blockRows` switch handles container default case
  - `pkg/html/translate/table.go` — `buildCell` builds `table.Cell` but does NOT set `Style`; that's the gap for `<tr>` backgrounds
  - `pkg/html/translate/list.go` — list-to-htmllist mapping
  - `pkg/html/css/computed.go` — `ComputedStyle.Apply` switch on property name; missing `border-radius` cases
  - `pkg/props/cell.go` — `Cell` struct; needs new `BorderRadius` fields
  - `pkg/components/htmllist/htmllist.go` — `Render` method, marker drawing; needs branch for `DecimalCircle`
  - `internal/providers/gofpdf/cellwriter/persideborder.go` — known limitation: uses margin-only positioning (see TODO at line 56). Fix using `Fpdf.GetXY()` which **already exists** in the interface (`gofpdfwrapper/fpdf.go:74`).
  - `internal/providers/gofpdf/cellwriter/builder.go` — chain assembly; new styler nodes go here
  - `cmd/html-demo/main.go` — visual demo
  - `docs/v2/html-support.md` — feature documentation
- **Gotchas:**
  - `perSideBorderStyler.Apply` runs BEFORE `cellWriter.Apply` draws the `CellFormat` background — that means rounded-corner fills must happen in a chain node positioned **before** `cellWriter` to override the rectangular fill it draws via `CellFormat(..., fill=true, ...)`. Solution: when `BorderRadius` is set, the radius styler draws fill+stroke via paths and clears `BackgroundColor` and `BorderType` on the prop before passing through.
  - `Fpdf.GetXY()` returns the **pen position**, which `CellFormat` uses as the cell's top-left. Snapshot it at the **start** of the styler chain — by the time `cellWriter.Apply` runs the pen has not moved yet.
  - `image.NewFromBytes` is in `pkg/components/image/bytesimage.go` — it expects `extension.Type` (`png`, `jpg`, `jpeg`, `gif`). We must rasterise SVG to PNG before delegating to it.
  - `oksvg.ReadIconStream(r)` parses SVG; `Icon.Draw(rasterx.Dasher, ...)` rasterises into a `*image.RGBA`. PNG-encode via `image/png.Encode`.
  - When measuring rasterised SVG size: requested width/height come from `<img width="…" height="…">` in mm. Convert to pixels at a sensible DPI (e.g., 150 DPI for crisp PDF embedding: `px = mm / 25.4 * 150`).
  - **`<tr>` does not have a `style="…"` attribute parsed today**: `pkg/html/dom/dom.go` exposes `InlineStyle()` per node, but `buildRow` in `table.go` ignores it. The fix is to read `<tr>`'s inline style (and class) and propagate to each child cell as a fallback when the cell has no own background.
- **Domain context:**
  - "Container backgrounds don't span children" means: today `<div style="bg:red"><p>A</p><p>B</p></div>` emits two separate rows, each with its own background. We want a single row containing a single column that renders a stacked-rows container with one background painted behind all of them.
  - "Title band" in the demo = `<h2>SUMMARY</h2>` rendered with a light grey background, dark left border, and padding around it.

## Runtime Environment

- **Build/run command:** `go run ./cmd/html-demo` (writes `test/output/html-demo.pdf`)
- **Test:** `go test ./...` (full suite); `go test ./pkg/html/...` (HTML pipeline only)
- **Verification of demo:** open `test/output/html-demo.pdf` after each major task to confirm the visual change

## Progress Tracking

- [x] Task 1: Fix per-side border positioning (use real cell X/Y)
- [x] Task 2: Propagate `<tr>` background/color to table cells
- [x] Task 3: Container component for `<div>` backgrounds/borders spanning children
- [x] Task 4: `border-radius` CSS parsing + `props.Cell` fields
- [x] Task 5: `border-radius` rendering in cellwriter chain
- [x] Task 6: Numbered-circle list markers (`<ol class="circle-numbers">`)
- [x] Task 7: Block-level `<img>` with PNG/JPG/SVG support (oksvg + rasterx)
- [x] Task 8: `.title-band` built-in class + demo update + docs

**Total Tasks:** 8 | **Completed:** 8 | **Remaining:** 0

## Implementation Tasks

### Task 1: Fix per-side border positioning

**Objective:** Replace the margin-only approximation in `perSideBorderStyler.Apply` with real cell coordinates from `Fpdf.GetXY()` so per-side borders (especially `border-bottom` on headings, used by Task 8) render at the correct position.

**Dependencies:** None

**Commit:** `fix(html): per-side borders use real cell coordinates`

**Files:**
- Modify: `internal/providers/gofpdf/cellwriter/persideborder.go`
- Test: `internal/providers/gofpdf/cellwriter/persideborder_test.go`

**Key Decisions / Notes:**
- `Fpdf.GetXY()` is already in the interface (`gofpdfwrapper/fpdf.go:74`). Call it once at the top of `Apply()` to snapshot the current pen position; that is the cell's top-left.
- Update the existing test's mock to return realistic X/Y values and assert the drawn line endpoints are in cell-relative space (not margin-relative).
- Remove the `TODO(v2)` comment on line 56.
- **Mock-update enumeration (testify-mock fails on unexpected calls):** every existing subtest in `persideborder_test.go` (currently at lines 42-105: "when only top border set", "when all four sides set", and the legacy regression subtest) must:
  - Remove the `EXPECT().GetMargins(...)` stub (the implementation no longer calls it)
  - Add `EXPECT().GetXY().Return(15.0, 30.0)` (or any realistic non-zero coordinates) — exactly once per Apply invocation
  - Update the `Line(x1,y1,x2,y2)` assertion to use `15.0`/`30.0` as the cell origin rather than the margin pair
- **`borderRadiusStyler` interop (cross-task):** `perSideBorderStyler.Apply` must early-return (pass through to next, draw nothing itself) when `prop != nil && prop.HasBorderRadius()`. The `HasBorderRadius()` method is delivered by Task 4 — when Task 4 lands, return here and add the guard plus a regression test.

**Definition of Done:**
- [ ] `perSideBorderStyler.Apply` uses `fpdf.GetXY()` for the cell origin
- [ ] All four `drawSide` calls receive coordinates derived from `GetXY()` + `width`/`height`
- [ ] Updated unit test verifies line endpoints relative to `GetXY()` mock return, not margins
- [ ] Existing subtests in `persideborder_test.go` have their `GetMargins` EXPECTs removed and `GetXY` EXPECTs added (see Mock-update enumeration above)
- [ ] `go test ./internal/providers/gofpdf/cellwriter/...` passes
- [ ] `gofmt`/`go vet`/`golangci-lint run` clean for changed files

**Verify:**
- `go test ./internal/providers/gofpdf/cellwriter/... ./pkg/components/table/... ./pkg/html/... -count=1` — covers existing per-side border consumers (tables, headings)
- Manual: run `go run ./cmd/html-demo`, open `test/output/html-demo.pdf`, confirm (a) `<h2 style="border-bottom: 1pt solid #aaa">Test</h2>` underline sits directly under the heading text (not at the page top), AND (b) the existing invoice table's borders still render at the same positions as before this task (no regression)

---

### Task 2: Propagate `<tr>` background/color to table cells

**Objective:** When `<tr>` has `style="background-color:…"` or `style="color:…"`, propagate it as a fallback to each child `<td>`/`<th>`'s `table.Cell.Style` and inline runs (so the table renders alternating row backgrounds and header bars).

**Dependencies:** None

**Commit:** `fix(html): propagate <tr> background and color to table cells`

**Files:**
- Modify: `pkg/html/translate/table.go` (extend `buildRow` and `buildCell`; convert to receiver methods on `*translator`)
- Create: `pkg/html/translate/style.go` — extract `blockCellStyle`, `applyInlineStyleToRuns`, and `isDisplayNone` from `translate.go` (preempts file-size pressure from Tasks 4 & 7 on `translate.go`, which is already 239 lines)
- Modify: `pkg/html/translate/translate.go` (remove the three extracted helpers; update the call site at line 101 to use `tr.tableRows(n)`)
- Test: `pkg/html/translate/table_test.go` (create if absent)
- Test: `pkg/html/translate/style_test.go` (move/extract any existing helper tests)

**Key Decisions / Notes:**
- Compute the `<tr>` `ComputedStyle` via `computeNodeStyle(tr.sheet, tr, nil)`. Pass it down to `buildCell` as a parent fallback.
- In `buildCell`, if the cell's own computed style has no `BackgroundColor`, fall back to the row's; same for `Color`.
- Build `table.Cell.Style` from `blockCellStyle` (currently in `translate.go:169`) — extract it to a shared helper accessible to `table.go`.
- `<tr>` color inheritance affects inline `runs`: if the row has a `Color`, apply it to runs whose `Color` is unset (mirror `applyInlineStyleToRuns`).
- Container `<table>` also accepts a `style="..."` at the `<table>` level — extend `tableRows` to compute its style and pass it down as the parent for row-level computations.

**Definition of Done:**
- [ ] Convert the four free functions in `pkg/html/translate/table.go` (`tableRows`, `buildTableMatrix`, `collectRows`/`buildRow`, `buildCell`) to methods on `*translator` so they can access `tr.sheet` for `computeNodeStyle` on `<table>` and `<tr>`
- [ ] Update the single call site at `pkg/html/translate/translate.go:101` from `tableRows(n)` to `tr.tableRows(n)`
- [ ] `<tr style="background-color:#1a3e72;color:#fff">` produces `table.Cell.Style.BackgroundColor` = #1a3e72 (on `props.Cell`, no `Color` field exists there) AND each cell's child `RichRun.Color` = #fff (text-colour propagates through the run path, not through `Cell.Style`)
- [ ] Per-cell `style` still wins over row-level fallback
- [ ] Demo PDF shows the dark header row in the invoice with white text
- [ ] `go test ./pkg/html/...` passes

**Verify:**
- `go test ./pkg/html/translate/... -count=1`
- `go run ./cmd/html-demo`, confirm the `<thead>` row in the invoice PDF has dark navy fill with white text, and zebra-stripe `<tbody>` rows render their backgrounds

---

### Task 3: Container component for `<div>` backgrounds/borders spanning children

**Objective:** When a `<div>` (or other generic container) has a background-color, border, or padding, render its children inside a single wrapping `core.Component` that paints one background/border behind all stacked sub-rows — instead of one styled row per child.

**Dependencies:** Task 1 (per-side border positioning fix needed so the container's borders render correctly), Task 2 (independent but commits cleanly together)

**Files:**
- Create: `pkg/html/translate/container.go` (new file — keeps `translate.go` under 300 lines)
- Modify: `pkg/html/translate/translate.go` (default branch in `blockRows` switch)
- Test: `pkg/html/translate/container_test.go`

**Key Decisions / Notes:**
- New component `blockContainer` mirrors `flexCellContent`'s shape: holds `[]core.Row`, implements `SetConfig`, `GetStructure`, `GetHeight`, `Render`.
- The container's `Render` paints background + per-side borders via `provider.CreateCol(width, height, config, style)` BEFORE rendering child rows on top — same pattern as `row.Row.Render` (`pkg/components/row/row.go:88-110`).
- In `translate.blockRows` default branch: when the container's computed style has `BackgroundColor != nil` or any `Border*Width > 0` or any `Padding* > 0`, gather all child rows and wrap them in a single row containing one col containing the `blockContainer`. Otherwise keep existing flat behaviour.
- Padding shrinks the inner cell: subtract `PaddingLeft+PaddingRight` from width and `PaddingTop+PaddingBottom` from total height; children render inside an offset inner cell.
- **Padding-only activation:** `blockCellStyle` at `pkg/html/translate/translate.go:169` currently returns `nil` when only padding is set (it only checks BackgroundColor and Border*Width). The container path must NOT rely on `blockCellStyle` to detect activation — `blockContainer` reads padding directly from the `ComputedStyle` and the activation check in `blockRows` looks at the raw `ComputedStyle` (not at `blockCellStyle`'s nullable return). Do not modify `blockCellStyle` (other callers depend on its current contract).
- **Pagination — known v1 limitation (documented, not fixed):** `row.Row.Render` (`pkg/components/row/row.go:88-110`) treats a row as atomic at the page boundary; the paginator can move a row to the next page but cannot split it. Wrapping N child rows into one outer row containing a `blockContainer` means content longer than the remaining page space cannot break — either the whole container is pushed to the next page (best case) or rendered clipped (worst case). This is acceptable for the demo's small cards but must be documented as a v1 limitation. **Add `blockContainer.GetHeight()` validation step:** if the computed height exceeds the page's printable height (`config.PageSize.Height - top - bottom margins`), log via `unsupportedHandler` ("html: container too tall to fit on one page; rendering may clip"). Do not panic.

**Definition of Done:**
- [ ] `blockContainer` type implements `core.Component` (SetConfig/GetStructure/GetHeight/Render)
- [ ] `GetHeight` returns the sum of child row heights + top+bottom padding
- [ ] `Render` paints the styled background once, then offsets the inner cell by padding and renders children with `r.Render(provider, innerCell)`
- [ ] `<div style="background-color:#eaf1fb; padding:5mm"><p>A</p><p>B</p></div>` produces a single styled row (verified via `GetStructure` containing one "container" node with two child rows)
- [ ] Demo PDF: the three party-info cards (`.card-blue`, `.card-teal`, `.card-amber`) render with their full background spanning all child content
- [ ] Existing tests in `pkg/html/translate/` still pass (no regression on plain `<div>` without styling)

**Verify:**
- `go test ./pkg/html/translate/... -count=1`
- Visual: `go run ./cmd/html-demo` → confirm `.card-blue/.card-teal/.card-amber` boxes render with continuous backgrounds covering their headings AND paragraphs

**Commit:** `feat(html): div backgrounds and borders span children via blockContainer`

---

### Task 4: `border-radius` CSS parsing + `props.Cell` fields

**Objective:** Add CSS `border-radius` (uniform shorthand) and per-corner longhands to the parser; add five new fields to `props.Cell` (one uniform + four per-corner).

**Dependencies:** None

**Commit:** _(no separate commit — combined with Task 5's `feat(html): CSS border-radius (uniform + per-corner)`)_

**Files:**
- Modify: `pkg/props/cell.go` (add `BorderRadius`, `BorderRadiusTopLeft`, `BorderRadiusTopRight`, `BorderRadiusBottomLeft`, `BorderRadiusBottomRight` fields)
- Modify: `pkg/html/css/computed.go` (add `BorderRadius`, `BorderRadius{TL,TR,BL,BR}` to `ComputedStyle`; handle `border-{top-left,top-right,bottom-left,bottom-right}-radius` cases in `Apply`)
- Modify: `pkg/html/css/shorthand.go` (add `border-radius` to `expandOne`; new helper `expandBorderRadius` parses 1–4 values per CSS spec — 1 = all; 2 = TL+BR, TR+BL; 3 = TL, TR+BL, BR; 4 = TL, TR, BR, BL)
- Modify: `pkg/html/translate/translate.go` (`blockCellStyle` populates new fields)
- Test: `pkg/props/cell_test.go` (extend `HasPerSideBorders`-style helper test, add `EffectiveRadii` helper test)
- Test: `pkg/html/css/css_test.go` (parse cases)

**Key Decisions / Notes:**
- Add helper `(c *Cell) EffectiveRadii() (tl, tr, br, bl float64)` returning per-corner with uniform fallback. Centralises the precedence rule (per-corner > uniform > 0).
- Add `(c *Cell) HasBorderRadius() bool` — needed by Task 5's chain node to decide whether to activate.
- The shorthand follows CSS spec: 4-value form is TL, TR, BR, BL (clockwise from top-left).

**Definition of Done:**
- [ ] `props.Cell` has 5 new exported radius fields with godoc comments
- [ ] `props.Cell.HasBorderRadius()` returns true when any radius > 0
- [ ] `props.Cell.EffectiveRadii()` returns the right precedence
- [ ] CSS parser accepts `border-radius: 4mm`, `border-radius: 4mm 8mm`, `border-radius: 4mm 8mm 2mm`, `border-radius: 1mm 2mm 3mm 4mm`
- [ ] CSS parser accepts `border-top-left-radius: 3mm` (etc. for all four corners)
- [ ] `blockCellStyle` populates these from `ComputedStyle`
- [ ] Unit tests cover all four shorthand arities and per-corner longhands
- [ ] `go test ./pkg/props/... ./pkg/html/css/... ./pkg/html/translate/... -count=1` passes

**Verify:**
- `go test ./pkg/props/... ./pkg/html/css/... ./pkg/html/translate/... -count=1`

---

### Task 5: `border-radius` rendering in cellwriter chain

**Objective:** Add a new `cellwriter` chain node that, when `prop.HasBorderRadius()` is true, draws the cell's filled background and stroked border as a rounded-rect path (using gofpdf's `MoveTo`, `LineTo`, `CurveTo`, `DrawPath`), and clears `BackgroundColor` + `BorderType` on the downstream prop so `CellFormat` doesn't redraw a square fill/border on top.

**Dependencies:** Task 1 (per-side border positioning — uses same `GetXY` approach), Task 4 (props.Cell fields)

**Commit:** `feat(html): CSS border-radius (uniform + per-corner)`

**Files:**
- Create: `internal/providers/gofpdf/cellwriter/borderradius.go`
- Test: `internal/providers/gofpdf/cellwriter/borderradius_test.go`
- Modify: `internal/providers/gofpdf/cellwriter/builder.go` (insert new node in chain BEFORE `cellWriter`)

**Key Decisions / Notes:**
- Use cubic Bezier arc approximation with magic number `k = 0.5522847498` (standard CSS engines use this for circular arcs).
- For each corner with radius `r`, build path segments: a top-left arc from `(0, r)` to `(r, 0)`, top edge to `(w-r, 0)`, top-right arc, right edge, etc. Use `MoveTo`, `LineTo`, `CurveBezierCubicTo`.
- Clamp each radius to `min(w/2, h/2)` to prevent degenerate paths.
- After drawing, clear `prop.BackgroundColor` and `prop.BorderType` and `prop.BorderTopThickness`/etc. on a copied prop, then `GoToNext`. This keeps per-side borders consistent: if `BorderRadius` is set we own the entire border render.
- For mixed radius + per-side-different-widths: stroke uses a single average thickness for v1 (most-common case is uniform border). Document this in the godoc.
- Use `DrawPath("DF")` when both fill+stroke present, `"F"` when fill-only, `"D"` when stroke-only.

**Definition of Done:**
- [ ] `borderRadiusStyler` implements `CellWriter` interface
- [ ] When `prop == nil` or `!prop.HasBorderRadius()`, pass through unchanged
- [ ] When active, draws a rounded path with fill (`SetFillColor` from `BackgroundColor`) and/or stroke (`SetDrawColor`/`SetLineWidth` from `BorderColor`/`BorderTopThickness` or per-side average)
- [ ] Per-corner radii are honored (uses `EffectiveRadii()`)
- [ ] Builder chain (`internal/providers/gofpdf/cellwriter/builder.go`) assembles in this exact order: `perSideBorderStyler → borderRadiusStyler → borderThicknessStyler → borderLineStyler → borderColorStyler → fillColorStyler → cellWriter`
- [ ] `builder_test.go` updated to assert the new chain name list exactly
- [ ] `perSideBorderStyler.Apply` adds a guard at its head: when `prop != nil && prop.HasBorderRadius()` it passes through to `GoToNext` unchanged (no per-side rectangular lines drawn), so the rounded styler owns the whole border
- [ ] `persideborder_test.go` adds a "skips when border-radius set" subtest asserting no `Line` calls when `HasBorderRadius()` is true
- [ ] `borderRadiusStyler.Apply` clears `BackgroundColor`, `BorderType`, and all four `BorderXThickness` fields on a COPIED prop before `GoToNext`, so downstream `fillColorStyler` and `cellWriter` do not redraw a rectangular fill or border
- [ ] Unit test mocks `Fpdf`, asserts `DrawPath`/`MoveTo`/`CurveBezierCubicTo` are called with the expected coordinates
- [ ] Unit test demonstrates mixed asymmetric per-side border widths (e.g., top=2pt, bottom=0.5pt) with radius set: assert `SetLineWidth` called once with the averaged thickness (1.25pt)
- [ ] Visual: `<div style="background-color:#eaf1fb; border-radius:4mm; padding:5mm">…</div>` renders with rounded corners
- [ ] `docs/v2/html-support.md` notes that `border-radius` combined with non-uniform per-side border widths uses an averaged stroke thickness (v1 limitation) — added as part of Task 8's docs update, but the requirement is owned by this task
- [ ] `go test ./internal/providers/gofpdf/cellwriter/... -count=1` passes

**Verify:**
- `go test ./internal/providers/gofpdf/cellwriter/... -count=1`
- `go run ./cmd/html-demo` and inspect `test/output/html-demo.pdf` — `.card-blue/.card-teal/.card-amber` cards now have rounded corners

---

### Task 6: Numbered-circle list markers (`<ol class="circle-numbers">`)

**Objective:** Add a new `htmllist.StyleType` `DecimalCircle` that renders each marker as a filled circle (background-color from a marker prop, defaulting to dark blue) with the index number centred inside in white. Translator detects `class="circle-numbers"` on `<ol>` and switches to this style.

**Dependencies:** None (depends conceptually on Task 5 only insofar as it draws circles — but `gofpdf.Circle` is in the wrapper interface already and is independent of the cellwriter chain)

**Commit:** `feat(html): numbered-circle list markers via .circle-numbers class`

**Files:**
- Modify: `pkg/components/htmllist/htmllist.go` (extend `Prop` with `MarkerBackground *props.Color`, `MarkerTextColor *props.Color`, and branch in `Render` on `Style == DecimalCircle`)
- Modify: `pkg/components/htmllist/marker.go` (add `DecimalCircle` constant; `FormatMarker(DecimalCircle, i)` returns the bare index string `"1"` (no period))
- Modify: `pkg/html/translate/list.go` (detect `class` attribute containing `circle-numbers` on `<ol>`, switch to `DecimalCircle` style with defaulted marker colors)
- Test: `pkg/components/htmllist/htmllist_test.go`
- Test: `pkg/html/translate/list_test.go` (create if absent, or extend `translate_test.go`)

**Key Decisions / Notes:**
- For circle markers, need a `core.Provider` method that draws a filled circle. Existing `provider.AddText` won't suffice. Solution: add a tiny method to `core.Provider` is invasive; instead, use the existing `AddImageFromBytes`-style approach via a custom `core.Component` per marker. Simpler: extend `htmllist` to delegate marker drawing to an injected `func(provider core.Provider, cell *entity.Cell, label string, p Prop)` so the default implementation is text-only, and `DecimalCircle` uses a circle-drawing variant.
- **Pragmatic implementation:** add a new narrow interface `core.ShapeProvider` with `DrawFilledCircle(cell *entity.Cell, prop *props.Rect)` — the method signature mirrors existing provider methods (`AddText`, `AddImageFromBytes`) which take a cell + a typed prop. `props.Rect` carries `Center bool` (already supported) and we will add `FillColor *props.Color` for the fill; if needed introduce a tiny `props.Circle` type instead, decided at implementation time.
- **Precedent for the optional-capability pattern:** `core.RichTextProvider` is already detected via type assertion in `pkg/components/htmllist/htmllist.go:141` — see the `if rtp, ok := provider.(core.RichTextProvider); ok { ... }` block. `ShapeProvider` follows the same shape: declare a narrow interface in `pkg/core/`, gofpdf implements it, htmllist type-asserts using the safe form `if sp, ok := provider.(core.ShapeProvider); ok { sp.DrawFilledCircle(...) } else { /* fallback: draw the index as a text marker, no circle */ }`. NEVER use the panicking form `provider.(core.ShapeProvider)`.
- Add `core.ShapeProvider` to `pkg/core/`. Implement in `internal/providers/gofpdf/provider.go` via `fpdf.SetFillColor(...)` + `fpdf.Circle(x, y, r, "F")`. Document that it's an optional capability — markers fall back to text-only (no circle) when unavailable.
- Circle radius = `min(gutter/2, itemRowHeight/2) * 0.45`. Number drawn via `AddText` with `MarkerTextColor` and centred horizontally/vertically within the marker cell.
- **`GetStructure` visibility:** when `Style == DecimalCircle`, the top-level htmllist `GetStructure()` Details map MUST include `"marker_style": "decimal-circle"` so snapshot tests can verify circle markers were emitted (otherwise the rendering bypasses the structure tree).

**Definition of Done:**
- [ ] `htmllist.DecimalCircle` constant added
- [ ] `htmllist.Prop` has `MarkerBackground` and `MarkerTextColor` (both `*props.Color`)
- [ ] `HTMLList.Render` branches on style: text marker (default) vs circle marker
- [ ] `core.ShapeProvider` interface with `DrawCircle(cell *entity.Cell, fill *props.Color)` declared in `pkg/core/`
- [ ] gofpdf provider implements `ShapeProvider` via `fpdf.Circle` and `fpdf.SetFillColor`
- [ ] Translator detects `class="circle-numbers"` (substring match like `strings.Contains` over space-separated classes) on `<ol>` and applies the style
- [ ] Demo PDF shows circle markers somewhere (e.g., "PAYMENT INSTRUCTIONS" ordered list)
- [ ] Unit test asserts `FormatMarker(DecimalCircle, 0) == "1"`
- [ ] Unit test using a fake provider asserts `DrawCircle` is called when `Style == DecimalCircle`
- [ ] `go test ./pkg/components/htmllist/... ./pkg/html/translate/... -count=1` passes

**Verify:**
- `go test ./pkg/components/htmllist/... ./pkg/html/translate/... -count=1`
- `go run ./cmd/html-demo` → confirm the payment-instructions list renders with circled numerals

---

### Task 7: Block-level `<img>` with PNG/JPG/SVG support (oksvg + rasterx)

**Objective:** When the translator sees a block-level `<img src="…" width="…" height="…" alt="…">`, load the bytes via a resolver, rasterise SVG to PNG if needed, and emit a row containing an `image.BytesImage`. Add `WithImageResolver(fn)` option so callers can override the default local-file resolver.

**Dependencies:** None (but lands after radius/markers so the demo can use both together)

**Commit:** `feat(html): block-level <img> with PNG/JPG/SVG (oksvg+rasterx)`

**Files:**
- Create: `pkg/html/translate/image.go` (block-level image row builder + SVG rasterisation)
- Modify: `pkg/html/translate/translate.go` (handle `case "img":` in `blockRows`)
- Modify: `pkg/html/html.go` (add `WithImageResolver` option; thread through `translate.Translate`)
- Modify: `pkg/html/translate/translate.go` (add `WithImageResolver` translator option mirroring html.WithImageResolver)
- Modify: `pkg/html/translate/inline.go` (when an inline `<img>` is the **only** child, prefer block rendering — actually keep inline behaviour unchanged; block-level dispatch happens in `blockRows`)
- Modify: `go.mod` / `go.sum` (`go get github.com/srwiley/oksvg github.com/srwiley/rasterx`)
- Test: `pkg/html/translate/image_test.go` (use a fixture SVG and a fixture PNG)
- Test fixture: `internal/fixture/icon.svg`, `internal/fixture/icon.png` (small 32×32)

**Key Decisions / Notes:**
- Signature: `WithImageResolver(func(src string) ([]byte, ext string, err error)) Option`. **Safe-by-default resolver:** when no `WithImageResolver` and no `WithImageBaseDir` is configured, the default ONLY accepts `data:` URIs (`data:image/png;base64,…`, `data:image/svg+xml;base64,…`); arbitrary `os.ReadFile(src)` is REFUSED to prevent path traversal on user-controlled HTML. Local filesystem reads require either `WithImageResolver(custom)` or `WithImageBaseDir(dir)`.
- New `WithImageBaseDir(dir string) Option`: opt-in local-file reads scoped to `dir`. The resolver uses `filepath.Clean` + `strings.HasPrefix(filepath.Clean(filepath.Join(dir, src)), filepath.Clean(dir))` to reject `..` traversal and absolute-path escapes; returns an error otherwise.
- Demo: `cmd/html-demo/main.go` passes `WithImageBaseDir("cmd/html-demo/assets")` so the SVG icon loads explicitly from that directory.
- `width="20mm"` / `height="20mm"` → `props.Rect{Percent: 100}` won't work directly because Rect sizes relative to cell. Solution: emit `row.New(heightMM).Add(col.New().Add(image.NewFromBytes(...)))` so the row has the requested height; the image fills it.
- For SVG: parse with `oksvg.ReadIconStream(bytes.NewReader(svgBytes))`, set target size with `icon.SetTarget(0, 0, float64(pxW), float64(pxH))`, draw into a `*image.RGBA` of that size via `rasterx.NewDasher(pxW, pxH, scanner)`, then `png.Encode` to bytes.
- DPI for raster output: 150 DPI (gives crisp results without bloating PDF size).
- Width/height parse: support `px`, `pt`, `mm`, `cm` via existing `css.ParseLength`. Drop `em`/`rem` for `<img>` width/height: the translator has no font-context at image-resolution time and they would silently use a wrong base. If only one of width/height is given, use the SVG's intrinsic aspect ratio (read from `icon.ViewBox.W/H`).
- If `src` is missing or resolver fails, fall back to `alt` text via the existing inline path (don't error out — log via `unsupportedHandler` if registered). Failure modes: (a) resolver returns err; (b) oksvg `ReadIconStream` returns err (rejects SVGs with unsupported features like filters/foreignObject); (c) `png.Encode` returns err. All three return paths log + alt-fallback; no `recover()` is needed because oksvg returns errors, not panics.
- File size budget: `pkg/html/translate/image.go` ≤ 200 lines.

**Definition of Done:**
- [ ] `go.mod` includes `github.com/srwiley/oksvg` and `github.com/srwiley/rasterx`
- [ ] `WithImageResolver` and `WithImageBaseDir` are exported on the `html` package and threaded to the translator
- [ ] `<img src="local.png" width="20mm" height="20mm">` emits a row with a `BytesImage` of width 20mm × height 20mm when a base dir or resolver is configured
- [ ] `<img src="icon.svg" width="15mm" height="15mm">` rasterises the SVG to PNG bytes at 150 DPI and emits a `BytesImage`
- [ ] **Safety:** Unit test verifies the default (zero-config) resolver REFUSES `<img src="/etc/passwd">` and `<img src="../../secret">` and falls back to alt text
- [ ] **Safety:** Unit test verifies `WithImageBaseDir("./assets")` accepts `<img src="icon.svg">` resolving to `./assets/icon.svg` AND refuses `<img src="../escape.png">` and `<img src="/abs/escape.png">`
- [ ] Missing-src or resolver-error path falls back to alt text (no panic)
- [ ] Unit test rasterises a fixture SVG and asserts the resulting PNG decodes to the requested pixel dimensions (±1 px)
- [ ] Unit test verifies the default resolver decodes `data:image/png;base64,…` and `data:image/svg+xml;base64,…` data URIs
- [ ] Unit test verifies a custom `WithImageResolver` is invoked
- [ ] Unit test verifies an SVG with unsupported features (e.g., `<svg><filter>…`) returns an error and falls back to alt text
- [ ] `go test ./pkg/html/... -count=1` passes
- [ ] `go mod tidy` clean

**Verify:**
- `go test ./pkg/html/translate/... -count=1`
- `go run ./cmd/html-demo` → confirm SVG icons render in the new demo content (added in Task 8)

---

### Task 8: `.title-band` built-in class + demo update + docs

**Objective:** Ship a built-in `.title-band` class that the parser injects into the cascade automatically (without user-provided `<style>`), update `cmd/html-demo` to showcase all six new features, and refresh `docs/v2/html-support.md` to add the new properties/classes and remove the corresponding v1 limitation entries.

**Dependencies:** Tasks 1–7 (all features must exist before the demo uses them)

**Commit:** `feat(demo): showcase visual enhancements + update HTML support docs`

**Files:**
- Modify: `pkg/html/translate/stylesheet.go` (prepend a built-in CSS block applied before user rules; defines `.title-band { background-color: #1a3e72; color: #fff; padding: 3mm; border-radius: 2mm }` and `.circle-numbers` may need no styling beyond the marker-style dispatch since Task 6 keys off the class name)
- Modify: `cmd/html-demo/main.go` (add an SVG/PNG logo to the header, a `.title-band` heading, rounded `<div>` cards, a circle-numbers ordered list)
- Add: `cmd/html-demo/assets/icon.svg` (a small demo SVG, e.g., a checkmark)
- Modify: `docs/v2/html-support.md` (remove "Container backgrounds/borders do not span children" limitation; add `border-radius`, block-level `<img>`, `.title-band`, `.circle-numbers`, SVG sections; update Image section to reflect new behaviour)
- Test: extend `pkg/html/translate/translate_test.go` to assert `.title-band`'s background color reaches the computed style without a user `<style>` block

**Key Decisions / Notes:**
- Built-in stylesheet: implement as a constant string prepended to the user's extracted style text in `parseStylesheet` (which is called from `translate.go:51`) via literal concatenation `builtinCSS + "\n" + userCSS`. CSS same-specificity rules win by source order, so a later user-defined `.title-band {...}` overrides the built-in. Inline `style="..."` attributes have the highest precedence (applied after the stylesheet cascade in `computeNodeStyle`).
- **Precedence tests (DoD required):** add three subtests in `translate_test.go`:
  1. `.title-band` from built-in resolves when user has no `<style>` (background-color comes through)
  2. user `<style>.title-band{background:red}</style>` overrides the built-in (background = red, not navy)
  3. inline `style="background:green"` on the element wins over both the built-in and user CSS
- Document the precedence (built-in < user CSS < inline style) in `docs/v2/html-support.md` as part of this task's docs update.
- Demo content additions:
  - Header: `<img src="cmd/html-demo/assets/icon.svg" width="12mm" height="12mm">` next to the title (resolver loads from the working directory — confirm in main `os.Chdir` to the repo root or pass an explicit resolver)
  - `<h2 class="title-band">SUMMARY</h2>` replaces the existing `.section-band > h2` pattern
  - `<ol class="circle-numbers">` for the payment instructions
  - `border-radius: 3mm` on `.card-blue/.card-teal/.card-amber`
- Docs structure: under "Supported CSS properties" → add new "Decoration" subsection with `border-radius`. Under "Supported HTML tags" → add an "Images" subsection promoting block-level `<img>` from limitation to supported. Remove the v1 limitation entry "Container backgrounds/borders do not span children" and replace with a feature note.

**Definition of Done:**
- [ ] `parseStylesheet` prepends a built-in CSS block including `.title-band`
- [ ] Translator test asserts `.title-band` background color is applied without user CSS
- [ ] `cmd/html-demo/main.go` uses `<img>`, `border-radius`, `.title-band`, `.circle-numbers`, and a `<tr>` header background (already existed, but now actually renders)
- [ ] `cmd/html-demo/assets/icon.svg` exists and renders in the output PDF
- [ ] `docs/v2/html-support.md` no longer lists "Container backgrounds/borders do not span children" as a limitation
- [ ] `docs/v2/html-support.md` adds documentation for `border-radius`, block-level `<img>` (PNG/JPG/SVG), `.title-band`, `.circle-numbers`, and the new `WithImageResolver` option
- [ ] `go run ./cmd/html-demo` produces a PDF where the visual layout matches all six feature claims
- [ ] All existing tests pass: `go test ./... -count=1`
- [ ] `gofmt -w .`, `go vet ./...`, `golangci-lint run` clean

**Verify:**
- `go test ./... -count=1`
- `go run ./cmd/html-demo` and visually inspect `test/output/html-demo.pdf`:
  - Header has an SVG icon next to the title
  - `.title-band` headings have a dark navy bar with white text and rounded corners
  - `.card-blue/.card-teal/.card-amber` boxes have continuous backgrounds + rounded corners spanning all children
  - Payment-instructions list renders with circled numerals
  - Invoice table has dark navy header row with white text + alternating row backgrounds (the table itself is NOT rounded — rounded table outer corners are out of scope for v1)

---

## Testing Strategy

- **Unit tests:**
  - CSS parser: each new property/shorthand has positive + edge-case tests
  - cellwriter chain nodes: mock `Fpdf`, assert correct drawing primitives are called
  - htmllist marker: format + render branch
  - translator: each tag/class produces the expected `core.Structure` tree
  - image: SVG rasterisation produces expected pixel dimensions; resolver fallback path
- **Integration tests:**
  - `pkg/html/translate/translate_test.go` end-to-end: HTML string → structure tree assertion
  - The existing `cmd/html-demo` build must succeed and produce a non-empty PDF
- **Manual verification:**
  - Open `test/output/html-demo.pdf` after Task 3, Task 5, Task 6, Task 7, Task 8 to confirm visual correctness
- **No PDF pixel-diffing** — kept out of scope; visual checks are manual

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
| - | - | - | - |
| `oksvg` or `rasterx` parsing fails on edge-case SVG | Medium | Low | Wrap rasterisation in `recover()`; on error fall back to alt text and call `unsupportedHandler` |
| Per-side border position fix breaks tables (which already use per-side borders) | Medium | High | Re-run full test suite after Task 1; visually inspect tables in demo PDF; the existing test in `persideborder_test.go` must continue passing with the same behaviour viewed through real coordinates |
| `border-radius` rendering interacts badly with `perSideBorderStyler` (both want to own border drawing) | Medium | Medium | `borderRadiusStyler` claims ownership and clears `BorderTopThickness`/etc. on the downstream prop when active; per-side styler runs first and checks `HasBorderRadius()` to skip if so |
| `blockContainer` GetHeight is called multiple times by paginator → recomputes child heights | Low | Low | Cache height on first call (same pattern as `Table.computeRowHeights`); invalidate via `SetConfig` |
| `blockContainer` content exceeds remaining page space (atomic row) | Medium | Medium | Documented as v1 limitation in Out of Scope; `blockContainer.GetHeight` logs via `unsupportedHandler` when the computed height exceeds the page's printable height; paginator will push the container to the next page when possible, content taller than a full page renders clipped |
| Default `<img>` resolver path-traversal exposure | Low | High | Default resolver only accepts `data:` URIs. Local-file reads require `WithImageBaseDir` or a custom `WithImageResolver`. Base-dir resolver validates with `filepath.Clean` + prefix check. |
| Demo PDF visual regression on existing features (flex, tables) | Medium | Medium | After every commit milestone, run `go run ./cmd/html-demo` and diff the page count + file size of `test/output/html-demo.pdf` against the pre-task baseline |
| SVG rasterised at 150 DPI bloats PDF size | Low | Low | Document the DPI; allow override via a future `WithImageDPI` option (deferred to v2) |
| `<tr>` style propagation accidentally overrides explicit `<td>` background | Low | Medium | Implementation explicitly checks "cell has no own BackgroundColor" before applying the row fallback |
| New deps (`oksvg`, `rasterx`) introduce license/security issues | Low | Medium | Both are pure-Go, BSD-2-Clause licensed (verify via `go mod why` and inspect their LICENSE); both are mature (oksvg 1k+ stars, used in fyne) |
| File-size budget exceeded (300 lines) | Medium | Low | Split early: `container.go`, `image.go`, `borderradius.go` are separate files from the start |

## Goal Verification

### Truths (what must be TRUE for the goal to be achieved)

- A `<div style="background-color:#eaf1fb; padding:5mm">` containing multiple `<p>` and `<h3>` children renders as a single background rectangle spanning all children (not per-child)
- A `<tr style="background-color:#1a3e72;color:#fff">` in the demo renders with the navy fill across every cell and white text
- A `<div style="border-radius:4mm">` renders with rounded corners
- An `<h2 style="border-bottom:1pt solid #aaa">` renders the underline at the heading's bottom edge, not at the page top margin
- A `<h2 class="title-band">` renders as a navy band with white text and rounded corners without any user-provided `<style>`
- An `<ol class="circle-numbers"><li>…</li></ol>` renders each `<li>` with a filled navy circle containing a centred white number
- An `<img src="icon.svg" width="15mm" height="15mm">` at block level renders the rasterised SVG as a 15mm × 15mm image
- An `<img src="logo.png" width="20mm">` renders the PNG at 20mm width (height auto from intrinsic ratio)

### Artifacts (what must EXIST to support those truths)

- `pkg/html/translate/container.go` — `blockContainer` component with real `Render` painting bg/border and stacking children
- `pkg/html/translate/image.go` — block-level `<img>` builder + SVG rasterisation pipeline
- `pkg/html/translate/table.go` — `<tr>` and `<table>` style propagation logic
- `pkg/props/cell.go` — 5 new radius fields + `EffectiveRadii()`/`HasBorderRadius()` methods
- `internal/providers/gofpdf/cellwriter/borderradius.go` — rounded path rendering node
- `internal/providers/gofpdf/cellwriter/persideborder.go` — fixed to use `GetXY()` instead of margins
- `pkg/components/htmllist/htmllist.go` + `marker.go` — `DecimalCircle` style and circle-marker render branch
- `pkg/core/shape_provider.go` (or similar) — `ShapeProvider` interface with `DrawCircle`
- `internal/providers/gofpdf/provider.go` — implements `ShapeProvider`
- `pkg/html/translate/stylesheet.go` — built-in CSS block including `.title-band`
- `pkg/html/html.go` — `WithImageResolver` option
- `cmd/html-demo/assets/icon.svg` — demo SVG asset
- `cmd/html-demo/main.go` — demo using all six features
- `docs/v2/html-support.md` — updated feature documentation

### Key Links (critical connections that must be WIRED)

- `translate.blockRows` default branch → `blockContainer` (when computed style has bg/border/padding)
- `translate.blockRows` `case "img":` → `image.buildImageRow` → either `BytesImage` (PNG/JPG) or `rasteriseSVG` + `BytesImage` (SVG)
- `translate.buildRow`/`buildCell` → `<tr>` ComputedStyle → `table.Cell.Style.BackgroundColor` fallback
- `html.FromString(opts)` → `WithImageResolver` → `translate.Translate(translatorOpts)` → image builder
- `cellwriter/builder.go` chain → inserts `borderRadiusStyler` BEFORE `cellWriter` so it can short-circuit fill/stroke
- `htmllist.HTMLList.Render` → checks `Style == DecimalCircle` → calls `provider.(ShapeProvider).DrawCircle` + `AddText`
- `translate.stylesheet.parseStylesheet` → prepends built-in CSS so `.title-band` resolves without user CSS
- `perSideBorderStyler.Apply` → `fpdf.GetXY()` → real cell coordinates → correctly positioned `Line` calls

## Open Questions

- Should `.title-band` use a left-border accent (like the existing `.section-band` in the demo)? Default plan keeps the demo's existing `.section-band` as a separate user-defined class and ships `.title-band` as a distinct built-in. User can choose at implementation time which one the demo uses.

### Deferred Ideas

- `WithImageDPI` option for tuning SVG rasterisation DPI (currently hardcoded 150)
- Elliptical border-radius (`Xpx / Ypx`)
- Per-cell `border-radius` on `<td>` (would require table cell rounding via clipping)
- Pseudo-elements `::before`/`::after`
- `box-shadow`
- CSS gradients
