// Command mockfix rewrites generated mocks in internal/mocks so they import
// the vendored mock runtime (internal/mocktest) instead of testify. The root
// module must stay free of third-party test dependencies; mockery emits
// testify imports, so `make mocks` runs this rewrite right after generation.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	testifyModule  = "github.com/stretchr/testify/mock"
	mocktestModule = "github.com/avdoseferovic/paper/internal/mocktest"
)

func main() {
	dir := "internal/mocks"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	changed, err := rewriteDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mockfix: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("mockfix: rewrote %d file(s) in %s\n", changed, dir)
}

// rewriteDir replaces the testify mock import with the internal mocktest
// import in every .go file directly inside dir. It returns the number of
// files that were modified.
func rewriteDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("reading mocks directory: %w", err)
	}

	changed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return changed, fmt.Errorf("reading %s: %w", path, err)
		}
		rewritten := bytes.ReplaceAll(content, []byte(testifyModule), []byte(mocktestModule))
		if bytes.Equal(content, rewritten) {
			continue
		}
		if err := os.WriteFile(path, rewritten, 0o600); err != nil {
			return changed, fmt.Errorf("writing %s: %w", path, err)
		}
		changed++
	}
	return changed, nil
}
