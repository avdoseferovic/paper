package css

import (
	"maps"
	"strings"
)

// ExpandShorthands takes a map of CSS property→value declarations and returns
// a new map with shorthand properties expanded to their longhand equivalents.
// Unrecognised properties are passed through unchanged.
func ExpandShorthands(decls map[string]string) map[string]string {
	out := make(map[string]string, len(decls))
	for prop, val := range decls {
		maps.Copy(out, expandOne(prop, val))
	}
	return out
}

// isIdentityExpansion reports whether m is just {prop: <something>} — i.e. prop
// was not a real shorthand and expandOne returned it unchanged. Used by
// ApplyCtx to decide whether a var()-resolved value needs longhand dispatch.
func isIdentityExpansion(prop string, m map[string]string) bool {
	if len(m) != 1 {
		return false
	}
	_, ok := m[prop]
	return ok
}

func expandOne(prop, val string) map[string]string {
	// A shorthand whose value still contains an unresolved var() reference cannot
	// be split reliably (e.g. expandBackground can't recognise "var(--x)" as a
	// colour). Leave it intact; ApplyCtx re-expands it after var() resolution,
	// once the value is concrete. See ComputedStyle.ApplyCtx.
	if strings.Contains(val, "var(") {
		return map[string]string{prop: val}
	}
	switch prop {
	case "flex":
		return expandFlex(val)
	case "border":
		return expandBorderAll(val)
	case "border-top":
		return expandBorderSide("top", val)
	case "border-right":
		return expandBorderSide(cssValueRight, val)
	case "border-bottom":
		return expandBorderSide("bottom", val)
	case "border-left":
		return expandBorderSide(cssValueLeft, val)
	case "border-radius":
		return expandBorderRadius(val)
	case "padding":
		return expandBox("padding", val)
	case "margin":
		return expandBox("margin", val)
	case "font":
		return expandFont(val)
	case "background":
		return expandBackground(val)
	default:
		return map[string]string{prop: val}
	}
}

func expandBackground(val string) map[string]string {
	val = strings.TrimSpace(val)
	if val == "" || hasTopLevelComma(val) {
		return map[string]string{"background": val}
	}
	tokens := splitBackgroundTokens(val)
	if len(tokens) == 0 {
		return map[string]string{"background": val}
	}
	out := map[string]string{}
	var position []string
	var size []string
	afterSlash := false
	for _, token := range tokens {
		lower := strings.ToLower(strings.TrimSpace(token))
		if lower == "" {
			continue
		}
		if lower == "/" {
			afterSlash = true
			continue
		}
		switch {
		case isBackgroundImageToken(lower):
			out["background-image"] = token
		case lower == cssValueNone:
			out["background-image"] = cssValueNone
		case ParseColor(token) != nil:
			out["background-color"] = token
		case isBackgroundRepeatToken(lower):
			out["background-repeat"] = lower
		case isBackgroundBoxToken(lower) || isBackgroundAttachmentToken(lower):
			// background-origin, background-clip, and background-attachment are
			// accepted in the shorthand but not represented by the current PDF
			// renderer.
			continue
		case afterSlash:
			size = append(size, lower)
		default:
			position = append(position, lower)
		}
	}
	if len(position) > 0 {
		out["background-position"] = strings.Join(position, " ")
	}
	if len(size) > 0 {
		out["background-size"] = strings.Join(size, " ")
	}
	if len(out) == 0 {
		return map[string]string{"background": val}
	}
	return out
}

func isBackgroundImageToken(lower string) bool {
	return strings.HasPrefix(lower, "url(") ||
		strings.HasPrefix(lower, "linear-gradient(") ||
		strings.HasPrefix(lower, "radial-gradient(") ||
		strings.HasPrefix(lower, "conic-gradient(")
}

func isBackgroundRepeatToken(lower string) bool {
	switch lower {
	case "repeat", "no-repeat", "repeat-x", "repeat-y":
		return true
	default:
		return false
	}
}

