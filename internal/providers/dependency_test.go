package providers_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProviderSourcesDoNotImportExternalGofpdf(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	externalBase := "github.com/" + "phpdave11/gofpdf"
	forbidden := []string{
		`"` + externalBase + `"`,
		`"` + externalBase + `/`,
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".worktrees" || entry.Name() == "docs" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(data)
		for _, value := range forbidden {
			if strings.Contains(source, value) {
				t.Errorf("%s imports external GoFPDF path %s", path, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve caller file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
