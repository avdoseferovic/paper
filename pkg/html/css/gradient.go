package css

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var (
	errLinearGradientParts = errors.New("linear-gradient needs at least 2 parts")
	errRadialGradientParts = errors.New("radial-gradient needs at least 2 parts")
	errConicGradientParts  = errors.New("conic-gradient needs at least 2 parts")
	errInvalidGradientFunc = errors.New("invalid gradient function")
	errGradientStopCount   = errors.New("gradient needs at least 2 stops")
	errGradientStopColor   = errors.New("unknown color in gradient stop")
	errGradientStopAngle   = errors.New("bad gradient stop angle")
)

// GradientKind identifies the type of gradient.
type GradientKind int

const (
	GradientLinear GradientKind = iota
	GradientRadial
	GradientConic
)

// Gradient is a union of LinearGradient and RadialGradient. Only the field
// corresponding to Kind is populated.
type Gradient struct {
	Kind   GradientKind
	Linear *LinearGradient
	Radial *RadialGradient
	Conic  *ConicGradient
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

// ConicGradient holds a parsed conic-gradient().
type ConicGradient struct {
	FromDeg float64
	CX, CY  float64 // centre as fraction of width/height (0.5 = center)
	Stops   []GradientStop
}

// ParseLinearGradient parses a CSS linear-gradient(...) function string,
// including the "linear-gradient(" prefix and closing ")".
func ParseLinearGradient(s string) (*LinearGradient, error) {
	inner, err := extractFuncArgs("linear-gradient", s)
	if err != nil {
		return nil, err
	}
	parts := splitTopLevel(inner)
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: got %q", errLinearGradientParts, inner)
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
	parts := splitTopLevel(inner)
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: got %q", errRadialGradientParts, inner)
	}
	circle, cx, cy, stopParts := parseRadialShape(parts)
	stops, err := parseStops(stopParts)
	if err != nil {
		return nil, fmt.Errorf("radial-gradient stops: %w", err)
	}
	distributeStops(stops)
	return &RadialGradient{Circle: circle, CX: cx, CY: cy, Stops: stops}, nil
}

// ParseConicGradient parses a CSS conic-gradient(...) function string.
func ParseConicGradient(s string) (*ConicGradient, error) {
	inner, err := extractFuncArgs("conic-gradient", s)
	if err != nil {
		return nil, err
	}
	parts := splitTopLevel(inner)
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: got %q", errConicGradientParts, inner)
	}
	fromDeg, cx, cy, stopParts := parseConicPrelude(parts)
	stops, err := parseStops(stopParts)
	if err != nil {
		return nil, fmt.Errorf("conic-gradient stops: %w", err)
	}
	distributeStops(stops)
	return &ConicGradient{FromDeg: fromDeg, CX: cx, CY: cy, Stops: stops}, nil
}

// extractFuncArgs strips "funcName(" prefix and trailing ")" and returns the
// content inside. Returns an error if the function name doesn't match.
func extractFuncArgs(name, s string) (string, error) {
	s = strings.TrimSpace(s)
	prefix := name + "("
	if !strings.HasPrefix(s, prefix) || !strings.HasSuffix(s, ")") {
		return "", fmt.Errorf("%w: expected %s(): %q", errInvalidGradientFunc, name, s)
	}
	return strings.TrimSpace(s[len(prefix) : len(s)-1]), nil
}

// splitTopLevel splits s on commas, but not when inside parentheses.
func splitTopLevel(s string) []string {
	var out []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
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
func parseLinearDirection(parts []string) (float64, []string) {
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
		if value, ok := strings.CutSuffix(first, "deg"); ok {
			v, err := strconv.ParseFloat(value, 64)
			if err == nil {
				return v, parts[1:]
			}
		}
		if value, ok := strings.CutSuffix(first, "turn"); ok {
			v, err := strconv.ParseFloat(value, 64)
			if err == nil {
				return v * 360, parts[1:]
			}
		}
		if value, ok := strings.CutSuffix(first, "rad"); ok {
			v, err := strconv.ParseFloat(value, 64)
			if err == nil {
				return v * 180 / math.Pi, parts[1:]
			}
		}
		// First part is a colour, not a direction.
		return 180, parts // default to "to bottom"
	}
}

