package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/props"
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
		assert.Equal(t, consts.AlignLeft, rt.Align)
		assert.Equal(t, 1.0, rt.LineHeight)
		assert.Equal(t, consts.BreakLineEmptySpace, rt.BreakLineStrategy)
	})

	t.Run("should not override existing values", func(t *testing.T) {
		t.Parallel()
		rt := &props.RichText{
			Align:      consts.AlignCenter,
			LineHeight: 1.5,
		}
		rt.MakeValid(nil)
		assert.Equal(t, consts.AlignCenter, rt.Align)
		assert.Equal(t, 1.5, rt.LineHeight)
	})
}
