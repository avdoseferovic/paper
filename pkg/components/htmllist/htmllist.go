// Package htmllist implements HTML-style bullet and numbered list components.
package htmllist

import (
	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// StyleType controls the marker style.
type StyleType string

const (
	Bullet        StyleType = "bullet"
	Decimal       StyleType = "decimal"
	DecimalCircle StyleType = "decimal-circle"
	LowerAlpha    StyleType = "lower-alpha"
	UpperAlpha    StyleType = "upper-alpha"
	LowerRoman    StyleType = "lower-roman"
	UpperRoman    StyleType = "upper-roman"
)

// Prop holds list-level configuration.
type Prop struct {
	Style         StyleType
	Indent        float64 // mm per nesting level
	MarkerPadding float64 // mm gap between marker and content
	GutterWidth   float64 // 0 = measure at render time

	// MarkerBackground is the fill color used for circle markers (DecimalCircle).
	// Default: dark navy (#1a3e72) when nil.
	MarkerBackground *props.Color
	// MarkerTextColor is the text color used for the number inside circle markers.
	// Default: white when nil.
	MarkerTextColor *props.Color
}

// Item is a single list entry.
type Item struct {
	Content core.Component
	SubList *HTMLList
}

// HTMLList is a core.Component rendering bullet/numbered lists.
type HTMLList struct {
	items  []Item
	prop   Prop
	config *entity.Config
}

// New creates an HTMLList component.
func New(items []Item, ps ...Prop) *HTMLList {
	prop := Prop{Style: Bullet, Indent: 5, MarkerPadding: 1}
	if len(ps) > 0 {
		prop = ps[0]
	}
	if prop.Style == "" {
		prop.Style = Bullet
	}
	if prop.Indent == 0 {
		prop.Indent = 5
	}
	if prop.MarkerPadding == 0 {
		prop.MarkerPadding = 1
	}
	return &HTMLList{items: items, prop: prop}
}

// SetConfig propagates Maroto config to all item components.
func (l *HTMLList) SetConfig(config *entity.Config) {
	l.config = config
	for _, item := range l.items {
		if item.Content != nil {
			item.Content.SetConfig(config)
		}
		if item.SubList != nil {
			item.SubList.SetConfig(config)
		}
	}
}

// GetStructure returns the component node for snapshots/debugging.
func (l *HTMLList) GetStructure() *node.Node[core.Structure] {
	style := l.prop.Style
	str := core.Structure{
		Type: "htmllist",
		Details: map[string]any{
			"style":        string(style),
			"items":        len(l.items),
			"marker_style": string(style),
		},
	}
	return node.New(str)
}

// GetHeight returns the total height of the list (items + sub-lists).
func (l *HTMLList) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	itemH := l.itemRowHeight(provider, cell)
	total := float64(len(l.items)) * itemH
	for _, item := range l.items {
		if item.SubList != nil {
			subCell := &entity.Cell{Width: cell.Width - l.prop.Indent, Height: cell.Height}
			total += item.SubList.GetHeight(provider, subCell)
		}
	}
	return total
}

// Render draws all list items into the PDF cell.
func (l *HTMLList) Render(provider core.Provider, cell *entity.Cell) {
	gutter := l.gutterWidth(provider)
	itemH := l.itemRowHeight(provider, cell)
	contentWidth := cell.Width - gutter
	y := cell.Y

	for i, item := range l.items {
		marker := FormatMarker(l.prop.Style, i)
		markerCell := &entity.Cell{X: cell.X, Y: y, Width: gutter, Height: itemH}
		l.renderMarker(provider, marker, markerCell)

		if item.Content != nil {
			contentCell := &entity.Cell{X: cell.X + gutter, Y: y, Width: contentWidth, Height: itemH}
			item.Content.Render(provider, contentCell)
		}
		y += itemH

		if item.SubList != nil {
			subCell := &entity.Cell{
				X:      cell.X + l.prop.Indent,
				Y:      y,
				Width:  cell.Width - l.prop.Indent,
				Height: cell.Height,
			}
			subH := item.SubList.GetHeight(provider, subCell)
			item.SubList.Render(provider, subCell)
			y += subH
		}
	}
}

// renderMarker draws a single list marker — either as text (default) or as a
// filled circle with the index centred inside (DecimalCircle style).
func (l *HTMLList) renderMarker(provider core.Provider, label string, cell *entity.Cell) {
	if l.prop.Style != DecimalCircle {
		provider.AddText(label, cell, l.markerTextProp())
		return
	}
	// Circle marker: best-effort via ShapeProvider; fallback to text-only.
	sp, ok := provider.(core.ShapeProvider)
	if !ok {
		provider.AddText(label, cell, l.markerTextProp())
		return
	}
	// Inscribe the circle in a square sized by the shorter side of the marker cell.
	diameter := cell.Width
	if cell.Height < diameter {
		diameter = cell.Height
	}
	circleCell := &entity.Cell{
		X:      cell.X + (cell.Width-diameter)/2,
		Y:      cell.Y + (cell.Height-diameter)/2,
		Width:  diameter,
		Height: diameter,
	}
	bg := l.prop.MarkerBackground
	if bg == nil {
		bg = &props.Color{Red: 26, Green: 62, Blue: 114} // #1a3e72
	}
	sp.DrawFilledCircle(circleCell, bg)

	// Number rendered as centred text inside the circle.
	tp := l.markerTextProp()
	tp.Align = "center"
	if l.prop.MarkerTextColor != nil {
		tp.Color = l.prop.MarkerTextColor
	} else {
		tp.Color = &props.Color{Red: 255, Green: 255, Blue: 255}
	}
	provider.AddText(label, circleCell, tp)
}

// gutterWidth returns the computed or configured gutter.
// When the provider implements core.RichTextProvider it measures the widest marker
// for accurate sizing; otherwise it falls back to a font-height heuristic.
func (l *HTMLList) gutterWidth(provider core.Provider) float64 {
	if l.prop.GutterWidth > 0 {
		return l.prop.GutterWidth
	}
	tp := l.markerTextProp()
	if rtp, ok := provider.(core.RichTextProvider); ok {
		widest := 0.0
		for i := range len(l.items) {
			m := FormatMarker(l.prop.Style, i)
			w := rtp.MeasureString(m, tp)
			if w > widest {
				widest = w
			}
		}
		if widest > 0 {
			return widest + l.prop.MarkerPadding
		}
	}
	// Fallback: heuristic based on font height (provider lacks MeasureString).
	return l.itemRowHeight(provider, &entity.Cell{Width: 100}) + l.prop.MarkerPadding
}

// itemRowHeight returns the height of one item row.
func (l *HTMLList) itemRowHeight(provider core.Provider, _ *entity.Cell) float64 {
	if l.config == nil || l.config.DefaultFont == nil {
		return 5.0
	}
	return provider.GetFontHeight(l.config.DefaultFont)
}

func (l *HTMLList) markerTextProp() *props.Text {
	tp := &props.Text{}
	if l.config != nil {
		tp.MakeValid(l.config.DefaultFont)
	}
	return tp
}
