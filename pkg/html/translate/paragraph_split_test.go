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

func TestStyledParagraphPreservesRunsAcrossSplit(t *testing.T) {
	p := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	paragraph := newSplittableParagraphRow([]props.RichRun{
		{Text: "one two three ", Size: 12},
		{Text: "four five six", Size: 10},
	}, props.RichText{}, nil)
	paragraph.SetConfig(&entity.Config{MaxGridSize: 12})

	first, rest, split := paragraph.SplitAt(p, 2, 100)
	require.True(t, split)
	require.NotNil(t, first)
	require.NotNil(t, rest)
	assert.Equal(t, []string{"one two three four"}, richTextValues([]core.Row{first}))
	assert.Equal(t, []string{"five six"}, richTextValues([]core.Row{rest}))
	firstRuns := first.(*splittableParagraphRow).runs
	restRuns := rest.(*splittableParagraphRow).runs
	require.Len(t, firstRuns, 2)
	require.Len(t, restRuns, 1)
	assert.Equal(t, 12.0, firstRuns[0].Size)
	assert.Equal(t, 10.0, firstRuns[1].Size)
	assert.Equal(t, 10.0, restRuns[0].Size)
}

func TestParagraphSplitPreservesTrailingWhitespace(t *testing.T) {
	p := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	paragraph := newSplittableParagraphRow([]props.RichRun{{Text: "one two three four five six   "}}, props.RichText{}, nil)
	paragraph.SetConfig(&entity.Config{MaxGridSize: 12})

	first, rest, split := paragraph.SplitAt(p, 2, 100)
	require.True(t, split)
	require.NotNil(t, first)
	require.NotNil(t, rest)
	assert.Equal(t, []string{"one two three four"}, richTextValues([]core.Row{first}))
	assert.Equal(t, []string{"five six   "}, richTextValues([]core.Row{rest}))
}
