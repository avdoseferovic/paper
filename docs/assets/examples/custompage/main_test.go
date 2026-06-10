package main

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/test"
)

func TestGetPaper(t *testing.T) {
	t.Parallel()
	// Act
	sut := GetPaper()

	// Assert
	test.New(t).Assert(sut.GetStructure()).Equals("examples/custompage.json")
}
