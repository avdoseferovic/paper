package layout

import (
	"sort"

	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
)

func DefaultGridSize() int {
	return int(pagesize.DefaultMaxGridSum)
}

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

func ProportionalUnits(weights []float64, total int) []int {
	return Hamilton(weights, total)
}

type ManualPlan struct {
	Units          []int
	GridSize       int
	TotalUnits     int
	Slack          int
	Overflow       int
	InvalidIndices []int
}

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

func UnitWidth(totalWidth float64, unit int, gridSize int) float64 {
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
