// Package htmllist implements HTML-style bullet and numbered list components.
package htmllist

import (
	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/paper/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/paper/v2/pkg/core"
	"github.com/johnfercher/paper/v2/pkg/core/entity"
	"github.com/johnfercher/paper/v2/pkg/props"
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

// SetConfig propagates Paper config to all item components.
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
	// Sum per-item heights (content's wrapped height, falling back to one line
	// when content is empty) so the list reports its true rendered height.
	gutter := l.gutterWidth(provider)
	contentWidth := cell.Width - gutter
	total := 0.0
	for _, item := range l.items {
		if item.Content != nil {
			contentInner := &entity.Cell{Width: contentWidth}
			if h := item.Content.GetHeight(provider, contentInner); h > 0 {
				total += h
			} else {
				total += itemH
			}
		} else {
			total += itemH
		}
		if item.SubList != nil {
			subCell := &entity.Cell{Width: cell.Width - l.prop.Indent, Height: cell.Height}
			total += item.SubList.GetHeight(provider, subCell)
		}
	}
	return total
}

// Render draws all list items into the PDF cell.
// Each item's row height is the actual wrapped height of its content (not a
// fixed single-line itemH) — this prevents markers from stacking when items
// have long text that wraps to multiple lines. The marker (text or circle)
// is always positioned at the top of its item's row, anchored to the first
// line of content.
func (l *HTMLList) Render(provider core.Provider, cell *entity.Cell) {
	gutter := l.gutterWidth(provider)
	lineH := l.itemRowHeight(provider, cell)
	contentWidth := cell.Width - gutter
	y := cell.Y

	for i, item := range l.items {
		// Per-item height: measure the content's actual rendered height at the
		// available content width. Falls back to lineH for empty items.
		itemH := lineH
		if item.Content != nil {
			contentInner := &entity.Cell{Width: contentWidth}
			if h := item.Content.GetHeight(provider, contentInner); h > 0 {
				itemH = h
			}
		}

		marker := FormatMarker(l.prop.Style, i)
		// Marker is anchored to the first line of the item, not stretched to itemH.
		markerCell := &entity.Cell{X: cell.X, Y: y, Width: gutter, Height: lineH}
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
	// Draw the circle slightly larger than the text line box. The marker cell's
	// gutter is intentionally wider than a line, so this keeps the disc readable
	// without stealing horizontal space from the list item.
	diameter := cell.Width
	lineDiameter := cell.Height * 1.28
	if lineDiameter < diameter {
		diameter = lineDiameter
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
	tp := l.circleMarkerTextProp()
	tp.Align = "center"
	fontH := provider.GetFontHeight(&props.Font{
		Family: tp.Family,
		Style:  tp.Style,
		Size:   tp.Size,
	})
	tp.Top = (circleCell.Height - fontH) / 2
	if tp.Top < 0 {
		tp.Top = 0
	}
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
// For DecimalCircle markers the gutter is enlarged to at least 1.6 × line-height
// so the inscribed circle is readable (otherwise a 2-char "10" measurement
// produces a circle smaller than the digit it contains).
func (l *HTMLList) gutterWidth(provider core.Provider) float64 {
	if l.prop.GutterWidth > 0 {
		return l.prop.GutterWidth
	}
	tp := l.markerTextProp()
	lineH := l.itemRowHeight(provider, &entity.Cell{Width: 100})
	textWidth := 0.0
	if rtp, ok := provider.(core.RichTextProvider); ok {
		for i := range len(l.items) {
			m := FormatMarker(l.prop.Style, i)
			w := rtp.MeasureString(m, tp)
			if w > textWidth {
				textWidth = w
			}
		}
	}
	if l.prop.Style == DecimalCircle {
		minGutter := lineH * 1.6
		if textWidth+l.prop.MarkerPadding < minGutter {
			return minGutter
		}
		return textWidth + l.prop.MarkerPadding
	}
	if textWidth > 0 {
		return textWidth + l.prop.MarkerPadding
	}
	return lineH + l.prop.MarkerPadding
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

func (l *HTMLList) circleMarkerTextProp() *props.Text {
	tp := l.markerTextProp()
	tp.Style = fontstyle.Bold
	tp.Size *= 0.72
	if tp.Size <= 0 {
		tp.Size = 7
	}
	return tp
}
