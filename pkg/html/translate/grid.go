package translate

import (
	"context"
	"strconv"
	"strings"

	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

type gridTrack struct {
	fixed float64
	flex  float64
}

type gridPlacement struct {
	child *dom.Node
	style *css.ComputedStyle
	row   int
	start int
	end   int
}

func (tr *translator) gridRows(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	children, styles := tr.gridItems(n, style)
	if len(children) == 0 {
		return nil
	}
	gridSize := normalizedFlexGridSize(tr.gridSize)
	tracks := parseGridTracks(style.GridTemplateColumns, style.FontSize, tr.flexContentWidth())
	if len(tracks) == 0 {
		tracks = []gridTrack{{flex: 1}}
	}
	columnCount := len(tracks)
	gapCols := tr.gapCols(style.ColumnGap, gridSize, columnCount)
	totalGap := gapCols * max(0, columnCount-1)
	available := gridSize - totalGap
	if available < columnCount {
		gapCols = 0
		totalGap = 0
		available = gridSize
	}
	sizes := gridTrackUnits(tracks, available, tr.flexContentWidth())
	placements := gridPlacements(children, styles, columnCount, style.GridTemplateAreas)
	rowTracks := parseGridTracks(style.GridTemplateRows, style.FontSize, style.Height)
	autoRowTracks := parseGridTracks(style.GridAutoRows, style.FontSize, style.Height)
	var rows []core.Row
	for rowIdx := range placements {
		rowHeight := gridRowTrackHeight(rowIdx, rowTracks, autoRowTracks)
		r := tr.gridRow(ctx, placements[rowIdx], sizes, gapCols, columnCount, rowHeight, style)
		if r != nil {
			if len(rows) > 0 && style.RowGap > 0 {
				rows = append(rows, spacerRow(style.RowGap))
			}
			rows = append(rows, r)
		}
	}
	_ = totalGap
	return rows
}

func (tr *translator) gridItems(n *dom.Node, parentStyle *css.ComputedStyle) ([]*dom.Node, []*css.ComputedStyle) {
	var children []*dom.Node
	var styles []*css.ComputedStyle
	for _, child := range n.Children() {
		if isWhitespaceNode(child) {
			continue
		}
		style := tr.computeBlockStyle(child, parentStyle)
		if isDisplayNone(child) || style.Display == displayNone {
			continue
		}
		children = append(children, child)
		styles = append(styles, style)
	}
	return children, styles
}

func (tr *translator) gridRow(
	ctx context.Context,
	placements []gridPlacement,
	sizes []int,
	gapCols int,
	columnCount int,
	rowHeight float64,
	containerStyle *css.ComputedStyle,
) core.Row {
	var cols []core.Col
	starts := map[int]gridPlacement{}
	for _, placement := range placements {
		starts[placement.start] = placement
	}
	for track := 0; track < columnCount; {
		if track > 0 && gapCols > 0 {
			cols = append(cols, col.New(gapCols))
		}
		if placement, ok := starts[track]; ok {
			err := translationCanceled(ctx)
			if err != nil {
				tr.err = err
				return nil
			}
			c := col.New(spannedGridUnits(sizes, placement.start, placement.end, gapCols))
			if comp := tr.flexItemContent(ctx, placement.child, placement.style); comp != nil {
				comp = tr.gridItemInlineAxisBox(comp, placement.child, containerStyle, placement.style)
				comp = flexItemCrossAxisBox(comp, containerStyle, placement.style)
				c = c.Add(comp)
			}
			cols = append(cols, c)
			track = placement.end
			continue
		}
		cols = append(cols, col.New(sizes[track]))
		track++
	}
	return rowFromColsWithHeight(cols, rowHeight)
}

func (tr *translator) gridItemInlineAxisBox(
	child core.Component,
	n *dom.Node,
	containerStyle *css.ComputedStyle,
	itemStyle *css.ComputedStyle,
) core.Component {
	if child == nil || containerStyle == nil {
		return child
	}
	align := normalizeGridInlineAxisAlign(containerStyle.JustifyItems)
	if align == "" {
		return child
	}
	width, ok := tr.estimateFlexOuterBasis(n, itemStyle)
	if !ok || width <= 0 {
		return child
	}
	return &inlineAxisBox{child: child, align: align, width: width}
}

func normalizeGridInlineAxisAlign(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case flexLogicalStart:
		return flexAlignStart
	case flexLogicalEnd:
		return flexAlignEnd
	case flexAlignCenter, flexAlignEnd, flexAlignStart:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func gridPlacements(
	children []*dom.Node,
	styles []*css.ComputedStyle,
	columnCount int,
	templateAreas [][]string,
) [][]gridPlacement {
	var rows [][]gridPlacement
	var occupied [][]bool
	ensureRow := func(row int) {
		for len(occupied) <= row {
			occupied = append(occupied, make([]bool, columnCount))
			rows = append(rows, nil)
		}
	}
	for i, child := range children {
		style := styles[i]
		row, start, span := explicitGridPlacement(style, columnCount, templateAreas)
		switch {
		case start >= 0 && row >= 0 && gridSpanFits(occupied, row, start, span, columnCount):
		case start >= 0:
			row = nextGridRowForStart(occupied, start, span, columnCount)
		default:
			row, start = nextGridAutoSlot(occupied, span, columnCount)
		}
		ensureRow(row)
		end := min(start+span, columnCount)
		for colIdx := start; colIdx < end; colIdx++ {
			occupied[row][colIdx] = true
		}
		rows[row] = append(rows[row], gridPlacement{
			child: child,
			style: style,
			row:   row,
			start: start,
			end:   end,
		})
	}
	return rows
}

func nextGridRowForStart(occupied [][]bool, start, span, columnCount int) int {
	for row := 0; ; row++ {
		if gridSpanFits(occupied, row, start, span, columnCount) {
			return row
		}
	}
}

func explicitGridPlacement(style *css.ComputedStyle, columnCount int, templateAreas [][]string) (int, int, int) {
	row := -1
	start := -1
	span := 1
	if style == nil {
		return row, start, span
	}
	if areaStart, areaEnd, areaRow, ok := resolveGridArea(style.GridArea, templateAreas); ok {
		row = areaRow
		start = areaStart
		span = areaEnd - areaStart
	}
	if style.GridRowStart > 0 {
		row = style.GridRowStart - 1
	}
	if style.GridColumnStart > 0 {
		start = style.GridColumnStart - 1
	}
	if style.GridColumnEnd > 0 {
		switch {
		case style.GridColumnStart > 0 && style.GridColumnEnd > style.GridColumnStart:
			span = style.GridColumnEnd - style.GridColumnStart
		case style.GridColumnStart == 0:
			span = style.GridColumnEnd
		}
	}
	span = max(span, 1)
	span = min(span, columnCount)
	if start >= columnCount {
		start = columnCount - 1
	}
	return row, start, span
}

func resolveGridArea(name string, areas [][]string) (int, int, int, bool) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || len(areas) == 0 {
		return 0, 0, 0, false
	}
	minCol, maxCol := -1, -1
	minRow := -1
	for rowIdx, row := range areas {
		for colIdx, area := range row {
			if area != name {
				continue
			}
			if minRow < 0 || rowIdx < minRow {
				minRow = rowIdx
			}
			if minCol < 0 || colIdx < minCol {
				minCol = colIdx
			}
			if maxCol < 0 || colIdx > maxCol {
				maxCol = colIdx
			}
		}
	}
	if minRow < 0 {
		return 0, 0, 0, false
	}
	return minCol, maxCol + 1, minRow, true
}

func nextGridAutoSlot(occupied [][]bool, span, columnCount int) (int, int) {
	span = min(span, columnCount)
	for row := 0; ; row++ {
		for start := 0; start+span <= columnCount; start++ {
			if gridSpanFits(occupied, row, start, span, columnCount) {
				return row, start
			}
		}
	}
}

func gridSpanFits(occupied [][]bool, row, start, span, columnCount int) bool {
	if row < 0 || start < 0 || span < 1 || start+span > columnCount {
		return false
	}
	if row >= len(occupied) {
		return true
	}
	for colIdx := start; colIdx < start+span; colIdx++ {
		if occupied[row][colIdx] {
			return false
		}
	}
	return true
}

func spannedGridUnits(sizes []int, start, end, gapCols int) int {
	units := 0
	for i := start; i < end && i < len(sizes); i++ {
		units += sizes[i]
		if i > start {
			units += gapCols
		}
	}
	return max(units, 1)
}

func gridRowTrackHeight(rowIdx int, templateRows, autoRows []gridTrack) float64 {
	if rowIdx < len(templateRows) {
		return templateRows[rowIdx].fixed
	}
	if len(autoRows) == 0 {
		return 0
	}
	autoIdx := (rowIdx - len(templateRows)) % len(autoRows)
	return autoRows[autoIdx].fixed
}

func parseGridTracks(value string, fontSize, contentWidth float64) []gridTrack {
	value = expandGridRepeat(strings.ToLower(strings.TrimSpace(value)))
	var tracks []gridTrack
	for _, token := range splitTopLevelFields(value) {
		switch {
		case token == "", token == cssValueNone:
			continue
		case token == cssValueAuto:
			tracks = append(tracks, gridTrack{flex: 1})
		case strings.HasSuffix(token, "fr"):
			flex, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(token, "fr")), 64)
			if err != nil || flex <= 0 {
				flex = 1
			}
			tracks = append(tracks, gridTrack{flex: flex})
		default:
			fixed := css.ParseLengthCtx(token, fontSize, contentWidth)
			if fixed > 0 {
				tracks = append(tracks, gridTrack{fixed: fixed})
			} else {
				tracks = append(tracks, gridTrack{flex: 1})
			}
		}
	}
	return tracks
}

