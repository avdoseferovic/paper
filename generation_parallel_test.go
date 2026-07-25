package paper_test

import (
	"bytes"
	"context"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
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

func generateWith(t *testing.T, cfg *entity.Config, rows []core.Row) []byte {
	t.Helper()

	m := paper.New(cfg)
	m.AddRows(rows...)
	doc, err := m.Generate(context.Background())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return doc.GetBytes()
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

// TestParallelPages_MatchesSequentialOutput is the correctness gate for
// parallel page rendering: the spliced document must draw the same text, in the
// same order, across the same number of pages as sequential generation, and
// must still parse as a valid PDF.
func TestParallelPages_MatchesSequentialOutput(t *testing.T) {
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
	sort.Slice(tokens, func(i, j int) bool { return tokens[i].index < tokens[j].index })

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
	rows := parallelTestRows(500)

	sequential := generateWith(t, config.NewBuilder().WithCompression(false).WithSequentialMode().Build(), rows)
	parallel := generateWith(t, config.NewBuilder().WithCompression(false).WithParallelPagesMode(8).Build(), rows)

	seqFonts := strings.Count(string(sequential), "/Type /Font")
	parFonts := strings.Count(string(parallel), "/Type /Font")
	if seqFonts != parFonts {
		t.Errorf("font objects = %d, want %d (workers must share one copy per font)", parFonts, seqFonts)
	}
}

// TestParallelPages_FallsBackForUnsupportedFeatures verifies that a document
// using a feature that cannot be spliced (here: outlines, which store absolute
// page indices) still generates correctly by falling back to sequential
// rendering, rather than failing.
func TestParallelPages_FallsBackForUnsupportedFeatures(t *testing.T) {
	rows := []core.Row{
		text.NewRow(12, "Chapter 1", props.Text{
			Size:    12,
			Outline: &props.Outline{Level: 1, Title: "Chapter 1"},
		}),
	}
	rows = append(rows, parallelTestRows(300)...)

	sequential := generateWith(t, config.NewBuilder().WithCompression(false).WithSequentialMode().Build(), rows)
	parallel := generateWith(t, config.NewBuilder().WithCompression(false).WithParallelPagesMode(4).Build(), rows)

	if _, err := reader.Parse(parallel); err != nil {
		t.Fatalf("fallback output does not parse: %v", err)
	}
	if got, want := countPages(parallel), countPages(sequential); got != want {
		t.Errorf("page count = %d, want %d", got, want)
	}
	if got, want := len(drawnText(parallel)), len(drawnText(sequential)); got != want {
		t.Errorf("drew %d strings, want %d", got, want)
	}
	if !bytes.Contains(parallel, []byte("/Outlines")) {
		t.Error("fallback output lost the outline")
	}
}

// TestParallelPages_Cancellation verifies the worker pool honours context
// cancellation instead of running to completion.
func TestParallelPages_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m := paper.New(config.NewBuilder().WithParallelPagesMode(4).Build())
	m.AddRows(parallelTestRows(200)...)

	if _, err := m.Generate(ctx); err == nil {
		t.Fatal("expected an error when the context is already canceled")
	}
}
