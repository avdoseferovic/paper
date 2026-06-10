package main

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/test"
)

func TestGetPaper(t *testing.T) {
	t.Parallel()
	// Act
	path := "docs/assets/images/certificate.png"
	sut := GetPaper(buildPath(path))

	// Assert
	test.New(t).Assert(sut.GetStructure()).Equals("examples/background.json")
}

func buildPath(file string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir = strings.ReplaceAll(dir, "docs/assets/examples/background", "")
	return path.Join(dir, file)
}