func isBackgroundBoxToken(lower string) bool {
	switch lower {
	case "border-box", "padding-box", "content-box":
		return true
	default:
		return false
	}
}

func isBackgroundAttachmentToken(lower string) bool {
	switch lower {
	case "scroll", "fixed", "local":
		return true
	default:
		return false
	}
}

// splitBackgroundTokens splits a background shorthand into tokens, emitting a
// top-level "/" (the position/size separator) as its own token.
func splitBackgroundTokens(value string) []string {
	return splitTopLevelTokens(value, true)
}

func hasTopLevelComma(value string) bool {
	depth := 0
	var quote rune
	for _, r := range value {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '(':
			depth++
		case r == ')':
			if depth > 0 {
				depth--
			}
		case r == ',' && depth == 0:
			return true
		}
	}
	return false
}

// expandBorderAll expands "border: <width> <style> <color>" to all 12 longhands.
func expandBorderAll(val string) map[string]string {
	width, style, color := parseBorderTriple(val)
	out := make(map[string]string, 12)
	for _, side := range []string{"top", cssValueRight, "bottom", cssValueLeft} {
		out["border-"+side+"-width"] = width
		out["border-"+side+"-style"] = style
		out["border-"+side+"-color"] = color
	}
	return out
}

// expandBorderSide expands "border-{side}: <width> <style> <color>" to 3 longhands.
func expandBorderSide(side, val string) map[string]string {
	width, style, color := parseBorderTriple(val)
	return map[string]string{
		"border-" + side + "-width": width,
		"border-" + side + "-style": style,
		"border-" + side + "-color": color,
	}
}

// parseBorderTriple splits a "1px solid red" border shorthand into its three parts.
// Parts may be in any order (width=has unit or width keyword, style=keyword,
// color=otherwise). Function colors like rgb(255, 0, 0) are kept as single
// tokens via paren-aware splitting.
func parseBorderTriple(val string) (string, string, string) {
	parts := splitTopLevelWhitespace(val)
	borderStyles := map[string]bool{
		cssValueNone: true, "hidden": true, "dotted": true, "dashed": true,
		"solid": true, "double": true, "groove": true, "ridge": true,
		"inset": true, "outset": true,
	}
	borderWidthKeywords := map[string]bool{"thin": true, cssValueMedium: true, "thick": true}
	width, style, colorVal := "", "", ""
	for _, p := range parts {
		lower := strings.ToLower(p)
		switch {
		case borderStyles[lower]:
			style = lower
		case isLengthValue(p) || borderWidthKeywords[lower]:
			width = p
		default:
			colorVal = p
		}
	}
	if width == "" {
		width = cssValueMedium
	}
	if style == "" {
		style = cssValueNone
	}
	if colorVal == "" {
		colorVal = "currentColor"
	}
	return width, style, colorVal
}

// expandBorderRadius expands the CSS border-radius shorthand into 4 per-corner longhands.
// Per CSS spec the value order is clockwise from top-left:
//
//	1 value:  all 4 corners
//	2 values: tl+br, tr+bl
//	3 values: tl, tr+bl, br
//	4 values: tl, tr, br, bl
func expandBorderRadius(val string) map[string]string {
	parts := strings.Fields(val)
	var tl, tr, br, bl string
	switch len(parts) {
	case 1:
		tl, tr, br, bl = parts[0], parts[0], parts[0], parts[0]
	case 2:
		tl, br = parts[0], parts[0]
		tr, bl = parts[1], parts[1]
	case 3:
		tl = parts[0]
		tr, bl = parts[1], parts[1]
		br = parts[2]
	case 4:
		tl, tr, br, bl = parts[0], parts[1], parts[2], parts[3]
	default:
		return map[string]string{"border-radius": val}
	}
	return map[string]string{
		"border-top-left-radius":     tl,
		"border-top-right-radius":    tr,
		"border-bottom-right-radius": br,
		"border-bottom-left-radius":  bl,
	}
}

