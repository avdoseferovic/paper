package main

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

func TestBuildShowcaseDocument(t *testing.T) {
	doc, err := buildShowcaseDocument()
	if err != nil {
		t.Fatalf("build showcase document: %v", err)
	}
	if len(doc.GetBytes()) == 0 {
		t.Fatal("expected generated PDF bytes")
	}
}

func TestBuildShowcasePaper_IncludesDesignHandoffSections(t *testing.T) {
	m, err := buildShowcasePaper()
	if err != nil {
		t.Fatalf("build showcase paper: %v", err)
	}

	values := structureValues(m.GetStructure())
	for _, want := range []string{
		"Generate PDFs from HTML and Go components",
		"Two Authoring Paths",
		"Component Library",
		"Layout Model",
		"HTML Pipeline",
		"Testing and Metrics",
		"Use Cases",
		"Generated PDF Example",
		"Install Paper",
	} {
		if !values[want] {
			t.Fatalf("expected showcase structure to include %q", want)
		}
	}
}

func structureValues(root *node.Node[core.Structure]) map[string]bool {
	values := make(map[string]bool)
	var walk func(*node.Node[core.Structure])
	walk = func(n *node.Node[core.Structure]) {
		if value, ok := n.GetData().Value.(string); ok {
			values[value] = true
		}
		for _, child := range n.GetNexts() {
			walk(child)
		}
	}
	walk(root)
	return values
}
