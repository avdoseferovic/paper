package cellwriter

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/props"
)

func alphaPtr(v float64) *float64 { return &v }

func TestClampAlpha(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"negative clamps to zero", -0.5, 0},
		{"above one clamps to one", 1.5, 1},
		{"in range passes through", 0.42, 0.42},
		{"zero passes through", 0, 0},
		{"one passes through", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, clampAlpha(tt.input))
		})
	}
}

func TestEffectiveAlpha(t *testing.T) {
	t.Parallel()

	t.Run("nil prop returns one", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1.0, effectiveAlpha(nil))
	})

	t.Run("prop without colors returns one", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1.0, effectiveAlpha(&props.Cell{}))
	})

	t.Run("colors without alpha return one", func(t *testing.T) {
		t.Parallel()
		prop := &props.Cell{
			BackgroundColor: &props.Color{Red: 1},
			BorderColor:     &props.Color{Red: 2},
		}
		assert.Equal(t, 1.0, effectiveAlpha(prop))
	})

	t.Run("background alpha is used when set", func(t *testing.T) {
		t.Parallel()
		prop := &props.Cell{BackgroundColor: &props.Color{Alpha: alphaPtr(0.5)}}
		assert.Equal(t, 0.5, effectiveAlpha(prop))
	})

	t.Run("border alpha wins when lower than background alpha", func(t *testing.T) {
		t.Parallel()
		prop := &props.Cell{
			BackgroundColor: &props.Color{Alpha: alphaPtr(0.5)},
			BorderColor:     &props.Color{Alpha: alphaPtr(0.3)},
		}
		assert.Equal(t, 0.3, effectiveAlpha(prop))
	})

	t.Run("higher border alpha does not override background alpha", func(t *testing.T) {
		t.Parallel()
		prop := &props.Cell{
			BackgroundColor: &props.Color{Alpha: alphaPtr(0.2)},
			BorderColor:     &props.Color{Alpha: alphaPtr(0.9)},
		}
		assert.Equal(t, 0.2, effectiveAlpha(prop))
	})

	t.Run("out-of-range alphas are clamped", func(t *testing.T) {
		t.Parallel()
		prop := &props.Cell{BackgroundColor: &props.Color{Alpha: alphaPtr(5)}}
		assert.Equal(t, 1.0, effectiveAlpha(prop))

		prop = &props.Cell{BackgroundColor: &props.Color{Alpha: alphaPtr(-2)}}
		assert.Equal(t, 0.0, effectiveAlpha(prop))
	})
}
