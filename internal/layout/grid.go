// Package layout turns column weights into whole grid units and applies cell
// margins. Hamilton distributes units so the rounded values still add up to the
// row total.
package layout

import (
	"sort"

	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
)

// DefaultGridSize returns the number of grid units a row spans when the caller
// has not chosen one.
func DefaultGridSize() int {
	return int(pagesize.DefaultMaxGridSum)
}

// NormalizeGridSize replaces a zero or negative grid size with the default.
func NormalizeGridSize(gridSize int) int {
	if gridSize <= 0 {
		return DefaultGridSize()
	}
	return gridSize
}

// Hamilton distributes total integer units across items proportionally to
// their weights using the largest-remainder method.
func Hamilton(weights []float64, total int) []int {
	if len(weights) == 0 {
		return []int{}
	}
	if total <= 0 {
		return make([]int, len(weights))
	}

	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if sum <= 0 {
		equal := make([]float64, len(weights))
		for i := range equal {
			equal[i] = 1
		}
		weights = equal
		sum = float64(len(weights))
	}

	exact := make([]float64, len(weights))
	floors := make([]int, len(weights))
	allocated := 0
	for i, w := range weights {
		exact[i] = w / sum * float64(total)
		floors[i] = int(exact[i])
		allocated += floors[i]
	}

	remainder := total - allocated
	type indexFrac struct {
		idx  int
		frac float64
	}
	fracs := make([]indexFrac, len(weights))
	for i, e := range exact {
		fracs[i] = indexFrac{idx: i, frac: e - float64(floors[i])}
	}
	sort.Slice(fracs, func(a, b int) bool {
		return fracs[a].frac > fracs[b].frac
	})

	result := make([]int, len(weights))
	copy(result, floors)
	for i := range remainder {
		result[fracs[i].idx]++
	}
	return result
}

// ProportionalUnits splits total units across items in proportion to weights.
func ProportionalUnits(weights []float64, total int) []int {
	return Hamilton(weights, total)
}

// ManualPlan is the result of checking column sizes the caller set by hand.
// Slack is how many units are left over, Overflow how many the row is above the
// grid size, and InvalidIndices lists the columns whose size was not positive.
type ManualPlan struct {
	Units          []int
	GridSize       int
	TotalUnits     int
	Slack          int
	Overflow       int
	InvalidIndices []int
}

// ManualUnits checks caller-supplied column sizes against the grid size and
// reports what it found, without changing the sizes.
func ManualUnits(units []int, gridSize int) ManualPlan {
	plan := ManualPlan{
		Units:    append([]int(nil), units...),
		GridSize: NormalizeGridSize(gridSize),
	}
	for i, unit := range units {
		plan.TotalUnits += unit
		if unit <= 0 {
			plan.InvalidIndices = append(plan.InvalidIndices, i)
		}
	}
	if plan.TotalUnits < plan.GridSize {
		plan.Slack = plan.GridSize - plan.TotalUnits
	}
	if plan.TotalUnits > plan.GridSize {
		plan.Overflow = plan.TotalUnits - plan.GridSize
	}
	return plan
}

// UnitWidth returns the width of a column spanning unit grid units out of
// gridSize, within totalWidth.
func UnitWidth(totalWidth float64, unit, gridSize int) float64 {
	if unit <= 0 {
		return 0
	}
	return totalWidth * (float64(unit) / float64(NormalizeGridSize(gridSize)))
}

// BumpZerosWithoutOverflow ensures every zero size becomes at least 1 when a
// larger sibling can donate a unit, preserving the original sum.
func BumpZerosWithoutOverflow(sizes []int, _ int) []int {
	if len(sizes) == 0 {
		return []int{}
	}
	out := append([]int(nil), sizes...)
	for i := range out {
		if out[i] != 0 {
			continue
		}
		largestIdx := -1
		for j := range out {
			if out[j] >= 2 && (largestIdx == -1 || out[j] > out[largestIdx]) {
				largestIdx = j
			}
		}
		if largestIdx == -1 {
			continue
		}
		out[i] = 1
		out[largestIdx]--
	}
	return out
}
