package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestNormalizeWatermark_WhenZeroValues_ShouldApplyDefaults(t *testing.T) {
	t.Parallel()

	w := props.NormalizeWatermark(props.Watermark{Text: "DRAFT"})

	assert.Equal(t, 48.0, w.Size)
	assert.Equal(t, 0.12, w.Alpha)
	assert.Equal(t, 45.0, w.Angle)
}

func TestNormalizeWatermark_WhenAlphaAboveOne_ShouldClamp(t *testing.T) {
	t.Parallel()

	w := props.NormalizeWatermark(props.Watermark{Alpha: 3})

	assert.Equal(t, 1.0, w.Alpha)
}

func TestNormalizeWatermark_WhenValuesSet_ShouldKeepThem(t *testing.T) {
	t.Parallel()

	w := props.NormalizeWatermark(props.Watermark{Size: 20, Alpha: 0.5, Angle: -30})

	assert.Equal(t, 20.0, w.Size)
	assert.Equal(t, 0.5, w.Alpha)
	assert.Equal(t, -30.0, w.Angle)
}

func TestCloneWatermark_WhenNil_ShouldReturnNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, props.CloneWatermark(nil))
}

func TestCloneWatermark_WhenSet_ShouldDeepCopyColor(t *testing.T) {
	t.Parallel()

	original := &props.Watermark{Text: "X", Color: &props.Color{Red: 1}}

	clone := props.CloneWatermark(original)
	clone.Color.Red = 99

	assert.Equal(t, 1, original.Color.Red)
}
