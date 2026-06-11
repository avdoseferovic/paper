package wasmconvert

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/avdoseferovic/paper/pkg/components/checkbox"
	"github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/components/line"
	"github.com/avdoseferovic/paper/pkg/components/signature"
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

// mutedColor is used for captions, labels, and placeholder text.
var mutedColor = &props.Color{Red: 154, Green: 143, Blue: 128}

// parseSpec decodes the JSON layout document.
func parseSpec(jsonSpec string) (Spec, error) {
	var spec Spec
	err := json.Unmarshal([]byte(jsonSpec), &spec)
	if err != nil {
		return Spec{}, fmt.Errorf("wasmconvert: invalid spec json: %w", err)
	}
	return spec, nil
}

// buildComponent maps one spec cell to one or more Paper components. The primary
// component is first; an optional caption (for codes/signatures) follows with an
// explicit Top offset, since components in a column do not flow-stack.
func buildComponent(c SpecComp) []core.Component {
	switch c.Type {
	case "text", "":
		return []core.Component{textComponent(c)}
	case "line":
		return []core.Component{lineComponent(c)}
	case "qrcode":
		return []core.Component{qrComponent(c)}
	case "barcode":
		return withCaption(barcodeComponent(c), c.Label, captionTopBarcode)
	case "signature":
		return withCaption(signatureComponent(c), c.Label, captionTopSignature)
	case "checkbox":
		return []core.Component{checkbox.New(plainText(c.Value), props.Checkbox{Checked: c.Checked})}
	case "pagenumber":
		// Static placeholder: a grid cell has no document-level page context, so
		// {n}/{t} resolve to "1". Real pagination is a document-level feature
		// (config.WithPageNumber), out of scope for the per-cell spec.
		page := strings.NewReplacer("{n}", "1", "{t}", "1").Replace(c.Value)
		return []core.Component{text.New(plainText(page), props.Text{Size: 9, Align: consts.AlignCenter, Color: mutedColor})}
	case "footer":
		return []core.Component{text.New(plainText(c.Value), props.Text{Size: 8, Color: mutedColor})}
	case "table":
		return []core.Component{tableComponent(c)}
	default:
		// image / unknown types render as a muted placeholder.
		return []core.Component{text.New(plainText(c.Value), props.Text{Size: 10, Color: mutedColor})}
	}
}

// caption Top offsets (mm) position a label below its component within the
// fixed-height row, since columns do not flow-stack.
const (
	captionTopBarcode   = 22.0
	captionTopSignature = 24.0
)

// withCaption appends a centered muted caption with an explicit Top offset so it
// renders below the primary component instead of overlapping it.
func withCaption(primary core.Component, label string, top float64) []core.Component {
	if strings.TrimSpace(label) == "" {
		return []core.Component{primary}
	}
	caption := text.New(plainText(label), props.Text{Top: top, Size: 8, Align: consts.AlignCenter, Color: mutedColor})
	return []core.Component{primary, caption}
}

func qrComponent(c SpecComp) core.Component {
	pct := c.Size
	if pct <= 0 {
		pct = 80
	}
	return code.NewQr(c.Value, props.Rect{Center: true, Percent: pct})
}

func barcodeComponent(c SpecComp) core.Component {
	// Honor an explicit width/height ratio when provided; otherwise default to a
	// slim 12:1.6 barcode (matching the showcase).
	proportion := props.Proportion{Width: 12, Height: 1.6}
	if c.Width > 0 && c.Height > 0 {
		proportion = props.Proportion{Width: c.Width, Height: c.Height}
	}
	return code.NewBar(c.Value, props.Barcode{
		Center:     true,
		Percent:    100,
		Proportion: proportion,
	})
}

func signatureComponent(c SpecComp) core.Component {
	return signature.New(plainText(c.Value), props.Signature{FontSize: 12})
}

// tableComponent builds a Paper table from the spec's head/rows/colAlign. On a
// construction error it falls back to a muted placeholder rather than panicking.
func tableComponent(c SpecComp) core.Component {
	cells := make([][]table.Cell, 0, len(c.Rows)+1)
	if len(c.Head) > 0 {
		head := make([]table.Cell, len(c.Head))
		for i, h := range c.Head {
			head[i] = table.Cell{Content: text.New(plainText(h), props.Text{Style: fontstyle.Bold, Size: 9, Align: colAlignAt(c.ColAlign, i)})}
		}
		cells = append(cells, head)
	}
	for _, r := range c.Rows {
		bodyRow := make([]table.Cell, len(r))
		for i, v := range r {
			bodyRow[i] = table.Cell{Content: text.New(plainText(v), props.Text{Size: 10, Align: colAlignAt(c.ColAlign, i)})}
		}
		cells = append(cells, bodyRow)
	}

	t, err := table.New(cells)
	if err != nil {
		return text.New("[unrenderable table]", props.Text{Size: 9, Color: mutedColor})
	}
	return t
}

func colAlignAt(colAlign []string, i int) consts.Align {
	if i < len(colAlign) {
		return alignFor(colAlign[i])
	}
	return consts.AlignLeft
}

// textComponent builds a text from a spec cell, mapping the design's style names
// to real Paper text props.
func textComponent(c SpecComp) core.Component {
	p := textPropsForStyle(c.Style)
	p.Align = alignFor(c.Align)
	return text.New(plainText(c.Value), p)
}

func lineComponent(c SpecComp) core.Component {
	thickness := 0.4
	color := &props.Color{Red: 43, Green: 38, Blue: 32}
	if c.Soft {
		thickness = 0.2
		color = &props.Color{Red: 216, Green: 210, Blue: 200}
	}
	return line.New(props.Line{Thickness: thickness, Color: color})
}

// textPropsForStyle maps the design's style tokens (h1/h2/num/label/body) to
// Paper text props.
func textPropsForStyle(style string) props.Text {
	switch style {
	case "h1":
		return props.Text{Size: 24, Style: fontstyle.Bold}
	case "h2":
		return props.Text{Size: 16, Style: fontstyle.Bold}
	case "num":
		return props.Text{Size: 18, Style: fontstyle.Bold}
	case "label":
		return props.Text{Size: 8, Color: mutedColor}
	default: // body
		return props.Text{Size: 10}
	}
}

// alignFor maps "left"/"center"/"right" (and single letters l/c/r) to a Paper
// alignment. Empty/unknown defaults to left.
func alignFor(a string) consts.Align {
	switch strings.ToLower(a) {
	case "c", "center":
		return consts.AlignCenter
	case "r", "right":
		return consts.AlignRight
	default:
		return consts.AlignLeft
	}
}

var tagPattern = regexp.MustCompile(`<[^>]*>`)

// plainText converts the small amount of inline HTML used in design values to
// plain text: <br> becomes a newline; all other tags are stripped. Paper's text
// component renders plain text only (inline emphasis is not preserved).
func plainText(s string) string {
	s = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(s, "\n")
	return strings.TrimSpace(tagPattern.ReplaceAllString(s, ""))
}
