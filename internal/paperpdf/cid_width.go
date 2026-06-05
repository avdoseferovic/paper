package paperpdf

import (
	"strconv"
	"strings"
)

type cidWidthRun struct {
	start    int
	widths   []int
	interval bool
}

type cidWidthRuns []cidWidthRun

func formatCIDWidthRuns(font *fontDefType, lastRune int) string {
	runs := mergeCIDWidthRuns(buildCIDWidthRuns(font, lastRune))

	var b fmtBuffer
	for _, run := range runs {
		if run.hasSingleWidth() {
			b.printf(" %d %d %d", run.start, run.end(), run.widths[0])
			continue
		}
		b.printf(" %d [ %s ]\n", run.start, joinCIDWidths(run.widths))
	}
	return b.String()
}

func buildCIDWidthRuns(font *fontDefType, lastRune int) cidWidthRuns {
	if font == nil || lastRune < 1 {
		return nil
	}

	runs := make(cidWidthRuns, 0)
	prevCID := -2
	prevWidth := -1
	interval := false

	for cid := 1; cid <= lastRune && cid < len(font.Cw); cid++ {
		width := font.Cw[cid]
		if width == 0 {
			continue
		}
		if width == 65535 {
			width = 0
		}
		if used, ok := font.usedRunes[cid]; cid > 255 && (!ok || used == 0) {
			continue
		}

		if cid == prevCID+1 && len(runs) > 0 {
			current := &runs[len(runs)-1]
			if width == prevWidth {
				if width == current.widths[0] {
					current.widths = append(current.widths, width)
				} else {
					current.widths = current.widths[:len(current.widths)-1]
					runs = append(runs, cidWidthRun{
						start:  prevCID,
						widths: []int{prevWidth, width},
					})
					current = &runs[len(runs)-1]
				}
				current.interval = true
				interval = true
			} else if interval {
				runs = append(runs, cidWidthRun{
					start:  cid,
					widths: []int{width},
				})
				interval = false
			} else {
				current.widths = append(current.widths, width)
			}
		} else {
			runs = append(runs, cidWidthRun{
				start:  cid,
				widths: []int{width},
			})
			interval = false
		}

		prevCID = cid
		prevWidth = width
	}

	return runs
}

func mergeCIDWidthRuns(runs cidWidthRuns) cidWidthRuns {
	merged := make(cidWidthRuns, 0, len(runs))
	nextStart := -1
	previousWasLongInterval := false

	for _, run := range runs {
		logicalLen := run.logicalLen()
		if run.start == nextStart && !previousWasLongInterval && (!run.interval || logicalLen < 4) {
			merged[len(merged)-1].widths = append(merged[len(merged)-1].widths, run.widths...)
		} else {
			merged = append(merged, run)
		}

		nextStart = run.start + logicalLen
		if run.interval {
			previousWasLongInterval = logicalLen > 3
			nextStart--
		} else {
			previousWasLongInterval = false
		}
	}

	return merged
}

func (run cidWidthRun) logicalLen() int {
	if run.interval {
		return len(run.widths) + 1
	}
	return len(run.widths)
}

func (run cidWidthRun) end() int {
	return run.start + len(run.widths) - 1
}

func (run cidWidthRun) hasSingleWidth() bool {
	for _, width := range run.widths[1:] {
		if width != run.widths[0] {
			return false
		}
	}
	return len(run.widths) > 0
}

func joinCIDWidths(widths []int) string {
	var b strings.Builder
	for i, width := range widths {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Itoa(width))
	}
	return b.String()
}
