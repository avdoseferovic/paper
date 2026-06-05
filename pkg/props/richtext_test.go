package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/breakline"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/stretchr/testify/assert"
)

func TestRichRun(t *testing.T) {
	t.Parallel()
	t.Run("should hold run fields", func(t *testing.T) {
		t.Parallel()
		link := "https://example.com"
		color := &props.Color{Red: 255}
		run := props.RichRun{
			Text:          "hello",
			Family:        "Helvetica",
			Style:         fontstyle.Bold,
			Size:          12,
			Color:         color,
			Underline:     true,
			Strikethrough: false,
			Hyperlink:     &link,
			VerticalAlign: "baseline",
		}
		assert.Equal(t, "hello", run.Text)
		assert.Equal(t, fontstyle.Bold, run.Style)
		assert.Equal(t, &link, run.Hyperlink)
		assert.True(t, run.Underline)
	})
}

func TestRichText_MakeValid(t *testing.T) {
	t.Parallel()
	t.Run("should set defaults when empty", func(t *testing.T) {
		t.Parallel()
		rt := &props.RichText{}
		rt.MakeValid(nil)
		assert.Equal(t, align.Left, rt.Align)
		assert.Equal(t, 1.0, rt.LineHeight)
		assert.Equal(t, breakline.EmptySpaceStrategy, rt.BreakLineStrategy)
	})

	t.Run("should not override existing values", func(t *testing.T) {
		t.Parallel()
		rt := &props.RichText{
			Align:      align.Center,
			LineHeight: 1.5,
		}
		rt.MakeValid(nil)
		assert.Equal(t, align.Center, rt.Align)
		assert.Equal(t, 1.5, rt.LineHeight)
	})
}
