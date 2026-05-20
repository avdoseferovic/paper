package css

import (
	"strings"
)

// maxVarDepth is a safety belt on absolute call depth. Cycle detection uses a
// visited set (correct), and this cap only fires if the visited set fails to
// catch a pathological substitution chain.
const maxVarDepth = 32

// ResolveVars resolves CSS var(--name [, fallback]) references in value
// against the given scope map. Returns the string with all var() references
// resolved; nested references are followed using a visited set to detect
// cycles. On a cycle, that var() resolves to its fallback (or empty).
//
// scope keys must include the leading "--" prefix (e.g. "--accent").
// scope may be nil; in that case, var() with no fallback resolves to "" and
// var(name, fb) resolves to fb.
func ResolveVars(value string, scope map[string]string) string {
	return resolveVarsWithVisited(value, scope, map[string]bool{}, 0)
}

func resolveVarsWithVisited(value string, scope map[string]string, visited map[string]bool, depth int) string {
	if depth >= maxVarDepth {
		return ""
	}
	if !strings.Contains(value, "var(") {
		return value
	}
	var b strings.Builder
	i := 0
	for i < len(value) {
		idx := strings.Index(value[i:], "var(")
		if idx < 0 {
			b.WriteString(value[i:])
			break
		}
		b.WriteString(value[i : i+idx])
		start := i + idx + 4 // past "var("
		// Find matching close paren, accounting for nested parens.
		depthP := 1
		j := start
		for j < len(value) && depthP > 0 {
			switch value[j] {
			case '(':
				depthP++
			case ')':
				depthP--
			}
			if depthP > 0 {
				j++
			}
		}
		if depthP != 0 {
			// Unmatched paren — bail.
			b.WriteString(value[i:])
			break
		}
		inside := value[start:j]
		i = j + 1 // past ')'

		name, fallback := splitVarArgs(inside)
		name = strings.TrimSpace(name)
		if !strings.HasPrefix(name, "--") {
			// Malformed — fall back to fallback or empty.
			b.WriteString(strings.TrimSpace(fallback))
			continue
		}
		if visited[name] {
			// Cycle detected — resolve to fallback (or empty).
			b.WriteString(strings.TrimSpace(fallback))
			continue
		}
		resolved, ok := scope[name]
		if !ok {
			b.WriteString(strings.TrimSpace(fallback))
			continue
		}
		// Recurse to resolve nested var() in the value.
		visited[name] = true
		nested := resolveVarsWithVisited(resolved, scope, visited, depth+1)
		delete(visited, name)
		b.WriteString(nested)
	}
	return b.String()
}

// splitVarArgs splits the contents of a var(...) expression on the FIRST
// comma so fallbacks containing commas are preserved.
func splitVarArgs(s string) (name, fallback string) {
	idx := strings.Index(s, ",")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}
