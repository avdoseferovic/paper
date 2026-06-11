package cellwriter

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestPickStrokeColor(t *testing.T) {
	t.Parallel()

	t.Run("uniform border color wins", func(t *testing.T) {
		t.Parallel()
		uniform := &props.Color{Red: 1}
		prop := &props.Cell{
			BorderColor:    uniform,
			BorderTopColor: &props.Color{Red: 2},
		}
		assert.Equal(t, uniform, pickStrokeColor(prop))
	})

	t.Run("first non-nil per-side color is picked in top-right-bottom-left order", func(t *testing.T) {
		t.Parallel()
		top := &props.Color{Red: 1}
		right := &props.Color{Red: 2}
		bottom := &props.Color{Red: 3}
		left := &props.Color{Red: 4}

		assert.Equal(t, top, pickStrokeColor(&props.Cell{BorderTopColor: top, BorderLeftColor: left}))
		assert.Equal(t, right, pickStrokeColor(&props.Cell{BorderRightColor: right, BorderBottomColor: bottom}))
		assert.Equal(t, bottom, pickStrokeColor(&props.Cell{BorderBottomColor: bottom, BorderLeftColor: left}))
		assert.Equal(t, left, pickStrokeColor(&props.Cell{BorderLeftColor: left}))
	})

	t.Run("no colors returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, pickStrokeColor(&props.Cell{}))
	})
}

func TestDrawStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fill, stroke bool
		expected     string
	}{
		{"fill and stroke", true, true, "DF"},
		{"fill only", true, false, "F"},
		{"stroke only", false, true, "D"},
		{"neither", false, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, drawStyle(tt.fill, tt.stroke))
		})
	}
}

func TestClampRadius(t *testing.T) {
	t.Parallel()

	t.Run("negative radius clamps to zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, clampRadius(-1, 10, 10))
	})

	t.Run("radius above half the smaller dimension clamps", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2.0, clampRadius(8, 10, 4))
		assert.Equal(t, 3.0, clampRadius(8, 6, 20))
	})

	t.Run("radius within bounds passes through", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1.5, clampRadius(1.5, 10, 10))
	})
}

func TestAverageBorderThickness(t *testing.T) {
	t.Parallel()

	t.Run("uniform thickness wins", func(t *testing.T) {
		t.Parallel()
		prop := &props.Cell{BorderThickness: 2, BorderTopThickness: 10}
		assert.Equal(t, 2.0, averageBorderThickness(prop))
	})

	t.Run("averages the non-zero per-side thicknesses", func(t *testing.T) {
		t.Parallel()
		prop := &props.Cell{BorderTopThickness: 1, BorderBottomThickness: 3}
		assert.Equal(t, 2.0, averageBorderThickness(prop))
	})

	t.Run("no thickness set returns zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, averageBorderThickness(&props.Cell{}))
	})
}
