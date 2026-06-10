package test_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/test"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// The package is a thin re-export of internal/test; this smoke test exercises
// only the public surface against a root test/paper golden file.
func TestNew_WhenAssertingMatchingStructure_ShouldPassAgainstRootGolden(t *testing.T) {
	t.Parallel()

	rootNode := node.New[core.Structure](core.Structure{Type: "paper"})
	rootNode.AddNext(node.New[core.Structure](core.Structure{Type: "page"}))

	test.New(t).Assert(rootNode).Equals("paper_test.json")
}

func TestNew_WhenAssertingMismatchedStructure_ShouldFail(t *testing.T) {
	t.Parallel()

	rootNode := node.New[core.Structure](core.Structure{Type: "not_paper"})
	innerT := &testing.T{}

	test.New(innerT).Assert(rootNode).Equals("paper_test.json")

	if !innerT.Failed() {
		t.Fatal("expected mismatched structure to fail the assertion")
	}
}
