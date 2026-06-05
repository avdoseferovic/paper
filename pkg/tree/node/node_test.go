package node_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/avdoseferovic/paper/pkg/tree/node"
)

func TestNew(t *testing.T) {
	t.Parallel()

	// Act
	sut := node.New("root")

	// Assert
	assert.Equal(t, "root", sut.GetData())
	assert.True(t, sut.IsRoot())
	assert.True(t, sut.IsLeaf())
}

func TestNode_AddNext(t *testing.T) {
	t.Parallel()

	t.Run("when child is added, should link parent and child", func(t *testing.T) {
		t.Parallel()

		// Arrange
		sut := node.New("root").WithID(1)
		child := node.New("child").WithID(2)

		// Act
		sut.AddNext(child)

		// Assert
		assert.False(t, sut.IsLeaf())
		assert.Equal(t, []*node.Node[string]{child}, sut.GetNexts())
		assert.Same(t, sut, child.GetPrevious())
		assert.False(t, child.IsRoot())
	})
}

func TestNode_Backtrack(t *testing.T) {
	t.Parallel()

	t.Run("when node has ancestors, should return path to root", func(t *testing.T) {
		t.Parallel()

		// Arrange
		root := node.New("root")
		child := node.New("child")
		leaf := node.New("leaf")
		root.AddNext(child)
		child.AddNext(leaf)

		// Act
		result := leaf.Backtrack()

		// Assert
		assert.Equal(t, []*node.Node[string]{leaf, child, root}, result)
	})
}

func TestNode_GetStructure(t *testing.T) {
	t.Parallel()

	t.Run("when node has descendants, should return parent child id pairs", func(t *testing.T) {
		t.Parallel()

		// Arrange
		sut := node.New("root").WithID(1)
		child := node.New("child").WithID(2)
		leaf := node.New("leaf").WithID(3)
		sut.AddNext(child)
		child.AddNext(leaf)

		// Act
		result := sut.GetStructure()

		// Assert
		assert.Equal(t, []string{"(NULL) -> (1), ", "(1) -> (2), ", "(2) -> (3)"}, result)
	})
}

func TestNode_Filter(t *testing.T) {
	t.Parallel()

	t.Run("when root is rejected, should return false", func(t *testing.T) {
		t.Parallel()

		// Arrange
		sut := node.New("root")

		// Act
		result, ok := sut.Filter(func(value string) bool {
			return value != "root"
		})

		// Assert
		assert.False(t, ok)
		assert.Nil(t, result)
	})

	t.Run("when descendants are filtered, should return copied matching subtree", func(t *testing.T) {
		t.Parallel()

		// Arrange
		sut := node.New("root").WithID(1)
		kept := node.New("kept").WithID(2)
		removed := node.New("removed").WithID(3)
		sut.AddNext(kept)
		sut.AddNext(removed)

		// Act
		result, ok := sut.Filter(func(value string) bool {
			return value != "removed"
		})

		// Assert
		assert.True(t, ok)
		assert.NotSame(t, sut, result)
		assert.Equal(t, "root", result.GetData())
		assert.Len(t, result.GetNexts(), 1)
		assert.Equal(t, "kept", result.GetNexts()[0].GetData())
		assert.Same(t, result, result.GetNexts()[0].GetPrevious())
	})
}
