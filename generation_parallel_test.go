package paper_test

import (
	"bytes"
	"cmp"
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/reader"
)

func parallelTestRows(count int) []core.Row {
	rows := make([]core.Row, 0, count)
	for i := range count {
		rows = append(rows, text.NewRow(9, "row-"+itoa(i)+" lorem ipsum dolor sit amet consectetur", props.Text{Size: 9, Top: 1}))
	}
	return rows
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

func generateDocument(t *testing.T, cfg *entity.Config, rows []core.Row) *core.Pdf {
	t.Helper()

	m := paper.New(cfg)
	m.AddRows(rows...)
	doc, err := m.Generate(t.Context())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return doc
}

func generateWith(t *testing.T, cfg *entity.Config, rows []core.Row) []byte {
	t.Helper()

	return generateDocument(t, cfg, rows).GetBytes()
}

// generateInParallel generates rows with the parallel pages mode and fails if
// generation actually ran sequentially.
//
// Without this check a test can pass while proving nothing: an unspliceable
// feature makes parallel generation transparently re-render the document
// sequentially, so the output is correct either way and the assertion that was
// supposed to cover splicing never exercises it.
func generateInParallel(t *testing.T, cfg *entity.Config, rows []core.Row) []byte {
	t.Helper()

	doc := generateDocument(t, cfg, rows)
	requireParallelGeneration(t, doc)
	return doc.GetBytes()
}

// parallelWorkerCounts spreads a document across chunk boundaries in as many
// ways as is cheap to check, because a splice bug shows up at a boundary.
var parallelWorkerCounts = []int{1, 2, 3, 4, 8, 16}

func requireParallelGeneration(t *testing.T, doc *core.Pdf) {
	t.Helper()

	report := doc.GetReport()
	if report == nil {
		t.Fatal("no metrics report, so there is no way to tell which mode generated the document")
	}
	if report.GenerationMode != consts.GenerationParallelPages {
		t.Fatalf("document was generated in %q mode, want %q: it fell back instead of rendering in parallel",
			report.GenerationMode, consts.GenerationParallelPages)
	}
}

var showTextRe = regexp.MustCompile(`\((?:[^()\\]|\\.)*\)\s*Tj`)

// drawnText returns every string passed to a Tj show-text operator, in order.
// With compression disabled the page content streams are plain text in the
// document, so this is a proxy for "what actually got drawn, in what order".
func drawnText(pdfBytes []byte) []string {
	matches := showTextRe.FindAll(pdfBytes, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, string(match))
	}
	return out
}

func countPages(pdfBytes []byte) int {
	return bytes.Count(pdfBytes, []byte("/Type /Page\n")) + bytes.Count(pdfBytes, []byte("/Type /Page/"))
}

func generateHTMLDocument(t *testing.T, cfg *entity.Config, htmlStr string) *core.Pdf {
	t.Helper()

	doc, err := paper.FromHTML(t.Context(), htmlStr, cfg)
	if err != nil {
		t.Fatalf("generate from html: %v", err)
	}
	return doc
}

func generateHTML(t *testing.T, cfg *entity.Config, htmlStr string) []byte {
	t.Helper()

	return generateHTMLDocument(t, cfg, htmlStr).GetBytes()
}

var (
	externalLinkRe = regexp.MustCompile(
		`/Subtype /Link /Rect \[([-0-9. ]+)\] /Border \[0 0 0\] /A <</S /URI /URI \(((?:[^()\\]|\\.)*)\)>>`)
	internalLinkRe = regexp.MustCompile(
		`/Subtype /Link /Rect \[([-0-9. ]+)\] /Border \[0 0 0\] /Dest \[(\d+) 0 R /XYZ 0 ([-0-9.]+) null\]`)
	outlineEntryRe = regexp.MustCompile(
		`/Title \(((?:[^()\\]|\\.)*)\)\n/Parent \d+ 0 R\n(?:/(?:Prev|Next|First|Last) \d+ 0 R\n)*` +
			`/Dest \[(\d+) 0 R /XYZ 0 ([-0-9.]+) null\]`)
)

// externalLinkAnnotations returns every URL link annotation as
// "rectangle→target", in document order.
func externalLinkAnnotations(pdfBytes []byte) []string {
	matches := externalLinkRe.FindAllSubmatch(pdfBytes, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, string(m[1])+"→"+string(m[2]))
	}
	return out
}

// linkDestination is a clickable area that jumps somewhere inside the document.
// destObject is the page object it points at, which is what has to survive
// splicing; a document that failed to resolve it points at the page tree root.
type linkDestination struct {
	rect       string
	destObject int
	destY      string
}