// expandBox expands a box shorthand (padding/margin) into 4 longhands.
// Invalid declarations (0 or 5+ values) are dropped entirely, matching the
// CSS rule that malformed declarations are ignored rather than zeroed.
func expandBox(prefix, val string) map[string]string {
	parts := strings.Fields(val)
	var top, right, bottom, left string
	switch len(parts) {
	case 1:
		top, right, bottom, left = parts[0], parts[0], parts[0], parts[0]
	case 2:
		top, right, bottom, left = parts[0], parts[1], parts[0], parts[1]
	case 3:
		top, right, bottom, left = parts[0], parts[1], parts[2], parts[1]
	case 4:
		top, right, bottom, left = parts[0], parts[1], parts[2], parts[3]
	default:
		return map[string]string{}
	}
	return map[string]string{
		prefix + "-top":    top,
		prefix + "-right":  right,
		prefix + "-bottom": bottom,
		prefix + "-left":   left,
	}
}

// expandFont handles the "font: [style] [weight] <size>[/<line-height>]
// <family>" shorthand.
func expandFont(val string) map[string]string {
	parts := strings.Fields(val)
	out := map[string]string{"font": val}
	fontStyles := map[string]bool{"italic": true, "oblique": true}
	fontWeights := map[string]bool{
		"bold": true, "bolder": true, "lighter": true,
		"100": true, "200": true, "300": true, "400": true, "500": true,
		"600": true, "700": true, "800": true, "900": true,
	}
	for i, p := range parts {
		sizePart, lineHeight, hasLineHeight := strings.Cut(p, "/")
		if !isLengthValue(sizePart) {
			continue
		}
		out["font-size"] = sizePart
		if hasLineHeight && lineHeight != "" {
			out["line-height"] = lineHeight
		}
		if i+1 < len(parts) {
			out["font-family"] = strings.Join(parts[i+1:], " ")
		}
		for _, prefix := range parts[:i] {
			lower := strings.ToLower(prefix)
			switch {
			case fontStyles[lower]:
				out["font-style"] = lower
			case fontWeights[lower]:
				out["font-weight"] = lower
			}
		}
		delete(out, "font")
		return out
	}
	return out
}

// isLengthValue returns true when the token looks like a CSS length.
func isLengthValue(s string) bool {
	for _, suffix := range []string{"px", "pt", "mm", "cm", "em", "rem", "%"} {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// isBasisToken returns true when the token is a valid flex-basis value
// (length, percentage, or the keyword "auto").
func isBasisToken(s string) bool {
	return s == cssValueAuto || isLengthValue(s)
}

// expandFlex expands the CSS flex shorthand into flex-grow, flex-shrink, flex-basis.
// Supports: none | auto | initial | <number> | <number> <basis> | <number> <number> | <grow> <shrink> <basis>
func expandFlex(val string) map[string]string {
	switch val {
	case cssValueNone:
		return map[string]string{"flex-grow": "0", "flex-shrink": "0", "flex-basis": cssValueAuto}
	case cssValueAuto:
		return map[string]string{"flex-grow": "1", "flex-shrink": "1", "flex-basis": cssValueAuto}
	case "initial":
		return map[string]string{"flex-grow": "0", "flex-shrink": "1", "flex-basis": cssValueAuto}
	}

	parts := strings.Fields(val)
	switch len(parts) {
	case 1:
		// Single number: flex-grow only; shrink=1, basis=0
		return map[string]string{"flex-grow": parts[0], "flex-shrink": "1", "flex-basis": "0"}
	case 2:
		if isBasisToken(parts[1]) {
			// <grow> <basis>
			return map[string]string{"flex-grow": parts[0], "flex-shrink": "1", "flex-basis": parts[1]}
		}
		// <grow> <shrink>
		return map[string]string{"flex-grow": parts[0], "flex-shrink": parts[1], "flex-basis": "0"}
	case 3:
		return map[string]string{"flex-grow": parts[0], "flex-shrink": parts[1], "flex-basis": parts[2]}
	default:
		return map[string]string{"flex": val}
	}
}
