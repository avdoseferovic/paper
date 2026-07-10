package translate

import "strings"

type selectorFunctionLocation struct {
	start     int
	argsStart int
	argsEnd   int
	end       int
}

func expandIsWhereSelectors(selector string) []string {
	expanded := []string{strings.TrimSpace(selector)}
	for {
		next := make([]string, 0, len(expanded))
		changed := false
		for _, sel := range expanded {
			items, didExpand := expandOneIsWhereSelector(sel)
			next = append(next, items...)
			if didExpand {
				changed = true
			}
		}
		expanded = next
		if !changed {
			return expanded
		}
	}
}

func expandOneIsWhereSelector(selector string) ([]string, bool) {
	loc, ok := findIsWhereSelectorFunction(selector)
	if !ok {
		if selector == "" {
			return nil, false
		}
		return []string{selector}, false
	}
	options := splitSelectorFunctionArgs(selector[loc.argsStart:loc.argsEnd])
	if len(options) == 0 {
		if selector == "" {
			return nil, false
		}
		return []string{selector}, false
	}
	var out []string
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option == "" {
			continue
		}
		nextSelector := strings.TrimSpace(selector[:loc.start] + option + selector[loc.end:])
		if nextSelector != "" {
			out = append(out, nextSelector)
		}
	}
	if len(out) == 0 {
		return []string{selector}, false
	}
	return out, true
}

func findIsWhereSelectorFunction(selector string) (selectorFunctionLocation, bool) {
	quote := byte(0)
	escaped := false
	bracketDepth := 0
	parenDepth := 0
	for i := range len(selector) {
		ch := selector[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
			continue
		case '[':
			bracketDepth++
			continue
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		case '(':
			parenDepth++
			continue
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			continue
		}
		if bracketDepth == 0 && parenDepth == 0 {
			if loc, ok := selectorFunctionAt(selector, i, ":is("); ok {
				return loc, true
			}
			if loc, ok := selectorFunctionAt(selector, i, ":where("); ok {
				return loc, true
			}
		}
	}
	return selectorFunctionLocation{}, false
}

func selectorFunctionAt(selector string, start int, prefix string) (selectorFunctionLocation, bool) {
	if start+len(prefix) > len(selector) || !strings.EqualFold(selector[start:start+len(prefix)], prefix) {
		return selectorFunctionLocation{}, false
	}
	open := start + len(prefix) - 1
	closeIdx, ok := findSelectorFunctionClose(selector, open)
	if !ok {
		return selectorFunctionLocation{}, false
	}
	return selectorFunctionLocation{
		start:     start,
		argsStart: open + 1,
		argsEnd:   closeIdx,
		end:       closeIdx + 1,
	}, true
}

func findSelectorFunctionClose(selector string, open int) (int, bool) {
	depth := 0
	bracketDepth := 0
	quote := byte(0)
	escaped := false
	for i := open; i < len(selector); i++ {
		ch := selector[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '(':
			if bracketDepth == 0 {
				depth++
			}
		case ')':
			if bracketDepth == 0 {
				depth--
				if depth == 0 {
					return i, true
				}
			}
		}
	}
	return -1, false
}

func splitSelectorFunctionArgs(args string) []string {
	var parts []string
	var b strings.Builder
	depth := 0
	bracketDepth := 0
	quote := byte(0)
	escaped := false
	flush := func() {
		part := strings.TrimSpace(b.String())
		if part != "" {
			parts = append(parts, part)
		}
		b.Reset()
	}
	for i := range len(args) {
		ch := args[i]
		if quote != 0 {
			b.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
			b.WriteByte(ch)
		case '[':
			bracketDepth++
			b.WriteByte(ch)
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			b.WriteByte(ch)
		case '(':
			if bracketDepth == 0 {
				depth++
			}
			b.WriteByte(ch)
		case ')':
			if bracketDepth == 0 && depth > 0 {
				depth--
			}
			b.WriteByte(ch)
		case ',':
			if depth == 0 && bracketDepth == 0 {
				flush()
				continue
			}
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	flush()
	return parts
}
