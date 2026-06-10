package css

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestResolveVar_Simple(t *testing.T) {
	t.Parallel()
	scope := map[string]string{"--accent": "#ff0000"}
	got := ResolveVars("var(--accent)", scope)
	assert.Equal(t, "#ff0000", got)
}

func TestResolveVar_WithFallback(t *testing.T) {
	t.Parallel()
	scope := map[string]string{}
	got := ResolveVars("var(--missing, green)", scope)
	assert.Equal(t, "green", got)
}

func TestResolveVar_NoFallback_Empty(t *testing.T) {
	t.Parallel()
	scope := map[string]string{}
	got := ResolveVars("var(--missing)", scope)
	assert.Equal(t, "", got)
}

func TestResolveVar_NestedReference(t *testing.T) {
	t.Parallel()
	scope := map[string]string{
		"--a": "var(--b)",
		"--b": "red",
	}
	got := ResolveVars("var(--a)", scope)
	assert.Equal(t, "red", got)
}

func TestResolveVar_DirectCycle(t *testing.T) {
	t.Parallel()
	scope := map[string]string{
		"--a": "var(--b)",
		"--b": "var(--a)",
	}
	// Cycle → fallback (empty here).
	got := ResolveVars("var(--a)", scope)
	assert.Equal(t, "", got)
}

func TestResolveVar_DeepLegitimate(t *testing.T) {
	t.Parallel()
	// 6 levels of legitimate chaining (no cycle).
	scope := map[string]string{
		"--xxl":  "16pt",
		"--xl":   "var(--xxl)",
		"--lg":   "var(--xl)",
		"--md":   "var(--lg)",
		"--sm":   "var(--md)",
		"--base": "var(--sm)",
	}
	got := ResolveVars("var(--base)", scope)
	assert.Equal(t, "16pt", got)
}

func TestResolveVar_EmbeddedInCompoundValue(t *testing.T) {
	t.Parallel()
	scope := map[string]string{"--accent": "red"}
	got := ResolveVars("1pt solid var(--accent)", scope)
	assert.Equal(t, "1pt solid red", got)
}

func TestResolveVar_NoVars_PassThrough(t *testing.T) {
	t.Parallel()
	got := ResolveVars("10mm", nil)
	assert.Equal(t, "10mm", got)
}
