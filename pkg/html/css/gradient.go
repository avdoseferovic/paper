package css

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// GradientKind identifies the type of gradient.
type GradientKind int

const (
	GradientLinear GradientKind = iota
	GradientRadial
)

// Gradient is a union of LinearGradient and RadialGradient. Only the field
// corresponding to Kind is populated.
type Gradient struct {
	Kind   GradientKind
	Linear *LinearGradient
	Radial *RadialGradient
}

// GradientStop is a single color+position stop in a gradient.
type GradientStop struct {
	Color    RGBColor
	Position float64 // 0.0–1.0; -1 means "auto" (evenly distributed)
}

// LinearGradient holds a parsed linear-gradient().
type LinearGradient struct {
	AngleDeg float64
	Stops    []GradientStop
}

// RadialGradient holds a parsed radial-gradient().
type RadialGradient struct {
	Circle bool
	CX, CY float64 // centre as fraction of width/height (0.5 = center)
	Stops  []GradientStop
}

// ParseLinearGradient parses a CSS linear-gradient(...) function string,
// including the "linear-gradient(" prefix and closing ")".
func ParseLinearGradient(s string) (*LinearGradient, error) {
	inner, err := extractFuncArgs("linear-gradient", s)
	if err != nil {
		return nil, err
	}
	parts := splitTopLevel(inner, ',')
	if len(parts) < 2 {
		return nil, fmt.Errorf("linear-gradient: need at least 2 parts, got %q", inner)
	}
	angleDeg, stopParts := parseLinearDirection(parts)
	stops, err := parseStops(stopParts)
	if err != nil {
		return nil, fmt.Errorf("linear-gradient stops: %w", err)
	}
	distributeStops(stops)
	return &LinearGradient{AngleDeg: angleDeg, Stops: stops}, nil
}

// ParseRadialGradient parses a CSS radial-gradient(...) function string.
func ParseRadialGradient(s string) (*RadialGradient, error) {
	inner, err := extractFuncArgs("radial-gradient", s)
	if err != nil {
		return nil, err
	}
	parts := splitTopLevel(inner, ',')
	if len(parts) < 2 {
		return nil, fmt.Errorf("radial-gradient: need at least 2 parts, got %q", inner)
	}
	circle, cx, cy, stopParts := parseRadialShape(parts)
	stops, err := parseStops(stopParts)
	if err != nil {
		return nil, fmt.Errorf("radial-gradient stops: %w", err)
	}
	distributeStops(stops)
	return &RadialGradient{Circle: circle, CX: cx, CY: cy, Stops: stops}, nil
}

// extractFuncArgs strips "funcName(" prefix and trailing ")" and returns the
// content inside. Returns an error if the function name doesn't match.
func extractFuncArgs(name, s string) (string, error) {
	s = strings.TrimSpace(s)
	prefix := name + "("
	if !strings.HasPrefix(s, prefix) || !strings.HasSuffix(s, ")") {
		return "", fmt.Errorf("not a %s() call: %q", name, s)
	}
	return strings.TrimSpace(s[len(prefix) : len(s)-1]), nil
}

// splitTopLevel splits s on sep, but not when inside parentheses.
func splitTopLevel(s string, sep rune) []string {
	var out []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case sep:
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

// parseLinearDirection extracts the angle from the first part and returns the
// remaining parts (colour stops).
func parseLinearDirection(parts []string) (angleDeg float64, stopParts []string) {
	first := strings.ToLower(strings.TrimSpace(parts[0]))
	switch first {
	case "to right":
		return 90, parts[1:]
	case "to left":
		return 270, parts[1:]
	case "to bottom":
		return 180, parts[1:]
	case "to top":
		return 0, parts[1:]
	case "to top right", "to right top":
		return 45, parts[1:]
	case "to bottom right", "to right bottom":
		return 135, parts[1:]
	case "to bottom left", "to left bottom":
		return 225, parts[1:]
	case "to top left", "to left top":
		return 315, parts[1:]
	default:
		if strings.HasSuffix(first, "deg") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(first, "deg"), 64)
			if err == nil {
				return v, parts[1:]
			}
		}
		if strings.HasSuffix(first, "turn") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(first, "turn"), 64)
			if err == nil {
				return v * 360, parts[1:]
			}
		}
		if strings.HasSuffix(first, "rad") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(first, "rad"), 64)
			if err == nil {
				return v * 180 / math.Pi, parts[1:]
			}
		}
		// First part is a colour, not a direction.
		return 180, parts // default to "to bottom"
	}
}

