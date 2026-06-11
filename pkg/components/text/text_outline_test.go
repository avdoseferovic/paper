package text_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestText_GetStructure_WhenOutlineSet_ShouldExposeOutlineDetails(t *testing.T) {
	t.Parallel()

	sut := text.New("Chapter 1", props.Text{Outline: &props.Outline{Level: 1, Title: "Ch. 1"}})

	details := sut.GetStructure().GetData().Details

	assert.Equal(t, 1, details["prop_outline_level"])
	assert.Equal(t, "Ch. 1", details["prop_outline_title"])
}

func TestText_GetStructure_WhenNoOutline_ShouldNotExposeOutlineDetails(t *testing.T) {
	t.Parallel()

	sut := text.New("Plain")

	details := sut.GetStructure().GetData().Details

	_, hasLevel := details["prop_outline_level"]
	assert.False(t, hasLevel)
}
