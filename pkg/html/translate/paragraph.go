package translate

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/line"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// paragraphRow converts a block element with inline content into a single auto-height row.
// When the computed style sets CSS padding, it is passed through as the richtext's
// Top/Right/Bottom/Left offsets so the text is inset from the styled background's
// edges instead of butting against them.
func (tr *translator) paragraphRowStyled(n *dom.Node, style *css.ComputedStyle) core.Row {
	runs := tr.inlineRunsStyled(n, blockInlineStyle(style))
	if len(runs) == 0 {
		runs = []props.RichRun{{Text: ""}}
	}
	// User CSS first, then heading-default fallback. applyBlockStyling only
	// sets runs[i].Size when it's still 0, so applying inline CSS first lets a
	// user `h2 { font-size: 12pt }` override the 20pt heading default.
	applyBlockStyling(n, runs)
	rtProp := richTextPropsFromStyle(style)
	if tr.outlineFromHeadings {
		if level, ok := headingOutlineLevel(n.Tag()); ok {
			rtProp.Outline = &props.Outline{Level: level}
		}
	}
	rt := richtext.New(runs, rtProp)
	if tr.anchorReg != nil {
		rt.WithAnchorRegistry(tr.anchorReg)
	}
	c := col.New().Add(rt)
	r := row.New().Add(c)
	if cellStyle := tr.blockCellStyle(style); cellStyle != nil {
		r = r.WithStyle(cellStyle)
	}
	return r
}

func richTextPropsFromStyle(style *css.ComputedStyle) props.RichText {
	if style == nil {
		return props.RichText{}
	}
	rt := props.RichText{
		Top:             style.PaddingTop,
		Right:           style.PaddingRight,
		Bottom:          style.PaddingBottom,
		Left:            style.PaddingLeft,
		Align:           richTextAlignFromCSS(style.TextAlign),
		FirstLineIndent: style.TextIndent,
		WhiteSpace:      style.WhiteSpace,
	}
	if style.LineHeight > 0 && style.LineHeight != 1 {
		rt.LineHeight = style.LineHeight
	}
	return rt
}

func richTextAlignFromCSS(value string) consts.Align {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case flexAlignCenter:
		return consts.AlignCenter
	case "right", "end":
		return consts.AlignRight
	case "justify":
		return consts.AlignJustify
	default:
		return ""
	}
}

// styledHrRow honours border-top-width, border-top-style, color on the <hr>
// element. Defaults match the original hrRow behaviour when no style is set.
func (tr *translator) styledHrRowWithStyle(_ *dom.Node, style *css.ComputedStyle) core.Row {
	lineProp := props.Line{}
	if style.BorderTopWidth > 0 {
		lineProp.Thickness = style.BorderTopWidth
	}
	if c := style.BorderTopColor; c != nil {
		lineProp.Color = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	} else if style.Color != nil {
		lineProp.Color = &props.Color{Red: style.Color.R, Green: style.Color.G, Blue: style.Color.B}
	}
	lineProp.Style = cssBorderStyleToLineStyle(style.BorderTopStyle)
	l := line.New(lineProp)
	c := col.New().Add(l)
	h := 1.0
	if style.BorderTopWidth > 0 {
		h = style.BorderTopWidth + 0.5
	}
	return row.New(h).Add(c)
}

// wrapTextRow handles raw text nodes at block level.
func wrapTextRowStyled(text string, style *css.ComputedStyle) []core.Row {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	run := props.RichRun{Text: text}
	applyInlineStyleToRun(style, &run)
	rt := richtext.New([]props.RichRun{run})
	c := col.New().Add(rt)
	return []core.Row{row.New().Add(c)}
}

// applyBlockStyling applies block-level heading defaults to the first run.
func applyBlockStyling(n *dom.Node, runs []props.RichRun) {
	tag := n.Tag()
	headingSizes := map[string]float64{
		"h1": 24, "h2": 20, "h3": 16, "h4": 14, "h5": 12, "h6": 10,
	}
	if size, ok := headingSizes[tag]; ok {
		for i := range runs {
			if runs[i].Size == 0 {
				runs[i].Size = size
			}
			if runs[i].Style == "" {
				runs[i].Style = fontstyle.Bold
			}
		}
	}
}
