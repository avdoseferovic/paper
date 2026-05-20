# Extend HTML + CSS Support — Modern Engine & Styling Implementation Plan

Created: 2026-05-20
Status: PENDING
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

**Goal:** Extend maroto's HTML→PDF pipeline with modern CSS engine features and broader tag/styling coverage so common modern stylesheets work without manual rewriting. This plan covers the **engine + styling** half of the work; a follow-up plan ("Plan B" — see Deferred Ideas) covers layout/paint/paginator-heavy features.

**Architecture:** All work lives in the existing layered stack — `pkg/html/css/*` (parsing/computed), `pkg/html/translate/*` (DOM walk + CSS apply), `pkg/components/htmllist` (lists), and `internal/providers/gofpdf` (alpha + font registration). New features are additive — no breaking changes to public APIs. CSS variables are resolved as a pre-pass; `calc()` is parsed at value-evaluation time; `@font-face` registers with `pkg/fontrepository`; external `<link>` stylesheets reuse the same safe resolver pattern as `<img>`.

**Tech Stack:** Go 1.26 + `golang.org/x/net/html` + `github.com/andybalholm/cascadia` + `github.com/aymerick/douceur` (all already in `go.mod`); `github.com/phpdave11/gofpdf` for PDF output; no new external dependencies expected.

## Scope

### In Scope

- **Modern colors:** full CSS named-color set (~147 entries), 3/4/6/8-digit hex (incl. alpha), `rgb()`, `rgba()`, `hsl()`, `hsla()`. Alpha tracked on `RGBColor`.
- **Opacity & alpha pipeline:** `opacity` CSS property; alpha applied via gofpdf `SetAlpha`, wrapped so it cannot leak into native Maroto rendering. New `Alpha` field on `props.Color`.
- **Typography:** `letter-spacing` (via gofpdf `SetCharSpacing`), `text-transform` (none/uppercase/lowercase/capitalize), `text-indent`, `white-space` (normal/nowrap/pre/pre-wrap/pre-line) — wired into the DOM whitespace handler.
- **Inline tag coverage:** `<mark>` (yellow bg), `<small>` (0.85× font), `<abbr title>` (dotted underline + `title` tooltip), `<code>` outside `<pre>` (monospace + light bg), `<kbd>` (boxed), `<samp>`, `<var>` (italic), `<cite>` (italic), `<q>` (auto-quoted), `<time>`.
- **Block tag coverage:** `<dl>/<dt>/<dd>` (definition lists), `<details>/<summary>` (always-expanded styled block with bold summary), `<hr>` styled by `color`/`border-top-width`/`border-top-style`, `<caption>` on tables, `<colgroup>/<col>` providing per-column width hints to the table builder.
- **Selectors:** verify Cascadia's `:nth-child(n)`, `:first-child`, `:last-child`, `:nth-of-type`, `:not(...)`, `:first-of-type`, `:last-of-type`, attribute selectors (`[name=value]`, `[name^=…]`, `[name$=…]`, `[name*=…]`, `[name~=…]`) all flow through and add regression tests + a zebra-rows demo using `tr:nth-child(even)`.
- **Internal anchors:** `id=""` on any element registers a named PDF destination; `<a href="#id">` produces an internal PDF link via gofpdf `Link`.
- **CSS variables:** `--name: value` declarations on any element; `var(--name, fallback)` resolved in any property value. Cascade-scoped (descendants inherit). Resolved before shorthand expansion.
- **`calc()` expressions:** supports `+`, `-`, `*`, `/` with one level of parentheses across mixed units (mm/cm/pt/px/em/rem/%). `%` resolved against the parent content width when known, otherwise yields 0 and logs via `unsupportedHandler`.
- **External stylesheets:** `<link rel="stylesheet" href="…">` loaded via a new `WithStylesheetResolver(fn)` / `WithStylesheetBaseDir(dir)` option family that mirrors the existing image-resolver safety model. Default rejects everything (data: URIs allowed).
- **`@font-face`:** parse `@font-face { font-family; src: url(...) format("truetype"|"opentype") }`; resolve `url(...)` via the stylesheet resolver; register the resulting TTF/OTF bytes with Maroto's font repository so subsequent `font-family: "MyFont"` declarations resolve.
- **Demo + docs:** extend `cmd/html-demo/main.go` to exercise every new feature; update `docs/v2/html-support.md`.

### Out of Scope (deferred to Plan B)

- **Backgrounds & effects:** `linear-gradient`, `radial-gradient`, `box-shadow`, `text-shadow`, `outline`.
- **Flexbox completeness:** `flex-wrap` (multi-row), `order`, `align-self`, `align-content`, true `*-reverse`.
- **Page break controls:** `page-break-*`, `break-inside: avoid`, splitting `blockContainer` backgrounds across page breaks.
- **Inline image flow inside paragraphs** (text-wrap around inline `<img>`).
- **Other intentional non-goals:**
  - JavaScript, CSS Grid, `position`, `transform`, `float`, `@media`, `@keyframes`, `@import`.
  - `:hover` / `:focus` / `:active` and any state-dependent selectors (PDFs are static).
  - `::before` / `::after` pseudo-elements.
  - Form-element interactivity (`<input>`, `<button>`).
  - Pixel-diff regression tests for the demo PDF.

## Prerequisites

- Existing HTML→PDF pipeline and CSS-flex feature merged (already in `master`).
- Working tree clean; no in-flight changes in `pkg/html/**`, `pkg/components/**`, `pkg/props/**`, `internal/providers/gofpdf/**`, `pkg/fontrepository/**`.

## Context for Implementer

- **Patterns to follow:**
  - **CSS property addition:** add to `pkg/html/css/computed.go` `Apply()` switch; add shorthand expansion in `pkg/html/css/shorthand.go` `expandOne()` if needed; extend `pkg/html/translate/style.go` `blockCellStyle` / `applyInlineStyleToRuns` to thread the new field into `props.Cell` or `props.RichRun`.
  - **Resolver-with-safety pattern:** mirror `WithImageResolver` + `WithImageBaseDir` in `pkg/html/translate/image.go` (`baseDirResolver` at line ~91 enforces `filepath.Clean` + prefix check). New `WithStylesheetResolver`/`WithStylesheetBaseDir` reuses the exact same shape — refuses everything by default except `data:` URIs.
  - **Cascadia selector matching:** `pkg/html/translate/stylesheet.go:78 applyToNode` already iterates rules and matches via `cascadia.Sel.Match(n)`. Pseudo-class selectors require nothing new — they Just Work via Cascadia's matcher; the task is verification + tests + documentation.
  - **Inline tag handling:** extend `mutateContext` in `pkg/html/translate/inline.go:90` for tags that affect styling (mark/small/abbr/code/kbd/samp/var/cite/q/time). Tags requiring a wrapping background (mark, kbd, code) need a per-run background-color thread (currently `props.RichRun` has no background — see Risks).
  - **Block tag handling:** extend `pkg/html/translate/translate.go:115 blockRows` switch; for `<dl>` reuse the list builder pattern from `list.go`; for `<details>/<summary>` treat as a styled container reusing `blockContainer`.
  - **gofpdf provider extension:** add new optional capability interfaces in `pkg/core/` (e.g., `AlphaProvider`, `CharSpacingProvider`, `LinkProvider`) and implement in `internal/providers/gofpdf/provider.go`. Consumer code type-asserts with the safe form `if x, ok := provider.(core.X); ok { ... }`. Precedent: `core.RichTextProvider` and `core.ShapeProvider`.
  - **Font registration:** see `pkg/fontrepository/` for existing API. `@font-face` calls into that with the resolved bytes + a derived `consts/fontfamily.Type`.
- **Conventions:**
  - Tests use `testify` (`assert`/`require`). Snapshot-style tests compare `core.Structure` trees via `pkg/test`.
  - Production files stay ≤ 300 lines (500 hard limit). Split early — new files like `pkg/html/css/color.go`, `pkg/html/css/calc.go`, `pkg/html/css/vars.go`.
  - Conventional commits with optional scope: `feat(html): …`, `fix(html): …`.
