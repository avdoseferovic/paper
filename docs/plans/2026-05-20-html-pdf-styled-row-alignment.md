# HTML/PDF Styled Row Alignment Fix Plan

Created: 2026-05-20
Status: VERIFIED
Approved: Yes
Iterations: 0
Worktree: No
Type: Bugfix

## Summary

**Goal:** Make `core.PositionProvider.SetCursor` and `core.ShapeProvider.DrawFilledCircle` honour the maroto cell-coordinate convention (margin-relative) so styled row backgrounds, per-side borders, rounded-radius fills, and list-marker circles land at the same absolute page coordinates as the text content the providers also paint for those cells.

**Root Cause:** `internal/providers/gofpdf/provider.go` — the new `SetCursor` and `DrawFilledCircle` methods call `fpdf.SetXY(x, y)` / `fpdf.Circle(cell.X+w/2, cell.Y+h/2, r, "F")` **without** adding the page margins. Every other provider method (`Text.Add`, `Text.AddRichText`, `Image.addImageToPdf`, `Checkbox.Add`, etc.) reads `left, top, _, _ := fpdf.GetMargins()` and uses `cell.X + left` / `cell.Y + top` because `entity.Cell` coordinates are margin-relative (see `entity.NewRootCell` which initialises X=0,Y=0 for the content area). The new methods break that invariant. Result: `SetCursor(20, 30)` with `margins=(20, 15)` lands the pen at absolute `(20, 30)` — at the page content's intended top-left, but the cellwriter chain's `CellFormat`/`Line`/`DrawPath` then paint at `(20, 30)` while the text component renders at absolute `(40, 45)` via the margin-adding `AddRichText`. Twenty millimetres of horizontal offset, fifteen of vertical — exactly the misalignment visible in the demo.

**Bug Condition (C):** A component (Row.Render, Table.Render, blockContainer.Render, flexCellContent.Render, htmllist.Render) calls `provider.(core.PositionProvider).SetCursor(cell.X, cell.Y)` or `provider.(core.ShapeProvider).DrawFilledCircle(cell, fill)` with margin-relative cell coordinates AND the document has non-zero left/top margins (default A4 demo: 20/15).

