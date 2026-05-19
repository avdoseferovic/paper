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
func (tr *translator) flexRow(n *dom.Node, containerStyle *css.ComputedStyle) core.Row {
	children := flexItems(n)
	if len(children) == 0 {
		return nil
	}

	gridSize := tr.gridSize
	if gridSize <= 0 {
		gridSize = defaultGridSize
	}

	styles := make([]*css.ComputedStyle, len(children))
	for i, child := range children {
		styles[i] = computeNodeStyle(tr.sheet, child, containerStyle)
	}
	weights := flexWeights(styles, gridSize)
	sizes := Hamilton(weights, gridSize)

	var cols []core.Col
	for i, child := range children {
		sz := sizes[i]
		if sz <= 0 {
			sz = 1
		}
		c := col.New(sz)
		comp := tr.flexItemContent(child, styles[i])
		if comp != nil {
			c = c.Add(comp)
		}
		cols = append(cols, c)
	}

	r := row.New()
	for _, c := range cols {
		r = r.Add(c)
	}
	return r
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

// flexWeights computes per-item proportional weights for Hamilton allocation.
// Percentage-basis items get a fixed fraction of gridSize; grow items share
// the remaining fraction proportionally. Mixed %-basis + grow is handled correctly.
func flexWeights(styles []*css.ComputedStyle, gridSize int) []float64 {
	// Sum all fixed percentage fractions.
	totalPct := 0.0
	for _, s := range styles {
		if s.FlexBasisPct > 0 {
			totalPct += s.FlexBasisPct / 100.0
		}
	}
	if totalPct > 1.0 {
		totalPct = 1.0
	}
	remaining := 1.0 - totalPct

	// Sum grow values across non-percentage items.
	totalGrow := 0.0
	for _, s := range styles {
		if s.FlexBasisPct == 0 {
			g := effectiveGrow(s)
			totalGrow += g
		}
	}

	weights := make([]float64, len(styles))
	for i, s := range styles {
		if s.FlexBasisPct > 0 {
			weights[i] = s.FlexBasisPct / 100.0 * float64(gridSize)
		} else {
			g := effectiveGrow(s)
			if totalGrow > 0 {
				weights[i] = g / totalGrow * remaining * float64(gridSize)
			} else {
				weights[i] = remaining / float64(len(styles)) * float64(gridSize)
			}
		}
	}
	return weights
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

	// Non-leaf: flatten to text (TODO Task 6 replaces with nested block renderer).
	text := strings.TrimSpace(n.TextContent())
	if text == "" {
		return nil
	}
	return richtext.New([]props.RichRun{{Text: text}})
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
