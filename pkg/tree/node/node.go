// Package node provides a small generic tree node used to describe Paper document structure.
package node

import "fmt"

// Node is a generic tree node.
type Node[T any] struct {
	id       int
	data     T
	previous *Node[T]
	nexts    []*Node[T]
}

// New creates a new node with data.
func New[T any](data T) *Node[T] {
	return &Node[T]{
		data: data,
	}
}

// WithID sets the node ID.
func (n *Node[T]) WithID(id int) *Node[T] {
	n.id = id
	return n
}

// GetData returns the node data.
func (n *Node[T]) GetData() T {
	return n.data
}

// GetID returns the node ID.
func (n *Node[T]) GetID() int {
	return n.id
}

// GetPrevious returns the parent node.
func (n *Node[T]) GetPrevious() *Node[T] {
	return n.previous
}

// GetNexts returns the child nodes.
func (n *Node[T]) GetNexts() []*Node[T] {
	return n.nexts
}

// IsRoot reports whether this node has no parent.
func (n *Node[T]) IsRoot() bool {
	return n.previous == nil
}

// IsLeaf reports whether this node has no children.
func (n *Node[T]) IsLeaf() bool {
	return len(n.nexts) == 0
}

// Backtrack returns a path from this node to the root.
func (n *Node[T]) Backtrack() []*Node[T] {
	var nodes []*Node[T]

	current := n
	for current != nil {
		nodes = append(nodes, current)
		current = current.previous
	}

	return nodes
}

// GetStructure returns parent-child ID pairs for this node and its descendants.
func (n *Node[T]) GetStructure() []string {
	structure := make([]string, 0, 1+len(n.nexts))
	var current string

	if n.previous == nil {
		current = fmt.Sprintf("(NULL) -> (%d)", n.id)
	} else {
		current = fmt.Sprintf("(%d) -> (%d)", n.previous.id, n.id)
	}

	if n.nexts != nil {
		current += ", "
	}

	structure = append(structure, current)

	for _, next := range n.nexts {
		structure = append(structure, next.GetStructure()...)
	}

	return structure
}

// AddNext adds a child node.
func (n *Node[T]) AddNext(child *Node[T]) {
	child.previous = n
	n.nexts = append(n.nexts, child)
}

// Filter returns a copy of this subtree containing only nodes accepted by filterFunc.
func (n *Node[T]) Filter(filterFunc func(obj T) bool) (*Node[T], bool) {
	if !filterFunc(n.GetData()) {
		return nil, false
	}

	newNode := New(n.GetData()).WithID(n.GetID())

	for _, next := range n.nexts {
		innerNode, ok := next.Filter(filterFunc)
		if ok {
			newNode.AddNext(innerNode)
		}
	}

	return newNode, true
}
