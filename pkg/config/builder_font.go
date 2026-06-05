package config

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/props"
)

// WithDefaultFont defines a custom font, other than arial. This can be used to define a custom font as default.
func (b *CfgBuilder) WithDefaultFont(font *props.Font) Builder {
	if font == nil {
		return b
	}

	if font.Family != "" {
		b.defaultFont.Family = font.Family
	}

	if font.Size != 0 {
		b.defaultFont.Size = font.Size
	}

	if font.Style != "" {
		b.defaultFont.Style = font.Style
	}

	if font.Color != nil {
		b.defaultFont.Color = props.CloneColor(font.Color)
	}

	return b
}

// WithPageNumber defines a string pattern to write the current page and total.
func (b *CfgBuilder) WithPageNumber(pageNumber ...props.PageNumber) Builder {
	var pageN props.PageNumber
	if len(pageNumber) > 0 {
		pageN = pageNumber[0]
	}

	if !strings.Contains(pageN.Pattern, "{current}") || !strings.Contains(pageN.Pattern, "{total}") {
		pageN.Pattern = "{current} / {total}"
	}

	if !pageN.Place.IsValid() {
		pageN.Place = props.Bottom
	}

	pageN.Color = props.CloneColor(pageN.Color)
	b.pageNumber = &pageN

	return b
}
