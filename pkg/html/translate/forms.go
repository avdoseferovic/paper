package translate

import (
	"context"
	"maps"
	"strings"
	"unicode/utf8"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

const (
	fieldsetDefaultBorderWidth  = 0.75 * 0.352778
	fieldsetDefaultBorderRadius = 3 * 0.352778
	fieldsetDefaultPadding      = 8 * 0.352778
)

type formControlKind int

const (
	formControlText formControlKind = iota
	formControlButton
	formControlSelect
	formControlTextarea
	formControlChoice
)

const (
	inputTypeText     = "text"
	inputTypeCheckbox = "checkbox"
	inputTypeRadio    = "radio"
	inputTypeSubmit   = "submit"
	inputTypeReset    = "reset"
	inputTypeButton   = "button"
	inputTypePassword = "password"
)

func (tr *translator) formControlRowsContext(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	runCtx := tr.styledRunContextContext(ctx, style)
	runs, handled := formControlRuns(n, runCtx)
	if !handled || len(runs) == 0 {
		return nil
	}
	rtProp := richTextPropsFromStyle(style)
	rtProp.Top, rtProp.Right, rtProp.Bottom, rtProp.Left = 0, 0, 0, 0
	rt := richtext.New(runs, rtProp)
	if tr.anchorReg != nil {
		rt.WithAnchorRegistry(tr.anchorReg)
	}
	return []core.Row{row.New().Add(col.New().Add(rt))}
}

func (tr *translator) fieldsetRows(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	fieldsetStyle := fieldsetStyleWithDefaults(style)
	rows := tr.fieldsetChildRows(ctx, n, fieldsetStyle)
	return []core.Row{tr.buildContainerRowContext(ctx, fieldsetStyle, rows)}
}

func (tr *translator) fieldsetChildRows(ctx context.Context, n *dom.Node, style *css.ComputedStyle) []core.Row {
	var rows []core.Row
	for _, child := range n.Children() {
		if isWhitespaceNode(child) {
			continue
		}
		if child.Tag() == tagLegend {
			rows = append(rows, tr.legendRows(ctx, child, style)...)
			continue
		}
		rows = append(rows, tr.blockRowsWithParent(ctx, child, style)...)
	}
	return rows
}

func (tr *translator) legendRows(ctx context.Context, n *dom.Node, parentStyle *css.ComputedStyle) []core.Row {
	style := tr.computeBlockStyle(n, parentStyle)
	if strings.TrimSpace(style.FontWeight) == "" {
		style = cloneComputedStyle(style)
		style.FontWeight = "bold"
	}
	return []core.Row{tr.paragraphRowStyledContext(ctx, n, style)}
}

func fieldsetStyleWithDefaults(style *css.ComputedStyle) *css.ComputedStyle {
	style = cloneComputedStyle(style)
	if style == nil {
		style = css.NewComputedStyle()
	}
	if style.PaddingTop == 0 && style.PaddingRight == 0 && style.PaddingBottom == 0 && style.PaddingLeft == 0 {
		style.PaddingTop = fieldsetDefaultPadding
		style.PaddingRight = fieldsetDefaultPadding
		style.PaddingBottom = fieldsetDefaultPadding
		style.PaddingLeft = fieldsetDefaultPadding
	}
	if !fieldsetHasVisibleBorder(style) && !fieldsetBorderSuppressed(style) {
		gray := css.NewRGBColor(150, 150, 150)
		style.BorderTopWidth = fieldsetDefaultBorderWidth
		style.BorderRightWidth = fieldsetDefaultBorderWidth
		style.BorderBottomWidth = fieldsetDefaultBorderWidth
		style.BorderLeftWidth = fieldsetDefaultBorderWidth
		style.BorderTopStyle = cssValueSolid
		style.BorderRightStyle = cssValueSolid
		style.BorderBottomStyle = cssValueSolid
		style.BorderLeftStyle = cssValueSolid
		style.BorderTopColor = &gray
		style.BorderRightColor = cloneCSSColor(&gray)
		style.BorderBottomColor = cloneCSSColor(&gray)
		style.BorderLeftColor = cloneCSSColor(&gray)
	}
	if style.BorderRadius == 0 &&
		style.BorderRadiusTopLeft == 0 &&
		style.BorderRadiusTopRight == 0 &&
		style.BorderRadiusBottomRight == 0 &&
		style.BorderRadiusBottomLeft == 0 {
		style.BorderRadius = fieldsetDefaultBorderRadius
	}
	return style
}

func cloneComputedStyle(style *css.ComputedStyle) *css.ComputedStyle {
	if style == nil {
		return nil
	}
	clone := *style
	clone.Color = cloneCSSColor(style.Color)
	clone.BackgroundColor = cloneCSSColor(style.BackgroundColor)
	clone.BorderTopColor = cloneCSSColor(style.BorderTopColor)
	clone.BorderRightColor = cloneCSSColor(style.BorderRightColor)
	clone.BorderBottomColor = cloneCSSColor(style.BorderBottomColor)
	clone.BorderLeftColor = cloneCSSColor(style.BorderLeftColor)
	clone.TextDecorationColor = cloneCSSColor(style.TextDecorationColor)
	clone.OutlineColor = cloneCSSColor(style.OutlineColor)
	clone.BackgroundGradient = style.BackgroundGradient
	clone.TextShadow = cloneCSSShadow(style.TextShadow)
	clone.TextShadows = cloneCSSShadows(style.TextShadows)
	clone.BoxShadow = cloneCSSShadows(style.BoxShadow)
	if len(style.Vars) > 0 {
		clone.Vars = make(map[string]string, len(style.Vars))
		maps.Copy(clone.Vars, style.Vars)
	}
	return &clone
}

func fieldsetHasVisibleBorder(style *css.ComputedStyle) bool {
	if style == nil {
		return false
	}
	return visibleBorderSide(style.BorderTopWidth, style.BorderTopStyle) ||
		visibleBorderSide(style.BorderRightWidth, style.BorderRightStyle) ||
		visibleBorderSide(style.BorderBottomWidth, style.BorderBottomStyle) ||
		visibleBorderSide(style.BorderLeftWidth, style.BorderLeftStyle)
}

func fieldsetBorderSuppressed(style *css.ComputedStyle) bool {
	if style == nil {
		return false
	}
	return borderStyleSuppressesDefault(style.BorderTopStyle) &&
		borderStyleSuppressesDefault(style.BorderRightStyle) &&
		borderStyleSuppressesDefault(style.BorderBottomStyle) &&
		borderStyleSuppressesDefault(style.BorderLeftStyle)
}

func borderStyleSuppressesDefault(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case cssValueNone, cssValueHidden:
		return true
	default:
		return false
	}
}

