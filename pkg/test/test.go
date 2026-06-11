// Package test exposes Paper's golden-structure test helper for library
// consumers. It is a thin re-export of the canonical implementation in
// internal/test, kept dependency-free so importing it never adds third-party
// modules to a consumer's build.
package test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/test"
)

// Errors returned while resolving the golden-file directory.
var (
	ErrCannotReadDir = test.ErrCannotReadDir
	ErrGoModNotFound = test.ErrGoModNotFound
)

type (
	// Node is the JSON shape used to compare document structures.
	Node = test.Node
	// PaperTest is the unit test instance.
	PaperTest = test.PaperTest
	// Config is the representation of a test config.
	Config = test.Config
)

// New creates the PaperTest instance for unit tests.
func New(t *testing.T) *PaperTest {
	t.Helper()
	return test.New(t)
}