func internalLinkAnnotations(pdfBytes []byte) []linkDestination {
	matches := internalLinkRe.FindAllSubmatch(pdfBytes, -1)
	out := make([]linkDestination, 0, len(matches))
	for _, m := range matches {
		object, err := strconv.Atoi(string(m[2]))
		if err != nil {
			continue
		}
		out = append(out, linkDestination{rect: string(m[1]), destObject: object, destY: string(m[3])})
	}
	return out
}

// outlineEntry is one bookmark: its title and the page object it jumps to.
type outlineEntry struct {
	title      string
	destObject int
	destY      string
}

func outlineEntries(pdfBytes []byte) []outlineEntry {
	matches := outlineEntryRe.FindAllSubmatch(pdfBytes, -1)
	out := make([]outlineEntry, 0, len(matches))
	for _, m := range matches {
		object, err := strconv.Atoi(string(m[2]))
		if err != nil {
			continue
		}
		out = append(out, outlineEntry{title: string(m[1]), destObject: object, destY: string(m[3])})
	}
	return out
}

// TestParallelPages_MatchesSequentialOutput is the correctness gate for
// parallel page rendering: the spliced document must draw the same text, in the
// same order, across the same number of pages as sequential generation, and
// must still parse as a valid PDF.
func TestParallelPages_MatchesSequentialOutput(t *testing.T) {
	t.Parallel()

	for _, rowCount := range []int{50, 200, 1000} {
		rows := parallelTestRows(rowCount)

		sequential := generateWith(t, config.NewBuilder().
			WithCompression(false).
			WithSequentialMode().
			Build(), rows)

		for _, workers := range []int{2, 4, 8} {
			parallel := generateWith(t, config.NewBuilder().
				WithCompression(false).
				WithParallelPagesMode(workers).
				Build(), rows)

			if _, err := reader.Parse(parallel); err != nil {
				t.Fatalf("rows=%d workers=%d: parallel output does not parse: %v", rowCount, workers, err)
			}

			seqPages, parPages := countPages(sequential), countPages(parallel)
			if seqPages != parPages {
				t.Errorf("rows=%d workers=%d: page count = %d, want %d", rowCount, workers, parPages, seqPages)
			}

			seqText, parText := drawnText(sequential), drawnText(parallel)
			if len(seqText) != len(parText) {
				t.Fatalf("rows=%d workers=%d: drew %d strings, want %d", rowCount, workers, len(parText), len(seqText))
			}
			for i := range seqText {
				if seqText[i] != parText[i] {
					t.Fatalf("rows=%d workers=%d: draw %d = %q, want %q", rowCount, workers, i, parText[i], seqText[i])
				}
			}
		}
	}
}

// drawOp is one text-showing operation together with the font state in effect
// when it runs.
type drawOp struct {
	fontID string
	size   string
	text   string
}

var (
	selectFontRe = regexp.MustCompile(`BT /F([0-9a-f]+) ([0-9.]+) Tf`)
	showTextOpRe = regexp.MustCompile(`\((?:[^()\\]|\\.)*\)\s*Tj`)
)

// documentDrawOps returns every drawn string paired with the font selected at
// that point. Comparing these sequences is stricter than comparing drawn text
// alone (it would catch text rendered at the wrong size or in the wrong font)
// while staying insensitive to redundant state operators, which legitimately
// differ between generation modes.
func documentDrawOps(pdfBytes []byte) []drawOp {
	type token struct {
		index          int
		isFontSelect   bool
		fontID, size   string
		shownTextValue string
	}

	var tokens []token
	for _, m := range selectFontRe.FindAllSubmatchIndex(pdfBytes, -1) {
		tokens = append(tokens, token{
			index:        m[0],
			isFontSelect: true,
			fontID:       string(pdfBytes[m[2]:m[3]]),
			size:         string(pdfBytes[m[4]:m[5]]),
		})
	}
	for _, m := range showTextOpRe.FindAllIndex(pdfBytes, -1) {
		tokens = append(tokens, token{index: m[0], shownTextValue: string(pdfBytes[m[0]:m[1]])})
	}
	slices.SortFunc(tokens, func(a, b token) int { return cmp.Compare(a.index, b.index) })

	var ops []drawOp
	var currentFont, currentSize string
	for _, tk := range tokens {
		if tk.isFontSelect {
			currentFont, currentSize = tk.fontID, tk.size
			continue
		}
		ops = append(ops, drawOp{fontID: currentFont, size: currentSize, text: tk.shownTextValue})
	}
	return ops
}