func formControlRuns(n *dom.Node, ctx runContext) ([]props.RichRun, bool) {
	if n == nil {
		return nil, false
	}
	switch n.Tag() {
	case tagInput:
		return inputControlRuns(n, ctx)
	case tagButton:
		text := strings.TrimSpace(n.TextContent())
		if text == "" {
			text = "Button"
		}
		return []props.RichRun{formControlRun(n, text, ctx, formControlButton, false)}, true
	case tagSelect:
		return []props.RichRun{formControlRun(n, selectControlText(n), ctx, formControlSelect, false)}, true
	case tagTextArea:
		text, placeholder := textareaControlText(n)
		return []props.RichRun{formControlRun(n, text, ctx, formControlTextarea, placeholder)}, true
	default:
		return nil, false
	}
}

func inputControlRuns(n *dom.Node, ctx runContext) ([]props.RichRun, bool) {
	inputType := strings.ToLower(strings.TrimSpace(n.Attr("type")))
	if inputType == "" {
		inputType = inputTypeText
	}
	switch inputType {
	case cssValueHidden:
		return nil, true
	case inputTypeCheckbox:
		if hasAttr(n, "checked") {
			return []props.RichRun{formControlRun(n, "☑", ctx, formControlChoice, false)}, true
		}
		return []props.RichRun{formControlRun(n, "☐", ctx, formControlChoice, false)}, true
	case inputTypeRadio:
		if hasAttr(n, "checked") {
			return []props.RichRun{formControlRun(n, "◉", ctx, formControlChoice, false)}, true
		}
		return []props.RichRun{formControlRun(n, "○", ctx, formControlChoice, false)}, true
	case inputTypeSubmit:
		return []props.RichRun{formControlRun(n, inputValueOrDefault(n, "Submit"), ctx, formControlButton, false)}, true
	case inputTypeReset:
		return []props.RichRun{formControlRun(n, inputValueOrDefault(n, "Reset"), ctx, formControlButton, false)}, true
	case inputTypeButton:
		return []props.RichRun{formControlRun(n, inputValueOrDefault(n, "Button"), ctx, formControlButton, false)}, true
	case inputTypePassword:
		text, placeholder := inputText(n)
		if !placeholder && text != "" {
			text = strings.Repeat("•", utf8.RuneCountInString(text))
		}
		return []props.RichRun{formControlRun(n, text, ctx, formControlText, placeholder)}, true
	default:
		text, placeholder := inputText(n)
		return []props.RichRun{formControlRun(n, text, ctx, formControlText, placeholder)}, true
	}
}

func inputValueOrDefault(n *dom.Node, fallback string) string {
	if value := strings.TrimSpace(n.Attr("value")); value != "" {
		return value
	}
	return fallback
}

