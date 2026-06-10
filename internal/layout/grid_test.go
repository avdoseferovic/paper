package layout

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestHamilton(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		weights []float64
		total   int
		want    []int
	}{
		{name: "empty", weights: nil, total: 12, want: []int{}},
		{name: "equal split", weights: []float64{1, 1, 1}, total: 12, want: []int{4, 4, 4}},
		{name: "largest remainder", weights: []float64{1, 1, 1, 1, 1}, total: 12, want: []int{3, 3, 2, 2, 2}},
		{name: "zero weights split equally", weights: []float64{0, 0, 0}, total: 12, want: []int{4, 4, 4}},
		{name: "non-positive total", weights: []float64{1, 1}, total: 0, want: []int{0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, Hamilton(tt.weights, tt.total))
		})
	}
}

func TestProportionalUnits(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []int{2, 6}, ProportionalUnits([]float64{1, 3}, 8))
	assert.Equal(t, []int{0, 0}, ProportionalUnits([]float64{1, 3}, -1))
}

func TestManualUnits(t *testing.T) {
	t.Parallel()

	t.Run("preserves exact fit explicit units", func(t *testing.T) {
		t.Parallel()

		plan := ManualUnits([]int{6, 6}, 12)

		assert.Equal(t, []int{6, 6}, plan.Units)
		assert.Equal(t, 12, plan.GridSize)
		assert.Equal(t, 12, plan.TotalUnits)
		assert.Equal(t, 0, plan.Slack)
		assert.Equal(t, 0, plan.Overflow)
		assert.Empty(t, plan.InvalidIndices)
	})

	t.Run("preserves underflow explicit units", func(t *testing.T) {
		t.Parallel()

		plan := ManualUnits([]int{4, 4}, 12)

		assert.Equal(t, []int{4, 4}, plan.Units)
		assert.Equal(t, 8, plan.TotalUnits)
		assert.Equal(t, 4, plan.Slack)
		assert.Equal(t, 0, plan.Overflow)
	})

	t.Run("preserves overflow explicit units", func(t *testing.T) {
		t.Parallel()

		plan := ManualUnits([]int{8, 8}, 12)

		assert.Equal(t, []int{8, 8}, plan.Units)
		assert.Equal(t, 16, plan.TotalUnits)
		assert.Equal(t, 0, plan.Slack)
		assert.Equal(t, 4, plan.Overflow)
	})

	t.Run("reports zero and negative explicit units without scaling", func(t *testing.T) {
		t.Parallel()

		plan := ManualUnits([]int{0, -2, 12}, 12)

		assert.Equal(t, []int{0, -2, 12}, plan.Units)
		assert.Equal(t, []int{0, 1}, plan.InvalidIndices)
		assert.Equal(t, 10, plan.TotalUnits)
		assert.Equal(t, 2, plan.Slack)
	})

	t.Run("normalizes invalid grid size to default", func(t *testing.T) {
		t.Parallel()

		plan := ManualUnits([]int{12}, 0)

		assert.Equal(t, 12, plan.GridSize)
		assert.Equal(t, []int{12}, plan.Units)
	})
}

func TestUnitWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 60.0, UnitWidth(120, 6, 12))
	assert.Equal(t, 60.0, UnitWidth(120, 4, 8))
	assert.Equal(t, 80.0, UnitWidth(120, 8, 12))
	assert.Equal(t, 31.666666666666664, UnitWidth(190, 2, 12))
	assert.Equal(t, 0.0, UnitWidth(120, 0, 12))
	assert.Equal(t, 0.0, UnitWidth(120, -1, 12))
	assert.Equal(t, 60.0, UnitWidth(120, 6, 0))
}

func TestBumpZerosWithoutOverflow(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []int{1, 5, 6}, BumpZerosWithoutOverflow([]int{0, 6, 6}, 12))
	assert.Equal(t, []int{0, 1}, BumpZerosWithoutOverflow([]int{0, 1}, 1))
	assert.Equal(t, []int{}, BumpZerosWithoutOverflow(nil, 12))
}
