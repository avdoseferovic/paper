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

// splittableParagraphRow places a plain paragraph's continuation on the next
// page when its measured lines exceed the available space.
type splittableParagraphRow struct {
	core.Row
	run       props.RichRun
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
	return &splittableParagraphRow{Row: inner, run: runs[0], prop: prop, anchorReg: anchors}
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

	text := r.run.Text
	var boundaries []int
	for offset, character := range text {
		if offset > 0 && unicode.IsSpace(character) {
			boundaries = append(boundaries, offset)
		}
	}
	if len(boundaries) == 0 {
		return nil, r, true
	}
	firstProp := r.prop
	firstProp.Bottom = 0
	firstRun := r.run
	fits := func(offset int) bool {
		firstRun.Text = strings.TrimRightFunc(text[:offset], unicode.IsSpace)
		return measurer.MeasureRichText([]props.RichRun{firstRun}, cell, &firstProp) <= remainingHeight+0.001
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
	firstRun.Text = strings.TrimRightFunc(text[:cut], unicode.IsSpace)
	restRun := r.run
	restRun.Text = strings.TrimLeftFunc(text[cut:], unicode.IsSpace)
	if firstRun.Text == "" || restRun.Text == "" {
		return nil, r, true
	}
	restProp := r.prop
	restProp.Top = 0
	restProp.FirstLineIndent = 0
	restProp.Outline = nil
	first := newSplittableParagraphRow([]props.RichRun{firstRun}, firstProp, r.anchorReg)
	rest := newSplittableParagraphRow([]props.RichRun{restRun}, restProp, r.anchorReg)
	if r.config != nil {
		first.SetConfig(r.config)
		rest.SetConfig(r.config)
	}
	return first, rest, true
}