// parseRadialShape extracts optional "circle at X" from the first part.
func parseRadialShape(parts []string) (circle bool, cx, cy float64, stopParts []string) {
	first := strings.ToLower(strings.TrimSpace(parts[0]))
	if strings.HasPrefix(first, "circle") || strings.HasPrefix(first, "ellipse") {
		circle = strings.HasPrefix(first, "circle")
		cx, cy = 0.5, 0.5 // default centre
		if idx := strings.Index(first, "at "); idx >= 0 {
			pos := strings.TrimSpace(first[idx+3:])
			cx, cy = parseRadialPosition(pos)
		}
		return circle, cx, cy, parts[1:]
	}
	// No shape qualifier; treat first part as a stop.
	return true, 0.5, 0.5, parts
}

func parseRadialPosition(pos string) (cx, cy float64) {
	switch pos {
	case "center", "center center":
		return 0.5, 0.5
	case "top":
		return 0.5, 0.0
	case "bottom":
		return 0.5, 1.0
	case "left":
		return 0.0, 0.5
	case "right":
		return 1.0, 0.5
	case "top left", "left top":
		return 0.0, 0.0
	case "top right", "right top":
		return 1.0, 0.0
	case "bottom left", "left bottom":
		return 0.0, 1.0
	case "bottom right", "right bottom":
		return 1.0, 1.0
	default:
		return 0.5, 0.5
	}
}

// parseStops converts each comma-separated token into a GradientStop.
func parseStops(parts []string) ([]GradientStop, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("gradient needs at least 2 stops")
	}
	var stops []GradientStop
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Try to split "color position" — e.g. "red 25%" or "#fff 0%"
		// The colour can itself contain spaces (e.g. "rgb(0, 0, 0)"), so we
		// split from the right: the last space-delimited token may be a %.
		colorStr, posStr := splitColorAndPosition(p)
		c := ParseColor(colorStr)
		if c == nil {
			return nil, fmt.Errorf("unknown color in stop: %q", colorStr)
		}
		pos := -1.0
		if posStr != "" {
			if strings.HasSuffix(posStr, "%") {
				v, err := strconv.ParseFloat(strings.TrimSuffix(posStr, "%"), 64)
				if err != nil {
					return nil, fmt.Errorf("bad stop position %q: %w", posStr, err)
				}
				pos = v / 100.0
			}
		}
		stops = append(stops, GradientStop{Color: *c, Position: pos})
	}
	return stops, nil
}

// splitColorAndPosition separates "red 25%" into ("red", "25%").
// When no position token is present, returns (s, "").
func splitColorAndPosition(s string) (colorStr, posStr string) {
	// Look for trailing "NNN%" or "NNNpx" or "NNNmm" token.
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return s, ""
	}
	last := fields[len(fields)-1]
	if strings.HasSuffix(last, "%") ||
		strings.HasSuffix(last, "px") || strings.HasSuffix(last, "mm") ||
		strings.HasSuffix(last, "pt") || strings.HasSuffix(last, "em") {
		colorStr = strings.TrimSpace(strings.Join(fields[:len(fields)-1], " "))
		return colorStr, last
	}
	return s, ""
}

// distributeStops fills in -1 positions evenly between known anchors.
func distributeStops(stops []GradientStop) {
	if len(stops) == 0 {
		return
	}
	// First and last default to 0 and 1 if not set.
	if stops[0].Position < 0 {
		stops[0].Position = 0
	}
	if stops[len(stops)-1].Position < 0 {
		stops[len(stops)-1].Position = 1
	}
	// Fill interior auto stops by linear interpolation between anchors.
	i := 0
	for i < len(stops) {
		if stops[i].Position >= 0 {
			i++
			continue
		}
		// Find next known position.
		j := i + 1
		for j < len(stops) && stops[j].Position < 0 {
			j++
		}
		// Interpolate between i-1 and j.
		startPos := stops[i-1].Position
		endPos := stops[j].Position
		count := j - i + 1
		for k := i; k < j; k++ {
			stops[k].Position = startPos + (endPos-startPos)*float64(k-i+1)/float64(count)
		}
		i = j + 1
	}
}
