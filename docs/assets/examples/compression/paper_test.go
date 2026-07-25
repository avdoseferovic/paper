package compression

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
	path := "docs/assets/images/frontpage.png"
	sut, err := GetPaper(buildPath(path))
	// Assert
	if err != nil {
		t.Fatalf("GetPaper returned an error: %v", err)
	}
	test.New(t).Assert(sut.GetStructure()).Equals("examples/compression.json")
}

// TestGetPaperMissingImage pins the contract that a missing image is reported as
// an error. The builder used to call os.Exit(1) here, which no recover() can
// catch: under wasm that tears down the instance and permanently disables every
// export on the page, and in a test run it kills the test binary outright.
func TestGetPaperMissingImage(t *testing.T) {
	t.Parallel()
	// Act
	sut, err := GetPaper(buildPath("docs/assets/images/does-not-exist.png"))

	// Assert
	if err == nil {
		t.Fatal("expected an error for a missing image, got nil")
	}
	if sut != nil {
		t.Errorf("expected a nil Paper alongside the error, got %T", sut)
	}
}

func buildPath(file string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir = strings.ReplaceAll(dir, "docs/assets/examples/compression", "")
	return path.Join(dir, file)
}
