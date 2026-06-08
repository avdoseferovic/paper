// Package richtext implements a PDF component for paragraphs with mixed inline styling.
package richtext

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

const whiteSpacePre = "pre"

// anchorResolverIface is a narrow interface satisfied by the HTML translator's
// anchorRegistry. It is defined here to avoid an import cycle between the
// richtext component package and the translate package.
type anchorResolverIface interface {
	EnsureLinkID(name string, lp core.LinkProvider) (int, bool)
}

// RichText is a paragraph component that renders inline runs with mixed styles.
// It implements core.Component and can be placed inside Col/Row like any other component.
type RichText struct {
	runs      []props.RichRun
	prop      props.RichText
	config    *entity.Config
	anchorReg anchorResolverIface // optional, set via WithAnchorRegistry

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
	return &RichText{runs: props.CloneRichRuns(runs), prop: prop}
}

// WithAnchorRegistry attaches an anchor registry so that runs with LocalAnchor
// produce precise per-run PDF link rectangles at render time.
func (r *RichText) WithAnchorRegistry(reg anchorResolverIface) *RichText {
	r.anchorReg = reg
	return r
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

// SetConfig propagates Paper configuration to the component.
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
	if r.prop.FirstLineIndent != 0 {
		details["first_line_indent"] = r.prop.FirstLineIndent
	}
	if r.prop.Left != 0 {
		details["left"] = r.prop.Left
	}
	if r.prop.Align != "" && r.prop.Align != align.Left {
		details["align"] = r.prop.Align
	}
	if ws := normalizeWhiteSpace(r.prop.WhiteSpace); ws != "normal" {
		details["white_space"] = ws
	}
	str := core.Structure{
		Type:    "richtext",
		Value:   r.allText(),
		Details: details,
	}
	return node.New(str)
}

// GetHeight returns the height the component will occupy in the cell.
// The result is memoised by cell width so Paper's two-call pattern (addRow + Render)
// doesn't drift even when SetConfig is called between invocations.
func (r *RichText) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	key := r.configKey()
	if r.cachedHeight > 0 && r.cachedCellWidth == cell.Width && r.cachedConfigKey == key {
		return r.cachedHeight
	}

	if measurer, ok := provider.(core.RichTextMeasurer); ok {
		prop := r.prop
		h := measurer.MeasureRichText(r.runsWithDefaultFont(), cell, &prop)
		r.cachedHeight = h
		r.cachedCellWidth = cell.Width
		r.cachedConfigKey = key
		return h
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
	if imageHeight := maxRichRunImageHeight(r.runs); imageHeight > lineHeight {
		lineHeight = imageHeight
	}
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
		prop := r.prop
		if r.anchorReg != nil {
			if lp, ok := provider.(core.LinkProvider); ok {
				reg := r.anchorReg
				prop.AnchorResolver = func(name string) int {
					id, _ := reg.EnsureLinkID(name, lp)
					return id
				}
			}
		}
		rtp.AddRichText(r.runsWithDefaultFont(), cell, &prop)
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

func (r *RichText) runsWithDefaultFont() []props.RichRun {
	if r.config == nil || r.config.DefaultFont == nil {
		return r.runs
	}
	out := make([]props.RichRun, len(r.runs))
	for i, run := range r.runs {
		out[i] = props.NormalizeRichRun(run, r.config.DefaultFont)
	}
	return out
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
	mode := normalizeWhiteSpace(r.prop.WhiteSpace)
	text := textForWhiteSpace(r.allText(), mode)
	if mode == "nowrap" || mode == whiteSpacePre {
		return countExplicitLines(text)
	}
	firstLineIndent := r.prop.FirstLineIndent
	for segment := range strings.SplitSeq(text, "\n") {
		if segment == "" {
			total++
			firstLineIndent = 0
			continue
		}
		width := colWidth
		if firstLineIndent > 0 {
			width -= firstLineIndent
			firstLineIndent = 0
			if width <= 0 {
				width = colWidth
			}
		}
		total += provider.GetLinesQuantity(segment, fontProp, width)
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
	sizeScale := 0.0
	if len(r.runs) > 0 {
		run := r.runs[0]
		tp.Family = run.Family
		tp.Style = run.Style
		tp.Size = run.Size
		tp.Color = run.Color
		sizeScale = run.SizeScale
	}
	if r.config != nil {
		tp.MakeValid(r.config.DefaultFont)
	}
	if sizeScale > 0 {
		tp.Size *= sizeScale
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

func normalizeWhiteSpace(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "nowrap", whiteSpacePre, "pre-wrap", "pre-line":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "normal"
	}
}

func textForWhiteSpace(text, mode string) string {
	switch mode {
	case whiteSpacePre, "pre-wrap":
		return text
	case "pre-line":
		parts := strings.Split(text, "\n")
		for i, part := range parts {
			parts[i] = strings.Join(strings.Fields(part), " ")
		}
		return strings.Join(parts, "\n")
	default:
		return strings.Join(strings.Fields(text), " ")
	}
}

func countExplicitLines(text string) int {
	if text == "" {
		return 1
	}
	return strings.Count(text, "\n") + 1
}

func maxRichRunImageHeight(runs []props.RichRun) float64 {
	maxHeight := 0.0
	for _, run := range runs {
		if run.Image != nil && run.Image.Height > maxHeight {
			maxHeight = run.Image.Height
		}
	}
	return maxHeight
}
