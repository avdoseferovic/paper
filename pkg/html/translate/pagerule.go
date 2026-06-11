package translate

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
)

// Document is the full result of translating an HTML document: the content
// rows plus document-level options parsed from `@page` rules.
type Document struct {
	Rows []core.Row
	// HeaderRows and FooterRows hold the translated content of the FIRST
	// top-level <header> / <footer> element (direct children of <body>).
	// They are meant for Paper's RegisterHeader/RegisterFooter so the bands
	// repeat on every page. Subsequent top-level header/footer elements and
	// nested ones render inline as part of Rows.
	HeaderRows []core.Row
	FooterRows []core.Row
	// Page holds size/margin options from a plain `@page { ... }` rule, or
	// nil when the document has none (or none that parsed to a usable value).
	Page *PageOptions
}

// PageOptions captures the supported subset of the CSS `@page` rule:
// `size` (named size, explicit dimensions, landscape/portrait keyword) and
// `margin` (shorthand or per-side).
type PageOptions struct {
	// PageSize is a normalized lowercase named size ("a4", "letter", ...);
	// empty when explicit Width/Height are set or only orientation was given.
	PageSize string
	// Width and Height are explicit page dimensions in mm; 0 when unset.
	Width  float64
	Height float64
	// Landscape reports the `landscape` keyword.
	Landscape bool
	// Margins in mm; -1 means unset (keep the document default).
	MarginLeft   float64
	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
}

var namedPageSizes = map[string]bool{
	"a1": true, "a2": true, "a3": true, "a4": true, "a5": true, "a6": true,
	"letter": true, "legal": true, "tabloid": true,
}

// pageOptions converts the stylesheet's collected @page declarations into
// PageOptions. Returns nil when nothing usable was declared.
func (tr *translator) pageOptions() *PageOptions {
	if tr.sheet == nil {
		return nil
	}
	return pageOptionsFromDecls(tr.sheet.pageDecls)
}

func pageOptionsFromDecls(decls []cssDeclaration) *PageOptions {
	if len(decls) == 0 {
		return nil
	}
	opts := &PageOptions{MarginLeft: -1, MarginTop: -1, MarginRight: -1, MarginBottom: -1}
	valid := false
	for _, decl := range decls {
		switch strings.ToLower(strings.TrimSpace(decl.property)) {
		case "size":
			valid = parsePageSize(opts, decl.value) || valid
		case "margin":
			valid = parsePageMarginShorthand(opts, decl.value) || valid
		case "margin-left":
			valid = setPageMargin(&opts.MarginLeft, decl.value) || valid
		case "margin-top":
			valid = setPageMargin(&opts.MarginTop, decl.value) || valid
		case "margin-right":
			valid = setPageMargin(&opts.MarginRight, decl.value) || valid
		case "margin-bottom":
			valid = setPageMargin(&opts.MarginBottom, decl.value) || valid
		}
	}
	if !valid {
		return nil
	}
	return opts
}

// parsePageSize handles `size: A4 | A4 landscape | landscape | 200mm 100mm`.
func parsePageSize(opts *PageOptions, value string) bool {
	parsed := false
	var lengths []float64
	for field := range strings.FieldsSeq(strings.ToLower(value)) {
		switch {
		case field == "landscape":
			opts.Landscape = true
			parsed = true
		case field == "portrait":
			opts.Landscape = false
			parsed = true
		case namedPageSizes[field]:
			opts.PageSize = field
			opts.Width, opts.Height = 0, 0
			parsed = true
		default:
			if v := css.ParseLength(field, 0); v > 0 {
				lengths = append(lengths, v)
			}
		}
	}
	switch len(lengths) {
	case 1:
		opts.PageSize = ""
		opts.Width, opts.Height = lengths[0], lengths[0]
		parsed = true
	case 2:
		opts.PageSize = ""
		opts.Width, opts.Height = lengths[0], lengths[1]
		parsed = true
	}
	return parsed
}

// parsePageMarginShorthand handles the 1-4 value CSS margin shorthand.
func parsePageMarginShorthand(opts *PageOptions, value string) bool {
	fields := strings.Fields(value)
	values := make([]float64, 0, len(fields))
	for _, field := range fields {
		v := css.ParseLength(field, 0)
		if v < 0 {
			v = 0
		}
		values = append(values, v)
	}
	switch len(values) {
	case 1:
		opts.MarginTop, opts.MarginRight, opts.MarginBottom, opts.MarginLeft = values[0], values[0], values[0], values[0]
	case 2:
		opts.MarginTop, opts.MarginBottom = values[0], values[0]
		opts.MarginRight, opts.MarginLeft = values[1], values[1]
	case 3:
		opts.MarginTop = values[0]
		opts.MarginRight, opts.MarginLeft = values[1], values[1]
		opts.MarginBottom = values[2]
	case 4:
		opts.MarginTop, opts.MarginRight, opts.MarginBottom, opts.MarginLeft = values[0], values[1], values[2], values[3]
	default:
		return false
	}
	return true
}

func setPageMargin(target *float64, value string) bool {
	v := css.ParseLength(strings.TrimSpace(value), 0)
	if v < 0 {
		v = 0
	}
	*target = v
	return true
}
