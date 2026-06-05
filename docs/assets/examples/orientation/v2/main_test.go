package main

import (
	"testing"

	"github.com/johnfercher/paper/v2/pkg/test"
)

func TestGetPaper(t *testing.T) {
	t.Parallel()
	// Act
	sut := GetPaper()

	// Assert
	test.New(t).Assert(sut.GetStructure()).Equals("examples/orientation.json")
}