func gridTrackUnits(tracks []gridTrack, available int, contentWidth float64) []int {
	if len(tracks) == 0 || available <= 0 {
		return nil
	}
	sizes := make([]int, len(tracks))
	fixedTotal := 0
	var flexIndices []int
	var flexWeights []float64
	for i, track := range tracks {
		if track.fixed > 0 && contentWidth > 0 {
			units := max(int(track.fixed/contentWidth*float64(available)+0.5), 1)
			sizes[i] = units
			fixedTotal += units
			continue
		}
		weight := track.flex
		if weight <= 0 {
			weight = 1
		}
		flexIndices = append(flexIndices, i)
		flexWeights = append(flexWeights, weight)
	}
	if fixedTotal > available {
		weights := make([]float64, len(tracks))
		for i, size := range sizes {
			weights[i] = float64(size)
		}
		return bumpZerosWithoutOverflow(layout.ProportionalUnits(weights, available), available)
	}
	remaining := available - fixedTotal
	if len(flexIndices) > 0 {
		flexSizes := bumpZerosWithoutOverflow(layout.ProportionalUnits(flexWeights, remaining), remaining)
		for i, idx := range flexIndices {
			sizes[idx] = flexSizes[i]
		}
	}
	return bumpZerosWithoutOverflow(sizes, available)
}

func expandGridRepeat(value string) string {
	for {
		idx := strings.Index(value, "repeat(")
		if idx < 0 {
			return value
		}
		closeIdx := matchingCloseParen(value, idx+len("repeat"))
		if closeIdx < 0 {
			return value
		}
		inner := value[idx+len("repeat(") : closeIdx]
		parts := splitTopLevelCommas(inner)
		if len(parts) != 2 {
			return value
		}
		count, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || count <= 0 {
			return value
		}
		repeated := strings.TrimSpace(parts[1])
		value = value[:idx] + strings.TrimSpace(strings.Repeat(repeated+" ", count)) + value[closeIdx+1:]
	}
}
