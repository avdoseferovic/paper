// Package translate — flex layout dispatch and item-content construction.
// Quantization, weight computation, and slack distribution live in flex_layout.go.
package translate

import (
	"sort"
	"strings"

	"github.com/avdoseferovic/paper/v2/pkg/components/col"
	"github.com/avdoseferovic/paper/v2/pkg/components/richtext"
	"github.com/avdoseferovic/paper/v2/pkg/components/row"
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/html/css"
	"github.com/avdoseferovic/paper/v2/pkg/html/dom"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

const defaultGridSize = 12

// flexRow converts a display:flex container node into a single core.Row
// containing one core.Col per flex item, sized by Hamilton's method.
// containerStyle is used as the parent for em-unit resolution in children.
// Returns nil for empty containers. For flex-direction:column it returns nil
// and the caller (flexColumnRows) handles vertical stacking instead.
func (tr *translator) flexRow(n *dom.Node, containerStyle *css.ComputedStyle) core.Row {
	children := flexItems(n)
	if len(children) == 0 {
		return nil
	}
	if isColumnDirection(containerStyle.FlexDirection) {
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

	gapCols := tr.gapCols(containerStyle.ColumnGap, gridSize, len(children))
	totalGap := gapCols * max(0, len(children)-1)
	available := gridSize - totalGap
	if available < len(children) {
		gapCols = 0
		totalGap = 0
		available = gridSize
	}

	sizes := computeFlexSizes(styles, available)
	sizes = bumpZerosWithoutOverflow(sizes, available)

	visualGap := 0.0
	if gapCols == 0 {
		visualGap = containerStyle.ColumnGap
	}

	itemCols := make([]core.Col, len(children))
	used := 0
	for i, child := range children {
		used += sizes[i]
		c := col.New(sizes[i])
		if comp := tr.flexItemContent(child, styles[i]); comp != nil {
			if visualGap > 0 && i > 0 {
				comp = &marginBox{child: comp, marginLeft: visualGap}
			}
			c = c.Add(comp)
		}
		itemCols[i] = c
	}

	slack := max(gridSize-used-totalGap, 0)
	finalCols := assembleFlexCols(itemCols, gapCols, slack, containerStyle.JustifyContent)

	r := row.New()
	for _, c := range finalCols {
		r = r.Add(c)
	}
	return r
}

// flexRows is the new entry point. It handles flex-wrap, order, and *-reverse,
// returning one []core.Row per visual flex line. When flex-wrap is off (default)
// it returns exactly one row — identical to the old flexRow behavior.
func (tr *translator) flexRows(n *dom.Node, containerStyle *css.ComputedStyle) []core.Row {
	children := flexItems(n)
	if len(children) == 0 {
		return nil
	}

	gridSize := tr.gridSize
	if gridSize <= 0 {
		gridSize = defaultGridSize
	}

	// Sort by order (CSS order property, DOM index tiebreak).
	type orderedChild struct {
		node     *dom.Node
		domIndex int
	}
	ordered := make([]orderedChild, len(children))
	for i, c := range children {
		ordered[i] = orderedChild{node: c, domIndex: i}
	}
	styles := make([]*css.ComputedStyle, len(children))
	for i, c := range children {
		styles[i] = computeNodeStyle(tr.sheet, c, containerStyle)
	}

	sort.SliceStable(ordered, func(a, b int) bool {
		oa := styles[ordered[a].domIndex].Order
		ob := styles[ordered[b].domIndex].Order
		if oa != ob {
			return oa < ob
		}
		return ordered[a].domIndex < ordered[b].domIndex
	})

	// Apply reverse for row-reverse.
	if containerStyle.FlexDirection == "row-reverse" {
		for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
			ordered[i], ordered[j] = ordered[j], ordered[i]
		}
	}

	// Reorder parallel slices to match sorted order.
	sortedChildren := make([]*dom.Node, len(ordered))
	sortedStyles := make([]*css.ComputedStyle, len(ordered))
	for i, o := range ordered {
		sortedChildren[i] = o.node
		sortedStyles[i] = styles[o.domIndex]
	}

	// Determine if we should wrap.
	wrap := containerStyle.FlexWrap == "wrap" || containerStyle.FlexWrap == "wrap-reverse"

	var logicalRows [][]*dom.Node
	var logicalRowStyles [][]*css.ComputedStyle

	if !wrap {
		// Single row.
		logicalRows = [][]*dom.Node{sortedChildren}
		logicalRowStyles = [][]*css.ComputedStyle{sortedStyles}
	} else {
		// Greedy wrap: fill each row until adding the next item would exceed the grid.
		rowChildren := []*dom.Node{}
		rowStyles := []*css.ComputedStyle{}
		usedPct := 0.0
		for i, child := range sortedChildren {
			s := sortedStyles[i]
			pct := s.FlexBasisPct
			if pct <= 0 {
				pct = 100.0 / float64(gridSize) // default: equal share
			}
			if len(rowChildren) > 0 && usedPct+pct > 100.001 {
				// Start a new row.
				logicalRows = append(logicalRows, rowChildren)
				logicalRowStyles = append(logicalRowStyles, rowStyles)
				rowChildren = []*dom.Node{}
				rowStyles = []*css.ComputedStyle{}
				usedPct = 0
			}
			rowChildren = append(rowChildren, child)
			rowStyles = append(rowStyles, s)
			usedPct += pct
		}
		if len(rowChildren) > 0 {
			logicalRows = append(logicalRows, rowChildren)
			logicalRowStyles = append(logicalRowStyles, rowStyles)
		}
	}

	// Reverse row order for wrap-reverse.
	if containerStyle.FlexWrap == "wrap-reverse" {
		for i, j := 0, len(logicalRows)-1; i < j; i, j = i+1, j-1 {
			logicalRows[i], logicalRows[j] = logicalRows[j], logicalRows[i]
			logicalRowStyles[i], logicalRowStyles[j] = logicalRowStyles[j], logicalRowStyles[i]
		}
	}

	// Build each logical row into a core.Row.
	var result []core.Row
	for rowIdx, rowChildren := range logicalRows {
		rowItemStyles := logicalRowStyles[rowIdx]
		gapCols := tr.gapCols(containerStyle.ColumnGap, gridSize, len(rowChildren))
		totalGap := gapCols * max(0, len(rowChildren)-1)
		available := gridSize - totalGap
		if available < len(rowChildren) {
			gapCols = 0
			totalGap = 0
			available = gridSize
		}

		sizes := computeFlexSizes(rowItemStyles, available)
		sizes = bumpZerosWithoutOverflow(sizes, available)

		visualGap := 0.0
		if gapCols == 0 {
			visualGap = containerStyle.ColumnGap
		}

		itemCols := make([]core.Col, len(rowChildren))
		used := 0
		for i, child := range rowChildren {
			used += sizes[i]
			c := col.New(sizes[i])
			if comp := tr.flexItemContent(child, rowItemStyles[i]); comp != nil {
				if visualGap > 0 && i > 0 {
					comp = &marginBox{child: comp, marginLeft: visualGap}
				}
				c = c.Add(comp)
			}
			itemCols[i] = c
		}

		slack := max(gridSize-used-totalGap, 0)
		finalCols := assembleFlexCols(itemCols, gapCols, slack, containerStyle.JustifyContent)

		r := row.New()
		for _, c := range finalCols {
			r = r.Add(c)
		}
		result = append(result, r)
	}
	return result
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
		out = append(out, tr.blockRows(child)...)
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

// flexItemContent builds the component rendered inside a flex col.
// Leaf items (only inline content) render as RichText. Non-leaf items render
// their block children sequentially via flexCellContent so nested headings and
// paragraphs preserve their formatting.
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
		rt := richtext.New(runs, richTextPropsFromStyle(style))
		if shouldUseContainer(style) {
			r := row.New().Add(col.New().Add(rt))
			return &blockContainer{
				rows:          []core.Row{r},
				style:         blockCellStyle(style),
				paddingTop:    style.PaddingTop,
				paddingRight:  style.PaddingRight,
				paddingBottom: style.PaddingBottom,
				paddingLeft:   style.PaddingLeft,
			}
		}
		return rt
	}

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
	// When this flex item has its own background/border/padding, wrap the
	// children in a styled blockContainer so the styling spans them all.
	if shouldUseContainer(style) {
		return &blockContainer{
			rows:          subRows,
			style:         blockCellStyle(style),
			paddingTop:    style.PaddingTop,
			paddingRight:  style.PaddingRight,
			paddingBottom: style.PaddingBottom,
			paddingLeft:   style.PaddingLeft,
		}
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