func inputText(n *dom.Node) (string, bool) {
	if value := n.Attr("value"); value != "" {
		return value, false
	}
	if placeholder := n.Attr("placeholder"); placeholder != "" {
		return placeholder, true
	}
	return "", false
}

func textareaControlText(n *dom.Node) (string, bool) {
	text := strings.TrimSpace(n.TextContent())
	if text != "" {
		return text, false
	}
	if placeholder := n.Attr("placeholder"); placeholder != "" {
		return placeholder, true
	}
	return "", false
}

func selectControlText(n *dom.Node) string {
	first, selected := selectOptionText(n.Children())
	text := selected
	if text == "" {
		text = first
	}
	if text == "" {
		return "▾"
	}
	return text + " ▾"
}

func selectOptionText(nodes []*dom.Node) (string, string) {
	first := ""
	selected := ""
	for _, child := range nodes {
		switch child.Tag() {
		case tagOption:
			text := strings.TrimSpace(child.TextContent())
			if first == "" {
				first = text
			}
			if hasAttr(child, "selected") {
				return first, text
			}
		case tagOptgroup:
			groupFirst, groupSelected := selectOptionText(child.Children())
			if first == "" {
				first = groupFirst
			}
			if groupSelected != "" {
				return first, groupSelected
			}
		default:
			continue
		}
	}
	return first, selected
}

func formControlRun(n *dom.Node, text string, ctx runContext, kind formControlKind, placeholder bool) props.RichRun {
	runCtx := ctx
	if placeholder && ctx.pseudoResolver != nil {
		if pseudo := ctx.pseudoResolver(n, ctx.style, "placeholder"); pseudo != nil {
			runCtx.style = pseudo
		}
	}
	run := richRunFromContext(text, runCtx)
	applyFormControlDefaults(&run, kind, ctx.style, placeholder)
	return run
}

func applyFormControlDefaults(run *props.RichRun, kind formControlKind, style *css.ComputedStyle, placeholder bool) {
	if run == nil {
		return
	}
	if placeholder && run.Color == nil {
		run.Color = &props.Color{Red: 117, Green: 117, Blue: 117}
	}
	if kind == formControlChoice {
		return
	}
	if run.Background == nil {
		switch kind {
		case formControlButton:
			run.Background = &props.Color{Red: 245, Green: 245, Blue: 245}
		case formControlText, formControlSelect, formControlTextarea:
			run.Background = props.CloneColor(&props.WhiteColor)
		case formControlChoice:
		}
	}
	if run.BorderColor == nil {
		run.BorderColor = &props.Color{Red: 150, Green: 150, Blue: 150}
	}
	if run.BorderWidth == 0 {
		run.BorderWidth = 0.25
	}
	if run.BgRadius == 0 {
		run.BgRadius = 0.8
	}
	applyFormControlPadding(run, kind, style)
	applyFormControlDimensions(run, kind, style)
}

func applyFormControlPadding(run *props.RichRun, kind formControlKind, style *css.ComputedStyle) {
	padX, padY := 2.0, 1.0
	if kind == formControlButton {
		padX, padY = 2.5, 1.0
	}
	left, right, top, bottom := padX, padX, padY, padY
	if style != nil {
		if style.PaddingLeft > 0 {
			left = style.PaddingLeft
		}
		if style.PaddingRight > 0 {
			right = style.PaddingRight
		}
		if style.PaddingTop > 0 {
			top = style.PaddingTop
		}
		if style.PaddingBottom > 0 {
			bottom = style.PaddingBottom
		}
	}
	if run.BgPadLeft == 0 {
		run.BgPadLeft = left
	}
	if run.BgPadRight == 0 {
		run.BgPadRight = right
	}
	if run.BgPadY == 0 {
		run.BgPadY = max(top, bottom)
	}
}

func applyFormControlDimensions(run *props.RichRun, kind formControlKind, style *css.ComputedStyle) {
	if style != nil {
		if style.Width > 0 && run.InlineBoxWidth == 0 {
			run.InlineBoxWidth = cssContentBoxWidth(style)
		}
		if style.Height > 0 && run.InlineBoxHeight == 0 {
			run.InlineBoxHeight = cssContentBoxHeight(style)
		}
	}
	if run.InlineBoxWidth == 0 {
		switch kind {
		case formControlText:
			run.InlineBoxWidth = 40
		case formControlSelect:
			run.InlineBoxWidth = 35
		case formControlTextarea:
			run.InlineBoxWidth = 70
		case formControlButton, formControlChoice:
		}
	}
	if kind == formControlTextarea && run.InlineBoxHeight == 0 {
		run.InlineBoxHeight = 16
	}
}
