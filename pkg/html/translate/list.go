package translate

import (
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/htmllist"
	"github.com/johnfercher/maroto/v2/pkg/components/richtext"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// listRows converts <ul>/<ol> into a single row containing an HTMLList component.
func listRows(n *dom.Node) []core.Row {
	list := buildList(n)
	if list == nil {
		return nil
	}
	c := col.New().Add(list)
	return []core.Row{row.New().Add(c)}
}

func buildList(n *dom.Node) *htmllist.HTMLList {
	style := htmllist.Bullet
	if n.Tag() == "ol" {
		style = listStyleFromType(n.Attr("type"))
		if hasClass(n, "circle-numbers") {
			style = htmllist.DecimalCircle
		}
	}

	var items []htmllist.Item
	for _, child := range n.Children() {
		if child.Tag() != "li" {
			continue
		}
		items = append(items, buildItem(child))
	}
	if len(items) == 0 {
		return nil
	}
	return htmllist.New(items, htmllist.Prop{Style: style})
}

// hasClass reports whether the node's class attribute contains the given class name.
func hasClass(n *dom.Node, name string) bool {
	for _, c := range strings.Fields(n.Attr("class")) {
		if c == name {
			return true
		}
	}
	return false
}

func buildItem(li *dom.Node) htmllist.Item {
	item := htmllist.Item{}

	// Recursively check for nested ul/ol; collect inline content into runs.
	var runs []props.RichRun
	for _, c := range li.Children() {
		switch c.Tag() {
		case "ul", "ol":
			item.SubList = buildList(c)
		default:
			// Use inline walker on each child to flatten its text and styling.
			walkInline(c, runContext{}, &runs)
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
