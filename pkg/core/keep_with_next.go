package core

// KeepWithNext asks the page builder to move a row and its successor to the
// next page when the two would otherwise be separated by a page boundary.
type KeepWithNext interface {
	KeepWithNext() bool
}
