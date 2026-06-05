package translate

import (
	"sort"

	"github.com/avdoseferovic/paper/v2/pkg/components/col"
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/html/css"
)

// Hamilton distributes total integer units across items proportionally to
// their weights using the largest-remainder method (Hamilton's method).
// Returns []int{} for empty input; zero-weight inputs get equal split.
func Hamilton(weights []float64, total int) []int {
	if len(weights) == 0 {
		return []int{}
	}

	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if sum <= 0 {
		w := make([]float64, len(weights))
		for i := range w {
			w[i] = 1.0
		}
		weights = w
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

// computeFlexSizes allocates gridSize integer cells across flex items.
// Percentage-basis items get a fixed share first; remaining cells go to grow items
// via Hamilton's largest-remainder. Items with neither basis nor grow get default
// grow=1 so unstyled flex children participate equally.
func computeFlexSizes(styles []*css.ComputedStyle, gridSize int) []int {
	sizes := make([]int, len(styles))
	fixedTotal := 0
	var growIndices []int
	var growWeights []float64

	for i, s := range styles {
		if s.FlexBasisPct > 0 {
			share := max(int(s.FlexBasisPct/100.0*float64(gridSize)+0.5), 1)
			sizes[i] = share
			fixedTotal += share
		} else {
			growIndices = append(growIndices, i)
			growWeights = append(growWeights, effectiveGrow(s))
		}
	}

	// Clamp percentages that collectively exceed gridSize by scaling them down.
	if fixedTotal > gridSize {
		scale := float64(gridSize) / float64(fixedTotal)
		fixedTotal = 0
		for i, s := range styles {
			if s.FlexBasisPct > 0 {
				sizes[i] = max(int(float64(sizes[i])*scale), 1)
				fixedTotal += sizes[i]
			}
		}
	}

	remaining := max(gridSize-fixedTotal, 0)

	if len(growIndices) > 0 && remaining > 0 {
		growSizes := Hamilton(growWeights, remaining)
		for j, idx := range growIndices {
			sizes[idx] = growSizes[j]
		}
	} else if len(growIndices) > 0 {
		for _, idx := range growIndices {
			sizes[idx] = 0
		}
	}
	return sizes
}

// effectiveGrow returns the flex-grow value, defaulting to 1 for items with no
// explicit basis or grow (the CSS default for auto-sized children in flex rows).
func effectiveGrow(s *css.ComputedStyle) float64 {
	if s.FlexGrow > 0 {
		return s.FlexGrow
	}
	if s.FlexBasis > 0 || s.FlexBasisAuto {
		return 0 // basis-only item, no growth
	}
	return 1.0 // default: item participates equally
}

// slackPlan describes how slack cols are distributed by justify-content.
type slackPlan struct {
	lead         int // leading offset col size
	trail        int // trailing offset col size
	betweenExtra int // extra cols added to each gap-spacer (on top of gapCols)
}

// computeSlackPlan derives lead/trail/between distribution from justify-content.
func computeSlackPlan(justify string, itemCount, slack int) slackPlan {
	switch justify {
	case "flex-end":
		return slackPlan{lead: slack}
	case "center":
		lead := slack / 2
		return slackPlan{lead: lead, trail: slack - lead}
	case "space-between":
		return planSpaceBetween(itemCount, slack)
	case "space-around":
		return planSpaceAround(itemCount, slack)
	default:
		return slackPlan{}
	}
}

// planSpaceBetween distributes slack across N-1 between-spacers uniformly
// (floor) with any remainder going to the trailing offset.
func planSpaceBetween(itemCount, slack int) slackPlan {
	if itemCount <= 1 || slack <= 0 {
		return slackPlan{}
	}
	gaps := itemCount - 1
	between := slack / gaps
	remainder := slack - between*gaps
	return slackPlan{betweenExtra: between, trail: remainder}
}

// planSpaceAround distributes slack across N+1 spacers (lead, between, trail)
// uniformly. Any remainder is distributed lead-first to keep symmetry roughly intact.
func planSpaceAround(itemCount, slack int) slackPlan {
	if slack <= 0 {
		return slackPlan{}
	}
	spots := itemCount + 1
	base := slack / spots
	remainder := slack - base*spots
	lead := base
	trail := base
	if remainder > 0 {
		lead++
		remainder--
	}
	if remainder > 0 {
		trail++
		remainder--
	}
	// Any further remainder (only when itemCount > 1) goes to between-extras.
	between := base
	if remainder > 0 && itemCount > 1 {
		// Spread across N-1 between gaps; uniform distribution.
		extra := remainder / (itemCount - 1)
		between += extra
	}
	return slackPlan{lead: lead, trail: trail, betweenExtra: between}
}

// bumpZerosWithoutOverflow ensures every size is at least 1 without exceeding
// the original total. For each 0 bumped to 1, decrement the largest size by 1.
// This preserves sum(sizes) == originalTotal so the row never overflows the grid.
func bumpZerosWithoutOverflow(sizes []int, originalTotal int) []int {
	out := make([]int, len(sizes))
	copy(out, sizes)
	for i := range out {
		if out[i] != 0 {
			continue
		}
		// Find largest size that can give up one cell (must stay ≥ 2 after donation).
		largestIdx := -1
		for j := range out {
			if out[j] >= 2 && (largestIdx == -1 || out[j] > out[largestIdx]) {
				largestIdx = j
			}
		}
		if largestIdx == -1 {
			// No size has slack; leave this position at 0 — caller may choose to drop it.
			continue
		}
		out[i] = 1
		out[largestIdx]--
	}
	_ = originalTotal // signature documents the invariant; sum unchanged by construction
	return out
}

// assembleFlexCols interleaves item cols with gap+slack spacers and applies
// justify-content slack distribution. Returns the final sequence of cols.
func assembleFlexCols(items []core.Col, gapCols, slack int, justify string) []core.Col {
	if len(items) == 0 {
		return nil
	}
	plan := computeSlackPlan(justify, len(items), slack)
	var out []core.Col
	if plan.lead > 0 {
		out = append(out, col.New(plan.lead))
	}
	for i, item := range items {
		out = append(out, item)
		if i < len(items)-1 {
			if gap := gapCols + plan.betweenExtra; gap > 0 {
				out = append(out, col.New(gap))
			}
		}
	}
	if plan.trail > 0 {
		out = append(out, col.New(plan.trail))
	}
	return out
}
