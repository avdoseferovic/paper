package translate

import (
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type paragraphMeasureProvider struct{ *cursorProvider }

func (p *paragraphMeasureProvider) MeasureRichText(runs []props.RichRun, _ *entity.Cell, prop *props.RichText) float64 {
	var text strings.Builder
	for _, run := range runs {
		text.WriteString(run.Text)
	}
	words := len(strings.Fields(text.String()))
	lines := (words + 1) / 2
	return float64(lines) + prop.Top + prop.Bottom
}

func TestParagraphRowSplitsAtMeasuredLineBoundary(t *testing.T) {
	p := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	row := newSplittableParagraphRow([]props.RichRun{{Text: "one two three four five six"}}, props.RichText{}, nil)
	row.SetConfig(&entity.Config{MaxGridSize: 12})

	first, rest, split := row.SplitAt(p, 2, 100)
	require.True(t, split)
	require.NotNil(t, first)
	require.NotNil(t, rest)
	assert.InDelta(t, 2.0, first.GetHeight(p, &entity.Cell{Width: 100}), 0.01)
	assert.InDelta(t, 1.0, rest.GetHeight(p, &entity.Cell{Width: 100}), 0.01)
}

func TestNestedParagraphCanContinueOnNextPage(t *testing.T) {
	p := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	paragraph := newSplittableParagraphRow([]props.RichRun{{Text: "one two three four five six"}}, props.RichText{}, nil)
	parent := newSplittableContainerRow(&blockContainer{rows: []core.Row{paragraph}})
	parent.SetConfig(&entity.Config{MaxGridSize: 12})

	first, rest, split := parent.SplitAt(p, 2, 100)
	require.True(t, split)
	require.NotNil(t, first)
	require.NotNil(t, rest)
	assert.InDelta(t, 2.0, first.GetHeight(p, &entity.Cell{Width: 100}), 0.01)
	assert.InDelta(t, 1.0, rest.GetHeight(p, &entity.Cell{Width: 100}), 0.01)
}
