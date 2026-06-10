// Package translate — flex layout dispatch and item-content construction.
// Quantization, weight computation, and slack distribution live in flex_layout.go.
package translate

import (
	"context"
	"sort"
	"strings"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

const defaultGridSize = 12

type orderedFlexChild struct {
	node     *dom.Node
	domIndex int
}

// flexRows handles flex-wrap, order, and *-reverse, returning one []core.Row
// per visual flex line. When flex-wrap is off (default) it returns exactly one
// row.
func (tr *translator) flexRows(ctx context.Context, n *dom.Node, containerStyle *css.ComputedStyle) []core.Row {
	children := flexItems(n)
	if len(children) == 0 {
		return nil
	}

	gridSize := normalizedFlexGridSize(tr.gridSize)
	sortedChildren, sortedStyles := tr.sortedFlexItems(children, containerStyle)
	logicalRows, logicalRowStyles := flexLogicalRows(sortedChildren, sortedStyles, containerStyle, gridSize)
	if containerStyle.FlexWrap == "wrap-reverse" {
		reverseFlexLogicalRows(logicalRows, logicalRowStyles)
	}
	return tr.buildFlexRows(ctx, logicalRows, logicalRowStyles, containerStyle, gridSize)
}

func normalizedFlexGridSize(gridSize int) int {
	if gridSize <= 0 {
		return defaultGridSize
	}
	return gridSize
}

func (tr *translator) sortedFlexItems(
	children []*dom.Node,
	containerStyle *css.ComputedStyle,
) ([]*dom.Node, []*css.ComputedStyle) {
	ordered := make([]orderedFlexChild, len(children))
	for i, c := range children {
		ordered[i] = orderedFlexChild{node: c, domIndex: i}
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
	if containerStyle.FlexDirection == "row-reverse" {
		reverseOrderedFlexChildren(ordered)
	}
	return orderedFlexItems(ordered, styles)
}

func reverseOrderedFlexChildren(children []orderedFlexChild) {
	for i, j := 0, len(children)-1; i < j; i, j = i+1, j-1 {
		children[i], children[j] = children[j], children[i]
	}
}

func orderedFlexItems(
	ordered []orderedFlexChild,
	styles []*css.ComputedStyle,
) ([]*dom.Node, []*css.ComputedStyle) {
	sortedChildren := make([]*dom.Node, len(ordered))
	sortedStyles := make([]*css.ComputedStyle, len(ordered))
	for i, o := range ordered {
		sortedChildren[i] = o.node
		sortedStyles[i] = styles[o.domIndex]
	}
	return sortedChildren, sortedStyles
}

func flexLogicalRows(
	sortedChildren []*dom.Node,
	sortedStyles []*css.ComputedStyle,
	containerStyle *css.ComputedStyle,
	gridSize int,
) ([][]*dom.Node, [][]*css.ComputedStyle) {
	wrap := containerStyle.FlexWrap == "wrap" || containerStyle.FlexWrap == "wrap-reverse"
	var logicalRows [][]*dom.Node
	var logicalRowStyles [][]*css.ComputedStyle
	if !wrap {
		return [][]*dom.Node{sortedChildren}, [][]*css.ComputedStyle{sortedStyles}
	}
	return wrapFlexLogicalRows(sortedChildren, sortedStyles, gridSize, logicalRows, logicalRowStyles)
}

func wrapFlexLogicalRows(
	sortedChildren []*dom.Node,
	sortedStyles []*css.ComputedStyle,
	gridSize int,
	logicalRows [][]*dom.Node,
	logicalRowStyles [][]*css.ComputedStyle,
) ([][]*dom.Node, [][]*css.ComputedStyle) {
	rowChildren := make([]*dom.Node, 0, len(sortedChildren))
	rowStyles := make([]*css.ComputedStyle, 0, len(sortedChildren))
	usedPct := 0.0
	for i, child := range sortedChildren {
		style := sortedStyles[i]
		pct := style.FlexBasisPct
		if pct <= 0 {
			pct = 100.0 / float64(gridSize)
		}
		if len(rowChildren) > 0 && usedPct+pct > 100.001 {
			logicalRows = append(logicalRows, rowChildren)
			logicalRowStyles = append(logicalRowStyles, rowStyles)
			rowChildren = []*dom.Node{}
			rowStyles = []*css.ComputedStyle{}
			usedPct = 0
		}
		rowChildren = append(rowChildren, child)
		rowStyles = append(rowStyles, style)
		usedPct += pct
	}
	if len(rowChildren) > 0 {
		logicalRows = append(logicalRows, rowChildren)
		logicalRowStyles = append(logicalRowStyles, rowStyles)
	}
	return logicalRows, logicalRowStyles
}

func reverseFlexLogicalRows(logicalRows [][]*dom.Node, logicalRowStyles [][]*css.ComputedStyle) {
	for i, j := 0, len(logicalRows)-1; i < j; i, j = i+1, j-1 {
		logicalRows[i], logicalRows[j] = logicalRows[j], logicalRows[i]
		logicalRowStyles[i], logicalRowStyles[j] = logicalRowStyles[j], logicalRowStyles[i]
	}
}

func (tr *translator) buildFlexRows(
	ctx context.Context,
	logicalRows [][]*dom.Node,
	logicalRowStyles [][]*css.ComputedStyle,
	containerStyle *css.ComputedStyle,
	gridSize int,
) []core.Row {
	var result []core.Row
	for rowIdx, rowChildren := range logicalRows {
		r, ok := tr.buildFlexRow(ctx, rowChildren, logicalRowStyles[rowIdx], containerStyle, gridSize)
		if !ok {
			return nil
		}
		result = append(result, r)
	}
	return result
}

func (tr *translator) buildFlexRow(
	ctx context.Context,
	rowChildren []*dom.Node,
	rowItemStyles []*css.ComputedStyle,
	containerStyle *css.ComputedStyle,
	gridSize int,
) (core.Row, bool) {
	gapCols := tr.gapCols(containerStyle.ColumnGap, gridSize, len(rowChildren))
	totalGap := gapCols * max(0, len(rowChildren)-1)
	available := gridSize - totalGap
	if available < len(rowChildren) {
		gapCols = 0
		totalGap = 0
		available = gridSize
	}

	sizes := bumpZerosWithoutOverflow(computeFlexSizes(rowItemStyles, available), available)
	visualGap := flexVisualGap(containerStyle, gapCols)
	itemCols, used, ok := tr.flexItemCols(ctx, rowChildren, rowItemStyles, containerStyle, sizes, visualGap)
	if !ok {
		return nil, false
	}
	slack := max(gridSize-used-totalGap, 0)
	return rowFromCols(assembleFlexCols(itemCols, gapCols, slack, containerStyle.JustifyContent)), true
}

func flexVisualGap(containerStyle *css.ComputedStyle, gapCols int) float64 {
	if gapCols == 0 {
		return containerStyle.ColumnGap
	}
	return 0
}

func (tr *translator) flexItemCols(
	ctx context.Context,
	rowChildren []*dom.Node,
	rowItemStyles []*css.ComputedStyle,
	containerStyle *css.ComputedStyle,
	sizes []int,
	visualGap float64,
) ([]core.Col, int, bool) {
	itemCols := make([]core.Col, len(rowChildren))
	used := 0
	for i, child := range rowChildren {
		err := translationCanceled(ctx)
		if err != nil {
			tr.err = err
			return nil, 0, false
		}
		used += sizes[i]
		itemCols[i] = tr.flexItemCol(ctx, child, rowItemStyles[i], containerStyle, sizes[i], visualGap, i)
	}
	return itemCols, used, true
}

func (tr *translator) flexItemCol(
	ctx context.Context,
	child *dom.Node,
	itemStyle *css.ComputedStyle,
	containerStyle *css.ComputedStyle,
	size int,
	visualGap float64,
	index int,
) core.Col {
	c := col.New(size)
	comp := tr.flexItemContent(ctx, child, itemStyle)
	if comp == nil {
		return c
	}
	comp = flexItemCrossAxisBox(comp, containerStyle, itemStyle)
	if visualGap > 0 && index > 0 {
		comp = &marginBox{child: comp, marginLeft: visualGap}
	}
	return c.Add(comp)
}

func rowFromCols(cols []core.Col) core.Row {
	r := row.New()
	for _, c := range cols {
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

// isColumnDirection returns true for column or column-reverse.
func isColumnDirection(d string) bool {
	return d == "column" || d == "column-reverse"
}

// flexColumnRows handles flex-direction:column by emitting one row per child,
// optionally inserting empty spacer rows between them based on row-gap.
func (tr *translator) flexColumnRows(ctx context.Context, n *dom.Node, containerStyle *css.ComputedStyle) []core.Row {
	children := flexItems(n)
	if len(children) == 0 {
		return nil
	}
	if containerStyle.FlexDirection == "column-reverse" {
		children = reverseNodes(children)
	}
	gapMM := containerStyle.RowGap
	var out []core.Row
	for i, child := range children {
		err := translationCanceled(ctx)
		if err != nil {
			tr.err = err
			return nil
		}
		out = append(out, tr.blockRows(ctx, child)...)
		if gapMM > 0 && i < len(children)-1 {
			out = append(out, spacerRow(gapMM))
		}
	}
	return out
}

func reverseNodes(nodes []*dom.Node) []*dom.Node {
	reversed := make([]*dom.Node, len(nodes))
	for i, n := range nodes {
		reversed[len(nodes)-1-i] = n
	}
	return reversed
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
func (tr *translator) flexItemContent(ctx context.Context, n *dom.Node, style *css.ComputedStyle) core.Component {
	if n.Tag() == "" {
		text := strings.TrimSpace(n.TextContent())
		if text == "" {
			return nil
		}
		return richtext.New([]props.RichRun{{Text: text}})
	}

	if isLeafFlexItem(n) {
		runs := tr.inlineRunsStyled(n, blockInlineStyle(style))
		if len(runs) == 0 {
			return nil
		}
		applyBlockStyling(n, runs)
		rt := richtext.New(runs, richTextPropsFromStyle(style))
		if shouldUseContainer(style) {
			r := row.New().Add(col.New().Add(rt))
			return &blockContainer{
				rows:          []core.Row{r},
				style:         tr.blockCellStyle(style),
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
		err := translationCanceled(ctx)
		if err != nil {
			tr.err = err
			return nil
		}
		if isWhitespaceNode(c) {
			continue
		}
		subRows = append(subRows, tr.blockRows(ctx, c)...)
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
			style:         tr.blockCellStyle(style),
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
