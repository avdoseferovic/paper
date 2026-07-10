package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/props"
)

func firstRunColorFromHTML(t *testing.T, htmlStr string) *props.Color {
	t.Helper()
	runs := runsFromHTML(t, htmlStr)
	require.NotEmpty(t, runs)
	return runs[0].Color
}

func TestImportantStylesheetBeatsInlineNormal(t *testing.T) {
	t.Parallel()

	color := firstRunColorFromHTML(t, `<style>p { color: red !important; }</style><p style="color: blue">hello</p>`)

	require.NotNil(t, color)
	assert.Equal(t, 255, color.Red)
	assert.Equal(t, 0, color.Blue)
}

func TestImportantInlineBeatsStylesheetImportant(t *testing.T) {
	t.Parallel()

	color := firstRunColorFromHTML(t, `<style>p { color: #ff0000 !important; }</style><p style="color: #00ff00 !important">hello</p>`)

	require.NotNil(t, color)
	assert.Equal(t, 0, color.Red)
	assert.Equal(t, 255, color.Green)
}

func TestImportantStylesheetBeatsLaterNormalStylesheet(t *testing.T) {
	t.Parallel()

	color := firstRunColorFromHTML(t, `<style>p { color: red !important; } p { color: blue; }</style><p>hello</p>`)

	require.NotNil(t, color)
	assert.Equal(t, 255, color.Red)
	assert.Equal(t, 0, color.Blue)
}

func TestImportantSuffixIsStrippedFromInlineValue(t *testing.T) {
	t.Parallel()

	color := firstRunColorFromHTML(t, `<p style="color: blue !important">hello</p>`)

	require.NotNil(t, color)
	assert.Equal(t, 0, color.Red)
	assert.Equal(t, 255, color.Blue)
}