// TestParallelPages_RendersSameContentAsSequential is the correctness gate for
// splicing: every drawn string must appear in the same order, with the same font
// and size in effect, as it would from sequential generation.
//
// This is deliberately not a byte comparison. Splicing emits one fewer redundant
// font-selection operator at each chunk boundary, because a worker's fresh
// document starts from the configured default font rather than inheriting the
// previous page's state. The rendered result is the same; only the redundant
// state operators differ, so the invariant worth pinning is the font state in
// effect at each draw. TestParallelPages_SingleWorkerIsByteIdentical covers the
// one case where byte equality does hold.
func TestParallelPages_RendersSameContentAsSequential(t *testing.T) {
	t.Parallel()

	for _, rowCount := range []int{1, 50, 600} {
		rows := parallelTestRows(rowCount)

		sequential := generateWith(t, config.NewBuilder().
			WithCompression(false).
			WithDeterministic(true).
			WithSequentialMode().
			Build(), rows)
		wantOps := documentDrawOps(sequential)
		if len(wantOps) == 0 {
			t.Fatalf("rows=%d: sequential output drew nothing", rowCount)
		}

		for _, workers := range []int{1, 2, 3, 4, 8, 16} {
			parallel := generateWith(t, config.NewBuilder().
				WithCompression(false).
				WithDeterministic(true).
				WithParallelPagesMode(workers).
				Build(), rows)

			gotOps := documentDrawOps(parallel)
			if len(gotOps) != len(wantOps) {
				t.Errorf("rows=%d workers=%d: %d draw operations, want %d",
					rowCount, workers, len(gotOps), len(wantOps))
				continue
			}
			for i := range wantOps {
				if gotOps[i] != wantOps[i] {
					t.Errorf("rows=%d workers=%d: draw %d = %+v, want %+v",
						rowCount, workers, i, gotOps[i], wantOps[i])
					break
				}
			}
			if got, want := countPages(parallel), countPages(sequential); got != want {
				t.Errorf("rows=%d workers=%d: page count = %d, want %d", rowCount, workers, got, want)
			}
		}
	}
}

// TestParallelPages_SingleWorkerIsByteIdentical pins the one configuration where
// splicing must reproduce sequential output exactly: a single worker renders the
// whole document as one chunk, so there is no chunk boundary to differ at.
func TestParallelPages_SingleWorkerIsByteIdentical(t *testing.T) {
	t.Parallel()

	for _, rowCount := range []int{1, 50, 600} {
		rows := parallelTestRows(rowCount)

		sequential := generateWith(t, config.NewBuilder().
			WithDeterministic(true).WithSequentialMode().Build(), rows)
		parallel := generateWith(t, config.NewBuilder().
			WithDeterministic(true).WithParallelPagesMode(1).Build(), rows)

		if !bytes.Equal(sequential, parallel) {
			t.Errorf("rows=%d: single-worker output differs from sequential (%d vs %d bytes)",
				rowCount, len(parallel), len(sequential))
		}
	}
}

// TestParallelPages_FontsDeduplicated verifies the resource union collapses the
// same font registered independently by every worker into one embedded object,
// rather than one per worker.
func TestParallelPages_FontsDeduplicated(t *testing.T) {
	t.Parallel()

	rows := parallelTestRows(500)

	sequential := generateWith(t, config.NewBuilder().WithCompression(false).WithSequentialMode().Build(), rows)
	parallel := generateWith(t, config.NewBuilder().WithCompression(false).WithParallelPagesMode(8).Build(), rows)

	seqFonts := strings.Count(string(sequential), "/Type /Font")
	parFonts := strings.Count(string(parallel), "/Type /Font")
	if seqFonts != parFonts {
		t.Errorf("font objects = %d, want %d (workers must share one copy per font)", parFonts, seqFonts)
	}
}

