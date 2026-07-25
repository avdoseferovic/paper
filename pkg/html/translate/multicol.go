package translate

import (
	"context"

	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func (tr *translator) resolvedColumnCount(style *css.ComputedStyle) int {
	if style == nil {
		return 0
	}
	if style.ColumnCount > 1 {
		return style.ColumnCount
	}
	if style.ColumnWidth <= 0 {
		return 0
	}
	width := tr.availableContentWidth()
	if style.Width > 0 {
		width = style.Width
	}
	gap := style.ColumnGap
	if width <= 0 || style.ColumnWidth <= 0 {
		return 0
	}
	count := int((width + gap) / (style.ColumnWidth + gap))
	if count > 1 {
		return count
	}
	return 0
}

func (tr *translator) multiColumnRows(
	ctx context.Context,
	n *dom.Node,
	style *css.ComputedStyle,
	columnCount int,
) []core.Row {
	var out []core.Row
	var segment []core.Row
	flushSegment := func() {
		if len(segment) == 0 {
			return
		}
		out = append(out, tr.multiColumnSegmentRow(style, columnCount, segment))
		segment = nil
	}

	for _, child := range n.Children() {
		err := translationCanceled(ctx)
		if err != nil {
			tr.err = err
			return nil
		}
		if isWhitespaceNode(child) {
			continue
		}
		childStyle := tr.computeBlockStyle(child, style)
		if isDisplayNone(child) || childStyle.Display == displayNone {
			continue
		}
		childRows := tr.blockRowsWithParent(ctx, child, style)
		if childStyle.ColumnSpan == cssValueAll {
			flushSegment()
			out = append(out, childRows...)
			continue
		}
		segment = append(segment, childRows...)
	}
	flushSegment()
	return out
}

func (tr *translator) multiColumnSegmentRow(
	style *css.ComputedStyle,
	columnCount int,
	rows []core.Row,
) core.Row {
	gridSize := normalizedFlexGridSize(tr.gridSize)
	gapCols := tr.gapCols(style.ColumnGap, gridSize, columnCount)
	totalGap := gapCols * max(0, columnCount-1)
	available := gridSize - totalGap
	if available < columnCount {
		gapCols = 0
		available = gridSize
	}

	weights := make([]float64, columnCount)
	for i := range weights {
		weights[i] = 1
	}
	sizes := bumpZerosWithoutOverflow(layout.Hamilton(weights, available), available)
	r := newBalancedMultiColumnRow(columnCount, gapCols, sizes, rows)
	return newColumnRuleRow(
		r,
		columnCount,
		gapCols,
		style.ColumnRuleWidth,
		style.ColumnRuleStyle,
		toPropsColor(style.ColumnRuleColor, effectiveOpacity(style)),
	)
}

func distributeColumnRows(rows []core.Row, columnCount int) [][]core.Row {
	columns := make([][]core.Row, columnCount)
	if columnCount <= 0 || len(rows) == 0 {
		return columns
	}
	chunkSize := (len(rows) + columnCount - 1) / columnCount
	for i, r := range rows {
		column := i / chunkSize
		if column >= columnCount {
			column = columnCount - 1
		}
		columns[column] = append(columns[column], r)
	}
	return columns
}
