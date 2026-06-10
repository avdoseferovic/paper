package examplepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	rootModule     = "github.com/avdoseferovic/paper"
	examplesModule = rootModule + "/examples"
)

// Repo returns a path rooted at the parent Paper repository when it can be found.
func Repo(relative string) string {
	if dir, ok := findModule(rootModule); ok {
		return filepath.Join(dir, relative)
	}
	return relative
}

// Module returns a path rooted at the examples module when it can be found.
func Module(relative string) string {
	if dir, ok := findModule(examplesModule); ok {
		return filepath.Join(dir, relative)
	}
	if dir, ok := findModule(rootModule); ok {
		return filepath.Join(dir, "examples", relative)
	}
	return relative
}

// EnsureParent creates the parent directory for path.
func EnsureParent(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("create parent directory %s: %w", dir, err)
	}
	return nil
}

func findModule(module string) (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		if found, ok := modulePath(filepath.Join(dir, "go.mod")); ok && found == module {
			return dir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func modulePath(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], true
		}
	}

	return "", false
}
