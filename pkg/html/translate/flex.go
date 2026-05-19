// Package translate — flex layout module.
// Converts display:flex containers into Maroto rows with Hamilton-quantized cols.
package translate

import (
	"sort"
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/richtext"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

const defaultGridSize = 12

// Hamilton distributes total integer units across items proportionally to
// their weights using the largest-remainder method (Hamilton's method).
// Returns []int{} for empty input; zero-weight inputs get equal split.
func Hamilton(weights []float64, total int) []int {
	if len(weights) == 0 {
		return []int{}
	}

	// Normalise zero-weight inputs to equal split.
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
	for i := 0; i < remainder; i++ {
		result[fracs[i].idx]++
	}
	return result
}

// flexRow converts a display:flex container node into a single core.Row
// containing one core.Col per flex item, sized by Hamilton's method.
// containerStyle is used as the parent for em-unit resolution in children.
// Returns nil for empty containers. For flex-direction:column it returns nil
// and the caller (flexBlockRows) handles vertical stacking instead.
func (tr *translator) flexRow(n *dom.Node, containerStyle *css.ComputedStyle) core.Row {
	children := flexItems(n)
	if len(children) == 0 {
		return nil
	}
	if isColumnDirection(containerStyle.FlexDirection) {
		return nil // caller will use flexColumnRows instead
	}

	gridSize := tr.gridSize
	if gridSize <= 0 {
		gridSize = defaultGridSize
	}

	styles := make([]*css.ComputedStyle, len(children))
	for i, child := range children {
		styles[i] = computeNodeStyle(tr.sheet, child, containerStyle)
	}

	// Compute gap cols (reserved between items).
	gapCols := tr.gapCols(containerStyle.ColumnGap, gridSize, len(children))
	totalGap := gapCols * max(0, len(children)-1)
	available := gridSize - totalGap
	if available < len(children) {
		gapCols = 0
		totalGap = 0
		available = gridSize
	}

	sizes := computeFlexSizes(styles, available)

	itemCols := make([]core.Col, len(children))
	used := 0
	for i, child := range children {
		sz := sizes[i]
		if sz <= 0 {
			sz = 1
		}
		used += sz
		c := col.New(sz)
		comp := tr.flexItemContent(child, styles[i])
		if comp != nil {
			c = c.Add(comp)
		}
		itemCols[i] = c
	}

	slack := gridSize - used - totalGap
	if slack < 0 {
		slack = 0
	}

	finalCols := assembleFlexCols(itemCols, gapCols, slack, containerStyle.JustifyContent)

	r := row.New()
	for _, c := range finalCols {
		r = r.Add(c)
	}
	return r
}

// gapCols converts a CSS gap (in mm) to integer spacer cols, clamped to half the grid.
// Uses the translator's contentWidthMM, defaulting to 170mm (A4 with 20mm L+R margins).
func (tr *translator) gapCols(gapMM float64, gridSize, itemCount int) int {
	if gapMM <= 0 || itemCount <= 1 {
		return 0
	}
	contentWidth := tr.contentWidthMM
	if contentWidth <= 0 {
		contentWidth = 170.0
	}
	mmPerCol := contentWidth / float64(gridSize)
	if mmPerCol <= 0 {
		return 0
	}
	cols := int(gapMM/mmPerCol + 0.5)
	maxGap := gridSize / 2
	if cols > maxGap {
		cols = maxGap
	}
	if cols < 0 {
		cols = 0
	}
	return cols
}

// assembleFlexCols interleaves item cols with gap spacers and applies
// justify-content slack distribution. Returns the final sequence of cols.
func assembleFlexCols(items []core.Col, gapCols, slack int, justify string) []core.Col {
	if len(items) == 0 {
		return nil
	}

	// Distribute slack based on justify-content.
	leadOffset, trailOffset := 0, 0
	betweenExtra := 0 // extra cols per between-spacer (added on top of gapCols)
	aroundLead, aroundBetween, aroundTrail := 0, 0, 0
	useAroundDistribution := false

	switch justify {
	case "flex-end":
		leadOffset = slack
	case "center":
		leadOffset = slack / 2
		trailOffset = slack - leadOffset
	case "space-between":
		if len(items) > 1 && slack > 0 {
			// Distribute slack across N-1 between-spacers via Hamilton.
			weights := make([]float64, len(items)-1)
			for i := range weights {
				weights[i] = 1.0
			}
			extras := Hamilton(weights, slack)
			// For simplicity, take the average; assembleFlexCols below uses betweenExtra uniformly.
			// To preserve Hamilton's exact allocation we'd need a per-slot array. Use first value as common case.
			if len(extras) > 0 {
				betweenExtra = extras[0]
				// If non-uniform, the remainder gets folded into trailOffset.
				totalExtras := 0
				for _, e := range extras {
					totalExtras += e
				}
				trailOffset += slack - totalExtras
				if betweenExtra*max(0, len(items)-1)+trailOffset != slack {
					// Adjust to ensure conservation.
					trailOffset = slack - betweenExtra*max(0, len(items)-1)
				}
			}
		}
		// If slack == 0, falls back to flex-start silently (documented).
	case "space-around":
		if slack > 0 {
			// N+1 spacers (lead, between, trail).
			weights := make([]float64, len(items)+1)
			for i := range weights {
				weights[i] = 1.0
			}
			extras := Hamilton(weights, slack)
			useAroundDistribution = true
			if len(extras) > 0 {
				aroundLead = extras[0]
				aroundTrail = extras[len(extras)-1]
				// Average of inner spacers as common between.
				if len(extras) > 2 {
					aroundBetween = extras[1]
				}
			}
		}
	default:
		// flex-start: trailing slack is implicit (sum of cols < gridSize leaves space on the right).
	}

	var out []core.Col

	if useAroundDistribution {
		if aroundLead > 0 {
			out = append(out, col.New(aroundLead))
		}
		for i, item := range items {
			out = append(out, item)
			if i < len(items)-1 {
				gap := gapCols + aroundBetween
				if gap > 0 {
					out = append(out, col.New(gap))
				}
			}
		}
		if aroundTrail > 0 {
			out = append(out, col.New(aroundTrail))
		}
		return out
	}

	if leadOffset > 0 {
		out = append(out, col.New(leadOffset))
	}
	for i, item := range items {
		out = append(out, item)
		if i < len(items)-1 {
			gap := gapCols + betweenExtra
			if gap > 0 {
				out = append(out, col.New(gap))
			}
		}
	}
	if trailOffset > 0 {
		out = append(out, col.New(trailOffset))
	}
	return out
}

// isColumnDirection returns true for column or column-reverse.
func isColumnDirection(d string) bool {
	return d == "column" || d == "column-reverse"
}

// flexColumnRows handles flex-direction:column by emitting one row per child,
// optionally inserting empty spacer rows between them based on row-gap.
func (tr *translator) flexColumnRows(n *dom.Node, containerStyle *css.ComputedStyle) []core.Row {
	children := flexItems(n)
	if len(children) == 0 {
		return nil
	}
	gapMM := containerStyle.RowGap
	var out []core.Row
	for i, child := range children {
		childRows := tr.blockRows(child)
		out = append(out, childRows...)
		if gapMM > 0 && i < len(children)-1 {
			out = append(out, spacerRow(gapMM))
		}
	}
	return out
}

// spacerRow returns an empty row of the given height in mm.
func spacerRow(heightMM float64) core.Row {
	return row.New(heightMM).Add(col.New())
}

// flexItems returns the non-whitespace children of a flex container.
func flexItems(n *dom.Node) []*dom.Node {
	var out []*dom.Node
	for _, c := range n.Children() {
		if isWhitespaceNode(c) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// isWhitespaceNode returns true for text nodes that contain only whitespace.
func isWhitespaceNode(n *dom.Node) bool {
	return n.Tag() == "" && strings.TrimSpace(n.TextContent()) == ""
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
			share := int(s.FlexBasisPct/100.0*float64(gridSize) + 0.5)
			if share < 1 {
				share = 1
			}
			sizes[i] = share
			fixedTotal += share
		} else {
			growIndices = append(growIndices, i)
			growWeights = append(growWeights, effectiveGrow(s))
		}
	}

	remaining := gridSize - fixedTotal
	if remaining < 0 {
		remaining = 0
	}

	if len(growIndices) > 0 && remaining > 0 {
		growSizes := Hamilton(growWeights, remaining)
		for j, idx := range growIndices {
			sizes[idx] = growSizes[j]
		}
	} else if len(growIndices) > 0 {
		// No remaining space; grow items get 0 (they collapse).
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

// flexItemContent builds the component rendered inside a flex col.
// Leaf items (inline content) become RichText; non-leaf items get flattened
// to text as a temporary stub (replaced in Task 6).
func (tr *translator) flexItemContent(n *dom.Node, style *css.ComputedStyle) core.Component {
	if n.Tag() == "" {
		text := strings.TrimSpace(n.TextContent())
		if text == "" {
			return nil
		}
		return richtext.New([]props.RichRun{{Text: text}})
	}

	if isLeafFlexItem(n) {
		runs := inlineRuns(n)
		if len(runs) == 0 {
			return nil
		}
		applyBlockStyling(n, runs)
		applyInlineStyleToRuns(style, runs)
		return richtext.New(runs, props.RichText{})
	}

	// Non-leaf: render each block child as a sub-row and aggregate via a wrapper.
	var subRows []core.Row
	for _, c := range n.Children() {
		if isWhitespaceNode(c) {
			continue
		}
		subRows = append(subRows, tr.blockRows(c)...)
	}
	if len(subRows) == 0 {
		text := strings.TrimSpace(n.TextContent())
		if text == "" {
			return nil
		}
		return richtext.New([]props.RichRun{{Text: text}})
	}
	return newFlexCellContent(subRows)
}

// isLeafFlexItem returns true when a node has no block-level children.
func isLeafFlexItem(n *dom.Node) bool {
	for _, c := range n.Children() {
		if c.IsBlock() {
			return false
		}
	}
	return true
}
