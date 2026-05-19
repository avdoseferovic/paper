// Package richtext implements a PDF component for paragraphs with mixed inline styling.
package richtext

import (
	"strings"

	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// RichText is a paragraph component that renders inline runs with mixed styles.
// It implements core.Component and can be placed inside Col/Row like any other component.
type RichText struct {
	runs   []props.RichRun
	prop   props.RichText
	config *entity.Config

	// memoised height — keyed by (cellWidth, configKey)
	cachedHeight    float64
	cachedCellWidth float64
	cachedConfigKey string
}

// New creates a RichText component.
func New(runs []props.RichRun, ps ...props.RichText) *RichText {
	prop := props.RichText{}
	if len(ps) > 0 {
		prop = ps[0]
	}
	return &RichText{runs: runs, prop: prop}
}

// NewCol wraps a RichText in a Col of the given grid size.
func NewCol(size int, runs []props.RichRun, ps ...props.RichText) core.Col {
	rt := New(runs, ps...)
	return col.New(size).Add(rt)
}

// NewRow wraps a RichText in a fixed-height Row.
func NewRow(height float64, runs []props.RichRun, ps ...props.RichText) core.Row {
	rt := New(runs, ps...)
	c := col.New().Add(rt)
	return row.New(height).Add(c)
}

// NewAutoRow wraps a RichText in an auto-height Row.
func NewAutoRow(runs []props.RichRun, ps ...props.RichText) core.Row {
	rt := New(runs, ps...)
	c := col.New().Add(rt)
	return row.New().Add(c)
}

// SetConfig propagates Maroto configuration to the component.
func (r *RichText) SetConfig(config *entity.Config) {
	r.config = config
	r.prop.MakeValid(config.DefaultFont)
	r.invalidateCache()
}

// GetStructure returns the component tree node for debugging/snapshot tests.
func (r *RichText) GetStructure() *node.Node[core.Structure] {
	details := map[string]any{
		"runs": len(r.runs),
	}
	if r.prop.LineHeight != 0 {
		details["line_height"] = r.prop.LineHeight
	}
	str := core.Structure{
		Type:    "richtext",
		Details: details,
	}
	return node.New(str)
}

// GetHeight returns the height the component will occupy in the cell.
// The result is memoised by cell width so Maroto's two-call pattern (addRow + Render)
// doesn't drift even when SetConfig is called between invocations.
func (r *RichText) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	key := r.configKey()
	if r.cachedHeight > 0 && r.cachedCellWidth == cell.Width && r.cachedConfigKey == key {
		return r.cachedHeight
	}

	colWidth := cell.Width - r.prop.Left - r.prop.Right
	if colWidth <= 0 {
		return 0
	}

	// Use the first run's font (or default) to estimate line height and count.
	// For a more precise multi-run height, we'd need RichTextProvider.MeasureString
	// for every word — but GetHeight only receives core.Provider (not RichTextProvider).
	// We approximate by splitting on explicit \n (from <br>) and word-wrapping each
	// segment independently, then summing. RichText height = total lines * fontHeight.
	fontProp := r.fontPropForFirstRun()
	fontHeight := provider.GetFontHeight(&props.Font{
		Family: fontProp.Family,
		Style:  fontProp.Style,
		Size:   fontProp.Size,
	})

	totalLines := max(r.countLines(provider, fontProp, colWidth), 1)

	lineMultiplier := r.prop.LineHeight
	if lineMultiplier <= 0 {
		lineMultiplier = 1.0
	}
	lineHeight := fontHeight * lineMultiplier
	h := float64(totalLines)*lineHeight + r.prop.Top + r.prop.Bottom

	r.cachedHeight = h
	r.cachedCellWidth = cell.Width
	r.cachedConfigKey = key
	return h
}

// Render draws the component. If the provider implements core.RichTextProvider, it
// delegates to AddRichText for accurate per-run styling. Otherwise it falls back to
// AddText with the first run's style.
func (r *RichText) Render(provider core.Provider, cell *entity.Cell) {
	if rtp, ok := provider.(core.RichTextProvider); ok {
		rtp.AddRichText(r.runs, cell, &r.prop)
		return
	}

	// Fallback: render concatenated text with first run's style.
	textProp := r.fontPropForFirstRun()
	textProp.Top = r.prop.Top
	textProp.Bottom = r.prop.Bottom
	textProp.Left = r.prop.Left
	textProp.Right = r.prop.Right
	textProp.Align = r.prop.Align
	provider.AddText(r.allText(), cell, textProp)
}

// countLines totals the visual line count across runs.
//
// IMPORTANT: a paragraph's logical lines come from BOTH inline word-wrap
// (the column being too narrow for the text) AND explicit \n breaks (from
// <br>). Splitting on \n produces one segment per logical break; each segment
// is then word-wrapped independently. Empty segments between consecutive \n
// count as blank lines. Runs are concatenated, so we accumulate the wrapped
// lines per segment without double-counting the line breaks themselves.
func (r *RichText) countLines(provider core.Provider, fontProp *props.Text, colWidth float64) int {
	total := 0
	for _, run := range r.runs {
		if run.Text == "" {
			continue
		}
		segments := strings.Split(run.Text, "\n")
		for i, seg := range segments {
			if seg == "" {
				// Blank segment only counts as a line when sandwiched between
				// \n breaks (i.e. not the trailing empty segment from "A\n").
				if i < len(segments)-1 {
					total++
				}
				continue
			}
			total += provider.GetLinesQuantity(seg, fontProp, colWidth)
		}
	}
	return total
}

// allText concatenates all run texts for fallback/measurement purposes.
func (r *RichText) allText() string {
	b := make([]byte, 0, r.totalTextLen())
	for _, run := range r.runs {
		b = append(b, run.Text...)
	}
	return string(b)
}

func (r *RichText) totalTextLen() int {
	n := 0
	for _, run := range r.runs {
		n += len(run.Text)
	}
	return n
}

// fontPropForFirstRun builds a props.Text from the first run (or config default).
func (r *RichText) fontPropForFirstRun() *props.Text {
	tp := &props.Text{}
	if len(r.runs) > 0 {
		run := r.runs[0]
		tp.Family = run.Family
		tp.Style = run.Style
		tp.Size = run.Size
		tp.Color = run.Color
	}
	if r.config != nil {
		tp.MakeValid(r.config.DefaultFont)
	}
	return tp
}

func (r *RichText) configKey() string {
	if r.config == nil || r.config.DefaultFont == nil {
		return ""
	}
	f := r.config.DefaultFont
	return f.Family + string(f.Style)
}

func (r *RichText) invalidateCache() {
	r.cachedHeight = 0
	r.cachedCellWidth = 0
	r.cachedConfigKey = ""
}
