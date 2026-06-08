package translate

import (
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/htmllist"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// listRows converts <ul>/<ol> into a single row containing an HTMLList component.
func (tr *translator) listRows(n *dom.Node) []core.Row {
	style := computeNodeStyleRooted(tr.sheet, n, tr.rootStyle)
	list := tr.buildList(n)
	if list == nil {
		return nil
	}
	var component core.Component = list
	if style.MarginTop > 0 || style.MarginRight > 0 || style.MarginBottom > 0 || style.MarginLeft > 0 {
		component = &marginBox{
			child:        component,
			marginTop:    style.MarginTop,
			marginRight:  style.MarginRight,
			marginBottom: style.MarginBottom,
			marginLeft:   style.MarginLeft,
		}
	}
	c := col.New().Add(component)
	return []core.Row{row.New().Add(c)}
}

func (tr *translator) buildList(n *dom.Node) *htmllist.HTMLList {
	style := htmllist.Bullet
	start := 0
	reversed := false
	if n.Tag() == "ol" {
		style = listStyleFromType(n.Attr("type"))
		start = atoiOr(n.Attr("start"), 0)
		reversed = hasAttr(n, "reversed")
	}
	cssStyle := computeNodeStyleRooted(tr.sheet, n, tr.rootStyle)
	if s, ok := listStyleFromCSS(cssStyle.ListStyleType); ok {
		style = s
	}

	var items []htmllist.Item
	for _, child := range n.Children() {
		if child.Tag() != "li" {
			continue
		}
		items = append(items, tr.buildItem(child, cssStyle))
	}
	if len(items) == 0 {
		return nil
	}
	return htmllist.New(items, htmllist.Prop{Style: style, Start: start, Reversed: reversed})
}

// listStyleFromCSS maps a CSS list-style-type value to an htmllist.StyleType.
// Returns ok=false when the value is empty or unrecognised so the caller keeps
// its existing default (bullet for ul, decimal/type-attr for ol).
func listStyleFromCSS(val string) (htmllist.StyleType, bool) {
	switch val {
	case "":
		return "", false
	case cssValueNone:
		return htmllist.None, true
	case "disc", "circle", "square":
		return htmllist.Bullet, true
	case listStyleDecimal:
		return htmllist.Decimal, true
	case "decimal-circle":
		return htmllist.DecimalCircle, true
	case "lower-alpha", "lower-latin":
		return htmllist.LowerAlpha, true
	case "upper-alpha", "upper-latin":
		return htmllist.UpperAlpha, true
	case "lower-roman":
		return htmllist.LowerRoman, true
	case "upper-roman":
		return htmllist.UpperRoman, true
	default:
		return "", false
	}
}

func (tr *translator) buildItem(li *dom.Node, parentStyle *css.ComputedStyle) htmllist.Item {
	item := htmllist.Item{}
	style := computeNodeStyle(tr.sheet, li, parentStyle)
	inlineStyle := blockInlineStyle(style)

	// Recursively check for nested ul/ol; collect inline content into runs.
	var runs []props.RichRun
	for _, c := range li.Children() {
		switch c.Tag() {
		case "ul", "ol":
			item.SubList = tr.buildList(c)
		default:
			// Use inline walker on each child to flatten its text and styling.
			walkInline(c, tr.styledRunContext(inlineStyle), &runs)
		}
	}
	if len(runs) > 0 {
		item.Content = richtext.New(runs)
	}
	return item
}

func listStyleFromType(t string) htmllist.StyleType {
	switch t {
	case "a":
		return htmllist.LowerAlpha
	case "A":
		return htmllist.UpperAlpha
	case "i":
		return htmllist.LowerRoman
	case "I":
		return htmllist.UpperRoman
	default:
		return htmllist.Decimal
	}
}

func hasAttr(n *dom.Node, name string) bool {
	if n == nil || n.RawNode() == nil {
		return false
	}
	for _, attr := range n.RawNode().Attr {
		if attr.Key == name {
			return true
		}
	}
	return false
}
