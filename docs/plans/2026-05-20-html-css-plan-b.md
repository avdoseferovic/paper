# HTML + CSS Plan B — Renderer, Effects, Flex, Pagination Implementation Plan

Created: 2026-05-20
Status: PENDING
Approved: Yes
Iterations: 0
Worktree: No
Type: Feature

> **Status Lifecycle:** PENDING → COMPLETE → VERIFIED
> **Approval Gate:** Implementation CANNOT proceed until `Approved: Yes`

## Summary

**Goal:** Land the renderer, effects, flexbox, and pagination work that the prior plan ("extend-html-css-support") deferred. Closes the render-side gaps surfaced during verification (RichRun.Background painting, RichRun.LetterSpacing consumption, per-run anchor link rects, calc()% in Apply, dashed/dotted `<hr>`), adds CSS Level 3 backgrounds & effects (gradients, box/text-shadow, outline), completes flexbox (wrap/order/align-self/*-reverse), and adds page-break controls including container splitting across page breaks. `align-content` is deferred — it requires explicit container height which is a separate substantial change.

**Architecture:** Two layers move in lockstep. The translator continues to be the source of truth for parsed CSS state; the gofpdf provider grows new capability interfaces (`core.GradientProvider`, `core.ShadowProvider`, `core.OutlineProvider`) and the `richtext`/`AddRichText` renderer is extended to consume `RichRun.Background`/`LocalAnchor`/`LetterSpacing`. Flex-wrap emits multi-row outputs from a new layout pass. **Page-break and container-split logic both run in `maroto.addRow()` (the build phase), NOT inside `Render()` — calling `fpdf.AddPage()` from Render would bypass the `core.Page` list and break headers/footers/numbering/concurrent generation.** Page breaks use a `core.PageBreaker` interface (`IsPageBreak() bool`); container splits use `core.Splittable.SplitAt(remainingHeight) (first, rest, didSplit)`. Both are type-asserted by `addRow()` and trigger existing `fillPageToAddNew()` flow.

**Tech Stack:** Go 1.26 + existing deps (`golang.org/x/net/html`, `andybalholm/cascadia`, `aymerick/douceur`, `phpdave11/gofpdf`). Gradients use an in-memory raster (`image/draw` + `png.Encode`) embedded via existing `Fpdf.RegisterImageReader` — no new external dependency. Shadows use offset-rect approximation; box-shadow's blur is approximated by overlaid translucent rects (configurable iterations) rather than a true Gaussian.

## Scope

### In Scope

**Render-side gap closures (from verification):**
- `RichRun.Background` painted by `AddRichText` as a filled rectangle behind each run that has it (covers `<mark>`, `<kbd>`, inline `<code>`, `<span style="background-color:…">`).
- `RichRun.LetterSpacing` consumed by `AddRichText` via `Fpdf.SetCharSpacing` per run (currently populated by the CSS pipeline but ignored by the renderer).
- `RichRun.LocalAnchor` registers a per-run clickable rectangle via `fpdf.Link` (currently only row-level via `anchorSource`).
- `calc()` percentages resolve correctly inside `ComputedStyle.Apply()` for width/height/padding/margin via threaded context width (`Apply` gains an optional context, or `computeNodeStyle` post-resolves length values).
- Dashed/dotted `<hr>` lines via `Fpdf.SetDashPattern` (already in wrapper) — `styledHrRow` picks dash pattern from `border-top-style`.

**CSS Level 3 backgrounds & effects:**
- `background-image: linear-gradient(...)` and `radial-gradient(...)` — rasterised to PNG bytes at the cell's pixel size and embedded as a background image (placement honours page margins, dedup via stable imgName cache to avoid duplicate embedded blobs). Supports up to 5 colour stops, angle keywords (`to right`/`to bottom`/`to top right`/etc.) and numeric degrees (`45deg`), plus shorthand and explicit stop positions (`red 0%, blue 100%`).
- `box-shadow: <x> <y> [<blur>] [<spread>] <color>` — drawn behind the container as an offset filled rect. Blur is approximated by N overlaid translucent rects (default N=3); a single `Inset` keyword switches to inset shadows (drawn inside the cell). Multi-shadow comma syntax supported (up to 4 shadows).
- `text-shadow: <x> <y> [<blur>] <color>` — per-run shadow drawn before the text via offset shifted `AddText` call.
- `outline: <width> <style> <color>` and per-side `outline-{top,right,bottom,left}-…` — drawn outside the cell box (1mm default offset, configurable via `outline-offset`).

**Flexbox completeness:**
- `flex-wrap: wrap | nowrap | wrap-reverse` — the quantizer is extended to emit multiple rows when total demand exceeds the grid. `wrap-reverse` reverses row order.
- `order: <integer>` — items are sorted by `order` before quantization (defaults to 0; lower comes first; tie-break by DOM order).
- `align-self: flex-start | flex-end | center | stretch | auto` — per-item cross-axis alignment overriding container `align-items`.
- `flex-direction: row-reverse | column-reverse` — actual reversal of child order during quantization (currently parsed but ignored).

**Page break controls:**
- `page-break-before: always` and `page-break-after: always` on block elements — emit a `pageBreakRow` that implements `core.PageBreaker` (`IsPageBreak() bool`); `addRow()` type-asserts and calls existing `fillPageToAddNew()`. `avoid` is best-effort or deferred (see Task 11).
- `break-inside: avoid` — flag on the container that `Splittable.SplitAt` honours by returning `(nil, self, true)` (push to next page) unless the container is itself taller than a page.
- **`blockContainer` splitting** — the largest single change: `blockContainer` implements `core.Splittable`. `maroto.addRow()` type-asserts and, on overflow, calls `SplitAt(remainingHeight)` to get `(first, rest)`, places `first` on the current page, then recursively addRow()s `rest` on a fresh page. Each slice carries a flag (first/middle/last) so Render() draws the correct corners and border edges.

**Demo + docs:**
- Demo additions covering every new feature (gradient hero, box-shadow card stack, page-break section, flex-wrap pill cloud, multi-line align-content, ordered flex, dashed hr).
- `docs/v2/html-support.md` rewritten so the "Known limitations" section shrinks dramatically.

### Out of Scope

**Genuine non-goals (not deferred — won't be done):**
- CSS Grid, `position`, `transform`, `float`.
- Conic gradients (`conic-gradient`).
- True Gaussian blur for shadows — we use a constant-time overlay approximation.
- Drop-shadow filter (`filter: drop-shadow(...)`).
- Inline image flow inside paragraphs (text-wrap around inline `<img>`) — still v2/3.
- `@media` rules — PDF is print-only; we apply all rules regardless.
- `clip-path`, `mask`, `mix-blend-mode`.
- `aspect-ratio`, `object-fit`, `object-position`.
- `:hover`/`:focus`/`:active`/`:visited` matching (PDFs are static).
- Custom font shaping (kerning pairs beyond gofpdf's defaults).

## Prerequisites

- Prior plan (`2026-05-20-extend-html-css-support.md`) merged and VERIFIED on `master` (already done).
- Clean working tree; no in-flight changes in `pkg/html/**`, `pkg/components/**`, `pkg/props/**`, `internal/providers/gofpdf/**`, `pkg/core/**`.

## Context for Implementer

- **Patterns to follow:**
  - **Capability interface pattern:** `pkg/core/{alpha,link,late_font,char_spacing}_provider.go` show the precedent — narrow interface, gofpdf provider implements via compile-time assertion, consumers type-assert with the safe form. New providers (`GradientProvider`, `ShadowProvider`, `OutlineProvider`, `PageBreakProvider`) follow the same shape.
  - **Render-time integration:** `pkg/html/translate/anchor.go` shows how to wrap a row at translate time and call the provider at Render time (target.Render does `lp.SetLink(...)`). Same pattern for the new shadow/outline wrappers.
  - **Resolver-with-safety:** N/A for Plan B — no new external loads beyond what's already in place.
  - **Cellwriter chain extension:** see `internal/providers/gofpdf/cellwriter/builder.go` for the order. New nodes (gradient fill, shadow paint, outline stroke) insert in specific positions.
  - **Multi-row flex quantization:** `pkg/html/translate/flex_layout.go` implements Hamilton's largest-remainder for single-row. For wrap, we emit multiple "logical rows" before passing to the quantizer; each logical row is independently allocated.
- **Conventions:**
  - testify + standard `testing`; production files ≤ 300 lines (test files exempt); conventional commits with scope.
  - Snapshot tests via `core.Structure` walks (see `walkRowStructure` in `anchor_test.go`).
  - Stubs for the gofpdf `Fpdf` interface use the embedding trick from `alpha_test.go` (embed `gofpdfwrapper.Fpdf`, override only the methods you need).
- **Key files:**
  - `pkg/html/css/computed.go` — `Apply` switch (new properties added here).
  - `pkg/html/css/length.go` — `ParseLength` (calc() entry point).
  - `pkg/html/css/calc.go` — calc evaluator (already supports `%` via `ParseLengthCtx`).
  - `pkg/html/translate/style.go` — `computeNodeStyle` + `computeNodeStyleRooted` (cascade entry).
  - `pkg/html/translate/translate.go` — block dispatcher + paragraph builder.
  - `pkg/html/translate/flex.go` + `flex_layout.go` — flex container builder + quantizer.
  - `pkg/html/translate/container.go` — `blockContainer` (extended for page splitting).
  - `pkg/components/richtext/richtext.go` — RichText component (extended for Background/LocalAnchor per-run rendering).
  - `internal/providers/gofpdf/richtext.go` — `AddRichText` (the renderer that needs to consume Background/LocalAnchor/LetterSpacing).
  - `internal/providers/gofpdf/cellwriter/{fillcolorstyler,bordercolorstyler,borderradius,persideborder}.go` — chain nodes; gradient fill inserts before fillcolor, shadow before everything.
  - `internal/providers/gofpdf/provider.go` — provider type + capability assertions.
  - `internal/providers/gofpdf/gofpdfwrapper/fpdf.go` — wrapper interface (already has `SetDashPattern`, `RegisterImageReader`, `LinearGradient`, `RadialGradient`).
  - `pkg/props/cell.go` — `props.Cell` (new fields for gradient, shadow, outline).
  - `pkg/props/richtext.go` — `RichRun` (already has Background/LocalAnchor; renderer just needs to consume them).
- **Gotchas:**
  - **`fpdf.LinearGradient` and `RadialGradient` are already in the wrapper interface** but cover only solid-axis 2-stop gradients. For multi-stop or non-axis-aligned gradients we rasterise to PNG and embed via `RegisterImageReader` (matches the SVG path in `image.go`).
  - **`box-shadow` color alpha is critical.** Without alpha (Plan A's pipeline), shadow bleed looks like a hard rectangle. Use `props.Color.Alpha` set to ~0.3 for the outermost overlay, 0.5 for middle, 0.8 for innermost when emulating blur.
  - **Splittable container (build-time split, NOT render-time):** Maroto is two-phase. `maroto.addRow()` (in `maroto.go` ~lines 228–255) builds the `pages[]` list; `generate()` then renders each pre-built `core.Page` atomically. Calling `fpdf.AddPage()` from inside `blockContainer.Render()` would create a gofpdf page that has no corresponding `core.Page` entry — breaking headers/footers/numbering/concurrent generation. The split must happen during `addRow()`: type-assert `core.Splittable.SplitAt(remainingHeight)` and place `first`/`rest` on separate pages via the existing `fillPageToAddNew()` machinery. `blockContainer.Render` stays pure (no `AddPage()` calls); each slice is a normal blockContainer with a first/middle/last flag controlling which corners round and which border edges draw.
  - **Flex-wrap state:** in single-row mode the layout function returns one slice of widths. In wrap mode it must return `[][]int`. To keep the call sites simple, introduce a `WrappedLayout` type that contains either a single row or a slice of rows.
  - **`order` and Cascadia matching:** sorting flex items by `order` happens AFTER cascade resolution but BEFORE quantization. Cascadia matches by DOM order, so the cascade is unaffected.
  - **Outline vs border:** outline is drawn OUTSIDE the cell box and doesn't count toward layout (unlike border). We need to know the parent's bounding box, which the cellwriter chain has access to via `Fpdf.GetXY`.
  - **Per-run Link (baseline-to-top correction is mandatory):** `Fpdf.Link` takes a rectangle in mm where `y` is the rect's TOP edge. The renderer's `y` is the text BASELINE (gofpdf.Text() convention). The existing Hyperlink path already handles this with `fpdf.LinkString(x, y-lineHeight, ...)`. The new per-run path MUST mirror this: pass `y - lineHeight`, NOT `y`. Without the correction, the clickable rectangle sits 3–5mm below the visible text and the link does not overlap its label.
  - **calc()% in Apply:** the cleanest fix is to pass the parent's content width through `computeNodeStyle` and into `Apply`. Either add a `ctxWidth` parameter to `Apply` or post-process the computed style by re-evaluating any stored calc() expressions. Latter is less invasive but requires storing raw expression strings.
- **Domain context:**
  - The prior plan completed engine+styling. Plan B is renderer + layout. Most work shifts from `pkg/html/css` (parsing) to `internal/providers/gofpdf` (drawing) and `pkg/html/translate/flex*` (layout).
  - The "Splittable container" work is the highest-risk item. It changes the pagination contract subtly: a row is no longer guaranteed atomic. The paginator (`maroto.addRow()`) must be backward-compatible — existing rows that don't implement `Splittable` continue to render atomically because the type assertion falls through to the existing height-check path.
  - **Build-vs-render-phase separation is a HARD invariant.** Plan B reviewers flagged that any attempt to call `fpdf.AddPage()` from inside `Render()` corrupts the `core.Page` list. Tasks 11 and 12 are explicitly scoped to the build phase (`addRow()`) — Render() stays pure.

## Runtime Environment

- **Build/run demo:** `go run ./cmd/html-demo` (writes `test/output/html-demo.pdf`)
- **Run all tests:** `go test ./... -count=1`
- **Run HTML pipeline tests only:** `go test ./pkg/html/... ./pkg/components/htmllist/... -count=1`
- **Lint:** `gofmt -w .`, `go vet ./...`, `golangci-lint run`
- **Visual verification:** open `test/output/html-demo.pdf` after each milestone

## Progress Tracking

**MANDATORY: Update this checklist as tasks complete. Change `[ ]` to `[x]`.**

- [x] Task 1: Paint RichRun.Background in AddRichText (per-run filled rect behind text)
- [x] Task 2: Per-run anchor link rectangles via Fpdf.Link + LetterSpacing via SetCharSpacing
- [x] Task 3: calc()% inside Apply() via context-aware length resolution
- [x] Task 4: Dashed/dotted styled `<hr>` via SetDashPattern
- [x] Task 5: CSS gradients (linear + radial) — parser + raster + cellwriter integration
- [ ] Task 6: box-shadow — parser + provider + render wrapper with multi-shadow support
- [ ] Task 7: text-shadow — per-run shadow before AddText
- [ ] Task 8: outline + outline-offset (drawn outside cell box)
- [ ] Task 9: flex-wrap, order, true *-reverse ordering (quantizer multi-row + sort)
- [ ] Task 10: align-self (per-item cross-axis alignment)
- [ ] Task 11: page-break-{before,after,inside} markers + paginator hint
- [ ] Task 12: Splittable blockContainer (split across page breaks, repaint bg+border)
- [ ] Task 13: Demo + docs sweep

**Total Tasks:** 13 | **Completed:** 5 | **Remaining:** 8

## Implementation Tasks

### Task 1: Paint RichRun.Background in AddRichText

**Objective:** Make `RichRun.Background` actually paint behind the run's text. Closes the prior plan's biggest render-side gap. After this task, `<mark>`, `<kbd>`, inline `<code>`, and `<span style="background-color:…">` all render with their background visible.

**Dependencies:** None

**Files:**
- Modify: `internal/providers/gofpdf/richtext.go` — `AddRichText` layout loop draws fill rect before text
- Test: `internal/providers/gofpdf/richtext_test.go` (new file or extend existing) — assert SetFillColor + Rect called when run has Background

**Key Decisions / Notes:**
- The `AddRichText` layout uses a three-pass algorithm with per-word bounding boxes already computed. Hook into the existing per-run draw call: when `run.Background != nil`, capture (x, y, width, height) and call `fpdf.Rect(x, y, w, h, "F")` with the background colour set first.
- Reset fill colour to the previous value after the fill rect (the existing fill colour state must not leak into following cells).
- Account for `run.Background.Alpha` — when alpha is non-nil and < 1, wrap in `WithAlpha`.

**Definition of Done:**
- [ ] `<mark>highlighted</mark>` produces a yellow filled rect behind the text in the rendered PDF
- [ ] `<kbd>Ctrl+C</kbd>` produces a light-grey rect behind the run
- [ ] Inline `<code>foo()</code>` (outside `<pre>`) produces a light-grey rect
- [ ] `<span style="background-color: rgba(255,0,0,0.3)">red</span>` renders translucent red
- [ ] Unit test asserts `SetFillColor` + `Rect("F")` are called per run with Background, with correct bounding box
- [ ] No regression on runs without Background (no SetFillColor calls, no extra draws)
- [ ] `go test ./... -count=1` passes

**Verify:**
- `go test ./internal/providers/gofpdf/... -count=1`
- `go run ./cmd/html-demo` → open PDF, visually confirm Modern Colours section has tinted span swatches and Expanded Tag Coverage shows yellow `<mark>` highlight

---

### Task 2: Per-run anchor link rectangles + LetterSpacing

**Objective:** Make `RichRun.LocalAnchor` produce a clickable rectangle exactly around the anchor text (not just the whole row), and make `RichRun.LetterSpacing` actually apply via `Fpdf.SetCharSpacing` per run. Both reuse Task 1's per-run bounding-box plumbing.

**Dependencies:** Task 1 (shares the per-run bounding box capture in `AddRichText`)

**Commit:** `feat(html): render RichRun.Background, LetterSpacing, and per-run anchor links` (covers Tasks 1+2)

**Files:**
- Modify: `internal/providers/gofpdf/richtext.go` — (a) when a run has `LocalAnchor != ""`, call `fpdf.Link(x, y, w, h, linkID)` after drawing the text; (b) when a run has `LetterSpacing > 0`, call `fpdf.SetCharSpacing(mmToPt(spacing))` before the run and `SetCharSpacing(0)` after (defer reset).
- Modify: `pkg/components/richtext/richtext.go` — pass a link-id resolver to the provider via SetConfig, or extend `core.RichTextProvider`'s `AddRichText` signature with a resolver param (cleaner: thread `*anchorRegistry` through `RichRun.LocalAnchor` lookups)
- Modify: `pkg/html/translate/anchor.go` — `anchorSource` becomes optional (only wraps a row when ALL runs in it share one anchor; otherwise per-run links suffice)
- Modify: `pkg/html/translate/translate.go` — `paragraphRow` no longer auto-wraps in anchorSource when individual runs carry LocalAnchor (per-run mode)
- Modify: `internal/providers/gofpdf/gofpdfwrapper/fpdf.go` — verify `SetCharSpacing(float64)` exists (it does, via `CharSpacingProvider` from Plan A); confirm mock signature matches

**Key Decisions / Notes:**
- Resolver pattern: `RichRun.LocalAnchor` is a string. The `*anchorRegistry` is held on the translator. To reach it from `AddRichText`, store the registry on the richtext component at construct time and thread it into the provider via a new field in `props.RichText` (e.g. `props.RichText.AnchorResolver func(name string) int`).
- Backward compatibility: when no resolver is provided, runs with LocalAnchor are rendered but produce no link (same behavior as before).
- LetterSpacing units: `RichRun.LetterSpacing` is stored in pt (matches RichRun.Size); pass through directly to `SetCharSpacing`. After each run, restore to `0` so non-spaced runs aren't affected. Defer the reset at the end of `AddRichText` as a safety net.
- Per-run anchor bbox formula: `width = fpdf.GetStringWidth(run.Text)` at the run's font size and family (set via SetFont before measurement); `height = lineHeight` (same as the run's line height in mm). The `(x, y)` arguments to `fpdf.Link` MUST use `y - lineHeight`, NOT `y` directly: gofpdf's `Link()` expects the rect's top edge, but the renderer's `y` is the text BASELINE. The existing `Hyperlink` code already follows this convention (`fpdf.LinkString(x, y-lineHeight, t.width, lineHeight, ...)`). The new per-run path must mirror it exactly. A unit test MUST assert that the y argument passed to `Link()` equals `(token_y - lineHeight)`, not `token_y`.

**Definition of Done:**
- [ ] `<a href="#section">click</a>` in the middle of a `<p>` produces a Link region covering ONLY "click", not the whole paragraph
- [ ] Unit test asserts `fpdf.Link(x, y, w, h, id)` is called exactly once per LocalAnchor run, with `w == GetStringWidth(text)`, `h == lineHeight`, and `y == (token_y - lineHeight)` (baseline-to-top correction matching the existing Hyperlink path)
- [ ] `<span style="letter-spacing: 0.4pt">spaced</span>` calls `SetCharSpacing(0.4)` before the run and `SetCharSpacing(0)` after (verified via mock recording call order)
- [ ] Runs without LetterSpacing produce zero `SetCharSpacing` calls (no regression)
- [ ] Backward-compatible: external `<a href="https://…">` continues to work (LinkString still called)
- [ ] Forward-reference test still passes (link before target in source order)
- [ ] `go test ./pkg/html/... ./internal/providers/gofpdf/... -count=1` passes

**Verify:**
- `go test ./pkg/html/... ./internal/providers/gofpdf/... -count=1`
- `go run ./cmd/html-demo` → open PDF, hover over the "Modern features" anchor in the Internal anchors section — only the link text should be clickable, not the whole paragraph

---

### Task 3: calc()% inside Apply() via context-aware length resolution

**Objective:** Make `calc(100% - 20mm)` and other CSS percentages resolve correctly when used in `width`, `height`, `padding`, `margin`, `top`, etc. Currently `Apply()` calls `ParseLength(val, fontSize)` which doesn't know the context width, so `%` evaluates to 0.

**Dependencies:** None

**Commit:** `fix(html): calc() percentages resolve against parent content width in Apply`

**Files:**
- Modify: `pkg/html/css/computed.go` — add a thread-through option: either (a) a new `Apply` method `ApplyCtx(prop, val string, parent *ComputedStyle, ctxWidth float64)`, or (b) store raw expressions for percentage-bearing properties and re-evaluate post-cascade
- Modify: `pkg/html/translate/style.go` — `computeNodeStyle` resolves the parent's content width and passes it through
- Test: `pkg/html/css/computed_test.go` (new or extend existing)

**Key Decisions / Notes:**
- Approach A (chosen for simplicity): add a new `Apply(prop, val, parent, ctxWidth)` method; the existing `Apply(prop, val, parent)` becomes a thin wrapper that passes 0. `computeNodeStyle` derives `ctxWidth` from the parent's width (or the document content width when no parent).
- `ParseLengthCtx` (already exists in `pkg/html/css/calc.go`) is the new dispatch target inside `Apply` for length-typed properties.
- Width-relative percentages: `width`, `max-width`, `padding-*`, `margin-*` (horizontal), `text-indent` resolve against parent content width. Height percentages stay 0 in v2 (document the limitation).

**Definition of Done:**
- [ ] `<div style="width: calc(100% - 20mm)">…</div>` at default A4 (170mm content) computes Width = 150mm
- [ ] `<div style="padding: calc(5% + 2mm)">…</div>` computes PaddingLeft/Right relative to parent content width
- [ ] Backward compatibility: existing `width: 50%` (without calc) continues to work
- [ ] Unit test verifies `ParseLengthCtx` is called for length-typed properties when val contains `%` or `calc(`
- [ ] `go test ./pkg/html/... -count=1` passes

**Verify:**
- `go test ./pkg/html/... -count=1 -run TestCalc`
- `go run ./cmd/html-demo` → confirm `.var-demo` now uses the explicit width from `calc(100% - 20mm)` (already in demo) — visible as a narrower box than the page width

---

### Task 4: Dashed/dotted styled `<hr>` lines

**Objective:** Honour `border-top-style: dashed | dotted | solid` on `<hr>` (and `<div style="border: 1pt dashed">`). gofpdf's `SetDashPattern` is already in the wrapper interface; the styledHrRow and per-side border styler just need to pick a pattern.

**Dependencies:** None

**Commit:** `feat(html): dashed and dotted border styles via SetDashPattern`

**Files:**
- Modify: `pkg/html/translate/translate.go` — `styledHrRow` picks dash pattern from `border-top-style`
- Modify: `internal/providers/gofpdf/cellwriter/persideborder.go` — apply dash pattern per side
- Modify: `pkg/components/line/line.go` — accept a `props.Line.Style` (already exists?) and call `SetDashPattern`
- Modify: `pkg/props/line.go` — extend Line struct with Style field if absent
- Test: `pkg/components/line/line_test.go`, `pkg/html/translate/block_tags_test.go`

**Key Decisions / Notes:**
- Pattern mapping: `solid` → `nil` (default), `dashed` → `[2.0, 1.0]` mm, `dotted` → `[0.4, 0.4]` mm.
- Restore the dash pattern to `nil` after each line/border stroke (defer reset, same pattern as alpha).
- `<hr style="border-top: 2pt dashed #888">` already parses correctly (Task 4 of the prior plan); this task just makes the renderer pick up `BorderTopStyle`.

**Definition of Done:**
- [ ] `<hr style="border-top: 2pt dashed #888">` renders as a dashed line at 2pt thickness in grey
- [ ] `<hr style="border-top: 1pt dotted red">` renders dotted red
- [ ] `<div style="border: 1pt dashed #aaa">` renders all four borders dashed
- [ ] Dash pattern is reset after each stroke (no leak into subsequent native rows)
- [ ] Unit test asserts `SetDashPattern([2,1], 0)` is called for dashed, `SetDashPattern([0.4,0.4], 0)` for dotted, and reset to `nil`
- [ ] `go test ./pkg/components/line/... ./pkg/html/... -count=1` passes

**Verify:**
- `go test ./pkg/components/line/... ./internal/providers/gofpdf/cellwriter/... ./pkg/html/... -count=1`
- `go run ./cmd/html-demo` → confirm the dashed hr in the Expanded tag coverage section now actually renders dashed

---

### Task 5: CSS gradients (linear + radial)

**Objective:** Support `background-image: linear-gradient(<angle>, <stops>...)` and `radial-gradient(<position>, <stops>...)` on block elements. Render via raster: build an in-memory RGBA image at the cell's pixel size, embed via `RegisterImageReader` as a PNG, and stretch to the cell.

**Dependencies:** None (parallel-safe)

**Commit:** `feat(html): linear-gradient and radial-gradient backgrounds via raster fallback`

**Files:**
- Create: `pkg/html/css/gradient.go` — `ParseLinearGradient`, `ParseRadialGradient` returning a `Gradient` struct with angle/position + colour stops
- Modify: `pkg/html/css/computed.go` — `Apply` handles `background-image` and stores on `ComputedStyle.Background`
- Modify: `pkg/props/cell.go` — add `BackgroundGradient *props.Gradient` field
- Create: `pkg/props/gradient.go` — public `Gradient` type for cell prop
- Create: `pkg/core/gradient_provider.go` — `GradientProvider` capability interface
- Modify: `internal/providers/gofpdf/provider.go` — implement `GradientProvider` (rasterise + embed)
- Create: `internal/providers/gofpdf/gradient.go` — rasterisation logic (`image.NewRGBA`, per-pixel interpolation, `png.Encode`, `RegisterImageReader`)
- Modify: `internal/providers/gofpdf/cellwriter/builder.go` — insert `gradientStyler` BEFORE `fillColorStyler` so gradient overrides solid fill
- Create: `internal/providers/gofpdf/cellwriter/gradientstyler.go`
- Test: `pkg/html/css/gradient_test.go`, `internal/providers/gofpdf/gradient_test.go`

**Key Decisions / Notes:**
- Use ~75 DPI for the raster (enough for visible gradients, keeps PDF size bounded). A 100mm × 50mm cell = 296 × 148 px = ~175 KB uncompressed RGBA, ~5–15 KB PNG.
- **Cache deduplication (correctness, not optimisation):** `image.go`'s existing pattern uses `uuid.NewRandom()` as the imgName for every `RegisterImageOptionsReader` call. This means gofpdf re-embeds identical PNG blobs per call. For gradients, this would inflate PDF size dramatically (3 columns × 6 gradient cells × ~10KB each = ~180KB of duplicated data on a single page). The gradient provider MUST maintain a `sync.Map` (cacheKey → registered imgName) on its struct. cacheKey = SHA-256 hex of (canonical gradient string, width mm rounded to 0.1, height mm rounded to 0.1, dpi). On first use: generate a deterministic imgName `"gradient-" + cacheKey[:16]`, `RegisterImageOptionsReader` with that name, store in map. On subsequent use: reuse the stored imgName and call `fpdf.Image` directly.
- **Coordinate convention — MUST add margin offsets:** Maroto cell coordinates are margin-relative (`cell.X=0` means the left content edge, not the left page edge). The gradient placement call MUST follow `image.go`'s convention exactly: `fpdf.Image(name, cell.X + margins.Left, cell.Y + margins.Top, cell.Width, cell.Height, false, "PNG", 0, "")`. The provider reads margins via `fpdf.GetMargins()` (already used elsewhere in `provider.go`). A unit test MUST assert the `x` argument equals `cell.X + leftMargin`, not `cell.X` alone.
- Angle conversion: CSS `0deg` = "to top" = vertical, going upward. We map to canonical `(dx, dy)` direction in pixel space.
- Multi-stop interpolation: 2-stop is `lerp(c0, c1, t)`. Multi-stop: find the segment, lerp within it.
- Radial gradient: support `radial-gradient(circle at center, <stops>)` and named positions (`at top right`). Skip elliptical sizing in v1 — always circular.
- Failure modes: unparseable gradient → store nil + log via `unsupportedHandler`. Don't fall back to solid fill.

**Definition of Done:**
- [ ] `<div style="background-image: linear-gradient(to right, red, blue)">…</div>` renders red→blue horizontal gradient
- [ ] `<div style="background-image: linear-gradient(45deg, #ff0000 0%, #00ff00 50%, #0000ff 100%)">` renders 3-stop diagonal gradient
- [ ] `<div style="background-image: radial-gradient(circle at center, white, black)">` renders white-centre to black-edge
- [ ] Gradient cache returns the same registered imgName for two cells with identical gradient + dims (unit test asserts `RegisterImageOptionsReader` is called exactly once for two identical gradients on the same page)
- [ ] `fpdf.Image` is called with `x == cell.X + leftMargin` and `y == cell.Y + topMargin` (unit test asserts the offset is applied — no missing-margin regression)
- [ ] Unparseable gradient (e.g. `linear-gradient(weird stuff)`) logs via `unsupportedHandler` and falls back to no background
- [ ] Unit test asserts the rasteriser produces correct pixel colour at known sample points (e.g. centre, corners)
- [ ] `go test ./pkg/html/css/... ./internal/providers/gofpdf/... -count=1` passes

**Verify:**
- `go test ./pkg/html/css/... ./internal/providers/gofpdf/... -count=1 -run Gradient`
- `go run ./cmd/html-demo` → confirm new "Gradients" demo section renders a couple of gradient hero blocks

---

### Task 6: box-shadow

**Objective:** Support `box-shadow: <offset-x> <offset-y> [<blur>] [<spread>] <color>` (single and comma-separated multi-shadow). Drawn behind the container as one or more offset filled rectangles. Blur is approximated by overlaid translucent rects at the corners.

**Dependencies:** None (parallel-safe)

**Commit:** `feat(html): box-shadow with blur approximation and multi-shadow support`

**Files:**
- Modify: `pkg/html/css/computed.go` — `Apply` handles `box-shadow` (parse via new helper)
- Create: `pkg/html/css/shadow.go` — `ParseShadow` returning `Shadow` slice (multi-shadow)
- Modify: `pkg/props/cell.go` — add `BoxShadow []props.Shadow` field
- Create: `pkg/props/shadow.go` — public `Shadow` type
- Create: `pkg/core/shadow_provider.go` — `ShadowProvider` capability
- Modify: `internal/providers/gofpdf/provider.go` — implement `ShadowProvider` (draw offset rects with alpha)
- Modify: `internal/providers/gofpdf/cellwriter/builder.go` — insert `shadowStyler` FIRST in chain so it draws beneath all other cell content
- Create: `internal/providers/gofpdf/cellwriter/shadowstyler.go`
- Test: `pkg/html/css/shadow_test.go`, `internal/providers/gofpdf/shadow_test.go`

**Key Decisions / Notes:**
- Single-shadow: draw one filled rect offset by (x, y) from the cell, with the shadow colour.
- Blur approximation: draw N=3 overlaid rects, each slightly larger than the previous, with alpha 0.3 / 0.5 / 0.8 (outermost most translucent). Total spread = `blur` parameter.
- Multi-shadow: parse comma-separated list (up to 4). Render in source order (first listed renders furthest back).
- `inset` keyword: draw inside the cell instead of behind. Flip the offset direction.
- `spread`: optional 4th length token, expands the shadow rect uniformly before applying blur.
- **Cursor save/restore (mandatory):** `shadowStyler.Apply` MUST capture `(x, y) = fpdf.GetXY()` BEFORE any draw operation, perform the shadow draws (which may move the cursor via internal gofpdf state), then call `fpdf.SetXY(x, y)` to restore the cursor BEFORE forwarding to the next chain node. The same save/restore pattern applies in `outlineStyler` (Task 8). Without this, downstream nodes that read cursor position (e.g. borderRadius, outlineStyler) draw at wrong coordinates. Add a regression test that places a box-shadow and an outline on the same cell and asserts the outline coordinates are independent of the shadow draw operations.
- Save/restore fill colour too (so the cellwriter chain's later fillColorStyler isn't polluted by the shadow's last SetFillColor call).

**Definition of Done:**
- [ ] `<div style="box-shadow: 2mm 2mm #00000040">…</div>` renders a translucent black shadow offset 2mm down and right
- [ ] `<div style="box-shadow: 0 4mm 6mm rgba(0,0,0,0.2)">` renders a blurred drop shadow (3 overlaid translucent rects)
- [ ] `<div style="box-shadow: 2mm 0 red, -2mm 0 blue">` renders both shadows (right red, left blue)
- [ ] `<div style="box-shadow: inset 0 2mm 4mm rgba(0,0,0,0.3)">` renders an inset shadow at the top
- [ ] Unit test asserts `Rect("F")` is called the expected number of times with offset + alpha values
- [ ] No regression on cells without box-shadow (zero Rect calls from shadow path)
- [ ] `go test ./pkg/html/css/... ./internal/providers/gofpdf/... -count=1` passes

**Verify:**
- `go test ./pkg/html/css/... ./internal/providers/gofpdf/... -count=1 -run Shadow`
- `go run ./cmd/html-demo` → confirm new Shadow demo section renders cards with drop shadows

---

### Task 7: text-shadow

**Objective:** Support `text-shadow: <offset-x> <offset-y> [<blur>] <color>` per element. Each run with text-shadow renders the text twice: once in the shadow colour at offset, then once normally on top.

**Dependencies:** Task 6 (shares shadow parsing)

**Commit:** `feat(html): text-shadow renders shifted shadow text behind run`

**Files:**
- Modify: `pkg/html/css/computed.go` — `Apply` handles `text-shadow` (reuses Shadow parser)
- Modify: `pkg/props/richtext.go` — add `RichRun.TextShadow *props.Shadow`
- Modify: `pkg/html/translate/style.go` — `applyInlineStyleToRuns` threads TextShadow
- Modify: `internal/providers/gofpdf/richtext.go` — when run has TextShadow, call `AddText` at offset with shadow colour before the normal render
- Test: `internal/providers/gofpdf/richtext_test.go`

**Key Decisions / Notes:**
- **Inline implementation (not a pre-pass):** for each token inside the normal `AddRichText` render loop, if `run.TextShadow != nil`: (1) save current text colour via `origColor`, (2) `SetTextColor(shadow.color)`, (3) draw the text at `(x + shadow.offsetX, y + shadow.offsetY)`, (4) restore `SetTextColor(origColor)`, (5) draw the text normally at `(x, y)`. This piggybacks on the existing per-token font/color setup and avoids the font-state hazard of a separate pre-pass (the existing `lastRunIdx` optimisation would otherwise reuse the prior run's font when shadow tokens cross run boundaries).
- Text-shadow blur is approximated by rendering the shadow twice with small additional offsets at half-alpha (cheap, looks acceptable for headings). Skip multi-shadow on text — only the first shadow listed is rendered.
- Skip text-shadow when alpha would be 0 or color is nil.

**Definition of Done:**
- [ ] `<h1 style="text-shadow: 1mm 1mm rgba(0,0,0,0.4)">Heading</h1>` renders the heading with a translucent shadow behind it
- [ ] Multi-shadow values (comma-separated) emit only the first shadow (limitation documented)
- [ ] Unit test asserts `AddText` is called with shadow colour at offset before normal text render
- [ ] No regression for runs without text-shadow
- [ ] `go test ./pkg/html/... ./internal/providers/gofpdf/... -count=1` passes

**Verify:**
- `go test ./internal/providers/gofpdf/... -count=1 -run TextShadow`
- `go run ./cmd/html-demo` → confirm Text-shadow demo section shows shadowed headings

---

### Task 8: outline + outline-offset

**Objective:** Support `outline: <width> <style> <color>` and `outline-offset: <length>`. Drawn outside the cell box (does not affect layout).

**Dependencies:** None (parallel-safe)

**Commit:** `feat(html): outline and outline-offset drawn outside cell box`

**Files:**
- Modify: `pkg/html/css/computed.go` — `Apply` handles `outline*` (similar to border)
- Modify: `pkg/props/cell.go` — add `OutlineWidth`, `OutlineStyle`, `OutlineColor`, `OutlineOffset` fields
- Create: `pkg/core/outline_provider.go` — `OutlineProvider` capability with `DrawOutline(cell, prop)`
- Modify: `internal/providers/gofpdf/provider.go` — implement
- Modify: `internal/providers/gofpdf/cellwriter/builder.go` — insert `outlineStyler` as LAST node (drawn on top, outside cell)
- Create: `internal/providers/gofpdf/cellwriter/outlinestyler.go`
- Test: `pkg/html/css/computed_test.go`, `internal/providers/gofpdf/cellwriter/outlinestyler_test.go`

**Key Decisions / Notes:**
- Outline width adds to the cell's visual size but NOT the layout box — the outline draws outside the cell bounds.
- Default `outline-offset` is 0; positive offsets push the outline further out, negative offsets pull it in (overlapping the cell border).
- Reuse dashed/dotted line styles from Task 4.
- Cursor save/restore in outlineStyler matches Task 6's shadowStyler pattern: capture `GetXY()` first, draw, restore before forwarding to the next chain node.
- **Known limitation — right-edge overdraw in flex rows:** in Maroto's left-to-right row rendering with no z-ordering, a flex item's right outline edge is painted over by the next item's fill rect. Only the rightmost item in a row will have a fully visible outline; left and middle items lose their right edge to the next item's background. Document this in `docs/v2/html-support.md`. A clean fix (deferred second pass after row completion) is left for a follow-up plan if users hit it.

**Definition of Done:**
- [ ] `<div style="outline: 0.5mm solid red; outline-offset: 1mm">…</div>` renders a red outline 1mm outside the cell border
- [ ] Outline does not affect parent layout (sibling elements don't shift when outline is added)
- [ ] `outline-style: dashed | dotted | solid` all work
- [ ] Unit test asserts `Rect` or `Line` calls are positioned OUTSIDE the cell's nominal bounds by `outline-offset + width/2`
- [ ] `go test ./pkg/html/css/... ./internal/providers/gofpdf/cellwriter/... -count=1` passes

**Verify:**
- `go test ./pkg/html/css/... ./internal/providers/gofpdf/cellwriter/... -count=1 -run Outline`
- `go run ./cmd/html-demo` → confirm Outline demo section shows boxes with outlines at varying offsets

---

### Task 9: flex-wrap, order, true *-reverse

**Objective:** Make the flex quantizer emit multiple rows when items wrap, sort items by `order` before quantizing, and actually reverse child order when `flex-direction: *-reverse` is set.

**Dependencies:** None (parallel-safe with most)

**Commit:** `feat(html): flex-wrap, order, and *-reverse ordering`

**Files:**
- Modify: `pkg/html/css/computed.go` — add `FlexWrap string` and `Order int` to `ComputedStyle`
- Modify: `pkg/html/translate/flex.go` — pre-sort items by Order (DOM order tiebreak); reverse for `*-reverse`
- Modify: `pkg/html/translate/flex_layout.go` — quantizer returns `WrappedLayout { Rows [][]int }` when `flex-wrap: wrap` and items overflow grid
- Modify: `pkg/html/translate/flex.go` — `flexRow` becomes `flexRows` returning `[]core.Row` when wrapping
- Modify: `pkg/html/translate/translate.go` — `dispatchBlockRows` flex branch emits all wrapped rows
- Test: `pkg/html/translate/flex_test.go` (extend with wrap + order + reverse cases)

**Key Decisions / Notes:**
- Wrap detection: when the sum of items' minimum widths (from flex-basis or content) exceeds the grid, start a new row.
- For simplicity, packing is greedy: each row fills until the next item won't fit, then start a new row. Optimal binpacking is overkill.
- `wrap-reverse` reverses the row order in the final emission, not within each row.
- Items without `order` default to 0 (matches CSS).

**Definition of Done:**
- [ ] 6 flex items at flex-basis: 33% with `flex-wrap: wrap` produce 2 rows of 3 items each
- [ ] 4 items with `order: 2, 1, 3, 0` render in DOM order [3, 1, 0, 2] (item with order=0 first, then 1, 2, 3)
- [ ] `flex-direction: row-reverse` reverses item order at quantization time
- [ ] `wrap-reverse` reverses the order of wrapped rows
- [ ] Unit tests cover all four behaviors with structure assertions
- [ ] **Golden test for single-row preservation:** before the WrappedLayout refactor lands, capture exact `[]int` outputs of `computeFlexSizes` and exact col-size outputs of `assembleFlexCols` for 5 representative single-row inputs (equal items, percentage basis, grow weights, mixed, with gap) as golden values. After the refactor, assert the same inputs produce byte-identical output. This turns the "bit-for-bit" claim into a verifiable test.
- [ ] No regression on single-row flex (zero wrap, no order)
- [ ] `go test ./pkg/html/translate/... -count=1` passes

**Verify:**
- `go test ./pkg/html/translate/... -count=1 -run TestFlex`
- `go run ./cmd/html-demo` → confirm new flex-wrap demo section shows a 2-row pill cloud

---

### Task 10: align-self (per-item cross-axis alignment)

**Objective:** Support per-item cross-axis alignment via `align-self` on flex children. Overrides the container's `align-items` for the specific child.

**⚠️ `align-content` removed from scope:** `align-content` only has effect when the flex container has an explicit height greater than its content height — but `blockContainer` currently auto-calculates height from its children (CSS `height: 200mm` is parsed into `ComputedStyle.Height` but never applied as a layout constraint). Implementing explicit-height containers is a separate substantial change (requires clamping, overflow handling, GetHeight() override). `align-content` is therefore deferred to a follow-up plan that introduces explicit container height. Documented under Deferred Ideas.

**Dependencies:** None (decoupled from Task 9 since it's per-item, not row-distribution)

**Commit:** `feat(html): align-self per-item flex alignment`

**Files:**
- Modify: `pkg/html/css/computed.go` — add `AlignSelf string`
- Modify: `pkg/html/translate/flex.go` — apply per-item alignment to each child col (override container `align-items`)
- Test: `pkg/html/translate/flex_test.go`

**Key Decisions / Notes:**
- `align-self: stretch` is the default; explicit `flex-start | flex-end | center` map to existing col-level alignment hooks (already used by `align-items`).
- `align-self: auto` (the CSS default) inherits container `align-items` — no override applied.

**Definition of Done:**
- [ ] `<div style="display:flex"><div style="align-self: flex-end">…</div></div>` renders that child bottom-aligned
- [ ] `<div style="display:flex; align-items:center"><div style="align-self:flex-start">…</div></div>` renders that child top-aligned (override wins)
- [ ] `align-self: auto` defers to container `align-items` (no override behaviour change)
- [ ] Unit tests cover all three (start, end, center) plus auto inheritance
- [ ] `go test ./pkg/html/translate/... -count=1` passes

**Verify:**
- `go test ./pkg/html/translate/... -count=1 -run TestAlign`
- `go run ./cmd/html-demo` → confirm align demo section shows aligned items

---

### Task 11: page-break-{before,after,inside} + break-inside

**Objective:** Emit page-break markers when CSS asks for them, and pass a "keep together" hint to the paginator for `break-inside: avoid` containers.

**⚠️ Architectural constraint — break runs in `addRow()`, not `Render()`:** Same constraint as Task 12. A zero-height marker row that calls `fpdf.AddPage()` from `Render()` corrupts the `core.Page` list. Instead, a pageBreakRow must report a sentinel height that causes `addRow()`'s existing height check to fail (overflow the page) and call `fillPageToAddNew()`. Render() of the marker is then a no-op.

**Dependencies:** None (Task 12 depends on this)

**Commit:** `feat(html): page-break controls and break-inside hint`

**Files:**
- Modify: `pkg/html/css/computed.go` — add `PageBreakBefore string`, `PageBreakAfter string`, `BreakInside string`
- Create: `pkg/html/translate/pagebreak.go` — `pageBreakRow` reports `GetHeight() == math.MaxFloat64 / 2` (a finite but unfittable sentinel); `Render()` is a no-op. This causes `addRow()`'s height overflow path to call `fillPageToAddNew()` naturally, then `addRow()` retries — the now-fresh page still cannot fit the sentinel, so the row is skipped (or wrapped to render nothing). The cleaner alternative: add `IsPageBreak() bool` capability on the row, type-asserted in `addRow()` to call `fillPageToAddNew()` and skip placing the marker row.
- Modify: `maroto.go` — in `addRow()`, type-assert `core.PageBreaker` (single-method `IsPageBreak() bool`); when true, call `fillPageToAddNew()` + `addHeader()` and continue without placing the row on any page.
- Modify: `pkg/html/translate/translate.go` — `dispatchBlockRows` prepends/appends pageBreakRow when computed style requests it
- Create: `pkg/core/page_break.go` — `PageBreaker` interface (`IsPageBreak() bool`)
- Test: `pkg/html/translate/pagebreak_test.go` — assert pageBreakRow implements PageBreaker
- Test: `maroto_test.go` — integration: HTML with `<p>A</p><div style="page-break-after:always">x</div><p>B</p>` produces 2 pages; page 1 contains rows for A + x's content, page 2 starts with B. Verify `core.Page` list length == 2 and each page's header/footer hooks fire.

**Key Decisions / Notes:**
- `IsPageBreak() bool` interface approach (chosen over sentinel height because it's explicit and survives all paginator code paths). Default rows that don't implement it behave normally — no regression.
- `page-break-before: avoid` is harder — it requires the paginator to retroactively decide not to break. v2 implements `avoid` as a hint stored on a row via a separate `BreakHint` interface; the paginator reads it during the height check. If implementation cost is high, defer `avoid` to a follow-up and document the limitation.
- `break-inside: avoid` on a `blockContainer` stores a flag that `Splittable.SplitAt(remainingHeight)` returns `(nil, self, true)` (no split, push whole thing to next page) UNLESS the container is taller than a full page.
- No `PageBreakProvider` capability or `BreakNow()` method is needed — the break happens via `addRow()` calling existing `fillPageToAddNew()` on the paginator's internal state.

**Definition of Done:**
- [ ] `<div style="page-break-before: always">…</div>` starts a new page before the div
- [ ] `<div style="page-break-after: always">…</div>` starts a new page after the div
- [ ] `<div style="break-inside: avoid">…</div>` flags the container; paginator behavior tested in Task 12
- [ ] Integration test asserts `len(maroto.pages) == 2` after a page-break-after directive; header/footer hooks fire for both pages
- [ ] Unit test asserts `pageBreakRow.IsPageBreak()` returns true and `pageBreakRow.Render()` produces zero fpdf draw calls
- [ ] `go test ./pkg/html/translate/... -count=1` passes

**Verify:**
- `go test ./pkg/html/translate/... -count=1 -run TestPageBreak`
- `go run ./cmd/html-demo` → verify a `<div style="page-break-after: always">` produces a page break in the output PDF

---

### Task 12: Splittable blockContainer

**Objective:** Allow a tall `blockContainer` to split across page breaks — repainting its background, border, and per-side padding on each page slice. The single biggest layout/render change in Plan B.

**⚠️ Architectural constraint — split happens in `addRow()`, not `Render()`:** Maroto is strictly two-phase. The `pages[]` list is built during `addRow()` (in `maroto.go`); `generate()` then renders each pre-built `core.Page` independently. Calling `fpdf.AddPage()` from inside `blockContainer.Render()` would create a gofpdf page that has no corresponding `core.Page` entry — breaking headers, footers, page numbering, and concurrent generation. The Splittable contract MUST therefore be honoured by `addRow()` via type assertion, not by Render().

**Dependencies:** Task 11 (uses pageBreakRow signalling)

**Commit:** `feat(html): blockContainer splits across page breaks with bg/border repaint`

**Files:**
- Create: `pkg/core/splittable.go` — `Splittable` interface (`SplitAt(remainingHeight float64) (first, rest core.Component, didSplit bool)`)
- Modify: `pkg/html/translate/container.go` — `blockContainer` implements `Splittable`. The split is a pure layout operation: given remaining height, return `(first, rest)` where `first.GetHeight() <= remainingHeight`. Render() remains pure — it just draws bg/border/children for the slice it was given. No `fpdf.AddPage()` calls from Render().
- Modify: `pkg/html/translate/container.go` — propagate "this is a first-slice" / "this is a last-slice" / "this is a middle-slice" flag to control which corners are rounded and which border edges to draw.
- Modify: `maroto.go` — in `addRow()` (around lines 228–255 where the height check happens): when a row's single component implements `core.Splittable` and `GetHeight() > remainingHeight`, call `SplitAt(remainingHeight)`. If `didSplit == true`, wrap `first` in a row and `addRow()` it (fits the current page), then call `fillPageToAddNew()` + `addHeader()`, then recursively `addRow()` a row wrapping `rest` (which itself may need splitting).
- Modify: `internal/providers/gofpdf/provider.go` — no `AddPage()` call from Splittable path; provider is unchanged here. The existing `Fpdf.GetY`/`GetPageSize` access via Render() is removed.
- Test: `pkg/html/translate/container_test.go` — assert that `blockContainer.SplitAt(h)` with a 300mm container and `h == 200mm` returns `(first, rest, true)` with `first.GetHeight() <= 200mm` and `rest.GetHeight() ≈ container.GetHeight() - first.GetHeight()`
- Test: `maroto_test.go` (new test or extend existing) — integration test: registering a 300mm-tall blockContainer via `addRow()` produces `len(pages) == 2`, each page has the correct slice's children, and core.Page metadata (number, header/footer hooks) is correct on both pages.

**Key Decisions / Notes:**
- `SplitAt(h)` returns `(first, rest, didSplit)`. `first` contains as many children as fit within `h` (≤ h); `rest` is the remainder. `didSplit == false` means "this row fits as-is" (caller proceeds normally without splitting); `didSplit == true && first == nil` means "atomic mode — push the whole row to the next page".
- Bg/border repaint: each slice's Render() simply paints its own bg + relevant border edges based on the slice flag (first/middle/last). No coordination between slices is needed because each slice is a normal `blockContainer` (with a different children subset + slice flag) addRow()'d into a different page.
- **Rounded-corner split behaviour (decided):** first-slice keeps the top corners rounded and renders a flat bottom edge; last-slice keeps the bottom corners rounded and renders a flat top edge; middle-slices have all four corners flat. Matches browser behaviour.
- Atomic mode: when `break-inside: avoid`, `SplitAt` returns `(nil, self, true)` to force the container to the next page. If the container is still taller than a full page after that, fall back to splitting and log a warning via `unsupportedHandler`.
- Cache invalidation: `GetHeight` cache on the original container is unaffected because `first` and `rest` are NEW `blockContainer` instances with their own caches.
- Concurrent generation safety: because all splits happen during `addRow()` (build phase), the resulting `pages[]` list is fully constructed before `generate()`/`generateConcurrently()` runs. Each page renders atomically — no cross-page state.

**Definition of Done:**
- [ ] A `blockContainer` with content totaling 300mm height on A4 (≈277mm printable) splits into two pages
- [ ] Background color repaints on the second-page slice — integration test asserts that after `AddPage()` during a split, `SetFillColor` is called again with the container background colour and `Rect("F")` is called to fill the new page slice BEFORE any child content renders on that page (verified via mock recording call order)
- [ ] Border-top renders on first-page slice; border-bottom on second-page slice; left/right on both (unit test asserts which border-side draw calls happen on each slice)
- [ ] Container with `border-radius: 4mm` split across two pages: first slice draws rounded top corners + flat bottom edge; second slice draws flat top edge + rounded bottom corners (unit test asserts the per-corner draw call set on each slice)
- [ ] `break-inside: avoid` on a fits-in-one-page container pushes it to the next page intact
- [ ] `break-inside: avoid` on a too-tall container logs warning + falls back to splitting
- [ ] Backward compatibility: rows that don't implement Splittable continue to render atomically
- [ ] Integration test asserts page count = 2 for the constructed input
- [ ] `go test ./pkg/html/translate/... -count=1` passes

**Verify:**
- `go test ./pkg/html/translate/... -count=1 -run TestContainer`
- `go run ./cmd/html-demo` → confirm a long demo section now spans two pages with consistent background colour

---

### Task 13: Demo + docs sweep

**Objective:** Extend the demo with sections that exercise every new feature, update `docs/v2/html-support.md` to remove the Known Limitations entries that are now implemented, and add documentation for each new feature.

**Dependencies:** Tasks 1-12

**Commit:** `feat(demo): showcase Plan B effects + docs sweep`

**Files:**
- Modify: `cmd/html-demo/main.go` — add sections: gradients, shadows, outline, flex-wrap+order, multi-line align, dashed hr, per-run anchors, page-break + container splitting
- Modify: `docs/v2/html-support.md` — remove limitations now resolved (RichRun.Background, per-run anchors, calc()% in Apply, dashed hr); add new sections for gradients, shadows, outline, flex-wrap, page-break controls
- Add: `cmd/html-demo/assets/extra.css` extended with `:root` vars used by new demo sections

**Definition of Done:**
- [ ] Demo PDF includes a visually-identifiable section for each of Tasks 1-12
- [ ] `docs/v2/html-support.md` Known Limitations section is reduced to only items that remain unresolved (gradient blur quality, gaussian-true blur, conic gradients)
- [ ] Selectors table documents new align-self / align-content / flex-wrap / order / *-reverse properties
- [ ] Page-break section documents page-break-before/after/inside semantics and the v1 splitting limitation that's now resolved
- [ ] `go run ./cmd/html-demo` exits 0 and produces a PDF > 50KB
- [ ] `go test ./... -count=1` passes
- [ ] `gofmt -w .`, `go vet ./...`, `golangci-lint run` clean for changed files

**Verify:**
- `go test ./... -count=1`
- `go run ./cmd/html-demo` and visually inspect `test/output/html-demo.pdf` — every demo section should look as described

---

## Testing Strategy

- **Unit tests:** parser cases for each new CSS property; provider methods asserted via lightweight Fpdf-embedded stubs; structure-tree assertions for translate-layer wiring
- **Integration tests:** end-to-end HTML→PDF for at least one full document combining several new features (helps catch interactions like "gradient + box-shadow + outline on the same div")
- **Visual regression:** the demo PDF is the manual visual checkpoint after each task
- **No pixel-diff tests** — kept out of scope; visual inspection by the user remains the bar

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
| - | - | - | - |
| `Splittable blockContainer` breaks the existing pagination contract for unrelated rows | Medium | High | The interface is opt-in (`Splittable` is detected via type assertion in `addRow()`). Rows that don't implement it fall through to current atomic behavior. Add a regression test using existing demo sections to confirm page counts haven't changed. |
| Calling `fpdf.AddPage()` from inside `Render()` (Tasks 11 + 12 fatal path) corrupts core.Page list — breaks headers, footers, page numbers, concurrent generation | Low (by design) | High | Tasks 11 and 12 are explicitly scoped to the BUILD phase (`maroto.addRow()`) — Render() never calls AddPage(). Code review checklist for both tasks includes "no AddPage() in Render() path". Integration test asserts `len(maroto.pages)` matches expected page count and headers/footers fire on every page after page-break-after / container-split. |
| Gradient PNG placed without margin offsets — every gradient appears displaced on the page | Medium | High | Gradient placement code follows `image.go`'s convention exactly: `fpdf.Image(name, cell.X + margins.Left, cell.Y + margins.Top, ...)`. Unit test asserts the x/y arguments include the offset. |
| Per-run anchor Link y argument uses text baseline instead of rect top | High (without explicit test) | High | Task 2 DoD requires a unit test asserting `y == (token_y - lineHeight)`, matching the existing Hyperlink path. Code review checklist references the `LinkString` baseline correction. |
| Gradient cache fails to deduplicate (image.go's UUID-per-call pattern) — PDF bloat | High (if not addressed) | Medium | Gradient provider maintains its own `sync.Map` (cacheKey → registered imgName) and reuses the stored imgName for repeat gradients. Unit test asserts `RegisterImageOptionsReader` is called once for two identical gradients. |
| shadow/outline styler perturbs cursor for downstream chain nodes | Medium | Medium | Both stylers capture `GetXY()` at entry and `SetXY(x, y)` at exit before forwarding. Regression test places shadow + outline on the same cell and asserts outline coordinates are unaffected. |
| Gradient raster cache explodes PDF size when many cells have unique gradients | Low | Medium | Cache keyed by (gradient hash, dims, dpi). Document the worst case in `docs/v2/html-support.md`. |
| Box-shadow blur approximation looks blocky | High | Low | Document as a known visual limitation; expose `WithShadowBlurIterations(n)` option in a follow-up if quality is unacceptable. Demo uses settings that look reasonable. |
| Per-run anchor rectangles drift due to font-metric inaccuracy | Medium | Low | Run bbox formula matches the existing Hyperlink path: `x = token_x`, `y = token_y - lineHeight` (baseline-to-top correction), `width = fpdf.GetStringWidth(run.Text)` at the run's font, `height = lineHeight`. Unit test asserts the Link call's `y` is exactly `token_y - lineHeight` and `w == GetStringWidth(text)`. |
| `calc()%` Apply refactor breaks existing callers | Medium | High | Add `ApplyCtx` as a NEW method alongside `Apply`. The old method becomes a thin wrapper that calls `ApplyCtx(...,0)`. All existing callers continue to compile and behave identically. |
| Flex-wrap quantizer regressions on single-row content | Medium | High | `WrappedLayout` struct tracks whether wrap actually occurred. Single-row path is preserved bit-for-bit when `flex-wrap: nowrap` (default). |
| `<a href="#x">` per-run links accidentally double-register (row + run) | Medium | Medium | When ANY run in a paragraph has `LocalAnchor`, skip the row-level `anchorSource` wrap. The per-run path becomes the only source of Link calls. |
| `box-shadow inset` interacts badly with `border-radius` | Low | Low | Document the limitation in v2 — inset shadows render as rectangular regardless of border-radius. Round-corner inset clipping deferred. |
| File-size budget exceeded (300 lines) | High | Low | Split early. New files: `gradient.go`, `shadow.go`, `pagebreak.go`, `gradient_provider.go`, `shadow_provider.go`, `outline_provider.go`, `splittable.go`. Existing files (`container.go`, `flex.go`, `flex_layout.go`) split if they approach 280 lines during implementation. |
| New deps added unnecessarily | Low | Low | Plan uses only existing deps. Raster gradients use `image/draw` + `image/png` (stdlib). |
| Page-break `avoid` heuristic produces visually worse output than always-break | Medium | Low | `avoid` is a hint, not a guarantee — document this. Provide a config option `WithStrictPageBreaks(true)` for callers who want hard guarantees. |
| Gradient parser permissive with malformed input → silent visual bugs | Medium | Medium | Strict parser. Malformed gradient → return error to caller, log via `unsupportedHandler`, no background applied. Test with table-driven cases including obviously broken inputs. |

## Goal Verification

### Truths (what must be TRUE for the goal to be achieved)

- `<mark>highlighted</mark>` renders with a visible yellow background behind only the marked text
- `<a href="#section">click</a>` mid-paragraph produces a Link region covering only the link text
- `<div style="width: calc(100% - 20mm)">` produces a div narrower than the page content width by exactly 20mm
- `<hr style="border-top: 2pt dashed #888">` renders as a dashed line, not solid
- `<div style="background-image: linear-gradient(to right, red, blue)">` renders a red-to-blue gradient
- `<div style="box-shadow: 2mm 2mm 4mm rgba(0,0,0,0.4)">` renders with a visible offset shadow
- `<h1 style="text-shadow: 1mm 1mm rgba(0,0,0,0.4)">` renders the heading with a shadow trail
- `<div style="outline: 0.5mm solid red; outline-offset: 1mm">` renders an outline 1mm outside the cell edge
- A 6-item flex container with `flex-wrap: wrap; flex-basis: 33%` renders in 2 rows of 3
- A 4-item flex container with `order: 2, 1, 3, 0` renders items in the visual sequence corresponding to order=0,1,2,3
- A `<div style="page-break-after: always">` creates a hard page break in the PDF; `len(maroto.pages) == 2` and headers/footers fire on both pages
- A blockContainer with content exceeding the printable page height splits across 2 pages, with its background colour repainted on the second page; the split is performed during `addRow()` (no `fpdf.AddPage()` calls from `Render()`); `len(maroto.pages) == 2` and headers/footers fire on both pages
- Per-run `<a href="#x">label</a>` Link rectangle's `y` argument equals `token_y - lineHeight` (baseline-to-top correction)
- Identical gradients on two cells produce exactly one `RegisterImageOptionsReader` call (dedup via stable imgName)
- Gradient `fpdf.Image` call positions include page margin offsets (`x = cell.X + leftMargin`, `y = cell.Y + topMargin`)

### Artifacts (what must EXIST to support those truths)

- `pkg/core/{gradient,shadow,outline}_provider.go` — capability interfaces for new draw operations
- `pkg/core/{page_break,splittable}.go` — interfaces consumed by `maroto.addRow()` (build phase, not provider)
- `pkg/html/css/{gradient,shadow}.go` — CSS parsers for the new properties
- `pkg/html/css/calc.go` (extended) — context-aware `%` resolution inside `Apply`
- `pkg/props/{gradient,shadow}.go` — public types for cell prop
- `pkg/props/cell.go` updated with BackgroundGradient, BoxShadow, OutlineWidth/Style/Color/Offset
- `pkg/props/richtext.go` updated with TextShadow
- `pkg/props/line.go` updated with Style (for dashed/dotted lines)
- `internal/providers/gofpdf/gradient.go` — rasterisation implementation
- `internal/providers/gofpdf/cellwriter/{gradient,shadow,outline}styler.go` — chain nodes
- `pkg/html/translate/{pagebreak}.go` — page break marker
- `pkg/html/translate/container.go` (extended) — Splittable impl
- `pkg/html/translate/flex.go` + `flex_layout.go` (extended) — wrap + order + reverse
- `cmd/html-demo/main.go` (extended) — demo sections
- `docs/v2/html-support.md` (extended) — feature documentation

### Key Links (critical connections that must be WIRED)

- `Apply("background-image", value)` → `ParseLinearGradient`/`ParseRadialGradient` → `Gradient` struct → `props.Cell.BackgroundGradient` → `gradientStyler.Apply` → `GradientProvider.DrawGradient` → rasterised PNG embedded via `RegisterImageReader`
- `Apply("box-shadow", value)` → `ParseShadow` → `Shadow` slice → `props.Cell.BoxShadow` → `shadowStyler.Apply` → `ShadowProvider.DrawShadow` → offset rect with alpha
- `Apply("text-shadow", value)` → `ParseShadow` → `RichRun.TextShadow` → `AddRichText` draws shadow + text
- `Apply("outline", value)` → border-triple parser → `OutlineStyler.Apply` → `Rect`/`Line` at cell + offset
- `Apply("calc(100% - 20mm)", parent, ctxWidth)` → `ParseLengthCtx(value, fontSize, ctxWidth)` → `evalCalc` resolves `%` against `ctxWidth`
- `RichRun.Background` → `AddRichText` measures run bounding box → `Rect("F")` with background fill
- `RichRun.LocalAnchor` → `RichText.AnchorResolver` → linkID → `fpdf.Link(x, y, w, h, linkID)` at the run's bbox
- `<div style="page-break-before: always">` → `dispatchBlockRows` prepends `pageBreakRow` → `maroto.addRow()` type-asserts `core.PageBreaker.IsPageBreak()` → calls existing `fillPageToAddNew()` + `addHeader()` for the next page
- `blockContainer` (tall) → `maroto.addRow()` type-asserts `core.Splittable.SplitAt(remainingHeight)` → places `first` on current page + recursively `addRow()`s `rest` on next page → each slice's `Render()` paints its own bg/border with first/middle/last flag determining corner rounding

## Open Questions

- **Gradient DPI:** is 75 DPI sufficient for the demo's gradient hero? If users report blocky edges, raise to 150 DPI behind a `WithGradientDPI(n)` option.
- **Shadow blur N:** is 3 overlay rects enough? 5 looks better but doubles draw calls. Default 3, document the trade-off.

### Deferred Ideas

- **Explicit container height + `align-content`**: requires `blockContainer` to honour `ComputedStyle.Height` as a layout constraint (clamp child rendering, handle overflow, override `GetHeight()`). Once explicit height is supported, `align-content: flex-start | flex-end | center | space-between | space-around | space-evenly | stretch` becomes implementable as spacer rows between wrapped flex lines.
- True Gaussian box-shadow blur — requires per-pixel blur kernel applied to a rasterised mask. Heavier than current overlay approximation.
- Conic gradients (`conic-gradient`) — requires angular interpolation around a center; uses similar raster approach.
- `filter: drop-shadow(...)` — same as box-shadow but follows non-rectangular shape (requires shape mask).
- CSS Grid — distinct enough to warrant its own plan.
- `position: relative/absolute` — requires layered rendering and stacking contexts; large change.
- Form element rendering (`<input>`, `<button>`, `<select>`) — render as styled boxes.
