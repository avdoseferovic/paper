package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestOutline_NormalizedLevel_WhenNegative_ShouldClampToZero(t *testing.T) {
	t.Parallel()

	o := &props.Outline{Level: -3}

	assert.Equal(t, 0, o.NormalizedLevel())
}

func TestOutline_NormalizedLevel_WhenPositive_ShouldReturnLevel(t *testing.T) {
	t.Parallel()

	o := &props.Outline{Level: 2}

	assert.Equal(t, 2, o.NormalizedLevel())
}

func TestOutline_ResolveTitle_WhenTitleEmpty_ShouldReturnFallback(t *testing.T) {
	t.Parallel()

	o := &props.Outline{}

	assert.Equal(t, "Chapter 1", o.ResolveTitle("Chapter 1"))
}

func TestOutline_ResolveTitle_WhenTitleSet_ShouldReturnTitle(t *testing.T) {
	t.Parallel()

	o := &props.Outline{Title: "Custom"}

	assert.Equal(t, "Custom", o.ResolveTitle("fallback"))
}

func TestOutline_ToMap_ShouldExposeLevelAndOptionalTitle(t *testing.T) {
	t.Parallel()

	bare := (&props.Outline{Level: 1}).ToMap(nil)
	titled := (&props.Outline{Level: 0, Title: "Intro"}).ToMap(map[string]any{"existing": true})

	assert.Equal(t, 1, bare["prop_outline_level"])
	_, hasTitle := bare["prop_outline_title"]
	assert.False(t, hasTitle)
	assert.Equal(t, "Intro", titled["prop_outline_title"])
	assert.Equal(t, true, titled["existing"])
}

func TestCloneOutline_WhenNil_ShouldReturnNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, props.CloneOutline(nil))
}

func TestCloneOutline_WhenSet_ShouldReturnIndependentCopy(t *testing.T) {
	t.Parallel()

	original := &props.Outline{Title: "A", Level: 1}

	clone := props.CloneOutline(original)
	clone.Title = "B"

	assert.Equal(t, "A", original.Title)
}
