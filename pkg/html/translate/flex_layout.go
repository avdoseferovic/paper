package translate

import (
	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
)

// Hamilton distributes total integer units across items proportionally to
// their weights using the largest-remainder method (Hamilton's method).
// Returns []int{} for empty input; zero-weight inputs get equal split.
func Hamilton(weights []float64, total int) []int {
	return layout.Hamilton(weights, total)
}

const defaultFlexContentWidthMM = 170.0

// computeFlexSizes allocates gridSize integer cells across flex items using the
// default A4-ish content width. Tests use this wrapper to preserve the old hook.
func computeFlexSizes(styles []*css.ComputedStyle, gridSize int) []int {
	return computeFlexSizesForWidth(styles, gridSize, defaultFlexContentWidthMM)
}

// computeFlexSizesForWidth allocates gridSize integer cells across flex items.
// Percentage and length-basis items get a fixed share first; remaining cells go
// to grow items via Hamilton's largest-remainder. Items with neither basis nor
// grow get default grow=1 so unstyled flex children participate equally.
func computeFlexSizesForWidth(styles []*css.ComputedStyle, gridSize int, contentWidthMM float64) []int {
	sizes := make([]int, len(styles))
	fixedTotal := 0
	var fixedIndices []int
	var growIndices []int
	var growWeights []float64

	for i, s := range styles {
		if share, ok := fixedFlexShare(s, gridSize, contentWidthMM); ok {
			sizes[i] = share
			fixedTotal += share
			fixedIndices = append(fixedIndices, i)
		} else {
			growIndices = append(growIndices, i)
			growWeights = append(growWeights, effectiveGrow(s))
		}
	}

	// Clamp percentages that collectively exceed gridSize by scaling them down.
	if fixedTotal > gridSize {
		weights := make([]float64, len(fixedIndices))
		for i, idx := range fixedIndices {
			weights[i] = float64(sizes[idx])
		}
		scaled := layout.BumpZerosWithoutOverflow(layout.ProportionalUnits(weights, gridSize), gridSize)
		fixedTotal = 0
		for i, idx := range fixedIndices {
			sizes[idx] = scaled[i]
			fixedTotal += scaled[i]
		}
	}

	remaining := max(gridSize-fixedTotal, 0)

	if len(growIndices) > 0 && remaining > 0 {
		growSizes := layout.ProportionalUnits(growWeights, remaining)
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

func fixedFlexShare(s *css.ComputedStyle, gridSize int, contentWidthMM float64) (int, bool) {
	if s == nil {
		return 0, false
	}
	if s.FlexBasisPct > 0 {
		return max(int(s.FlexBasisPct/100.0*float64(gridSize)+0.5), 1), true
	}
	if s.FlexBasis <= 0 {
		return 0, false
	}
	if contentWidthMM <= 0 {
		contentWidthMM = defaultFlexContentWidthMM
	}
	return max(int(s.FlexBasis/contentWidthMM*float64(gridSize)+0.5), 1), true
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
	case flexAlignEnd:
		return slackPlan{lead: slack}
	case flexAlignCenter:
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
	return layout.BumpZerosWithoutOverflow(sizes, originalTotal)
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
