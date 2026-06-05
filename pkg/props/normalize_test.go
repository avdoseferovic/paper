package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/barcode"
	"github.com/avdoseferovic/paper/pkg/consts/breakline"
	"github.com/avdoseferovic/paper/pkg/consts/fontfamily"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/consts/linestyle"
	"github.com/avdoseferovic/paper/pkg/consts/orientation"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/stretchr/testify/assert"
)

func TestColorHelpersReturnIndependentValues(t *testing.T) {
	t.Parallel()

	black := props.Black()
	black.Red = 99

	assert.Equal(t, 0, props.BlackColor.Red)
	assert.Equal(t, 0, props.Black().Red)
}

func TestNormalizeTextReturnsDefaultedCopy(t *testing.T) {
	t.Parallel()

	input := props.Text{Left: -10, Top: -5}
	font := &props.Font{
		Family: fontfamily.Helvetica,
		Style:  fontstyle.Bold,
		Size:   11,
		Color:  &props.RedColor,
	}

	normalized := props.NormalizeText(input, font)

	assert.Equal(t, props.Text{Left: -10, Top: -5}, input)
	assert.Equal(t, fontfamily.Helvetica, normalized.Family)
	assert.Equal(t, fontstyle.Bold, normalized.Style)
	assert.Equal(t, 11.0, normalized.Size)
	assert.Equal(t, align.Left, normalized.Align)
	assert.Equal(t, breakline.EmptySpaceStrategy, normalized.BreakLineStrategy)
	assert.Equal(t, 0.0, normalized.Left)
	assert.Equal(t, 0.0, normalized.Top)
	assert.NotSame(t, font.Color, normalized.Color)
	assert.Equal(t, *font.Color, *normalized.Color)
}

func TestNormalizeShapePropsReturnDefaultedCopies(t *testing.T) {
	t.Parallel()

	rect := props.Rect{Left: -1, Top: -2, Percent: -5, Center: true}
	normalizedRect := props.NormalizeRect(rect)
	assert.Equal(t, props.Rect{Left: -1, Top: -2, Percent: -5, Center: true}, rect)
	assert.Equal(t, 100.0, normalizedRect.Percent)
	assert.Equal(t, 0.0, normalizedRect.Left)
	assert.Equal(t, 0.0, normalizedRect.Top)

	checkbox := props.Checkbox{Left: -1, Top: -2, Size: -3}
	normalizedCheckbox := props.NormalizeCheckbox(checkbox)
	assert.Equal(t, props.Checkbox{Left: -1, Top: -2, Size: -3}, checkbox)
	assert.Equal(t, 5.0, normalizedCheckbox.Size)
	assert.Equal(t, 0.0, normalizedCheckbox.Left)
	assert.Equal(t, 0.0, normalizedCheckbox.Top)

	codeProp := props.Barcode{Left: -1, Top: -2, Percent: -3}
	normalizedBarcode := props.NormalizeBarcode(codeProp)
	assert.Equal(t, props.Barcode{Left: -1, Top: -2, Percent: -3}, codeProp)
	assert.Equal(t, 100.0, normalizedBarcode.Percent)
	assert.Equal(t, props.Proportion{Width: 1, Height: 0.2}, normalizedBarcode.Proportion)
	assert.Equal(t, barcode.Code128, normalizedBarcode.Type)
}

func TestNormalizeColorBearingPropsClonePointers(t *testing.T) {
	t.Parallel()

	lineColor := &props.Color{Red: 1, Green: 2, Blue: 3}
	line := props.NormalizeLine(props.Line{Color: lineColor, OffsetPercent: 1000})
	assert.Equal(t, linestyle.Solid, line.Style)
	assert.Equal(t, orientation.Horizontal, line.Orientation)
	assert.NotSame(t, lineColor, line.Color)
	lineColor.Red = 99
	assert.Equal(t, 1, line.Color.Red)

	fontColor := &props.Color{Red: 4, Green: 5, Blue: 6}
	normalizedSignature := props.NormalizeSignature(props.Signature{
		FontColor: fontColor,
		LineColor: lineColor,
	}, fontfamily.Arial)
	assert.NotSame(t, fontColor, normalizedSignature.FontColor)
	assert.NotSame(t, lineColor, normalizedSignature.LineColor)
	fontColor.Red = 99
	assert.Equal(t, 4, normalizedSignature.FontColor.Red)

	background := &props.Color{Red: 7, Green: 8, Blue: 9}
	shadowColor := &props.Color{Red: 10, Green: 11, Blue: 12}
	defaultFont := &props.Font{Family: fontfamily.Helvetica, Style: fontstyle.Italic, Size: 13, Color: fontColor}
	run := props.NormalizeRichRun(props.RichRun{
		Color:      fontColor,
		Background: background,
		TextShadow: &props.Shadow{Color: shadowColor},
	}, defaultFont)
	assert.Equal(t, fontfamily.Helvetica, run.Family)
	assert.Equal(t, fontstyle.Italic, run.Style)
	assert.Equal(t, 13.0, run.Size)
	assert.NotSame(t, fontColor, run.Color)
	assert.NotSame(t, background, run.Background)
	assert.NotSame(t, shadowColor, run.TextShadow.Color)
	background.Red = 99
	shadowColor.Red = 99
	assert.Equal(t, 7, run.Background.Red)
	assert.Equal(t, 10, run.TextShadow.Color.Red)
}

func TestNormalizeRichTextAndCloneCellCopyNestedPointers(t *testing.T) {
	t.Parallel()

	rich := props.NormalizeRichText(props.RichText{})
	assert.Equal(t, align.Left, rich.Align)
	assert.Equal(t, 1.0, rich.LineHeight)
	assert.Equal(t, breakline.EmptySpaceStrategy, rich.BreakLineStrategy)
	assert.Equal(t, "normal", rich.WhiteSpace)

	alpha := 0.5
	background := &props.Color{Red: 1, Green: 2, Blue: 3, Alpha: &alpha}
	border := &props.Color{Red: 4, Green: 5, Blue: 6}
	shadowColor := &props.Color{Red: 7, Green: 8, Blue: 9}
	cell := &props.Cell{
		BackgroundColor: background,
		BorderColor:     border,
		BorderTopColor:  border,
		OutlineColor:    border,
		BackgroundGradient: &props.Gradient{
			Stops: []props.GradientStop{{Color: props.Red(), Position: 0}},
		},
		BoxShadow: []props.Shadow{{Color: shadowColor}},
	}

	clone := props.CloneCell(cell)
	assert.NotSame(t, cell, clone)
	assert.NotSame(t, background, clone.BackgroundColor)
	assert.NotSame(t, border, clone.BorderColor)
	assert.NotSame(t, shadowColor, clone.BoxShadow[0].Color)
	assert.NotSame(t, cell.BackgroundGradient, clone.BackgroundGradient)

	background.Red = 99
	border.Red = 99
	shadowColor.Red = 99
	cell.BackgroundGradient.Stops[0].Color.Red = 99

	assert.Equal(t, 1, clone.BackgroundColor.Red)
	assert.Equal(t, 4, clone.BorderColor.Red)
	assert.Equal(t, 7, clone.BoxShadow[0].Color.Red)
	assert.Equal(t, 255, clone.BackgroundGradient.Stops[0].Color.Red)
	assert.Equal(t, 0.5, *clone.BackgroundColor.Alpha)
}
