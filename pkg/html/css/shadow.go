package css

import (
	"fmt"
	"strings"
)

// Shadow holds the parsed values of a single CSS shadow entry (box-shadow or text-shadow).
type Shadow struct {
	OffsetX    float64   // mm
	OffsetY    float64   // mm
	BlurRadius float64   // mm; 0 when omitted
	Spread     float64   // mm; 0 when omitted (box-shadow only)
	Color      *RGBColor // nil when no color specified
	Inset      bool      // true when "inset" keyword is present
}

// ParseShadow parses a CSS shadow value string (box-shadow or text-shadow format).
// It supports multi-shadow comma-separated lists (up to 4 entries).
// Returns an error when the value is empty or unparseable.
func ParseShadow(val string) ([]Shadow, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil, fmt.Errorf("shadow: empty value")
	}
	parts := splitTopLevel(val, ',')
	var shadows []Shadow
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		s, err := parseSingleShadow(p)
		if err != nil {
			return nil, fmt.Errorf("shadow %q: %w", p, err)
		}
		shadows = append(shadows, s)
		if len(shadows) >= 4 {
			break
		}
	}
	if len(shadows) == 0 {
		return nil, fmt.Errorf("shadow: no valid shadows in %q", val)
	}
	return shadows, nil
}

// parseSingleShadow parses one shadow entry: [inset] <x> <y> [blur] [spread] [color].
func parseSingleShadow(val string) (Shadow, error) {
	tokens := strings.Fields(val)
	if len(tokens) == 0 {
		return Shadow{}, fmt.Errorf("empty shadow entry")
	}

	var s Shadow
	remaining := tokens

	// Check for "inset" keyword (can appear first or last per spec).
	var lengths []string
	var colorStr string
	for _, tok := range remaining {
		lower := strings.ToLower(tok)
		if lower == "inset" {
			s.Inset = true
			continue
		}
		// Try parsing as a length.
		if isLengthToken(tok) {
			lengths = append(lengths, tok)
		} else {
			// Assume it's (part of) a color. Since color functions can be
			// multi-token (e.g. "rgb(0, 0, 0)") they won't appear here because
			// we split on Fields — so we join non-length, non-inset tokens.
			colorStr = tok
		}
	}

	if len(lengths) < 2 {
		return Shadow{}, fmt.Errorf("shadow needs at least x and y offsets, got %q", val)
	}

	s.OffsetX = ParseLength(lengths[0], 0)
	s.OffsetY = ParseLength(lengths[1], 0)
	if len(lengths) >= 3 {
		s.BlurRadius = ParseLength(lengths[2], 0)
	}
	if len(lengths) >= 4 {
		s.Spread = ParseLength(lengths[3], 0)
	}

	if colorStr != "" {
		s.Color = ParseColor(colorStr)
	}

	return s, nil
}

// isLengthToken returns true when tok looks like a CSS length value.
func isLengthToken(tok string) bool {
	if tok == "0" {
		return true
	}
	suffixes := []string{"mm", "cm", "in", "pt", "px", "em", "rem", "%"}
	for _, sfx := range suffixes {
		if strings.HasSuffix(tok, sfx) {
			return true
		}
	}
	return false
}