**Postcondition (P):** After SetCursor, subsequent gofpdf calls that read the pen via `GetXY`/`GetX`/`GetY` (the cellwriter chain's `borderRadiusStyler`, `perSideBorderStyler`, `cellWriter.Apply → CellFormat`) draw the styled background at the **same absolute page coords** that the cell's inner content (text via `Text.Add`/`AddRichText`, images via `Image.addImageToPdf`, etc.) reaches. For DrawFilledCircle: the circle's centre lands at `(cell.X + cell.Width/2 + margins.Left, cell.Y + cell.Height/2 + margins.Top)` so it aligns with the marker text drawn over it by `AddText` (which adds margins).

**Symptom:** In the demo PDF (`test/output/html-demo.pdf`):
- `<h2 class="title-band">SUMMARY/PAYMENT INSTRUCTIONS/NOTES</h2>` → empty navy rounded bands at the page left edge with the heading text rendered 20mm to the right and a few mm below — never inside the band.
- `<tr style="background-color:#1a3e72">` thead and zebra-row backgrounds → navy/pink rectangles extending past the left page edge instead of aligning with the table column area.
- `<ol class="circle-numbers">` markers → vertical column of small navy dots clipped to the page margin instead of sitting next to the list items.
- `<img src="icon.svg">` followed by `<h1>` → the H1 INVOICE title appears to be missing (likely because earlier `SetCursor` mis-position cascaded forward and a downstream `CreateRow → Ln` advanced past the H1's intended location).

## Behavior Contract

### Must Change (C ⟹ P)

- **WHEN** a styled row, table cell, blockContainer, flexCellContent, or htmllist marker calls `SetCursor(cellX, cellY)` or `DrawFilledCircle(cell, fill)` **AND** the document has non-zero margins, **THEN** the resulting drawing primitive lands at absolute page `(cellX + margins.Left, cellY + margins.Top)` — the same convention every other provider method already uses.
- **Regression test 1:** `internal/providers/gofpdf/provider_test.go::TestProvider_SetCursor_AppliesPageMargins` — calls `provider.SetCursor(10, 20)` against a mock fpdf with `GetMargins().Return(5, 7, 0, 0)`, asserts the mock was called with `SetXY(15, 27)`. Fails on current code because SetCursor currently calls `SetXY(10, 20)`.
- **Regression test 2:** `internal/providers/gofpdf/provider_test.go::TestProvider_DrawFilledCircle_AppliesPageMargins` — calls `provider.DrawFilledCircle(&entity.Cell{X:10, Y:20, Width:6, Height:6}, fill)` with mock margins `(5, 7, 0, 0)`, asserts `Circle(18, 30, 3, "F")` was called. Fails on current code because cx/cy currently lack the margin offset.
- **Regression test 3 (integration):** `pkg/html/translate/translate_test.go::TestTranslate_StyledRowBgAlignsWithText` — parses a minimal HTML doc `<html><body><h2 class="title-band">X</h2></body></html>`, renders to PDF bytes via the existing `pkg/test` helpers OR via maroto's full pipeline, and asserts that the bg-fill draw command and the text draw command for the heading share the same Y coordinate (within 1mm). Fails on current code because they differ by margins.Top mm.

### Must NOT Change (¬C ⟹ unchanged)

The fix lives entirely inside `(*provider).SetCursor` and `(*provider).DrawFilledCircle`. These are NEW methods added by the visual-enhancements feature — no pre-existing caller uses them. All other existing rendering paths (plain paragraphs, headings without `.title-band`, tables without `<tr style>`, ordered lists with text markers, flex layouts without container backgrounds) call neither method and so are not on the path of this fix.

- **Preservation:** Existing test suite (`go test ./... -count=1`) covers preservation — no explicit additional preservation tests needed. The fix is additive (margin translation in two new methods).

## Scope

**Change:**
- `internal/providers/gofpdf/provider.go` — fix `SetCursor` and `DrawFilledCircle` to add margins.

**Test:**
- `internal/providers/gofpdf/provider_test.go` — add 2 unit tests (mock-based).
- `pkg/html/translate/translate_test.go` (or a new `pkg/html/translate/alignment_test.go`) — add 1 integration-style test that exercises a styled row through `Translate` + rendering to verify bg/text Y alignment via the structure tree OR via the actual rendered PDF bytes if structure tree is insufficient.

**Out of scope:**
- Pagination of long blockContainers (still a documented v1 limitation — Out of Scope in the visual-enhancements plan).
- Rounded outer corners on tables.
- Refactoring the cellwriter chain to be coordinate-agnostic (would be much larger; the margin-aware `SetCursor` is the minimum fix that restores correctness).

## Context for Implementer

- **Root cause file:** `internal/providers/gofpdf/provider.go`. Search for `func (g *provider) SetCursor` (added recently) and `func (g *provider) DrawFilledCircle` (also new).
- **Pattern to follow:** Every other method in `provider.go` and its siblings (e.g. `Text.Add` in `text.go`, `Image.addImageToPdf` in `image.go`, `Line.Add` in `line.go`) reads `left, top, _, _ := s.pdf.GetMargins()` (or via the wrapped fpdf in the provider) and adds them to cell coordinates. Mirror that. Concretely the fix is:
  ```go
  func (g *provider) SetCursor(x, y float64) {
      left, top, _, _ := g.fpdf.GetMargins()
      g.fpdf.SetXY(x+left, y+top)
  }
  func (g *provider) DrawFilledCircle(cell *entity.Cell, fill *props.Color) {
      // ... existing nil/zero guards, color setup ...
      left, top, _, _ := g.fpdf.GetMargins()
      cx := cell.X + cell.Width/2 + left
      cy := cell.Y + cell.Height/2 + top
      g.fpdf.Circle(cx, cy, radius, "F")
  }
  ```
- **Why the visual-enhancements plan's `SetCursor` calls aren't themselves wrong:** The callers (Row.Render, Table.Render, blockContainer.Render, flexCellContent.Render) all pass margin-relative cell coordinates — that IS the maroto convention. The bug is on the provider side, not the caller side. Reverting the caller-side SetCursor invocations would re-introduce the pre-fix symptoms (pen drift); the right move is to make the new provider methods honour the existing convention.
- **Test fixtures / mocks:**
  - The existing `mocks.NewFpdf(t)` (testify-mock) is used throughout `internal/providers/gofpdf/*_test.go`. Use it.
  - For the unit tests: `fpdf.EXPECT().GetMargins().Return(5.0, 7.0, 0.0, 0.0)` then `fpdf.EXPECT().SetXY(15.0, 27.0)` for the first test; analogous for DrawFilledCircle.
  - For the integration test: see existing `pkg/test` snapshot helpers — `core.Structure` doesn't carry coordinates so the assertion will probably need to render to a real PDF buffer via maroto.New(...).AddRows + Generate and use `pdfcpu` (already a dep — see `go.mod`) to extract the text+graphic content stream, then compare Y coordinates of the band rectangle and the heading text. If pdfcpu's API is too cumbersome for this test, fall back to a smoke test that just verifies `go run ./cmd/html-demo` produces a PDF whose file size is in the expected range AND that re-running it after the fix produces a DIFFERENT file size than before (proving rendering changed) — clear with the user during implementation if pdfcpu introspection is too invasive.
- **Verify after the fix:** `go run ./cmd/html-demo`, then open `test/output/html-demo.pdf`. Confirm visually that (a) the H1 INVOICE title appears, (b) `.title-band` headings show their heading text rendered in white **inside** the navy rounded band, (c) cards' backgrounds start at the left margin (not the page edge), (d) `<ol class="circle-numbers">` markers sit immediately to the left of each list item, and (e) the table thead navy row aligns with the column area (not extending past the left margin).

## Progress Tracking

- [x] Task 1: Reproduce & fix margin-relative coords in SetCursor/DrawFilledCircle
- [x] Task 2: Verify (full suite + demo visual check)

**Tasks:** 2 | **Done:** 2

## Implementation Tasks

### Task 1: Reproduce & fix margin-relative coords in SetCursor/DrawFilledCircle

**Objective:** Make `(*provider).SetCursor` and `(*provider).DrawFilledCircle` apply page-margin offsets so the new PositionProvider/ShapeProvider capabilities respect the same margin-relative cell-coordinate convention that every other provider method already follows.

**Files:**
- Test: `internal/providers/gofpdf/provider_test.go` (add 2 unit tests)
- Test: `pkg/html/translate/alignment_test.go` (new — 1 integration test)
- Modify: `internal/providers/gofpdf/provider.go` (fix 2 methods)

**TDD Flow:**
1. Write `TestProvider_SetCursor_AppliesPageMargins`. Use `mocks.NewFpdf(t)` with `EXPECT().GetMargins().Return(5.0, 7.0, 0.0, 0.0)`. Call `(&provider{fpdf: fpdf}).SetCursor(10, 20)`. Assert `EXPECT().SetXY(15.0, 27.0)`. Run — confirm FAIL because current code calls `SetXY(10, 20)`.
2. Write `TestProvider_DrawFilledCircle_AppliesPageMargins`. Use `mocks.NewFpdf(t)` with `EXPECT().GetMargins().Return(5.0, 7.0, 0.0, 0.0)` and `EXPECT().GetFillColor().Return(0,0,0)`, `EXPECT().SetFillColor(...)` (twice — once for set, once for restore). Call `DrawFilledCircle(&entity.Cell{X:10, Y:20, Width:6, Height:6}, &props.Color{Red:1,Green:2,Blue:3})`. Assert `Circle(18.0, 30.0, 3.0, "F")` was called (cx = 10+3+5 = 18, cy = 20+3+7 = 30). Run — confirm FAIL.
3. Write `pkg/html/translate/alignment_test.go::TestStyledRowBgAndTextShareY`. Build a `*Maroto` with default A4 + 20mm L margins, `AddHTML("<h2 class=\"title-band\">X</h2>")`, Generate, then either (a) parse the resulting PDF page content stream via `pdfcpu` to extract the bg rect Y coordinate and the text Y coordinate and assert they are within `lineHeight` of each other, OR (b) if the pdfcpu API is too invasive in this codebase, capture the underlying provider's calls via a thin spy wrapper and assert that `SetXY`/path Y and the text-positioning Y match. (Decision is allowed at implementation time — both are valid; if the spy version is faster to land, use it.) Run — confirm FAIL.
4. Apply the fix to `provider.go`:
   - In `SetCursor`: read `left, top, _, _ := g.fpdf.GetMargins()` and call `g.fpdf.SetXY(x+left, y+top)`.
   - In `DrawFilledCircle`: read margins inside the existing nil/zero guards and add `+ left` / `+ top` to cx, cy.
5. Run all three tests — confirm GREEN.
6. Run `go test ./... -count=1` — confirm no regressions in the existing suite (all pre-existing tests stay green; the visual-enhancements suite is now structurally consistent).

**Verify:** `go test ./... -count=1`

---

### Task 2: Verify (full suite + demo visual check)

**Objective:** Confirm the fix lands the demo PDF in a visibly correct state and that the full test suite passes.

**Commit:** `fix(html): apply page margins in SetCursor and DrawFilledCircle so styled row bgs and circle markers align with their inner content`

**Steps:**
1. `gofmt -w .`
2. `go vet ./...`
3. `go test ./... -count=1` — all green
4. `go run ./cmd/html-demo` — confirm exit 0 and `test/output/html-demo.pdf` regenerated.
5. Open `test/output/html-demo.pdf` and visually confirm:
   - Header "MAROTO INVOICE SYSTEM" + divider line at top.
   - SVG check-icon at top-centre (~14mm × 14mm).
   - `<h1>INVOICE #2026-0042</h1>` heading visible BELOW the icon.
   - `<p class="subtitle">Issued 19 May 2026 …</p>` BELOW the H1.
   - Three party cards (`.card-blue`, `.card-teal`, `.card-amber`) start at the LEFT MARGIN (~20mm from page edge), each with their full content visible inside their rounded background.
   - `<h2 class="title-band">SUMMARY</h2>` renders as a navy rounded band with **white "SUMMARY" text inside** (not above or below).
   - `<p>Thank you for your continued business …</p>` below.
   - Invoice table with `<thead>` navy bg under the column headings (ITEM, QUANTITY, UNIT PRICE, TOTAL) and zebra-striped `<tbody>` rows.
   - `<h2 class="title-band">PAYMENT INSTRUCTIONS</h2>` and `<h2 class="title-band">NOTES</h2>` side-by-side via flex, each with their heading text inside the band.
   - Under PAYMENT INSTRUCTIONS: ordered list with **navy circles around each numeral**, the circles sitting immediately to the left of each item's text (not in the page margin).
   - Under NOTES: bullet list.
   - Footer "Maroto Invoice System — confidential" and "Page 1 of N".

**Verify:** `go test ./... -count=1 && go run ./cmd/html-demo`

---

## Open Questions

- For the integration test (Task 1, step 3): is pdfcpu introspection of the rendered PDF acceptable, or should we use a thin spy wrapper around the provider? Decide at implementation time based on which is faster to land cleanly.

### Deferred Ideas

- Container backgrounds that split across page breaks (still v2).
- Rounded outer table corners (v2).
- A general regression test that renders the demo to PDF and snapshots the byte size (catches future positioning regressions automatically).
