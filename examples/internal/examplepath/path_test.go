package examplepath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepo(t *testing.T) {
	t.Parallel()

	path := Repo("go.mod")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read root go.mod: %v", err)
	}
	if want := "module " + rootModule; !containsLine(data, want) {
		t.Fatalf("expected %s to contain %q", path, want)
	}
}

func TestModule(t *testing.T) {
	t.Parallel()

	path := Module("go.mod")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read examples go.mod: %v", err)
	}
	if want := "module " + examplesModule; !containsLine(data, want) {
		t.Fatalf("expected %s to contain %q", path, want)
	}
}

func TestEnsureParent(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "file.txt")

	if err := EnsureParent(path); err != nil {
		t.Fatalf("ensure parent: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("stat parent: %v", err)
	}
}

func containsLine(data []byte, want string) bool {
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}

	return false
}