// TestParallelPages_ReportsTheModeItUsed pins the signal every other test in
// this file relies on to tell a real parallel render from a silent fallback.
func TestParallelPages_ReportsTheModeItUsed(t *testing.T) {
	t.Parallel()

	// Each subtest builds its own rows: AddRows calls Row.SetConfig, so sharing
	// one slice across parallel subtests would be a write race on the fixture.
	t.Run("a spliceable document reports parallel pages", func(t *testing.T) {
		t.Parallel()
		rows := parallelTestRows(300)
		doc := generateDocument(t, config.NewBuilder().WithParallelPagesMode(4).Build(), rows)

		if got := doc.GetReport().GenerationMode; got != consts.GenerationParallelPages {
			t.Errorf("generation mode = %q, want %q", got, consts.GenerationParallelPages)
		}
	})

	// Tagged PDF is a document-catalog feature: it is written once for the whole
	// document, so chunked generation cannot reproduce it and the configured
	// mode is overridden.
	t.Run("a document-catalog feature reports the sequential fallback", func(t *testing.T) {
		t.Parallel()
		rows := parallelTestRows(300)
		cfg := config.NewBuilder().WithParallelPagesMode(4).Build()
		m := paper.New(cfg)
		m.SetTagged(true)
		m.AddRows(rows...)

		doc, err := m.Generate(t.Context())
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if got := doc.GetReport().GenerationMode; got != consts.GenerationSequential {
			t.Errorf("generation mode = %q, want %q", got, consts.GenerationSequential)
		}
	})
}

// TestParallelPages_RendersExternalLinksInParallel covers hyperlinked documents
// — invoices and reports with URLs — which used to abandon the parallel render
// entirely. External links carry a URL rather than a page index, so splicing
// only has to carry the annotations across.
func TestParallelPages_RendersExternalLinksInParallel(t *testing.T) {
	t.Parallel()

	target := "https://example.com/invoice"
	rows := make([]core.Row, 0, 300)
	for i := range 300 {
		rows = append(rows, text.NewRow(9, "row-"+itoa(i)+" see the terms online",
			props.Text{Size: 9, Top: 1, Hyperlink: &target}))
	}

	sequential := generateWith(t, config.NewBuilder().WithCompression(false).WithSequentialMode().Build(), rows)
	wantLinks := externalLinkAnnotations(sequential)
	if len(wantLinks) == 0 {
		t.Fatal("sequential output emitted no external link annotations")
	}

	for _, workers := range parallelWorkerCounts {
		parallel := generateInParallel(t, config.NewBuilder().
			WithCompression(false).WithParallelPagesMode(workers).Build(), rows)

		if _, err := reader.Parse(parallel); err != nil {
			t.Fatalf("workers=%d: parallel output does not parse: %v", workers, err)
		}
		gotLinks := externalLinkAnnotations(parallel)
		if len(gotLinks) != len(wantLinks) {
			t.Fatalf("workers=%d: emitted %d external link annotations, want %d",
				workers, len(gotLinks), len(wantLinks))
		}
		for i := range wantLinks {
			if gotLinks[i] != wantLinks[i] {
				t.Fatalf("workers=%d: external link %d = %q, want %q", workers, i, gotLinks[i], wantLinks[i])
			}
		}
	}
}

// TestParallelPages_RendersOutlinesInParallel covers the single most common
// reason documents used to fall back. Bookmarks store an absolute page number,
// which splicing shifts by the pages already in the document.
func TestParallelPages_RendersOutlinesInParallel(t *testing.T) {
	t.Parallel()

	rows := make([]core.Row, 0, 320)
	for chapter := range 8 {
		rows = append(rows, text.NewRow(12, "Chapter "+itoa(chapter), props.Text{
			Size:    12,
			Outline: &props.Outline{Level: 0, Title: "Chapter " + itoa(chapter)},
		}))
		rows = append(rows, parallelTestRows(40)...)
	}

	sequential := generateWith(t, config.NewBuilder().WithCompression(false).WithSequentialMode().Build(), rows)
	wantOutline := outlineEntries(sequential)
	if len(wantOutline) != 8 {
		t.Fatalf("sequential output has %d outline entries, want 8", len(wantOutline))
	}

	for _, workers := range parallelWorkerCounts {
		parallel := generateInParallel(t, config.NewBuilder().
			WithCompression(false).WithParallelPagesMode(workers).Build(), rows)

		if _, err := reader.Parse(parallel); err != nil {
			t.Fatalf("workers=%d: parallel output does not parse: %v", workers, err)
		}
		if !bytes.Contains(parallel, []byte("/Outlines")) {
			t.Fatalf("workers=%d: parallel output has no outline", workers)
		}
		if got, want := countPages(parallel), countPages(sequential); got != want {
			t.Errorf("workers=%d: page count = %d, want %d", workers, got, want)
		}

		gotOutline := outlineEntries(parallel)
		if len(gotOutline) != len(wantOutline) {
			t.Fatalf("workers=%d: parallel output has %d outline entries, want %d",
				workers, len(gotOutline), len(wantOutline))
		}
		for i := range wantOutline {
			if gotOutline[i] != wantOutline[i] {
				t.Errorf("workers=%d: outline entry %d = %v, want %v", workers, i, gotOutline[i], wantOutline[i])
			}
		}
	}
}

