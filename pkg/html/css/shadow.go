package css

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errShadowEmpty        = errors.New("shadow value is empty")
	errShadowNoValidEntry = errors.New("shadow has no valid entries")
	errDropShadowMissing  = errors.New("filter has no drop-shadow")
	errFilterInvalidFunc  = errors.New("filter has an invalid function")
	errShadowEntryEmpty   = errors.New("shadow entry is empty")
	errShadowOffsets      = errors.New("shadow needs at least x and y offsets")
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
		return nil, errShadowEmpty
	}
	parts := splitTopLevel(val)
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
		return nil, fmt.Errorf("%w: %q", errShadowNoValidEntry, val)
	}
	return shadows, nil
}

// ParseFilterDropShadow extracts CSS filter drop-shadow(...) functions and
// parses them into ordinary shadows for the PDF cell shadow renderer.
func ParseFilterDropShadow(val string) ([]Shadow, error) {
	val = strings.TrimSpace(val)
	if val == "" || strings.EqualFold(val, cssValueNone) {
		return nil, errDropShadowMissing
	}
	var shadows []Shadow
	for val != "" {
		val = strings.TrimLeft(val, " \t\r\n\f")
		if val == "" {
			break
		}
		nameEnd := strings.IndexByte(val, '(')
		if nameEnd < 0 {
			break
		}
		name := strings.ToLower(strings.TrimSpace(val[:nameEnd]))
		args, rest, ok := readFilterFunction(val[nameEnd:])
		if !ok {
			return nil, fmt.Errorf("%w: %q", errFilterInvalidFunc, val)
		}
		if name == "drop-shadow" {
			shadow, err := parseSingleShadow(args)
			if err != nil {
				return nil, fmt.Errorf("drop-shadow %q: %w", args, err)
			}
			shadow.Spread = 0
			shadow.Inset = false
			shadows = append(shadows, shadow)
			if len(shadows) >= 4 {
				break
			}
		}
		val = rest
	}
	if len(shadows) == 0 {
		return nil, errDropShadowMissing
	}
	return shadows, nil
}

func readFilterFunction(value string) (string, string, bool) {
	if !strings.HasPrefix(value, "(") {
		return "", "", false
	}
	depth := 1
	var quote rune
	escaped := false
	for i, r := range value[1:] {
		pos := i + 1
		switch {
		case quote != 0:
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
		default:
			switch r {
			case '"', '\'':
				quote = r
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					return value[1:pos], value[pos+1:], true
				}
			}
		}
	}
	return "", "", false
}

// parseSingleShadow parses one shadow entry: [inset] <x> <y> [blur] [spread] [color].
func parseSingleShadow(val string) (Shadow, error) {
	tokens := strings.Fields(val)
	if len(tokens) == 0 {
		return Shadow{}, errShadowEntryEmpty
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
		return Shadow{}, fmt.Errorf("%w: got %q", errShadowOffsets, val)
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