- **Key files:**
  - `pkg/html/css/computed.go` — `ComputedStyle.Apply()` switch; needs new property cases.
  - `pkg/html/css/length.go` — `ParseLength`; needs `calc()` hook and per-feature unit additions.
  - `pkg/html/css/shorthand.go` — `ExpandShorthands`; needs CSS-variable pre-pass.
  - `pkg/html/css/computed.go` `parseColor` — needs functional notation + alpha + 147 named colors.
  - `pkg/html/translate/stylesheet.go` — `parseStylesheet`; needs `@font-face` and `<link>` handling.
  - `pkg/html/translate/style.go` — `blockCellStyle`, `applyInlineStyleToRuns`; needs alpha + char-spacing threading.
  - `pkg/html/translate/inline.go` — `mutateContext`, `appendTextRun`; needs new tag handling + text-transform.
  - `pkg/html/translate/translate.go` — `blockRows` switch; needs new block tag cases.
  - `pkg/html/translate/list.go` — pattern for `<dl>/<dt>/<dd>`.
  - `pkg/html/translate/table.go` — needs `<caption>` and `<colgroup>/<col>` integration.
  - `pkg/html/dom/dom.go` — `collapseWhitespace`; needs `white-space` opt-out hook.
  - `pkg/html/html.go` — public options; needs `WithStylesheetResolver`/`WithStylesheetBaseDir`.
  - `pkg/props/color.go` (or wherever `props.Color` lives) — add `Alpha float64`.
  - `pkg/props/text.go` / `pkg/props/cell.go` — needs `Alpha`/`CharSpacing` fields where relevant.
  - `pkg/core/` — new optional capability interfaces.
  - `internal/providers/gofpdf/provider.go` — implement them.
  - `pkg/fontrepository/` — `@font-face` registration target.
  - `cmd/html-demo/main.go` — demo of all features.
  - `docs/v2/html-support.md` — feature documentation.
- **Gotchas:**
  - **gofpdf `SetAlpha` is global state.** Must save/restore around any drawing that uses alpha. Wrap in a helper that always resets to 1.0 after the affected primitive call so Maroto-native rows don't render translucently.
  - **`props.RichRun` has no background-color field today.** Inline tags that need a background (mark, kbd, code) require either (a) adding `Background *props.Color` to `RichRun` and threading it through `richtext.Render`, or (b) emitting them as a separate inline component. Option (a) is the right move — touches `pkg/components/richtext` lightly. Acknowledge upfront so the inline-tags task plans for it.
  - **CSS variables scope.** Variables must inherit down the DOM. Build a `varScope` map keyed on element pointer with parent-pointer chain. Resolve `var(--x, fallback)` before shorthand expansion AND before `Apply()` — both inline `style=""` and rules from `<style>`.
  - **`calc()` percentages need a known basis.** Width-relative percentages (`calc(100% - 20mm)`) require the parent's content width. The translator already passes `contentWidthMM` through. Pass it into the length parser context. Heights are best-effort — document that `calc()` with `%` on height yields 0 outside flex columns.
  - **`@font-face` `src: url(local("foo"), local("bar"), url("file.ttf") format("truetype"))`** — only the `url()` references with TTF/OTF format are loaded; `local()` and `format("woff")` / `format("woff2")` are skipped (the gofpdf font system needs TTF/OTF only). Log skipped sources via `unsupportedHandler`.
  - **Pseudo-classes `:hover`, `:focus`, `:active`, `:visited`** match nothing in static PDF — Cascadia will silently never match them, which is correct. Document explicitly so users don't think rules are dropped.
  - **`<details>` open/closed:** treat all `<details>` as **always expanded**. Note: PDF has no toggle. The `<summary>` renders as bold above the body. Skip `open` attribute handling.
  - **`text-transform: capitalize`** is locale-dependent. Use `unicode.ToUpper` on the first rune of each word per `strings.Fields`. Accept that it's English-centric — document this.
  - **`@font-face` font naming:** `font-family: "Foo"` declarations must match exactly (case-insensitive) the family name registered. Maroto's `consts/fontfamily.Type` is a `string` — register the lower-cased name to keep matching predictable.
  - **External `<link>` stylesheets** must be loaded **before** inline `<style>` blocks in source order, mirroring browser behavior. Concatenate in DOM order via the doc walker that finds `<style>` today (`pkg/html/dom/dom.go:122 extractStyles`) — extend it to also walk `<link rel=stylesheet>` and call the resolver.
- **Domain context:** Maroto's HTML pipeline is a pure-Go subset — no browser, no JS, no external CSS engine. Every CSS property is a deliberate addition: it must be parsed (`pkg/html/css`), computed (`ComputedStyle`), and rendered (`pkg/html/translate` → `props.Cell`/`RichRun` → gofpdf cellwriter). The cascade is built-in CSS < user CSS < inline style. The renderer's grid is integer-quantized; layout always produces `core.Row`s.

## Runtime Environment

- **Build/run demo:** `go run ./cmd/html-demo` (writes `test/output/html-demo.pdf`)
- **Run all tests:** `go test ./... -count=1`
- **Run HTML pipeline tests only:** `go test ./pkg/html/... ./pkg/components/htmllist/... -count=1`
- **Lint:** `gofmt -w .`, `go vet ./...`, `golangci-lint run`
- **Visual verification:** open `test/output/html-demo.pdf` after each milestone

## Progress Tracking

**MANDATORY: Update this checklist as tasks complete. Change `[ ]` to `[x]`.**

- [x] Task 1: Modern color formats (147 named, hex variants, rgb/rgba/hsl/hsla)
- [x] Task 2: Opacity & alpha pipeline (CSS opacity + props.Color.Alpha + gofpdf SetAlpha)
- [x] Task 3: Typography properties (letter-spacing, text-transform, text-indent, white-space)
- [ ] Task 4: Expanded tag coverage (inline mark/small/abbr/code/kbd/samp/var/cite/q/time + block dl/dt/dd/details/summary/hr/caption/colgroup)
- [ ] Task 5: Pseudo-class & nth selector verification + tests
- [ ] Task 6: Internal PDF anchors (id + href="#id" via gofpdf Link)
- [ ] Task 7: CSS variables (--var declarations + var() resolution)
- [ ] Task 8: calc() length expressions
- [ ] Task 9: External stylesheets (<link rel=stylesheet> + resolver)
- [ ] Task 10: @font-face web font registration
- [ ] Task 11: Demo + docs sweep

**Total Tasks:** 11 | **Completed:** 3 | **Remaining:** 8

## Implementation Tasks

### Task 1: Modern color formats

**Objective:** Replace the current 16-entry named-color map and 3/6-digit hex parser with full CSS color support: ~147 named colors, 3/4/6/8-digit hex (4 and 8 digit include alpha), `rgb(r,g,b)`, `rgba(r,g,b,a)`, `hsl(h,s,l)`, `hsla(h,s,l,a)`. Track alpha on `RGBColor`.

**Dependencies:** None

**Files:**
- Create: `pkg/html/css/color.go` (move all color parsing here; keep `pkg/html/css/computed.go` lean)
- Create: `pkg/html/css/named_colors.go` (the 147-entry table)
- Modify: `pkg/html/css/computed.go` (remove old `parseColor`, `namedColors`; replace `RGBColor` with `{R,G,B int; A float64 /* 0–1, default 1 */}`)
- Test: `pkg/html/css/color_test.go`

**Key Decisions / Notes:**
- Add `RGBColor.A float64` with default 1.0. `NewComputedStyle()` and existing zero-value callers MUST keep producing A=1 (use a constructor `NewRGBColor(r,g,b)` returning A:1 to avoid forgetting).
- `rgb()`/`rgba()` accept integer 0–255 and percentage forms (`50%` → 127.5 → round to nearest). `hsl()` uses standard HSL→RGB conversion (`H` in degrees 0–360, `S/L` as 0–100 percentages). Reject malformed input by returning `nil` (current contract).
- 4-digit hex `#rgba` → `#rrggbbaa` expansion (same as 3 → 6).
- Named-color table sourced from CSS Color Module Level 4; verify alphabetical sort for grep-ability.

**Definition of Done:**
- [ ] `RGBColor` has `R, G, B int; A float64` with `A` defaulting to 1.0 on parse success
- [ ] `parseColor("rgb(255, 0, 0)")` returns `{255, 0, 0, 1.0}`
- [ ] `parseColor("rgba(0, 0, 0, 0.5)")` returns `{0, 0, 0, 0.5}`
- [ ] `parseColor("#ff000080")` returns `{255, 0, 0, ~0.5}`
- [ ] `parseColor("#f008")` (4-digit) returns `{255, 0, 0, ~0.53}`
- [ ] `parseColor("hsl(0, 100%, 50%)")` returns `{255, 0, 0, 1.0}` (red)
- [ ] `parseColor("hsla(120, 100%, 25%, 0.8)")` returns `{0, 128, 0, 0.8}` (dark green at 80%)
- [ ] All 147 CSS named colors resolve (spot-test ≥ 10 of them)
- [ ] Malformed input returns `nil` (no panic)
- [ ] All existing color-parsing callers in `pkg/html/translate/` continue to compile and behave identically when alpha = 1.0
- [ ] No file exceeds 300 lines (named_colors.go is exempt as a data file but stays in its own file)
- [ ] `go test ./pkg/html/css/... -count=1` passes

