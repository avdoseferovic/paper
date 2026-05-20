// Package translate — flex layout dispatch and item-content construction.
// Quantization, weight computation, and slack distribution live in flex_layout.go.
package translate

import (
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

	itemCols := make([]core.Col, len(children))
	used := 0
	for i, child := range children {
		used += sizes[i]
		c := col.New(sizes[i])
		if comp := tr.flexItemContent(child, styles[i]); comp != nil {
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
		return richtext.New(runs, props.RichText{})
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