// TestParallelPages_ResolvesInternalLinksAcrossChunks is the case index
// rewriting exists for: the clickable area and the page it jumps to are
// rendered by different workers, so neither worker's document can resolve the
// link on its own.
func TestParallelPages_ResolvesInternalLinksAcrossChunks(t *testing.T) {
	t.Parallel()

	var page strings.Builder
	page.WriteString(`<p><a href="#the-end">jump to the end</a></p>`)
	for i := range 400 {
		page.WriteString("<p>filler paragraph " + itoa(i) + " lorem ipsum dolor sit amet consectetur</p>")
	}
	page.WriteString(`<h2 id="the-end">The End</h2>`)

	sequential := generateHTML(t, config.NewBuilder().WithCompression(false).WithSequentialMode().Build(), page.String())
	wantLinks := internalLinkAnnotations(sequential)
	if len(wantLinks) == 0 {
		t.Fatal("sequential output emitted no internal link annotation")
	}

	for _, workers := range parallelWorkerCounts {
		parallelDoc := generateHTMLDocument(t, config.NewBuilder().
			WithCompression(false).WithParallelPagesMode(workers).Build(), page.String())
		requireParallelGeneration(t, parallelDoc)
		parallel := parallelDoc.GetBytes()

		if _, err := reader.Parse(parallel); err != nil {
			t.Fatalf("workers=%d: parallel output does not parse: %v", workers, err)
		}
		if pages := countPages(parallel); pages < 4 {
			t.Fatalf("document spans %d pages, too few to place the target in another chunk", pages)
		}

		gotLinks := internalLinkAnnotations(parallel)
		if len(gotLinks) != len(wantLinks) {
			t.Fatalf("workers=%d: emitted %d internal link annotations, want %d",
				workers, len(gotLinks), len(wantLinks))
		}
		for i := range wantLinks {
			// An unresolved destination points at object 1, the page tree root,
			// which is what a naive splice produces when the target page was
			// rendered by a different worker.
			if gotLinks[i].destObject <= 1 {
				t.Errorf("workers=%d: internal link %d points at object %d, which is not a page: the target was not resolved",
					workers, i, gotLinks[i].destObject)
			}
			if gotLinks[i] != wantLinks[i] {
				t.Errorf("workers=%d: internal link %d = %+v, want %+v", workers, i, gotLinks[i], wantLinks[i])
			}
		}
	}
}

// TestParallelPages_RendersWatermarksInParallel covers blend modes, which are
// how watermarks draw. Their PDF name used to be their position in the graphics
// state list, so two workers would disagree on what /GS1 meant.
func TestParallelPages_RendersWatermarksInParallel(t *testing.T) {
	t.Parallel()

	const sentinel = "XWMKSENTINEL"
	rows := parallelTestRows(300)

	sequential := generateWith(t, config.NewBuilder().
		WithCompression(false).WithWatermark(sentinel).WithSequentialMode().Build(), rows)
	parallel := generateInParallel(t, config.NewBuilder().
		WithCompression(false).WithWatermark(sentinel).WithParallelPagesMode(4).Build(), rows)

	if _, err := reader.Parse(parallel); err != nil {
		t.Fatalf("parallel output does not parse: %v", err)
	}
	pages := countPages(parallel)
	if got, want := countPages(sequential), pages; got != want {
		t.Errorf("page count = %d, want %d", want, got)
	}
	if got := bytes.Count(parallel, []byte(sentinel)); got != pages {
		t.Errorf("watermark drawn %d times across %d pages", got, pages)
	}

	// Every page sets alpha to the same two values, so the whole document needs
	// exactly the graphics states sequential generation needs — not one set per
	// worker.
	gotStates := bytes.Count(parallel, []byte("/Type /ExtGState"))
	wantStates := bytes.Count(sequential, []byte("/Type /ExtGState"))
	if gotStates != wantStates {
		t.Errorf("emitted %d ExtGState objects, want %d (workers must share one copy per state)",
			gotStates, wantStates)
	}
	if gotStates == 0 {
		t.Fatal("no ExtGState objects at all, so the deduplication assertion proves nothing")
	}
}

// TestParallelPages_Cancellation verifies the worker pool honours context
// cancellation instead of running to completion.
func TestParallelPages_Cancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	m := paper.New(config.NewBuilder().WithParallelPagesMode(4).Build())
	m.AddRows(parallelTestRows(200)...)

	if _, err := m.Generate(ctx); err == nil {
		t.Fatal("expected an error when the context is already canceled")
	}
}
