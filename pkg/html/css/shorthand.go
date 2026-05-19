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

func expandOne(prop, val string) map[string]string {
	switch prop {
	case "flex":
		return expandFlex(val)
	case "border":
		return expandBorderAll(val)
	case "border-top":
		return expandBorderSide("top", val)
	case "border-right":
		return expandBorderSide("right", val)
	case "border-bottom":
		return expandBorderSide("bottom", val)
	case "border-left":
		return expandBorderSide("left", val)
	case "padding":
		return expandBox("padding", val)
	case "margin":
		return expandBox("margin", val)
	case "font":
		return expandFont(val)
	default:
		return map[string]string{prop: val}
	}
}

// expandBorderAll expands "border: <width> <style> <color>" to all 12 longhands.
func expandBorderAll(val string) map[string]string {
	width, style, color := parseBorderTriple(val)
	out := make(map[string]string, 12)
	for _, side := range []string{"top", "right", "bottom", "left"} {
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
// Parts may be in any order (width=has unit, style=keyword, color=otherwise).
func parseBorderTriple(val string) (string, string, string) {
	parts := strings.Fields(val)
	borderStyles := map[string]bool{
		"none": true, "hidden": true, "dotted": true, "dashed": true,
		"solid": true, "double": true, "groove": true, "ridge": true,
		"inset": true, "outset": true,
	}
	width, style, colorVal := "", "", ""
	for _, p := range parts {
		switch {
		case borderStyles[p]:
			style = p
		case isLengthValue(p):
			width = p
		default:
			colorVal = p
		}
	}
	if width == "" {
		width = "medium"
	}
	if style == "" {
		style = "none"
	}
	if colorVal == "" {
		colorVal = "currentColor"
	}
	return width, style, colorVal
}

// expandBox expands a box shorthand (padding/margin) into 4 longhands.
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
		top, right, bottom, left = "0", "0", "0", "0"
	}
	return map[string]string{
		prefix + "-top":    top,
		prefix + "-right":  right,
		prefix + "-bottom": bottom,
		prefix + "-left":   left,
	}
}

// expandFont handles the simplified "font: <size> <family>" shorthand.
func expandFont(val string) map[string]string {
	parts := strings.Fields(val)
	out := map[string]string{"font": val}
	for i, p := range parts {
		if isLengthValue(p) {
			out["font-size"] = p
			if i+1 < len(parts) {
				out["font-family"] = strings.Join(parts[i+1:], " ")
			}
			delete(out, "font")
			return out
		}
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
	return s == "auto" || isLengthValue(s)
}

// expandFlex expands the CSS flex shorthand into flex-grow, flex-shrink, flex-basis.
// Supports: none | auto | initial | <number> | <number> <basis> | <number> <number> | <grow> <shrink> <basis>
func expandFlex(val string) map[string]string {
	switch val {
	case "none":
		return map[string]string{"flex-grow": "0", "flex-shrink": "0", "flex-basis": "auto"}
	case "auto":
		return map[string]string{"flex-grow": "1", "flex-shrink": "1", "flex-basis": "auto"}
	case "initial":
		return map[string]string{"flex-grow": "0", "flex-shrink": "1", "flex-basis": "auto"}
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
