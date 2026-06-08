package translate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// readFileInRoot reads name (a path relative to dir) while preventing any
// escape from dir. Confinement is enforced by the kernel via os.Root
// (Go 1.24+), which blocks ".." traversal, absolute paths, and — unlike a
// lexical filepath.Clean prefix check — symlinks that point outside dir.
//
// An empty dir is refused outright: collapsing it to "." would silently scope
// reads to the process working directory, defeating the resolver's
// refuse-by-default contract.
func readFileInRoot(dir, name string) ([]byte, error) {
	if dir == "" {
		return nil, errBaseDirEmpty
	}
	if filepath.IsAbs(name) {
		return nil, fmt.Errorf("%w: %q", errAbsolutePathRefused, name)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("html: opening base dir %q: %w", dir, err)
	}
	defer root.Close()

	f, err := root.Open(filepath.Clean(filepath.FromSlash(name)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("html: reading %q from base dir: %w", name, err)
		}
		// os.Root rejects paths that escape the root (".." or out-of-root
		// symlinks) with a non-ErrNotExist error.
		return nil, fmt.Errorf("%w: %q: %w", errPathEscapesBaseDir, name, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("html: reading %q from base dir: %w", name, err)
	}
	return data, nil
}
