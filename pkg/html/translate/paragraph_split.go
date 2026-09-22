package translate

import (
	"strings"
	"unicode"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// splittableParagraphRow places a paragraph's continuation on the next
// page when its measured lines exceed the available space.
type splittableParagraphRow struct {
	core.Row
	runs      []props.RichRun
	prop      props.RichText
	anchorReg *anchorRegistry
	config    *entity.Config
}

func newSplittableParagraphRow(runs []props.RichRun, prop props.RichText, anchors *anchorRegistry) *splittableParagraphRow {
	rt := richtext.New(runs, prop)
	if anchors != nil {
		rt.WithAnchorRegistry(anchors)
	}
	inner := row.New().Add(col.New().Add(rt))
	return &splittableParagraphRow{Row: inner, runs: props.CloneRichRuns(runs), prop: prop, anchorReg: anchors}
}

func (r *splittableParagraphRow) SetConfig(config *entity.Config) {
	r.config = config
	r.Row.SetConfig(config)
}

func (r *splittableParagraphRow) SplitAt(provider core.Provider, remainingHeight, width float64) (core.Row, core.Row, bool) {
	cell := &entity.Cell{Width: width}
	if r.GetHeight(provider, cell) <= remainingHeight {
		return nil, nil, false
	}
	measurer, ok := provider.(core.RichTextMeasurer)
	if !ok || remainingHeight <= 0 {
		return nil, r, true
	}

	type boundary struct{ run, offset int }
	var boundaries []boundary
	for runIndex, run := range r.runs {
		if run.Image != nil || run.ForceBreak {
			continue
		}
		for offset, character := range run.Text {
			if unicode.IsSpace(character) {
				first, rest := splitParagraphRuns(r.runs, runIndex, offset)
				if len(first) > 0 && len(rest) > 0 {
					boundaries = append(boundaries, boundary{runIndex, offset})
				}
			}
		}
	}
	if len(boundaries) == 0 {
		return nil, r, true
	}
	firstProp := r.prop
	firstProp.Bottom = 0
	fits := func(cut boundary) bool {
		firstRuns, _ := splitParagraphRuns(r.runs, cut.run, cut.offset)
		return measurer.MeasureRichText(firstRuns, cell, &firstProp) <= remainingHeight+0.001
	}
	low, high := 0, len(boundaries)
	for low < high {
		middle := (low + high) / 2
		if fits(boundaries[middle]) {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low == 0 {
		return nil, r, true
	}
	cut := boundaries[low-1]
	firstRuns, restRuns := splitParagraphRuns(r.runs, cut.run, cut.offset)
	if len(firstRuns) == 0 || len(restRuns) == 0 {
		return nil, r, true
	}
	restProp := r.prop
	restProp.Top = 0
	restProp.FirstLineIndent = 0
	restProp.Outline = nil
	first := newSplittableParagraphRow(firstRuns, firstProp, r.anchorReg)
	rest := newSplittableParagraphRow(restRuns, restProp, r.anchorReg)
	if r.config != nil {
		first.SetConfig(r.config)
		rest.SetConfig(r.config)
	}
	return first, rest, true
}

func splitParagraphRuns(runs []props.RichRun, runIndex, offset int) ([]props.RichRun, []props.RichRun) {
	first := props.CloneRichRuns(runs[:runIndex])
	rest := props.CloneRichRuns(runs[runIndex+1:])
	head, tail := runs[runIndex], runs[runIndex]
	head.Text = runesTrimRight(head.Text[:offset])
	tail.Text = strings.TrimLeftFunc(tail.Text[offset:], unicode.IsSpace)
	if head.Text != "" {
		first = append(first, head)
	}
	if tail.Text != "" {
		rest = append([]props.RichRun{tail}, rest...)
	}
	for len(first) > 0 {
		last := &first[len(first)-1]
		if last.Image != nil || last.ForceBreak {
			break
		}
		last.Text = runesTrimRight(last.Text)
		if last.Text != "" {
			break
		}
		first = first[:len(first)-1]
	}
	for len(rest) > 0 {
		start := &rest[0]
		if start.Image != nil || start.ForceBreak {
			break
		}
		start.Text = strings.TrimLeftFunc(start.Text, unicode.IsSpace)
		if start.Text != "" {
			break
		}
		rest = rest[1:]
	}
	return first, rest
}

func runesTrimRight(text string) string {
	return strings.TrimRightFunc(text, unicode.IsSpace)
}
