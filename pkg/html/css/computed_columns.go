package css

import (
	"strconv"
	"strings"
	"unicode"
)

func (s *ComputedStyle) applyColumnProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "column-count":
		s.applyColumnCount(ctx.val)
	case "column-width":
		s.applyColumnWidth(ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "columns":
		s.applyColumns(ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "column-span":
		switch strings.ToLower(strings.TrimSpace(ctx.val)) {
		case cssValueAll:
			s.ColumnSpan = cssValueAll
		case cssValueNone:
			s.ColumnSpan = cssValueNone
		}
	case "column-gap":
		if strings.EqualFold(strings.TrimSpace(ctx.val), "normal") {
			return true
		}
		s.ColumnGap = ParseLengthCtx(ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "column-rule-width":
		s.applyColumnRuleWidth(ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "column-rule-style":
		s.applyColumnRuleStyle(ctx.val)
	case "column-rule-color":
		if c := ParseColor(ctx.val); c != nil {
			s.ColumnRuleColor = c
		}
	case "column-rule":
		s.applyColumnRule(ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	default:
		return false
	}
	return true
}

func (s *ComputedStyle) applyColumnCount(value string) {
	if strings.EqualFold(strings.TrimSpace(value), cssValueAuto) {
		return
	}
	count, err := strconv.Atoi(strings.TrimSpace(value))
	if err == nil && count > 0 {
		s.ColumnCount = count
	}
}

func (s *ComputedStyle) applyColumnWidth(value string, parentFontSize, ctxWidth float64) {
	if strings.EqualFold(strings.TrimSpace(value), cssValueAuto) {
		return
	}
	width := ParseLengthCtx(value, parentFontSize, ctxWidth)
	if width > 0 {
		s.ColumnWidth = width
	}
}

func (s *ComputedStyle) applyColumns(value string, parentFontSize, ctxWidth float64) {
	for token := range strings.FieldsSeq(value) {
		if strings.EqualFold(token, cssValueAuto) {
			continue
		}
		count, err := strconv.Atoi(token)
		if err == nil && count > 0 {
			s.ColumnCount = count
			continue
		}
		if isColumnWidthToken(token) {
			s.applyColumnWidth(token, parentFontSize, ctxWidth)
		}
	}
}

func isColumnWidthToken(token string) bool {
	lower := strings.ToLower(strings.TrimSpace(token))
	if lower == "" {
		return false
	}
	if strings.HasPrefix(lower, "calc(") {
		return true
	}
	for _, suffix := range []string{"mm", "cm", "in", "pt", "px", "em", "rem"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func (s *ComputedStyle) applyColumnRuleWidth(value string, parentFontSize, ctxWidth float64) {
	width, ok := parseColumnRuleWidth(value, parentFontSize, ctxWidth)
	if ok {
		s.ColumnRuleWidth = width
	}
}

func (s *ComputedStyle) applyColumnRuleStyle(value string) {
	if style := normalizeColumnRuleStyle(value); style != "" {
		s.ColumnRuleStyle = style
	}
}

func (s *ComputedStyle) applyColumnRule(value string, parentFontSize, ctxWidth float64) {
	style := constColumnRuleSolid
	var width float64
	var color *RGBColor

	for _, token := range splitTopLevelWhitespace(value) {
		lower := strings.ToLower(strings.TrimSpace(token))
		if nextStyle := normalizeColumnRuleStyle(lower); nextStyle != "" {
			style = nextStyle
			continue
		}
		if c := ParseColor(token); c != nil {
			color = c
			continue
		}
		if nextWidth, ok := parseColumnRuleWidth(lower, parentFontSize, ctxWidth); ok {
			width = nextWidth
		}
	}

	if style == cssValueNone || style == cssValueHidden {
		width = 0
	}
	s.ColumnRuleWidth = width
	s.ColumnRuleStyle = style
	s.ColumnRuleColor = color
}

const constColumnRuleSolid = "solid"

func normalizeColumnRuleStyle(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case constColumnRuleSolid, "dashed", "dotted", "double", cssValueNone, cssValueHidden,
		"groove", "ridge", "inset", "outset":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func parseColumnRuleWidth(value string, parentFontSize, ctxWidth float64) (float64, bool) {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "thin":
		return mmPerPx, true
	case cssValueMedium:
		return 3 * mmPerPx, true
	case "thick":
		return 5 * mmPerPx, true
	case "":
		return 0, false
	}
	width := ParseLengthCtx(lower, parentFontSize, ctxWidth)
	if width != 0 || isZeroColumnRuleWidth(lower) {
		return width, true
	}
	return 0, false
}

func isZeroColumnRuleWidth(value string) bool {
	if value == "0" {
		return true
	}
	for _, suffix := range []string{"mm", "cm", "in", "pt", "px", "em", "rem", "%"} {
		num, ok := strings.CutSuffix(value, suffix)
		if !ok {
			continue
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
		return err == nil && parsed == 0
	}
	return false
}

func splitTopLevelWhitespace(value string) []string {
	var tokens []string
	var b strings.Builder
	depth := 0
	var quote rune
	flush := func() {
		token := strings.TrimSpace(b.String())
		if token != "" {
			tokens = append(tokens, token)
		}
		b.Reset()
	}

	for _, r := range value {
		switch {
		case quote != 0:
			b.WriteRune(r)
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
			b.WriteRune(r)
		case r == '(':
			depth++
			b.WriteRune(r)
		case r == ')':
			if depth > 0 {
				depth--
			}
			b.WriteRune(r)
		case depth == 0 && unicode.IsSpace(r):
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return tokens
}
