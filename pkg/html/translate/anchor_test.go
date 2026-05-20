package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
)

func TestAnchor_LocalAnchor_PopulatedOnRichRun(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><a href="#target">jump</a></p>`)
	require.NotEmpty(t, runs)
	var found bool
	for _, r := range runs {
		if r.localAnchor == "target" {
			found = true
		}
	}
	assert.True(t, found, "RichRun.LocalAnchor must be set for href=\"#…\"")
}

func TestAnchor_ExternalHrefStillWorks(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><a href="https://example.com">x</a></p>`)
	require.NotEmpty(t, runs)
	for _, r := range runs {
		if r.text == "x" {
			assert.Empty(t, r.localAnchor)
		}
	}
}

func TestAnchor_CollectAnchorIDs_Forward(t *testing.T) {
	t.Parallel()
	// Forward reference: link appears BEFORE target. Drive collectAnchorIDs
	// indirectly through Translate, then assert the body Walk picked up the id.
	html := `<a href="#later">jump</a><h2 id="later">Target</h2>`
	doc, err := dom.Parse(html)
	require.NoError(t, err)
	rows, err := Translate(doc)
	require.NoError(t, err)
	var foundTarget bool
	for _, r := range rows {
		walkRowStructure(r.GetStructure(), func(s core.Structure) {
			if s.Type == "anchor_target" {
				foundTarget = true
			}
		})
	}
	assert.True(t, foundTarget, "anchor_target for forward-referenced id should exist")
}

func TestAnchor_TranslatorStructure_BothEnds(t *testing.T) {
	t.Parallel()
	// End-to-end: <p><a href="#s1">jump</a></p>…<h2 id="s1">Title</h2>
	// produces a structure containing BOTH an anchor_source and an
	// anchor_target node referencing "s1".
	doc, err := dom.Parse(`<p><a href="#s1">jump</a></p><h2 id="s1">Title</h2>`)
	require.NoError(t, err)
	rows, err := Translate(doc)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 2)

	var sources, targets []string
	for _, r := range rows {
		walkRowStructure(r.GetStructure(), func(s core.Structure) {
			switch s.Type {
			case "anchor_source":
				if v, ok := s.Details["anchors"].([]string); ok && len(v) > 0 {
					sources = append(sources, v[0])
				}
			case "anchor_target":
				if v, ok := s.Details["name"].(string); ok {
					targets = append(targets, v)
				}
			}
		})
	}
	assert.Contains(t, sources, "s1", "expected anchor_source for \"s1\"")
	assert.Contains(t, targets, "s1", "expected anchor_target for \"s1\"")
}

func TestAnchor_ForwardReference_StructurePresent(t *testing.T) {
	t.Parallel()
	// Link appears BEFORE target in source order — both must still register.
	doc, err := dom.Parse(`<p><a href="#later">jump</a></p><h2 id="later">Target</h2>`)
	require.NoError(t, err)
	rows, err := Translate(doc)
	require.NoError(t, err)

	var foundSource, foundTarget bool
	for _, r := range rows {
		walkRowStructure(r.GetStructure(), func(s core.Structure) {
			if s.Type == "anchor_source" {
				foundSource = true
			}
			if s.Type == "anchor_target" {
				foundTarget = true
			}
		})
	}
	assert.True(t, foundSource, "anchor_source must exist (forward reference)")
	assert.True(t, foundTarget, "anchor_target must exist (forward reference)")
}

func walkRowStructure(n *node.Node[core.Structure], fn func(core.Structure)) {
	if n == nil {
		return
	}
	fn(n.GetData())
	for _, c := range n.GetNexts() {
		walkRowStructure(c, fn)
	}
}
