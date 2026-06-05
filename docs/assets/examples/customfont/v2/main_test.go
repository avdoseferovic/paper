package main

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/pkg/test"
)

func TestGetPaper(t *testing.T) {
	t.Parallel()
	// Act
	sut := GetPaper(buildPath("docs/assets/fonts/arial-unicode-ms.ttf"))

	// Assert
	test.New(t).Assert(sut.GetStructure()).Equals("examples/customfont.json")
}

func buildPath(file string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir = strings.ReplaceAll(dir, "docs/assets/examples/customfont/v2", "")
	return path.Join(dir, file)
}