**Verify:**
- `go test ./pkg/html/css/... -count=1`
- `go test ./pkg/html/... -count=1` — ensures no caller regressed on the type change

---

### Task 2: Opacity & alpha pipeline

**Objective:** Wire color alpha through to PDF output via `props.Color.Alpha`, gofpdf `SetAlpha`, and a new `core.AlphaProvider` capability. Add the CSS `opacity` property (multiplies into every color's alpha for that element + descendants). Ensure alpha never leaks into Maroto-native rendering.

**Dependencies:** Task 1

**Commit:** `feat(html): modern colors, opacity, and alpha pipeline through gofpdf` (covers Tasks 1+2)

**Files:**
- Modify: `pkg/props/color.go` (add `Alpha *float64 // nil = opaque; 0.0 = transparent; 0.5 = 50%`) — **pointer type to avoid breaking 96 existing `props.Color{Red:…,Green:…,Blue:…}` literals across 29 files**. A nil `Alpha` is treated as opaque (1.0) in every render path; only a non-nil pointer activates the alpha pipeline.
- Modify: `pkg/html/css/computed.go` (add `Opacity float64; OpacitySet bool` to ComputedStyle, handle `opacity` in `Apply`)
- Modify: `pkg/html/translate/style.go` (`blockCellStyle`/`applyInlineStyleToRuns` set `Alpha = &finalAlpha` only when `color.A < 1` OR `style.Opacity < 1`; otherwise leave nil)
- Create: `pkg/core/alpha_provider.go` — `AlphaProvider` interface with `WithAlpha(a float64, fn func())` (encapsulates save/restore around `fn`)
- Modify: `internal/providers/gofpdf/provider.go` (implement `AlphaProvider`; uses `Fpdf.SetAlpha(a, "Normal")` and resets to 1.0 via `defer` so panics also reset)
- Modify: `internal/providers/gofpdf/cellwriter/fillcolorstyler.go`, `bordercolorstyler.go`, `borderradius.go`, and `pkg/components/richtext/richtext.go` — wrap fill/stroke/text calls in `WithAlpha` when `prop.Color.Alpha != nil && *prop.Color.Alpha < 1`. **Enumerate every render path that reads `props.Color`**: `fillcolorstyler`, `bordercolorstyler`, `borderradius` rounded-path renderer, `richtext` text rendering, `line.Render`. Each must have a unit test verifying that `Alpha == nil` produces the same gofpdf call sequence as before this change (byte-for-byte equivalence on the zero-value path).
- **No wrapper interface change needed:** `SetAlpha(alpha float64, blendModeStr string)` is **already present** in `internal/providers/gofpdf/gofpdfwrapper/fpdf.go` (line ~109) and the generated mock at `mocks/` already implements it. Verify the existing signature matches what `WithAlpha` calls; **do NOT regenerate the mock from scratch** (would clobber hand-edits in other mocked methods).
- Test: `pkg/props/color_test.go`, `pkg/html/translate/style_test.go`, `internal/providers/gofpdf/...`

**Key Decisions / Notes:**
- `Alpha == nil` (default for all 96 existing struct literals) bypasses the alpha pipeline entirely — gofpdf call sequences and resulting PDFs are byte-identical to pre-change output.
- A non-nil `Alpha` with value `1.0` is also a bypass (same as nil); only `0 ≤ *Alpha < 1` triggers `WithAlpha`.
- CSS `opacity: 0.5` on a `<div>` multiplies into every color's alpha for that element AND descendants. Store as `Opacity float64 (default 1)` on `ComputedStyle`; multiply it during `computeNodeStyle` finalisation.
- **Safety:** `WithAlpha` MUST restore to 1.0 even when `fn` panics — use `defer`. Failure to restore would tint subsequent native-Maroto rows.
- Text alpha: text color alpha flows into `richtext.Render` by wrapping the `AddText` call in `WithAlpha` when the text run's color has a non-nil Alpha < 1.
- **Render-path inventory (DoD verification):** before declaring Task 2 done, `grep -rn "props.Color" pkg internal | grep -v _test` must show that every consumer either (a) ignores `Alpha` entirely (safe), or (b) reads `Alpha` through the nil-aware helper. Document the list in a code comment on `props.Color.Alpha`.

**Definition of Done:**
- [ ] `props.Color` has `Alpha float64` with godoc note "0 = transparent, 1 = opaque, default 1"
- [ ] `core.AlphaProvider` interface declared with `WithAlpha(a float64, fn func())`
- [ ] gofpdf provider implements `AlphaProvider`; verified to call `SetAlpha(a, "Normal")` then `SetAlpha(1, "Normal")` in defer
- [ ] CSS `opacity: 0.5` on a `<div>` multiplies into the alpha of every child's color
- [ ] `<p style="color: rgba(255,0,0,0.5)">` renders translucent red text
- [ ] `<div style="background: rgba(0,0,0,0.2)">…</div>` renders translucent grey background
- [ ] Existing tests pass — alpha=1 path is byte-identical to pre-change output
- [ ] Unit test verifies `WithAlpha` resets to 1.0 even when `fn` panics
- [ ] `go test ./... -count=1` passes

**Verify:**
- `go test ./pkg/props/... ./pkg/html/... ./internal/providers/gofpdf/... -count=1`
- Visual: demo PDF (after Task 11) shows translucent overlays correctly

---

### Task 3: Typography properties

**Objective:** Add `letter-spacing`, `text-transform`, `text-indent`, and `white-space` CSS properties. `letter-spacing` threads to gofpdf `SetCharSpacing` via a new `CharSpacingProvider`. `text-transform` mutates extracted text. `text-indent` adds a leading indent to paragraphs. `white-space` controls DOM whitespace collapsing per-element.

**Dependencies:** None (parallel-safe with Task 2)

**Commit:** `feat(html): typography properties (letter-spacing, text-transform, text-indent, white-space)`

**Files:**
- Modify: `pkg/html/css/computed.go` (add `LetterSpacing float64; TextTransform string; TextIndent float64; WhiteSpace string` to `ComputedStyle`; handle in `Apply`)
- Modify: `pkg/html/translate/inline.go` (apply `TextTransform` to extracted text in `appendTextRun`; pass `WhiteSpace` into the DOM walker)
- Modify: `pkg/html/dom/dom.go` (extend `extractText` to honor `white-space: pre*` overrides — currently only the `<pre>`/`<code>` tag check exists)
- Modify: `pkg/html/translate/style.go` (thread `LetterSpacing` into `props.RichRun` via a new field; thread `TextIndent` into `props.RichText` as an additional `Left` offset on the first line)
- Modify: `pkg/props/richrun.go` (or wherever `RichRun` lives) — add `LetterSpacing float64 // mm; 0 = default`
- Create: `pkg/core/char_spacing_provider.go` — `CharSpacingProvider` with `WithCharSpacing(mm float64, fn func())`
- Modify: `internal/providers/gofpdf/provider.go` (implement `CharSpacingProvider` using `Fpdf.SetCharSpacing`)
- Modify: `pkg/components/richtext/richtext.go` — call `WithCharSpacing` around runs that have non-zero `LetterSpacing`
- Test: `pkg/html/css/computed_test.go`, `pkg/html/translate/inline_test.go`, `pkg/html/translate/style_test.go`, `pkg/components/richtext/richtext_test.go`

**Key Decisions / Notes:**
- `text-transform: capitalize` uses `unicode.ToUpper` on the first rune of each Unicode-aware whitespace-split word. Document English-centric limitation.
- `white-space: pre-line` collapses spaces but preserves newlines. `pre-wrap` preserves both and allows wrapping. `nowrap` collapses whitespace but disables wrapping (passed through as a hint to RichText, which currently always wraps — note as a limitation if the renderer can't honour it; the CSS parser still accepts and stores the value).
- `text-indent` applies only to the first line of a block. Implemented by inserting a leading padding-style left margin on the first run via the `Top`/`Left` field on `props.RichText` (best-effort; full first-line indent inside multi-line wrapping requires renderer-side support — accept the limitation and document it).
- `letter-spacing` is in mm; conversion from `0.5pt` etc. handled by `ParseLength`.

**Definition of Done:**
- [ ] CSS parser accepts `letter-spacing: 0.5pt`, `text-transform: uppercase`, `text-indent: 5mm`, `white-space: pre-wrap`
- [ ] `<p style="text-transform:uppercase">hello</p>` renders "HELLO"
- [ ] `<p style="text-transform:capitalize">hello world</p>` renders "Hello World"
- [ ] `<pre>` and `<code>` continue to preserve whitespace (no regression)
- [ ] `<span style="white-space:pre">  spaced  </span>` preserves internal spaces in the extracted run
- [ ] `RichRun.LetterSpacing` is set to the mm equivalent of the CSS value when `letter-spacing` is specified, and `CharSpacingProvider.WithCharSpacing` is invoked with that mm value during render (verified via unit test using a mock provider)
- [ ] `text-indent: 5mm` shifts the first line of a paragraph 5mm right (verified by asserting `props.RichText.Left` or equivalent first-line offset = 5mm on the structure)
- [ ] CSS parser accepts `white-space: nowrap` and stores the value on `ComputedStyle.WhiteSpace` without error; the renderer-always-wraps limitation is documented in `docs/v2/html-support.md` (Task 11)
- [ ] `props.RichRun.LetterSpacing` defaults to 0 (no regression on existing callers)
- [ ] `go test ./pkg/html/... ./pkg/components/richtext/... -count=1` passes

**Verify:**
- `go test ./pkg/html/... ./pkg/components/richtext/... -count=1`
- Visual: demo PDF (Task 11) has a "Typography" section showcasing each property

---

### Task 4: Expanded tag coverage

**Objective:** Add support for a broader set of HTML5 tags both inline and block. Inline: `<mark>`, `<small>`, `<abbr title>`, `<code>` outside `<pre>`, `<kbd>`, `<samp>`, `<var>`, `<cite>`, `<q>`, `<time>`. Block: `<dl>/<dt>/<dd>` (definition lists), `<details>/<summary>`, styled `<hr>`, `<caption>`, `<colgroup>/<col>` (per-column width hints).

**Dependencies:** Task 1 (for new built-in colors used by `<mark>`); Task 2 not strictly required but combined commit is cleaner

**Commit:** `feat(html): expanded inline and block tag coverage`

**Files:**
- Modify: `pkg/html/translate/inline.go` (**prerequisite redesign of `runContext`** — add `fontSize float64`, `fontFamily string`, `background *css.RGBColor`, `monospace bool` fields; extend `walkInline` to accept `parentStyle *css.ComputedStyle` so the inline walk knows the paragraph's computed font size; `appendTextRun` writes the context's fontSize / family / background into the emitted `RichRun`; `mutateContext` for `<small>` sets `next.fontSize = ctx.fontSize * 0.85` (falling back to parent's computed size when 0); for `<mark>` sets `next.background = &RGBColor{255,255,0,1}`; for `<code>`/`<kbd>`/`<samp>` sets `next.monospace = true` and a light-grey background; for `<var>`/`<cite>` sets `next.italic = true`)
- Modify: `pkg/html/translate/translate.go` — `paragraphRow` and `flexItemContent` pass the paragraph's computed style into `inlineRuns` so the inline walk has a starting fontSize/fontFamily
- Modify: `pkg/props/richrun.go` — add `Background *props.Color` (needed for mark/kbd/code-inline bg)
- Modify: `pkg/components/richtext/richtext.go` — render `Background` behind a run by drawing a filled rect before the text (use existing cell math)
- Modify: `pkg/html/translate/translate.go` (add `case "dl", "details", "hr", …` block branches; refactor switch into helper functions if file exceeds 280 lines)
- Create: `pkg/html/translate/definition_list.go` (`<dl>/<dt>/<dd>` builder; dt bold, dd indented)
- Create: `pkg/html/translate/details.go` (`<details>` renders as styled container; `<summary>` rendered as bold heading row)
- Modify: `pkg/html/translate/translate.go` — `hrRow` reads style from `ComputedStyle` (color, border-top-width as thickness, border-top-style as solid/dashed/dotted)
- Modify: `pkg/html/translate/table.go` — read `<caption>` and emit it as a centred row above the table; read `<colgroup>/<col>` `width` attribute or `style="width:…"` and pass column widths into `table.New` (if supported, otherwise document as a hint that biases column allocation)
- Modify: `pkg/html/translate/stylesheet.go` — add built-in styling for `mark` (yellow bg), `small` (font-size 0.85em), `dt` (bold), `dd` (margin-left:5mm), `q::before/::after` is not possible — use the inline auto-quote in inline.go instead
- Test: extend `pkg/html/translate/inline_test.go`, create `definition_list_test.go`, `details_test.go`, extend `table_test.go`

**Key Decisions / Notes:**
- `<abbr title="…">` — render the abbr as dotted-underline via `props.RichRun.Underline` (no dotted style support today; document as a limitation — use solid underline) and append the `title` as a footnote-style superscript OR just preserve the text. Simplest: solid underline + log via `unsupportedHandler` if title is set so users know it's not surfaced.
- `<q>Hello</q>` emits `"Hello"` with `“` and `”` (or `"` and `"` ASCII based on a config — keep ASCII for safety).
- `<details>` always renders expanded. `<summary>` is bold + slightly larger.
- `<colgroup>/<col>` — only `width="N%"` and `width="Nmm"` honored; converted to integer col counts via the same `mmPerCol` math as flex gaps.
- File-size budget: `translate.go` is already 229 lines; tag additions push it over 300. Extract the switch's larger branches into per-file builders.

**Definition of Done:**
- [ ] `<mark>highlighted</mark>` renders with a yellow background behind the run
- [ ] `<small>fine print</small>` renders at 85% of the parent font size
- [ ] `<code>x = 1</code>` outside `<pre>` renders in monospace with light grey background
- [ ] `<kbd>Ctrl+C</kbd>` renders boxed (background + thin border via a built-in CSS rule)
- [ ] `<q>quoted</q>` renders as `"quoted"` (ASCII quotes)
- [ ] `<abbr title="…">text</abbr>` renders with a solid underline (dotted-style limitation logged via `unsupportedHandler`)
- [ ] `<var>x</var>` and `<cite>title</cite>` render in italic style
- [ ] `<samp>output</samp>` renders in monospace font
- [ ] `<time datetime="…">2026</time>` renders its inner text without error and preserves the `datetime` attribute on the structure for downstream consumers
- [ ] `<dl><dt>Term</dt><dd>Definition</dd></dl>` renders with bold term and indented definition
- [ ] `<details><summary>Title</summary><p>Body</p></details>` renders Title bold above Body
- [ ] `<hr style="border-top: 2pt dashed #888">` renders a dashed grey horizontal line 2pt thick
- [ ] `<table><caption>Title</caption>…</table>` renders Title as a centred row above the table
- [ ] When the table builder accepts explicit column widths, parsed `<col width="20%">` + `<col width="80%">` produces a column allocation measurably closer to 20/80 than the default equal split (verified via `core.Structure` assertion on the table's column metadata or the rendered cell widths)
- [ ] When the table builder cannot accept explicit widths in v1, `unsupportedHandler` is called with a message identifying the `colgroup`/`col` feature so users get diagnostic feedback rather than silent degradation
- [ ] `props.RichRun.Background` defaults to nil (no regression)
- [ ] No production file exceeds 300 lines
- [ ] `go test ./pkg/html/... ./pkg/components/richtext/... -count=1` passes

**Verify:**
- `go test ./pkg/html/... ./pkg/components/richtext/... -count=1`
- Visual: demo PDF (Task 11) has a "Tag coverage" section using every new tag

---

### Task 5: Pseudo-class & nth selector verification

**Objective:** Verify that Cascadia's existing support for `:nth-child(n)`, `:first-child`, `:last-child`, `:nth-of-type`, `:first-of-type`, `:last-of-type`, `:not(...)`, and attribute selectors (`[name=value]`, `[name^=…]`, `[name$=…]`, `[name*=…]`, `[name~=…]`, `[name|=…]`) flows correctly through `parseStylesheet → applyToNode`. Add regression tests. Document static-only selectors (`:hover`, `:focus`, `:active`, `:visited` — silently never match).

**Dependencies:** None

**Commit:** `feat(html): pseudo-class selectors and nth-child support` (verified, tested, documented)

**Files:**
- Test: `pkg/html/translate/stylesheet_test.go` (add selector-focused subtests)
- Modify: `pkg/html/translate/stylesheet.go` (no logic change expected; if Cascadia silently rejects a state-dependent selector, ensure we don't crash — defensive parsing already in place)
- Modify: `docs/v2/html-support.md` — section on selector support

**Key Decisions / Notes:**
- Concretely test:
  - `tr:nth-child(even) { background-color: #f0f0f0 }` produces zebra rows
  - `li:first-child { font-weight: bold }` bolds the first item only
  - `p:not(.intro) { color: grey }` skips the intro paragraph
  - `[data-status="ok"] { color: green }` matches attribute equality
  - `a[href^="https://"] { color: blue }` matches prefix
- State-dependent pseudo-classes (`:hover`, `:focus`, `:active`, `:visited`) — verify they silently match nothing without erroring.

**Definition of Done:**
- [ ] Subtests verify ≥ 6 selector forms produce the expected `ComputedStyle` on the matched node and NOT on non-matched nodes
- [ ] Zebra-rows test: a 5-row table with `tr:nth-child(even) { background-color: red }` produces 2 rows with red background, 3 without
- [ ] State-dependent selectors (`:hover`) do not error and do not match
- [ ] `docs/v2/html-support.md` adds a "Selectors" section listing supported pseudo-classes + attribute selectors + the static-only caveat for `:hover`/`:focus`/`:active`/`:visited`
- [ ] `go test ./pkg/html/translate/... -count=1` passes

**Verify:**
- `go test ./pkg/html/translate/... -count=1 -run TestStylesheet`

---

### Task 6: Internal PDF anchors

**Objective:** Support `id=""` on any element as a named PDF destination, and `<a href="#id">` as an internal PDF link. Uses gofpdf `Link` + the existing `Hyperlink` flow on `RichRun` for href-style links; adds a new path for anchor target registration.

**Dependencies:** None (parallel-safe)

**Commit:** `feat(html): internal anchors via id and href="#id"`

**Files:**
- Modify: `pkg/html/translate/inline.go` (in `mutateContext` for `<a>`, when `href` starts with `#`, emit a marker run that includes the local-anchor target rather than a URL)
- Modify: `pkg/components/richtext/richtext.go` — when `RichRun.LocalAnchor != ""`, call gofpdf's `Link(x,y,w,h, internalLink)` with the resolved page+y from `SetLink/AddLink`
- Modify: `pkg/props/richrun.go` — add `LocalAnchor string // anchor name; takes precedence over Hyperlink when set`
- Modify: `pkg/html/translate/translate.go` — register `id` attributes on block elements as PDF destinations via a new `LinkProvider.AddLink(name)` capability
- Create: `pkg/core/link_provider.go` — `LinkProvider` with `AddLink() int` (returns a link ID), `SetLink(id, x, y, w, h, page)`, `InternalLink(cell, id)` (or similar minimal API)
- Modify: `internal/providers/gofpdf/provider.go` — implement via `Fpdf.AddLink()`/`SetLink()`/`Link()`
- Modify: `pkg/html/translate/translate.go` — first pass walks the DOM and pre-registers `id` → link-id; second pass (existing walker) wires links at render time
- Test: `pkg/html/translate/anchor_test.go` (assert that `<a href="#foo">click</a>` followed by `<h2 id="foo">` produces the right structure)

**Key Decisions / Notes:**
- **Two-pass approach (committed, not optional):** add `collectAnchorIDs(body *dom.Node) map[string]int` that walks the DOM once before `blockRows`, reserves a PDF link ID via `LinkProvider.AddLink()` for every element with an `id` attribute, and stores the id→linkID map on the `translator` struct. Pass 2 (existing `blockRows` walk) then resolves `<a href="#id">` by looking up the linkID. Forward references (link before target) work because both passes complete before render. Lazy registration is REJECTED: forward references would break because `SetLink(linkID, x, y, page)` requires the target's coordinates, which aren't known until the target renders.
- **Note: `AddLink`, `SetLink`, and `Link` already exist on the gofpdf wrapper interface** (`internal/providers/gofpdf/gofpdfwrapper/fpdf.go` lines ~19, ~142) and on the generated mock. Do NOT regenerate the mock; verify signatures match what `LinkProvider` calls.
- `<a href="https://…">` continues to work as a normal external link (existing path).

**Definition of Done:**
- [ ] `core.LinkProvider` interface declared
- [ ] gofpdf provider implements it via `Fpdf.AddLink`/`SetLink`/`Link`
- [ ] `<h2 id="section1">Title</h2>…<a href="#section1">jump</a>` produces a working internal link in the PDF
- [ ] External `<a href="https://…">` still works (regression)
- [ ] Unit test asserts `RichRun.LocalAnchor` is populated when `href` starts with `#`
- [ ] Unit test asserts that when the HTML contains `<h2 id="s1">Title</h2>` followed by `<a href="#s1">jump</a>`, the resulting `core.Structure` contains BOTH (a) a node registering a PDF destination for "s1" (via the `LinkProvider` capability call recorded by a mock provider) AND (b) a `RichRun` with `LocalAnchor = "s1"` resolved to the same registered link ID
- [ ] Unit test using a mock `LinkProvider` asserts `AddLink` and `SetLink` are invoked once per `id="…"` and `Link` is invoked once per `<a href="#…">` with the matching link ID
- [ ] Forward-reference test: HTML `<a href="#later">jump</a>…<h2 id="later">Target</h2>` (link appears BEFORE target in source) produces a working internal link — the `<a>`'s linkID matches the `<h2>`'s registered ID, confirming the two-pass design correctly handles forward references
- [ ] `go test ./pkg/html/... -count=1` passes

**Verify:**
- `go test ./pkg/html/... -count=1`
- Visual (best-effort, not gating): open the demo PDF in a viewer; clicking an internal link jumps to the right page

---

### Task 7: CSS variables

**Objective:** Support `--name: value` custom property declarations and `var(--name, fallback)` resolution in any CSS value. Variables inherit through the DOM via the existing cascade.

**Dependencies:** None (parallel-safe with most; combined with Task 8 in one commit)

**Commit:** `feat(html): CSS variables and calc() expressions` (covers Tasks 7+8)

**Files:**
- Create: `pkg/html/css/vars.go` — `VarScope` map + `ResolveVars(value string, scope *VarScope) string` (handles `var(--x)` and `var(--x, fallback)`)
- Modify: `pkg/html/translate/style.go` — `computeNodeStyle` builds a `VarScope` chained to the parent's scope, stores `--*` declarations from rules + inline, then calls `ResolveVars` on every value before `Apply`
- Modify: `pkg/html/translate/stylesheet.go` — `applyToNode` accepts the scope and resolves values before applying
- Test: `pkg/html/css/vars_test.go`, `pkg/html/translate/vars_test.go`

**Key Decisions / Notes:**
- Resolution is text-level substitution — performed before shorthand expansion so `border: var(--accent) solid 2pt` becomes `border: red solid 2pt` then expands correctly.
- Nested `var(...)` references resolve via a **visited set** (`map[string]bool`) — when a variable name is already in the set during its own resolution, that branch returns its fallback (or empty if none) and logs via `unsupportedHandler`. A depth counter is NOT used because it silently truncates legitimate deep chains (e.g., `--size-base → --size-sm → --size-md → --size-lg → --size-xl → --size-xxl`). Legitimate depth is unbounded; only true cycles are rejected. As a safety belt, also cap absolute call depth at 32 to bound stack use.
- Scope inheritance: parent's `--x` is visible in child unless child redeclares. `:root { --x: red }` works because `:root` matches `<html>`.
- Multi-arg `var(--x, fallback with spaces)` — the fallback can contain commas if we use last-comma split; keep simple by splitting on first comma.

**Definition of Done:**
- [ ] `:root { --accent: #ff0000 } p { color: var(--accent) }` produces red text in `<p>`
- [ ] `<div style="--bg: blue"><p style="background-color: var(--bg)">…</p></div>` produces blue background on the `<p>`
- [ ] `var(--missing, green)` resolves to `green`
- [ ] `var(--missing)` (no fallback) resolves to empty string; `ParseLength`/`parseColor` treat as `nil`/0 (no panic)
- [ ] Cycle detection: `--a: var(--b); --b: var(--a)` logs and resolves to empty
- [ ] Child redeclaration shadows parent
- [ ] `go test ./pkg/html/css/... ./pkg/html/translate/... -count=1` passes

**Verify:**
- `go test ./pkg/html/css/... ./pkg/html/translate/... -count=1 -run TestVars`

---

### Task 8: calc() length expressions

**Objective:** Support `calc()` in any length value: `width: calc(100% - 20mm)`, `padding: calc(2mm + 1em)`, `font-size: calc(10pt * 1.2)`. Supports `+`, `-`, `*`, `/` with optional parentheses (one level deep). Mixed units resolve via existing `ParseLength` per token; `%` resolved against a context width (best-effort).

**Dependencies:** None functionally, but bundled with Task 7 commit

**Commit:** _(combined with Task 7 above)_

**Files:**
- Create: `pkg/html/css/calc.go` — small shunting-yard or recursive-descent evaluator
- Modify: `pkg/html/css/length.go` — `ParseLength` detects `calc(…)` prefix and dispatches to the evaluator
- Modify: `pkg/html/css/computed.go` — new signature `ParseLengthCtx(value string, parentFontSize, contextWidthMM float64) float64` and helper that wraps it; existing `ParseLength(value, parentFontSize)` defers with `contextWidthMM = 0`
- Modify: `pkg/html/translate/style.go` — `computeNodeStyle` passes the known content width through (from `tr.contentWidthMM`)
- Test: `pkg/html/css/calc_test.go`

**Key Decisions / Notes:**
- Grammar (informal): `expr = term (('+'|'-') term)*` ; `term = factor (('*'|'/') factor)*` ; `factor = NUMBER UNIT? | '(' expr ')'`. **Whitespace around `+`/`-` is lenient** (browsers accept `calc(100%-20mm)` even though the CSS spec disallows it). Tokenizer disambiguates unary minus from binary minus by tracking whether the previous token was a value (binary) or an operator/open-paren/start (unary).
- Mixed-unit arithmetic: convert each operand to mm via `ParseLength`, then arithmetic in mm, then return mm.
- Division by zero returns 0 (log via `unsupportedHandler`).
- `%` resolved against `contextWidthMM` if > 0, else 0 (logged).
- Nesting: support one level of `()` to keep complexity bounded. `calc(a + (b * c))` works; `calc(a + (b * (c - d)))` returns 0 + log.

**Definition of Done:**
- [ ] `calc(10mm + 5mm)` → 15.0 mm
- [ ] `calc(2cm - 5mm)` → 15.0 mm
- [ ] `calc(10pt * 1.5)` → ~5.29 mm
- [ ] `calc(100% - 20mm)` at contextWidth=170mm → 150.0 mm
- [ ] `calc(100% / 4)` at contextWidth=160mm → 40.0 mm
- [ ] `calc((10mm + 2mm) * 2)` → 24.0 mm (one level parens)
- [ ] Malformed `calc(...)` returns 0 and logs via `unsupportedHandler`
- [ ] `go test ./pkg/html/css/... -count=1` passes

**Verify:**
- `go test ./pkg/html/css/... -count=1 -run TestCalc`

---

### Task 9: External stylesheets via <link>

**Objective:** Load `<link rel="stylesheet" href="…">` content into the cascade. Add `WithStylesheetResolver(fn)` / `WithStylesheetBaseDir(dir)` options mirroring the image-resolver safety model. Default refuses non-`data:` URIs.

**Dependencies:** None (parallel-safe with most)

**Commit:** `feat(html): external <link rel=stylesheet> support with safe resolver`

**Files:**
- Modify: `pkg/html/dom/dom.go` — **keep `StyleText() string` unchanged for backward compatibility** (returns only inline `<style>` text). Add a NEW method `StyleSources() (inlineCSS string, linkHrefs []string)` that walks the DOM ONCE and returns both the inline CSS text and the ordered list of `<link rel="stylesheet">` href values. Internally, `StyleText()` can delegate to `StyleSources()` and return just the first value. This avoids breaking any external caller of `StyleText()`.
- Modify: `pkg/html/translate/translate.go` — call `doc.StyleSources()` instead of `doc.StyleText()`; for each href, invoke the resolver and accumulate the fetched text in DOM order; then concatenate **fetched stylesheets BEFORE inline `<style>` text** in source order (mirror browser behavior). Wrap each resolver call and each `parser.Parse` call on external content in `defer recover()` so a malformed external sheet logs+skips rather than crashing the caller.
- Modify: `pkg/html/html.go` — add `WithStylesheetResolver(fn func(href string) ([]byte, error))` and `WithStylesheetBaseDir(dir string)` options; thread through `translate.Translate`
- Modify: `pkg/html/translate/translate.go` — accept the resolver as a translator option
- Create: `pkg/html/translate/stylesheet_resolver.go` — `safeDefaultStylesheetResolver` (only data: URIs), `stylesheetBaseDirResolver(dir)` with the same `filepath.Clean` + prefix-check pattern as images
- Test: `pkg/html/translate/stylesheet_resolver_test.go` (positive/safety tests)
- Modify: `pkg/html/dom/dom_test.go` (assert `<link>` href extraction)

**Key Decisions / Notes:**
- Failure modes: missing resolver → log + skip the link. Resolver error → log + skip. Resolver panic → `defer recover()` → log + skip. Empty body → no-op. `parser.Parse` error or panic on the fetched content → log + skip (the external sheet is dropped from the cascade, inline `<style>` rules still apply). Never panic out of `Translate`. Add an "error-level" log marker (vs. debug) so users can distinguish "intentionally not supported" from "load failed" in the `unsupportedHandler` callback (pass distinct `kind` strings like `link.skipped` vs `link.error`).
- `media="print"` is not honoured — all stylesheets apply (PDF == print).
- Test fixture: a small `.css` file in `internal/fixture/`.

**Definition of Done:**
- [ ] `WithStylesheetResolver` and `WithStylesheetBaseDir` exported on the `html` package
- [ ] Default resolver REFUSES `<link href="file:///etc/passwd">` and `<link href="../../secret.css">`
- [ ] `WithStylesheetBaseDir("./assets")` accepts `<link href="theme.css">` resolving to `./assets/theme.css`; refuses `<link href="../escape.css">`
- [ ] `data:text/css,...` URIs decode and apply
- [ ] Linked stylesheet rules apply BEFORE inline `<style>` rules (verified by overriding the same selector)
- [ ] Unsupported resolver path falls back to ignoring the link (logged, no error)
- [ ] `go test ./pkg/html/... -count=1` passes

**Verify:**
- `go test ./pkg/html/translate/... -count=1 -run TestStylesheetResolver`

---

### Task 10: @font-face web font registration

**Objective:** Parse `@font-face { font-family: "Foo"; src: url("./foo.ttf") format("truetype") }` from any reachable stylesheet (built-in, `<style>`, or `<link>`); resolve `url(...)` via the stylesheet resolver; register the resulting TTF/OTF bytes with Maroto's font repository so subsequent `font-family: "Foo"` declarations resolve.

**Dependencies:** Task 9 (uses the same resolver)

**Commit:** `feat(html): @font-face web font registration via font repository`

**Files:**
- **Sub-task 10.0 (pre-implementation spike, ≤ 30 min):** write a throwaway test that parses a sample `@font-face { font-family: "Foo"; src: url("./x.ttf") format("truetype") }` through douceur `parser.Parse` and prints `rule.Kind`, `rule.Name`, and each `declaration.Value`. Confirm what shape the `src:` value takes (likely a single raw string like `url("./x.ttf") format("truetype")`). Pin the expected shape as a comment in `fontface.go`. Reason: douceur's AST for `@font-face` is **not verified** in the plan; we will not blindly assume the structure.
- Modify: `pkg/html/translate/stylesheet.go` — **CRITICAL: change the existing `rule.Kind != 0` filter at line ~49** so that AtRules with `Name == "font-face"` are NOT skipped. Current code unconditionally drops every AtRule (`@font-face`, `@media`, `@keyframes`, `@import`). Replace with: `if rule == nil { continue }; if rule.Kind != 0 && rule.Name != "font-face" { continue }`. Then dispatch `@font-face` rules to `processFontFace`.
- Create: `pkg/html/translate/fontface.go` — `processFontFace(rule *parser.Rule, resolver StylesheetResolver, repo fontrepository.Repository) error` extracts family + src, calls resolver, registers
- Modify: `pkg/html/translate/translate.go` — add a new translator field `fontRepository fontrepository.Repository` and option `WithFontRepository(repo)`. The translator passes `tr.fontRepository` to `processFontFace`. If the field is nil, `@font-face` rules are silently skipped (with `unsupportedHandler` log).
- Modify: `pkg/html/html.go` — add public `WithFontRepository(repo fontrepository.Repository) Option` mirroring the translator option.
- Modify: `pkg/fontrepository/` — verify the existing API. Per repo inspection, the concrete type exposes registration; if there is no method to add raw TTF/OTF bytes with a chosen family name, add `AddBytes(family string, style fontstyle.Type, bytes []byte) error` and wire it into the gofpdf provider's font cache via `Fpdf.AddUTF8FontFromBytes`.
- **Late-registration capability:** if gofpdf's font loading happens during `m.Generate()` BEFORE HTML translation runs (common in Maroto today), the font registered by `@font-face` will never reach `Fpdf.AddUTF8FontFromBytes`. To handle this, define `core.LateFontProvider` with `RegisterFont(family string, style fontstyle.Type, bytes []byte) error` implemented by the gofpdf provider as a direct call to `Fpdf.AddUTF8FontFromBytes` at registration time (no caching needed since the PDF is built incrementally). `processFontFace` calls this provider AT TRANSLATION TIME so the font is available when subsequent rows render.
- Test: `pkg/html/translate/fontface_test.go` (fixture TTF in `internal/fixture/`) — includes a regression test that built-in fonts (Helvetica, Times) still resolve after registering a custom font, ensuring `@font-face` does not poison the font cache.

**Key Decisions / Notes:**
- **AtRule filter change is a prerequisite — without it Task 10 silently does nothing.** Highlighted in DoD.
- **fontrepository must be threaded through the translator via `WithFontRepository`** — without this option the `@font-face` rule parses but cannot register, again silently dropping the font. Highlighted in DoD.
- Only `format("truetype")` and `format("opentype")` URLs are loaded; `format("woff")`/`format("woff2")` skipped (gofpdf doesn't support these without an additional decompressor).
- `src: local("Foo"), url("foo.ttf") format("truetype")` — `local()` is skipped; first valid `url() format()` is loaded.
- `font-weight: bold` and `font-style: italic` inside `@font-face` produce per-style registration. Default: register as Normal style.
- Failure modes: missing src, resolver refusal, malformed TTF, parser panic — all caught (defer recover around `processFontFace`), logged, never propagate. The font lookup then falls back to default.
- Fixture: ship a tiny CC0 / OFL-licensed TTF (≤ 100 KB), kept under `internal/fixture/` with a `LICENSE-fontname.txt` companion file.

**Definition of Done:**
- [ ] Sub-task 10.0 spike completed: douceur AST shape for `@font-face` `src:` verified and documented as a comment in `fontface.go`
- [ ] `parseStylesheet`'s AtRule filter updated to let `font-face` AtRules through (other AtRules like `@media`/`@keyframes` still dropped)
- [ ] Unit test asserts that `@font-face` rules SURVIVE the parseStylesheet filtering step (positive test of the filter change)
- [ ] `WithFontRepository` option exported on the `html` package and threaded through `translate.Translate` into the translator struct
- [ ] `core.LateFontProvider` interface declared; gofpdf provider implements it via direct `Fpdf.AddUTF8FontFromBytes`
- [ ] `@font-face { font-family: "Foo"; src: url("./fixture.ttf") format("truetype") }` followed by `font-family: "Foo"` on a paragraph renders that paragraph in the registered font (end-to-end test with the fixture)
- [ ] When `WithFontRepository` is NOT supplied, `@font-face` rules log via `unsupportedHandler` and are skipped (no panic, no error to caller)
- [ ] Unsupported `format()` values (woff/woff2) are skipped with a specific log
- [ ] `local("...")` entries are skipped; first valid `url() format()` is loaded
- [ ] Missing fixture file logs and falls back; no panic
- [ ] Malformed TTF logs and falls back; no panic
- [ ] Regression test: after registering a custom font via `@font-face`, built-in fonts (Helvetica, Times) still resolve correctly
- [ ] Unit test: stylesheet → fixture TTF → font family registered → `LateFontProvider.RegisterFont` called (mocked)
- [ ] `go test ./pkg/html/translate/... ./pkg/fontrepository/... -count=1` passes

**Verify:**
- `go test ./pkg/html/translate/... ./pkg/fontrepository/... -count=1`
- Visual: demo PDF (Task 11) loads a fixture font via `@font-face` and renders text in it

---

### Task 11: Demo + docs sweep

**Objective:** Extend `cmd/html-demo/main.go` to exercise every new feature added in Tasks 1-10, and update `docs/v2/html-support.md` to document each.

**Dependencies:** Tasks 1-10

**Commit:** `feat(demo): showcase modern CSS extensions + docs sweep`

**Files:**
- Modify: `cmd/html-demo/main.go` (add sections for: modern colors, opacity, typography, new tags, pseudo-classes, anchors, CSS vars, calc, external CSS, @font-face)
- Add: `cmd/html-demo/assets/extra.css` (an external stylesheet using `:root` vars, calc, and a pseudo-class rule)
- Add: `cmd/html-demo/assets/demo-font.ttf` (a small TTF for `@font-face` — pick a CC0/public-domain font)
- Modify: `docs/v2/html-support.md` (new sections for each feature family; updated "Supported CSS properties" and "Supported HTML tags" lists; new "Selectors" section; updated "Options" section listing all new `With*` options)
- Modify: `README.md` if it has an HTML feature summary (skim and update only if needed)

**Key Decisions / Notes:**
- Use a CC0-licensed TTF (e.g., a subset of a public-domain font) under 100 KB to keep the repo lean.
- Demo sections should be visually distinct so reviewers can confirm each feature renders correctly.

**Definition of Done:**
- [ ] Demo PDF includes a visually-identifiable section for each of Tasks 1-10
- [ ] `cmd/html-demo/assets/extra.css` loads via `<link>` and contributes rules visible in the output
- [ ] `@font-face` demo paragraph renders in the fixture font
- [ ] `docs/v2/html-support.md` documents every new property/tag/option
- [ ] `go run ./cmd/html-demo` exits 0 and produces a PDF > 10KB
- [ ] `go test ./... -count=1` passes
- [ ] `gofmt -w .`, `go vet ./...`, `golangci-lint run` all clean

**Verify:**
- `go test ./... -count=1`
- `go run ./cmd/html-demo` and visually inspect `test/output/html-demo.pdf`

---

## Testing Strategy

- **Unit tests:** every new CSS parser branch (colors, calc, vars), every new tag handler, resolver safety paths
- **Integration tests:** end-to-end `HTML string → core.Structure` assertions in `pkg/html/translate/*_test.go`
- **Selector tests:** new file `stylesheet_test.go` subtests covering ≥ 6 pseudo-class/attribute selector forms
- **Safety tests:** every new resolver (stylesheet, font) has positive + negative path-traversal tests
- **No PDF pixel-diff** — visual checks are manual

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
| - | - | - | - |
| `props.Color.Alpha` default of 0 breaks existing rendering (zero value = transparent) | ~~High~~ Eliminated | High | **Resolved by design:** `Alpha` is `*float64` (pointer). 96 existing `props.Color{Red:…,Green:…,Blue:…}` literals across 29 files get `Alpha == nil`, which is treated as opaque in EVERY render path (fillcolorstyler, bordercolorstyler, borderradius, richtext, line). The byte-for-byte equivalence test on the nil path enforces the contract. |
| gofpdf `SetAlpha` global-state leak into Maroto-native rows | Medium | High | All alpha use goes through `core.AlphaProvider.WithAlpha(a, fn)` which `defer`s a reset to 1.0. Direct `SetAlpha` calls forbidden outside the provider. Lint rule: `grep -r "SetAlpha" --include="*.go" pkg internal` should show only the provider file. |
| `RGBColor` type change (adding `A float64` with default 0) silently corrupts downstream | ~~Medium~~ Low | High | `RGBColor.A` initialized via `NewRGBColor(r,g,b)` returning `A: 1.0`. `parseColor` ALWAYS sets `A` (default 1 on success). Existing callers that construct `RGBColor{R,G,B}` literally are searched and updated in Task 1's audit step. |
| `@font-face` rules silently dropped by existing parseStylesheet AtRule filter | High | High | **Resolved:** Task 10 explicitly changes the `rule.Kind != 0` guard to admit `font-face` AtRules. Positive unit test verifies @font-face survives filtering. |
| `fontrepository` not threaded into translator (no path to register fonts) | High | High | **Resolved:** Task 10 adds `WithFontRepository` option AND a new `core.LateFontProvider` capability that the gofpdf provider implements via direct `Fpdf.AddUTF8FontFromBytes`. Registration happens at translation time, not at `Generate()` time. |
| `dom.Document.StyleText()` API change breaks external callers (Task 9) | Medium | Medium | **Resolved:** keep `StyleText()` signature unchanged; add new `StyleSources()` method returning both inline text and link hrefs. Internal callers migrate to `StyleSources()`. |
| `runContext` missing fontSize/family/background fields (Task 4 cannot wire `<small>`/`<mark>`/`<code>`) | High | High | **Resolved:** Task 4 explicitly redesigns `runContext` to add `fontSize`, `fontFamily`, `background`, `monospace` fields, and threads parent computed style into `walkInline`. |
| Two-pass anchor walk has no infrastructure (forward references break with lazy registration) | Medium | High | **Resolved:** Task 6 commits to a two-pass design with `collectAnchorIDs` pre-pass + explicit forward-reference unit test. |
| CSS variable depth-5 limit silently truncates legitimate deep chains | Medium | Medium | **Resolved:** Task 7 uses a visited set (not depth counter); only true cycles are rejected. Bonus 32-call stack safety belt for runaway recursion. |
| `<mark>`/`<kbd>`/`<code>` need `RichRun.Background` (new field) | High | Medium | Plan adds the field in Task 4 with default nil = no background. `richtext.Render` checks for nil before drawing a fill rect. No regression for existing callers. |
| CSS variable resolution depth/cycle attacks | Low | Low | Cap recursion at 5 levels; log overflow via `unsupportedHandler`; return empty value. |
| `calc()` expression evaluator complexity creep | Medium | Low | Cap at one level of parens. Reject anything more complex with a log + fallback to 0. Document the limitation. |
| External stylesheet path traversal | Low | High | Default resolver refuses non-`data:` URIs. Base-dir resolver reuses the same `filepath.Clean` + prefix-check tested in `image.go`. New file `stylesheet_resolver.go` shares helpers — DRY through a `safeResolver` helper. |
| `@font-face` registration breaks existing font lookup (Maroto's font system caches per family name) | Medium | High | Register under the exact case-folded family name; verify the existing font cache doesn't reject duplicates. Add a regression test that builtins (Helvetica, Times) still resolve after registering a custom font. |
| Cascadia silently accepts `:hover` selectors that never match — users may not realise rules are dropped | Low | Low | Document explicitly in "Selectors" section. Optionally: emit a one-time log via `unsupportedHandler` when a `:hover/:focus/:active/:visited` selector is encountered. |
| 11-task plan is too big to land in one milestone | Medium | Medium | Commits are grouped: (1+2), (3), (4), (5), (6), (7+8), (9), (10), (11) — 9 commits. Each is independently buildable + tested + verifiable. The verify phase can run per-commit if needed. |
| File-size budget exceeded (300 lines) | High | Low | Plan splits new code across many files: `color.go`, `named_colors.go`, `vars.go`, `calc.go`, `stylesheet_resolver.go`, `fontface.go`, `definition_list.go`, `details.go`. Existing files extracted as needed (Task 4 may extract switch helpers from `translate.go`). |
| Demo PDF size growth from TTF fixture | Low | Low | Pick a TTF ≤ 100 KB. Demo PDF size growth is bounded by the font itself + a few sample paragraphs. |

## Goal Verification

> The spec-reviewer-goal agent verifies these criteria during verification.

### Truths (what must be TRUE for the goal to be achieved)

- A user can write `color: rgba(0,0,0,0.5)` and the PDF renders translucent black text
- A user can write `opacity: 0.3` on a container and all descendants render at 30% opacity
- A user can write `letter-spacing: 0.5pt` and the rendered text has wider character spacing
- A user can write `text-transform: uppercase` and lowercase HTML text appears as uppercase in the PDF
- A user can use `<mark>`, `<kbd>`, `<code>` (outside `<pre>`), `<small>`, `<abbr title>`, `<details>/<summary>`, `<dl>/<dt>/<dd>`, `<hr>` with style, and `<caption>` and they render with semantically appropriate visual styling
- A user can write `tr:nth-child(even) { background-color: … }` and zebra rows render correctly
- A user can write `<a href="#section">` and clicking it in the PDF jumps to the element with `id="section"`
- A user can declare `--accent: #ff0000` and reference it via `var(--accent)` in any color property
- A user can write `width: calc(100% - 20mm)` and the value resolves against the known content width
- A user can write `<link rel="stylesheet" href="theme.css">` and (with a configured resolver) the rules apply
- A user can write `@font-face` and reference the registered font via `font-family: "MyFont"`
- The default behavior is byte-identical to pre-change output for HTML that didn't use any new features (no regression)

### Artifacts (what must EXIST to support those truths)

- `pkg/html/css/color.go` + `named_colors.go` — full color parser
- `pkg/html/css/vars.go` — variable resolver
- `pkg/html/css/calc.go` — calc evaluator
- `pkg/html/translate/stylesheet_resolver.go` — safe stylesheet resolver
- `pkg/html/translate/fontface.go` — `@font-face` handler
- `pkg/html/translate/definition_list.go` — `<dl>/<dt>/<dd>` builder
- `pkg/html/translate/details.go` — `<details>/<summary>` builder
- `pkg/core/alpha_provider.go`, `char_spacing_provider.go`, `link_provider.go` — capability interfaces
- `internal/providers/gofpdf/provider.go` — implementations of the three new providers
- `pkg/props/color.go` updated with `Alpha`
- `pkg/props/richrun.go` updated with `Background`, `LetterSpacing`, `LocalAnchor`
- `cmd/html-demo/main.go` + `assets/extra.css` + `assets/demo-font.ttf` — demo
- `docs/v2/html-support.md` — updated docs

### Key Links (critical connections that must be WIRED)

- `parseColor` → returns `*RGBColor` with `A` populated → `blockCellStyle` → `props.Color{Alpha:…}` → cellwriter detects `Alpha < 1` → `AlphaProvider.WithAlpha`
- `computeNodeStyle` → builds parent-chained `VarScope` → `ResolveVars` on every declaration value → `ExpandShorthands` → `Apply`
- `ParseLength` → detects `calc(…)` → `evalCalc(value, parentFontSize, contextWidthMM)` → returns mm
- `pkg/html/dom/dom.go` `extractStyles` → also collects `<link rel=stylesheet>` href list → `translate.Translate` → resolver → concat into stylesheet text BEFORE inline `<style>`
- `parseStylesheet` → finds `@font-face` rule → `processFontFace` → resolver → `fontrepository.Register` → gofpdf font cache
- `<a href="#…">` → inline.go sets `RichRun.LocalAnchor` → `richtext.Render` calls `LinkProvider.Link(cell, anchorID)`
- `<mark>`/`<kbd>` → `mutateContext` sets `runContext.background` → `appendTextRun` writes `RichRun.Background` → `richtext.Render` paints background rect before text

## Open Questions

- **Font fixture license:** which TTF to ship in `cmd/html-demo/assets/`? Pick a CC0 / OFL / public-domain font ≤ 100 KB during Task 11. Suggestion: a subset of [Inter](https://github.com/rsms/inter) or [JetBrains Mono](https://github.com/JetBrains/JetBrainsMono) — both are OFL-licensed.
- **`color()` and `color-mix()` Level 4 functions:** out of scope here; could be added later via the `color.go` parser.

### Deferred Ideas (Plan B candidates)

A follow-up plan should cover the layout/paint-heavy features the user also selected:

- **Backgrounds & effects:** `linear-gradient`, `radial-gradient` (require gofpdf gradient API or rasterised fallback), `box-shadow` (offset + blur emulation), `text-shadow`, `outline` (separate from border, drawn outside the cell box).
- **Flexbox completeness:** `flex-wrap` (multi-row emission — biggest architectural change since the current quantizer assumes a single row), `order` (sort items before quantization), `align-self` (per-item cross-axis), `align-content` (cross-axis distribution with wrap), true `*-reverse` ordering.
- **Page break controls:** `page-break-before/after: always|avoid`, `break-inside: avoid` (hint to the paginator to keep a container together), splitting `blockContainer` backgrounds across page breaks (the largest item — requires Render to know the page boundary and re-emit the background on the next page).
- **Inline image flow inside paragraphs:** text-wrap around `<img>` inside `<p>` (currently falls back to alt text).

These all involve renderer/paginator changes rather than parser/cascade changes, hence the split.
