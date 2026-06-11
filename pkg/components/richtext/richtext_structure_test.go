package richtext_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

type fakeAnchorRegistry struct {
	linkID int
}

func (f *fakeAnchorRegistry) EnsureLinkID(_ string, _ core.LinkProvider) (int, bool) {
	return f.linkID, f.linkID != 0
}

func TestRichText_WithAnchorRegistry(t *testing.T) {
	t.Parallel()

	runs := []props.RichRun{{Text: "see intro", LocalAnchor: "intro"}}

	sut := richtext.New(runs).WithAnchorRegistry(&fakeAnchorRegistry{linkID: 7})

	// The registry attaches fluently and the component stays usable.
	assert.Equal(t, "richtext", sut.GetStructure().GetData().Type)
}

func TestRichText_GetStructureIncludesNonDefaultProps(t *testing.T) {
	t.Parallel()

	runs := []props.RichRun{{Text: "styled"}}
	sut := richtext.New(runs, props.RichText{
		LineHeight:      2,
		FirstLineIndent: 4,
		Left:            6,
		Align:           consts.AlignCenter,
		WhiteSpace:      "pre",
	})
	sut.SetConfig(defaultConfig())

	details := sut.GetStructure().GetData().Details

	assert.Equal(t, 2.0, details["line_height"])
	assert.Equal(t, 4.0, details["first_line_indent"])
	assert.Equal(t, 6.0, details["left"])
	assert.Equal(t, consts.AlignCenter, details["align"])
	assert.Equal(t, "pre", details["white_space"])
}