// parseRadialShape extracts optional "circle at X" from the first part.
func parseRadialShape(parts []string) (bool, float64, float64, []string) {
	first := strings.ToLower(strings.TrimSpace(parts[0]))
	if strings.HasPrefix(first, "circle") || strings.HasPrefix(first, "ellipse") {
		circle := strings.HasPrefix(first, "circle")
		cx, cy := 0.5, 0.5 // default centre
		if _, pos, ok := strings.Cut(first, "at "); ok {
			pos = strings.TrimSpace(pos)
			cx, cy = parseRadialPosition(pos)
		}
		return circle, cx, cy, parts[1:]
	}
	// No shape qualifier; treat first part as a stop.
	return true, 0.5, 0.5, parts
}

func parseRadialPosition(pos string) (float64, float64) {
	switch pos {
	case "center", "center " + "center":
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

func parseConicPrelude(parts []string) (float64, float64, float64, []string) {
	fromDeg, cx, cy := 0.0, 0.5, 0.5
	first := strings.ToLower(strings.TrimSpace(parts[0]))
	if !strings.HasPrefix(first, "from ") && !strings.HasPrefix(first, "at ") && !strings.Contains(first, " at ") {
		return fromDeg, cx, cy, parts
	}
	fields := strings.Fields(first)
	for i := 0; i < len(fields); i++ {
		switch fields[i] {
		case "from":
			if i+1 < len(fields) {
				if deg, ok := parseAngleDeg(fields[i+1]); ok {
					fromDeg = deg
				}
				i++
			}
		case "at":
			if i+1 < len(fields) {
				cx, cy = parseRadialPosition(strings.Join(fields[i+1:], " "))
			}
			i = len(fields)
		}
	}
	return fromDeg, cx, cy, parts[1:]
}

// parseStops converts each comma-separated token into a GradientStop.
func parseStops(parts []string) ([]GradientStop, error) {
	if len(parts) < 2 {
		return nil, errGradientStopCount
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
			return nil, fmt.Errorf("%w: %q", errGradientStopColor, colorStr)
		}
		pos := -1.0
		if posStr != "" {
			parsed, ok, err := parseGradientStopPosition(posStr)
			if err != nil {
				return nil, err
			}
			if ok {
				pos = parsed
			}
		}
		stops = append(stops, GradientStop{Color: *c, Position: pos})
	}
	return stops, nil
}

// splitColorAndPosition separates "red 25%" into ("red", "25%").
// When no position token is present, returns (s, "").
func splitColorAndPosition(s string) (string, string) {
	// Look for trailing "NNN%" or "NNNpx" or "NNNmm" token.
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return s, ""
	}
	last := fields[len(fields)-1]
	if strings.HasSuffix(last, "%") ||
		strings.HasSuffix(last, "deg") || strings.HasSuffix(last, "turn") ||
		strings.HasSuffix(last, "rad") ||
		strings.HasSuffix(last, "px") || strings.HasSuffix(last, "mm") ||
		strings.HasSuffix(last, "pt") || strings.HasSuffix(last, "em") {
		colorStr := strings.TrimSpace(strings.Join(fields[:len(fields)-1], " "))
		return colorStr, last
	}
	return s, ""
}

func parseGradientStopPosition(posStr string) (float64, bool, error) {
	switch {
	case strings.HasSuffix(posStr, "%"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(posStr, "%"), 64)
		if err != nil {
			return 0, false, fmt.Errorf("bad stop position %q: %w", posStr, err)
		}
		return v / 100.0, true, nil
	case strings.HasSuffix(posStr, "deg") || strings.HasSuffix(posStr, "turn") || strings.HasSuffix(posStr, "rad"):
		deg, ok := parseAngleDeg(posStr)
		if !ok {
			return 0, false, fmt.Errorf("%w: %q", errGradientStopAngle, posStr)
		}
		return deg / 360.0, true, nil
	default:
		return -1, false, nil
	}
}

func parseAngleDeg(value string) (float64, bool) {
	switch {
	case strings.HasSuffix(value, "deg"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(value, "deg"), 64)
		return v, err == nil
	case strings.HasSuffix(value, "turn"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(value, "turn"), 64)
		return v * 360, err == nil
	case strings.HasSuffix(value, "rad"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(value, "rad"), 64)
		return v * 180 / math.Pi, err == nil
	default:
		return 0, false
	}
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
