package css

import (
	"strconv"
	"strings"
)

func (s *ComputedStyle) applyGridProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "grid-template-columns":
		s.GridTemplateColumns = strings.ToLower(strings.TrimSpace(ctx.val))
	case "grid-template-rows":
		s.GridTemplateRows = strings.ToLower(strings.TrimSpace(ctx.val))
	case "grid-auto-flow":
		s.GridAutoFlow = strings.ToLower(strings.TrimSpace(ctx.val))
	case "grid-auto-rows":
		s.GridAutoRows = strings.ToLower(strings.TrimSpace(ctx.val))
	case "grid-template-areas":
		s.GridTemplateAreas = parseGridTemplateAreas(ctx.val)
	case "grid-area":
		s.GridArea = strings.TrimSpace(ctx.val)
	case "justify-items":
		if align := normalizeGridItemAlign(ctx.val); align != "" {
			s.JustifyItems = align
		}
	case "grid-column":
		s.GridColumnStart, s.GridColumnEnd = parseGridLine(ctx.val)
	case "grid-row":
		s.GridRowStart, s.GridRowEnd = parseGridLine(ctx.val)
	case "grid-column-start":
		if v, ok := parseGridLineNumber(ctx.val); ok {
			s.GridColumnStart = v
		}
	case "grid-column-end":
		if v, ok := parseGridLineNumber(ctx.val); ok {
			s.GridColumnEnd = v
		}
	case "grid-row-start":
		if v, ok := parseGridLineNumber(ctx.val); ok {
			s.GridRowStart = v
		}
	case "grid-row-end":
		if v, ok := parseGridLineNumber(ctx.val); ok {
			s.GridRowEnd = v
		}
	default:
		return false
	}
	return true
}

func normalizeGridItemAlign(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case cssValueStart, cssValueEnd, cssValueCenter, cssValueStretch:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func parseGridTemplateAreas(value string) [][]string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, cssValueNone) {
		return nil
	}
	var areas [][]string
	for i := 0; i < len(value); {
		qStart := -1
		for j := i; j < len(value); j++ {
			if value[j] == '"' || value[j] == '\'' {
				qStart = j
				break
			}
		}
		if qStart < 0 {
			break
		}
		quote := value[qStart]
		qEnd := -1
		for j := qStart + 1; j < len(value); j++ {
			if value[j] == quote {
				qEnd = j
				break
			}
		}
		if qEnd < 0 {
			break
		}
		if row := strings.Fields(strings.TrimSpace(value[qStart+1 : qEnd])); len(row) > 0 {
			areas = append(areas, row)
		}
		i = qEnd + 1
	}
	return areas
}

func parseGridLine(value string) (int, int) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) == 1 {
		return parseSingleGridLine(parts[0])
	}
	start, _ := parseGridLineNumber(parts[0])
	end, _ := parseGridLineEnd(parts[1], start)
	return start, end
}

func parseSingleGridLine(value string) (int, int) {
	value = strings.TrimSpace(strings.ToLower(value))
	if strings.HasPrefix(value, "span") {
		if span, ok := parseSpanCount(value); ok {
			return 0, span
		}
		return 0, 0
	}
	start, _ := parseGridLineNumber(value)
	return start, 0
}

func parseGridLineEnd(value string, start int) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if strings.HasPrefix(value, "span") {
		span, ok := parseSpanCount(value)
		if !ok {
			return 0, false
		}
		if start > 0 {
			return start + span, true
		}
		return span, true
	}
	return parseGridLineNumber(value)
}

func parseSpanCount(value string) (int, bool) {
	count := strings.TrimSpace(strings.TrimPrefix(value, "span"))
	v, err := strconv.Atoi(count)
	return v, err == nil && v > 0
}

func parseGridLineNumber(value string) (int, bool) {
	v, err := strconv.Atoi(strings.TrimSpace(value))
	return v, err == nil && v > 0
}
