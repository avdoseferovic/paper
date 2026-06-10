package css

import (
	"strconv"
	"strings"
	"unicode"
)

// ParseLengthCtx is a context-aware variant of ParseLength that resolves CSS
// percentage values against contextWidthMM (the parent's content width in mm)
// and dispatches calc(...) to the small expression evaluator below.
// parentFontSize is used for em/rem unit resolution.
//
// When contextWidthMM is 0, percentages inside calc() resolve to 0.
func ParseLengthCtx(value string, parentFontSize, contextWidthMM float64) float64 {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "calc(") && strings.HasSuffix(value, ")") {
		expr := strings.TrimSpace(value[5 : len(value)-1])
		v, ok := evalCalc(expr, parentFontSize, contextWidthMM)
		if !ok {
			return 0
		}
		return v
	}
	// Percent without calc(): return as mm fraction of contextWidth (best
	// effort; existing ParseLength returns 0 for "X%").
	if strings.HasSuffix(value, "%") && contextWidthMM > 0 {
		if pct, ok := ParsePercentage(value); ok {
			return pct * contextWidthMM
		}
	}
	return ParseLength(value, parentFontSize)
}

// evalCalc evaluates a CSS calc() expression with +, -, *, /, one level of
// parentheses, mixed units, and lenient whitespace around operators.
// Returns (mm, true) on success or (0, false) on malformed input.
func evalCalc(expr string, parentFontSize, contextWidthMM float64) (float64, bool) {
	tokens, ok := tokenizeCalc(expr)
	if !ok || len(tokens) == 0 {
		return 0, false
	}
	v, _, ok := parseCalcExpr(tokens, 0, parentFontSize, contextWidthMM)
	return v, ok
}

// calcToken is a single token: a number+unit, an operator, or a paren.
type calcToken struct {
	kind  string // "num", "op", "lparen", "rparen"
	text  string
	value float64 // pre-resolved value in mm for num tokens
}

func tokenizeCalc(expr string) ([]calcToken, bool) {
	var out []calcToken
	i := 0
	for i < len(expr) {
		token, next, ok := nextCalcToken(expr, i, out)
		if !ok {
			return nil, false
		}
		if token.kind != "" {
			out = append(out, token)
		}
		i = next
	}
	return out, true
}

func nextCalcToken(expr string, pos int, previous []calcToken) (calcToken, int, bool) {
	ch := expr[pos]
	switch {
	case ch == ' ' || ch == '\t':
		return calcToken{}, pos + 1, true
	case ch == '(':
		return calcToken{kind: "lparen"}, pos + 1, true
	case ch == ')':
		return calcToken{kind: "rparen"}, pos + 1, true
	case ch == '+', ch == '*', ch == '/':
		return calcToken{kind: "op", text: string(ch)}, pos + 1, true
	case ch == '-':
		return nextMinusCalcToken(expr, pos, previous)
	case unicode.IsDigit(rune(ch)) || ch == '.':
		text, next, ok := readCalcNumberText(expr, pos)
		return calcToken{kind: "num", text: text}, next, ok
	default:
		return calcToken{}, pos, false
	}
}

func nextMinusCalcToken(expr string, pos int, previous []calcToken) (calcToken, int, bool) {
	if len(previous) > 0 && (previous[len(previous)-1].kind == "num" || previous[len(previous)-1].kind == "rparen") {
		return calcToken{kind: "op", text: "-"}, pos + 1, true
	}

	text, next, ok := readCalcNumberText(expr, pos+1)
	if !ok {
		return calcToken{}, pos, false
	}
	text = "-" + text
	v, ok := parseCalcNum(text, 0, 0)
	if !ok {
		return calcToken{}, pos, false
	}
	return calcToken{kind: "num", value: v, text: text}, next, true
}

func readCalcNumberText(expr string, pos int) (string, int, bool) {
	next := pos
	for next < len(expr) && (unicode.IsDigit(rune(expr[next])) || expr[next] == '.') {
		next++
	}
	if pos == next {
		return "", pos, false
	}
	for next < len(expr) && unicode.IsLetter(rune(expr[next])) {
		next++
	}
	if next < len(expr) && expr[next] == '%' {
		next++
	}
	return expr[pos:next], next, true
}

// parseCalcNum converts a numeric token (e.g. "10mm", "5pt", "50%") to mm.
// parentFontSize and contextWidthMM are used for em and % resolution.
func parseCalcNum(text string, parentFontSize, contextWidthMM float64) (float64, bool) {
	if strings.HasSuffix(text, "%") {
		if contextWidthMM <= 0 {
			return 0, true
		}
		if pct, ok := ParsePercentage(text); ok {
			return pct * contextWidthMM, true
		}
		return 0, false
	}
	v := ParseLength(text, parentFontSize)
	if v == 0 {
		// Could be 0 OR could be a bare number; try parsing directly.
		n, err := strconv.ParseFloat(text, 64)
		if err == nil {
			return n, true
		}
	}
	return v, true
}

// parseCalcExpr parses a calc expression at the given position. Returns
// (value, next position, ok).
func parseCalcExpr(tokens []calcToken, pos int, parentFontSize, ctxW float64) (float64, int, bool) {
	left, pos, ok := parseCalcTerm(tokens, pos, parentFontSize, ctxW)
	if !ok {
		return 0, pos, false
	}
	for pos < len(tokens) && tokens[pos].kind == "op" && (tokens[pos].text == "+" || tokens[pos].text == "-") {
		op := tokens[pos].text
		pos++
		right, npos, ok := parseCalcTerm(tokens, pos, parentFontSize, ctxW)
		if !ok {
			return 0, pos, false
		}
		pos = npos
		if op == "+" {
			left += right
		} else {
			left -= right
		}
	}
	return left, pos, true
}

func parseCalcTerm(tokens []calcToken, pos int, parentFontSize, ctxW float64) (float64, int, bool) {
	left, pos, ok := parseCalcFactor(tokens, pos, parentFontSize, ctxW)
	if !ok {
		return 0, pos, false
	}
	for pos < len(tokens) && tokens[pos].kind == "op" && (tokens[pos].text == "*" || tokens[pos].text == "/") {
		op := tokens[pos].text
		pos++
		right, npos, ok := parseCalcFactor(tokens, pos, parentFontSize, ctxW)
		if !ok {
			return 0, pos, false
		}
		pos = npos
		if op == "*" {
			left *= right
		} else {
			if right == 0 {
				return 0, pos, false
			}
			left /= right
		}
	}
	return left, pos, true
}

func parseCalcFactor(tokens []calcToken, pos int, parentFontSize, ctxW float64) (float64, int, bool) {
	if pos >= len(tokens) {
		return 0, pos, false
	}
	tk := tokens[pos]
	switch tk.kind {
	case "lparen":
		v, npos, ok := parseCalcExpr(tokens, pos+1, parentFontSize, ctxW)
		if !ok {
			return 0, npos, false
		}
		if npos >= len(tokens) || tokens[npos].kind != "rparen" {
			return 0, npos, false
		}
		return v, npos + 1, true
	case "num":
		v, ok := parseCalcNum(tk.text, parentFontSize, ctxW)
		if !ok {
			return 0, pos, false
		}
		return v, pos + 1, true
	}
	return 0, pos, false
}
