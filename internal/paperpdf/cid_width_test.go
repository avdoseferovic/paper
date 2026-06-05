package paperpdf

import (
	"strings"
	"testing"
)

func TestCIDWidthRunsFormatEqualWidthInterval(t *testing.T) {
	font := cidWidthTestFont(12, map[int]int{
		10: 500,
		11: 500,
		12: 500,
	}, nil)

	got := formatCIDWidthRuns(font, 12)
	want := " 10 12 500"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCIDWidthRunsFormatMixedWidthArray(t *testing.T) {
	font := cidWidthTestFont(12, map[int]int{
		10: 500,
		11: 520,
		12: 540,
	}, nil)

	got := formatCIDWidthRuns(font, 12)
	want := " 10 [ 500 520 540 ]\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCIDWidthRunsMergeAdjacentShortRuns(t *testing.T) {
	font := cidWidthTestFont(13, map[int]int{
		10: 500,
		11: 500,
		12: 520,
		13: 540,
	}, nil)

	got := formatCIDWidthRuns(font, 13)
	want := " 10 [ 500 500 520 540 ]\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCIDWidthRunsKeepLongIntervalSeparate(t *testing.T) {
	font := cidWidthTestFont(14, map[int]int{
		10: 500,
		11: 500,
		12: 500,
		13: 520,
		14: 540,
	}, nil)

	got := formatCIDWidthRuns(font, 14)
	want := " 10 12 500 13 [ 520 540 ]\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCIDWidthRunsFiltersHighCIDsByUsedRunes(t *testing.T) {
	font := cidWidthTestFont(258, map[int]int{
		250: 300,
		256: 400,
		257: 410,
		258: 420,
	}, map[int]int{257: 257, 258: 0})

	got := formatCIDWidthRuns(font, 258)
	want := " 250 250 300 257 257 410"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCIDWidthRunsMapsMissingWidthSentinelToZero(t *testing.T) {
	font := cidWidthTestFont(20, map[int]int{20: 65535}, nil)

	got := formatCIDWidthRuns(font, 20)
	want := " 20 20 0"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCIDWidthRunsSparseRanges(t *testing.T) {
	font := cidWidthTestFont(12, map[int]int{10: 500, 12: 520}, nil)

	got := formatCIDWidthRuns(font, 12)
	want := " 10 10 500 12 12 520"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCIDWidthRunsHonorsLastRuneBoundary(t *testing.T) {
	font := cidWidthTestFont(11, map[int]int{10: 500, 11: 520}, nil)

	got := formatCIDWidthRuns(font, 10)
	want := " 10 10 500"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestGenerateCIDFontMapWritesWidthObjectEntry(t *testing.T) {
	font := cidWidthTestFont(10, map[int]int{10: 500}, nil)
	f := &Fpdf{}

	f.generateCIDFontMap(font, 10)

	got := f.buffer.String()
	if !strings.Contains(got, "/W [ 10 10 500 ]") {
		t.Fatalf("expected /W entry in output, got %q", got)
	}
}

func cidWidthTestFont(lastRune int, widths map[int]int, usedRunes map[int]int) *fontDefType {
	cw := make([]int, lastRune+1)
	for cid, width := range widths {
		cw[cid] = width
	}
	if usedRunes == nil {
		usedRunes = map[int]int{}
	}
	return &fontDefType{
		Cw:        cw,
		usedRunes: usedRunes,
	}
}
