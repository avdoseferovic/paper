// Block margin, page-break, and page-control helpers for the translator
// walker (split from translate.go).
package translate

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func applyBlockMargins(n *dom.Node, style *css.ComputedStyle, rows []core.Row) []core.Row {
	if style == nil || n.Tag() == "" || len(rows) == 0 || selfHandlesMargin(n.Tag()) {
		return rows
	}
	// Honor block-level margins outside the element's content box. Horizontal
	// margins narrow/offset each produced content row; vertical margins remain
	// flow spacers so they can collapse with adjacent block margins.
	if style.MarginLeft != 0 || style.MarginRight != 0 {
		rows = wrapRowsWithHorizontalMargins(rows, style.MarginLeft, style.MarginRight)
	}
	if style.MarginTop > 0 {
		rows = append([]core.Row{newMarginSpacer(style.MarginTop)}, rows...)
	}
	if style.MarginBottom > 0 {
		rows = append(rows, newMarginSpacer(style.MarginBottom))
	}
	// CSS margins collapse: adjacent/last-child margins reduce to the larger,
	// they do not sum. Merge touching margin spacers so stacked block margins
	// don't accumulate. (Containers with padding/border are opaque rows, so
	// margins never collapse through them — matching CSS.)
	return collapseMarginSpacers(rows)
}

func withPageBreakRows(style *css.ComputedStyle, rows []core.Row) []core.Row {
	if style != nil {
		if style.PageBreakBefore == "always" {
			rows = append([]core.Row{NewPageBreakRow()}, rows...)
		}
		if style.PageBreakAfter == "always" {
			rows = append(rows, NewPageBreakRow())
		}
	}
	return rows
}

func wrapRowsWithHorizontalMargins(rows []core.Row, marginLeft, marginRight float64) []core.Row {
	out := make([]core.Row, 0, len(rows))
	for _, r := range rows {
		if _, ok := r.(marginSpacerRow); ok {
			out = append(out, r)
			continue
		}
		out = append(out, &horizontalMarginRow{
			child:       r,
			marginLeft:  marginLeft,
			marginRight: marginRight,
		})
	}
	return out
}

func pageControlFromStyle(style, parent *css.ComputedStyle) (core.PageControl, bool) {
	if style == nil || len(style.Vars) == 0 {
		return core.PageControl{}, false
	}
	pageNumberCount := cssVarDeclaredEquals(style, parent, "--paper-page-number", "count")
	control := core.PageControl{
		Blank:                     cssVarDeclaredEquals(style, parent, "--paper-page", "blank"),
		SuppressFooter:            cssVarDeclaredEquals(style, parent, "--paper-page-footer", "none"),
		SuppressPageNumber:        cssVarDeclaredEquals(style, parent, "--paper-page-number", "none") || pageNumberCount,
		CountPageNumber:           pageNumberCount,
		DecorFirstPageOnly:        cssVarDeclaredEquals(style, parent, "--paper-page-decor-scope", "first-page"),
		TopMarginFirstPageOnly:    cssVarDeclaredEquals(style, parent, "--paper-page-top-margin-scope", "first-page"),
		RenderOffsetFirstPageOnly: cssVarDeclaredEquals(style, parent, "--paper-page-render-offset-scope", "first-page"),
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-top-margin"); ok {
		top := css.ParseLength(value, style.FontSize)
		control.TopMargin = &top
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-top-margin-continuation"); ok {
		top := css.ParseLength(value, style.FontSize)
		control.ContinuationTopMargin = &top
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-render-offset-x"); ok {
		x := css.ParseLength(value, style.FontSize)
		control.RenderOffsetX = &x
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-render-offset-y"); ok {
		y := css.ParseLength(value, style.FontSize)
		control.RenderOffsetY = &y
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-render-offset-x-continuation"); ok {
		x := css.ParseLength(value, style.FontSize)
		control.ContinuationRenderOffsetX = &x
	}
	if value, ok := cssVarDeclaredValue(style, parent, "--paper-page-render-offset-y-continuation"); ok {
		y := css.ParseLength(value, style.FontSize)
		control.ContinuationRenderOffsetY = &y
	}
	return control, pageControlHasEffect(control)
}

func pageControlHasEffect(control core.PageControl) bool {
	return control.Blank ||
		control.SuppressFooter ||
		control.SuppressPageNumber ||
		control.CountPageNumber ||
		control.DecorFirstPageOnly ||
		control.TopMargin != nil ||
		control.TopMarginFirstPageOnly ||
		control.ContinuationTopMargin != nil ||
		control.RenderOffsetX != nil ||
		control.RenderOffsetY != nil ||
		control.RenderOffsetFirstPageOnly ||
		control.ContinuationRenderOffsetX != nil ||
		control.ContinuationRenderOffsetY != nil
}

func cssVarDeclaredEquals(style, parent *css.ComputedStyle, name, want string) bool {
	value, ok := cssVarDeclaredValue(style, parent, name)
	return ok && strings.EqualFold(strings.TrimSpace(value), want)
}

func cssVarDeclaredValue(style, parent *css.ComputedStyle, name string) (string, bool) {
	value, ok := style.Vars[name]
	if !ok {
		return "", false
	}
	if parent == nil || len(parent.Vars) == 0 {
		return value, true
	}
	parentValue, inherited := parent.Vars[name]
	return value, !inherited || parentValue != value
}

// marginSpacerRow is an empty flow spacer emitted for a block's vertical margin.
// It is a distinct type so collapseMarginSpacers can recognise and merge
// adjacent ones (CSS margin collapsing).
type marginSpacerRow struct {
	core.Row
	height float64
}

func newMarginSpacer(h float64) marginSpacerRow {
	return marginSpacerRow{Row: spacerRow(h), height: h}
}

// collapseMarginSpacers merges each run of adjacent margin spacers into a single
// spacer of the largest height, approximating CSS adjacent and parent/last-child
// margin collapsing so stacked block margins reduce to the larger rather than
// summing. Non-margin rows (content, page breaks, flex gaps) are untouched.
func collapseMarginSpacers(rows []core.Row) []core.Row {
	out := make([]core.Row, 0, len(rows))
	for _, r := range rows {
		if ms, ok := r.(marginSpacerRow); ok && len(out) > 0 {
			if prev, prevOK := out[len(out)-1].(marginSpacerRow); prevOK {
				if ms.height > prev.height {
					out[len(out)-1] = ms
				}
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// selfHandlesMargin reports whether a tag's dedicated translation already
// applies its own CSS margins (via marginBox), so blockRowsWithParent must not
// add margin spacers on top (which would double them). Lists wrap their content
// in a marginBox that also carries horizontal indent.
func selfHandlesMargin(tag string) bool {
	switch tag {
	case "ul", "ol", "dl":
		return true
	default:
		return false
	}
}
