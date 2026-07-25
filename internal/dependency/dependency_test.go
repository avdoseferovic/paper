package dependency_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const paperModule = "github.com/avdoseferovic/paper"

type listedPackage struct {
	ImportPath   string
	Standard     bool
	Module       *listedModule
	Imports      []string
	TestImports  []string
	XTestImports []string
}

type listedModule struct {
	Path string
}

type moduleSpec struct {
	name    string
	dir     string
	pattern string
}

func TestDependencyGraphsUseOnlyApprovedModules(t *testing.T) {
	root := repositoryRoot(t)
	for _, spec := range []moduleSpec{
		{name: "root", dir: root, pattern: "./..."},
		{name: "docs", dir: filepath.Join(root, "docs"), pattern: "./assets/examples/..."},
		{name: "examples", dir: filepath.Join(root, "examples"), pattern: "./..."},
	} {
		t.Run(spec.name, func(t *testing.T) {
			packages := listPackages(t, spec.dir, "-deps", spec.pattern)
			for _, pkg := range packages {
				if allowedPackageModule(pkg.Standard, modulePath(pkg.Module)) {
					continue
				}
				t.Errorf("%s resolves through disallowed module %q", pkg.ImportPath, modulePath(pkg.Module))
			}
		})
	}
}

func TestPaperSourceImportsUseOnlyApprovedExternalPackages(t *testing.T) {
	root := repositoryRoot(t)
	for _, spec := range []moduleSpec{
		{name: "root", dir: root, pattern: "./..."},
		{name: "docs", dir: filepath.Join(root, "docs"), pattern: "./assets/examples/..."},
		{name: "examples", dir: filepath.Join(root, "examples"), pattern: "./..."},
	} {
		t.Run(spec.name, func(t *testing.T) {
			packages := listPackages(t, spec.dir, spec.pattern)
			for _, pkg := range packages {
				if !isPaperSourcePackage(pkg) {
					continue
				}
				for _, importPath := range packageImports(pkg) {
					if !allowedSourceImport(importPath) {
						t.Errorf("%s directly imports disallowed package %q", pkg.ImportPath, importPath)
					}
				}
			}
		})
	}
}

func TestAllowedPackageModule(t *testing.T) {
	tests := []struct {
		name     string
		standard bool
		module   string
		want     bool
	}{
		{name: "standard library", standard: true, want: true},
		{name: "root module", module: paperModule, want: true},
		{name: "docs module", module: paperModule + "/docs", want: true},
		{name: "examples module", module: paperModule + "/examples", want: true},
		{name: "retained net", module: "golang.org/x/net", want: true},
		{name: "retained image", module: "golang.org/x/image", want: true},
		{name: "image text closure", module: "golang.org/x/text", want: true},
		{name: "unresolved nonstandard package", want: false},
		{name: "lookalike module", module: "golang.org/x/net/http2", want: false},
		{name: "removed selector library", module: "github.com/andybalholm/cascadia", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := allowedPackageModule(test.standard, test.module); got != test.want {
				t.Fatalf("allowedPackageModule(%t, %q) = %t, want %t", test.standard, test.module, got, test.want)
			}
		})
	}
}

func TestAllowedSourceImport(t *testing.T) {
	tests := []struct {
		importPath string
		want       bool
	}{
		{importPath: "bytes", want: true},
		{importPath: paperModule + "/pkg/html", want: true},
		{importPath: "golang.org/x/net/html", want: true},
		{importPath: "golang.org/x/image/vector", want: true},
		{importPath: "golang.org/x/net/http2", want: false},
		{importPath: "golang.org/x/text/encoding/charmap", want: false},
		{importPath: "github.com/srwiley/oksvg", want: false},
	}
	for _, test := range tests {
		t.Run(test.importPath, func(t *testing.T) {
			if got := allowedSourceImport(test.importPath); got != test.want {
				t.Fatalf("allowedSourceImport(%q) = %t, want %t", test.importPath, got, test.want)
			}
		})
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locating dependency guard source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func listPackages(t *testing.T, dir string, args ...string) []listedPackage {
	t.Helper()
	command := exec.CommandContext(t.Context(), "go", append([]string{"list", "-json"}, args...)...)
	command.Dir = dir
	command.Env = os.Environ()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list in %s: %v\n%s", dir, err, output)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var packages []listedPackage
	for decoder.More() {
		var pkg listedPackage
		if err := decoder.Decode(&pkg); err != nil {
			t.Fatalf("decode go list output from %s: %v", dir, err)
		}
		packages = append(packages, pkg)
	}
	if len(packages) == 0 {
		t.Fatalf("go list in %s returned no packages", dir)
	}
	return packages
}

func isPaperSourcePackage(pkg listedPackage) bool {
	return strings.HasPrefix(pkg.ImportPath, paperModule) && strings.HasPrefix(modulePath(pkg.Module), paperModule)
}

func packageImports(pkg listedPackage) []string {
	imports := make(map[string]struct{}, len(pkg.Imports)+len(pkg.TestImports)+len(pkg.XTestImports))
	for _, group := range [][]string{pkg.Imports, pkg.TestImports, pkg.XTestImports} {
		for _, importPath := range group {
			imports[importPath] = struct{}{}
		}
	}
	result := make([]string, 0, len(imports))
	for importPath := range imports {
		result = append(result, importPath)
	}
	sort.Strings(result)
	return result
}

func modulePath(module *listedModule) string {
	if module == nil {
		return ""
	}
	return module.Path
}

func allowedPackageModule(standard bool, path string) bool {
	if standard {
		return true
	}
	switch path {
	case paperModule, paperModule + "/docs", paperModule + "/examples", "golang.org/x/net", "golang.org/x/image", "golang.org/x/text":
		return true
	default:
		return false
	}
}

func allowedSourceImport(importPath string) bool {
	return !strings.Contains(importPath, ".") ||
		strings.HasPrefix(importPath, paperModule) ||
		importPath == "golang.org/x/net/html" ||
		strings.HasPrefix(importPath, "golang.org/x/image/") ||
		importPath == "golang.org/x/image"
}
