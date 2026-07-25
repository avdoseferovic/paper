package exampleregistry_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/examples/internal/examplepath"
	"github.com/avdoseferovic/paper/examples/internal/exampleregistry"
)

// wantNames is every documented example whose PDF preview is generated in the
// browser. The four whose inputs are too awkward to ship (background and
// disablepagebreak need a 792KB PNG, customfont a 23MB TTF, mergepdf an existing
// PDF) stay static fixtures and are deliberately absent, as is unittests, which
// has no builder.
var wantNames = []string{
	"addpage", "autorow", "barcodegrid", "billing", "bookmark", "cellstyle",
	"checkbox", "compression", "customdimensions", "custompage", "datamatrixgrid",
	"footer", "header", "imagegrid", "line", "list", "lowmemory", "margins",
	"maxgridsum", "metadatas", "orientation", "pagenumber", "parallelism",
	"protection", "qrgrid", "signaturegrid", "simplest", "textgrid", "watermark",
}

func TestNamesAreExactAndSorted(t *testing.T) {
	t.Parallel()

	got := exampleregistry.Names()

	if !slices.Equal(got, wantNames) {
		t.Errorf("Names() mismatch:\n got: %v\nwant: %v", got, wantNames)
	}
	if !slices.IsSorted(got) {
		t.Error("Names() must be sorted for deterministic output")
	}
}

func TestStaticFixturesAreNotRegistered(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"background", "customfont", "disablepagebreak", "mergepdf", "unittests"} {
		if _, err := exampleregistry.Lookup(name); err == nil {
			t.Errorf("%q must not be registered: it stays a static fixture", name)
		}
	}
}

func TestLookupUnknownReturnsError(t *testing.T) {
	t.Parallel()

	if _, err := exampleregistry.Lookup("nope"); err == nil {
		t.Error("expected an error for an unknown example, got nil")
	}
}

func TestEveryEntryBuildsAndDeclaresRealAssets(t *testing.T) {
	t.Parallel()

	for _, name := range exampleregistry.Names() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			entry, err := exampleregistry.Lookup(name)
			if err != nil {
				t.Fatalf("Lookup(%q): %v", name, err)
			}

			// Asset paths are repo-root relative because that is the string the
			// library's os.ReadFile receives, both here and behind the browser's
			// in-memory filesystem. Tests run with cwd set to this package, so
			// they need examplepath.Repo to find the real file.
			for _, asset := range entry.Assets {
				if _, err := os.Stat(examplepath.Repo(asset)); err != nil {
					t.Errorf("declared asset %q does not exist: %v", asset, err)
				}
			}

			if doc := entry.Build(); doc == nil {
				t.Error("Build() returned nil")
			}
		})
	}
}

// TestDeclaredAssetsCoverSource parses each registered example's paper.go and
// checks that every image path hardcoded there is also declared in the registry.
// Without this, adding an image to an example silently breaks its browser
// preview: nothing fetches the file, the read fails, and the library quietly
// draws an error box instead of returning an error.
//
// The check is containment rather than equality because a builder can also
// receive a path as an argument that the registry supplies — compression takes
// its frontpage.png that way, so that path is declared but appears in no literal
// inside paper.go.
func TestDeclaredAssetsCoverSource(t *testing.T) {
	t.Parallel()

	for _, name := range exampleregistry.Names() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			entry, err := exampleregistry.Lookup(name)
			if err != nil {
				t.Fatalf("Lookup(%q): %v", name, err)
			}

			src := examplepath.Repo(filepath.Join("docs", "assets", "examples", name, "paper.go"))
			for _, found := range imagePathsIn(t, src) {
				if !slices.Contains(entry.Assets, found) {
					t.Errorf("%s reads %q but the registry does not declare it (declared: %v)",
						src, found, entry.Assets)
				}
			}
		})
	}
}

// imagePathsIn returns the distinct string literals passed as the path argument
// to the image.NewFromFile* constructors in the given file.
func imagePathsIn(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var out []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !strings.HasPrefix(sel.Sel.Name, "NewFromFile") && !strings.HasPrefix(sel.Sel.Name, "NewAutoFromFile") {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "image" {
			return true
		}
		for _, arg := range call.Args {
			lit, ok := arg.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil || !strings.HasPrefix(value, "docs/assets/") {
				continue
			}
			if !slices.Contains(out, value) {
				out = append(out, value)
			}
		}
		return true
	})
	return out
}
