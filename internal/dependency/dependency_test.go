package dependency_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGoModulesExcludeRemovedDependencies(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	output := runGo(t, root, "list", "-m", "all")

	assertMissing(t, output, forbiddenModulePaths()...)
}

func TestGoDependenciesExcludeRemovedDependenciesAndLegacyModule(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	output := runGo(t, root, "list", "-deps", "./...")

	assertMissing(t, output, forbiddenModulePaths()...)
	assertMissing(t, output, legacyModulePath())
	assertMissing(t, output, previousOwnerPaperModulePath())
}

func TestGoTestDependenciesExcludeRemovedDependencies(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	output := runGo(t, root, "list", "-deps", "-test", "./...")

	assertMissing(t, output, forbiddenModulePaths()...)
}

func TestActiveGoSourceExcludesRemovedSourceImports(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	forbidden := forbiddenGoSourceImportPaths()

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)

		if entry.IsDir() {
			if shouldSkipDir(relativePath) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(relativePath) != ".go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, value := range forbidden {
			if strings.Contains(string(data), value) {
				t.Errorf("%s contains removed source import %q", relativePath, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestActiveTextExcludesRemovedDependenciesAndLegacyName(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	forbidden := forbiddenTextPatterns()

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)

		if entry.IsDir() {
			if shouldSkipDir(relativePath) {
				return filepath.SkipDir
			}
			return nil
		}
		if !shouldScanFile(relativePath) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, pattern := range forbidden {
			if pattern.regex.Match(data) {
				t.Errorf("%s contains forbidden %s", relativePath, pattern.name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestVersionedProjectArtifactsStayRemoved(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	for _, path := range removedVersionedPaths() {
		assertPathMissing(t, filepath.Join(root, path))
	}
	for _, pattern := range removedVersionedGlobs(root) {
		assertNoGlobMatches(t, pattern)
	}
}

func TestRemovedDeadCodeArtifactsStayRemoved(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	for _, path := range removedDeadCodePaths() {
		assertPathMissing(t, filepath.Join(root, path))
	}

	forbidden := removedDeadCodeSourceLiterals()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)

		if entry.IsDir() {
			if shouldSkipDir(relativePath) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(relativePath) != ".go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, value := range forbidden {
			if strings.Contains(content, value) {
				t.Errorf("%s contains removed dead-code symbol %q", relativePath, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMaintenanceReorganizationArtifactsExist(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	for _, file := range maintenanceReorganizationFiles() {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Fatalf("expected maintenance reorganization artifact %s: %v", file, err)
		}
	}
}

type forbiddenPattern struct {
	name  string
	regex *regexp.Regexp
}

func forbiddenModulePaths() []string {
	return []string{
		"github.com/" + "pdfcpu/pdfcpu",
		"github.com/" + "phpdave11/gofpdf",
		"github.com/" + "johnfercher/go-tree",
		"github.com/" + "f-amaral/go-" + "async",
	}
}

func forbiddenGoSourceImportPaths() []string {
	return []string{
		"gopkg.in/" + "yaml.v3",
	}
}

func legacyModulePath() string {
	return "github.com/johnfercher/" + legacyName()
}

func previousOwnerPaperModulePath() string {
	return "github.com/" + previousOwner() + "/paper"
}

func forbiddenTextPatterns() []forbiddenPattern {
	legacy := legacyName()
	return []forbiddenPattern{
		{name: "versioned Paper module path", regex: regexp.MustCompile(regexp.QuoteMeta("github.com/avdoseferovic/paper/" + "v2"))},
		{name: "versioned docs path", regex: regexp.MustCompile(regexp.QuoteMeta("docs/" + "v1"))},
		{name: "versioned docs path", regex: regexp.MustCompile(regexp.QuoteMeta("docs/" + "v2"))},
		{name: "versioned example path", regex: regexp.MustCompile(regexp.QuoteMeta("docs/assets/examples/") + `[^[:space:]"']+/` + regexp.QuoteMeta("v2"))},
		{name: "removed merge dependency", regex: regexp.MustCompile(regexp.QuoteMeta("github.com/" + "pdfcpu/pdfcpu"))},
		{name: "removed PDF backend dependency", regex: regexp.MustCompile(regexp.QuoteMeta("github.com/" + "phpdave11/gofpdf"))},
		{name: "removed tree dependency", regex: regexp.MustCompile(regexp.QuoteMeta("github.com/" + "johnfercher/go-tree"))},
		{name: "removed async dependency", regex: regexp.MustCompile(regexp.QuoteMeta("github.com/" + "f-amaral/go-" + "async"))},
		{name: "removed Paper test config file", regex: regexp.MustCompile(regexp.QuoteMeta(".paper" + ".yml"))},
		{name: "removed Paper test config key", regex: regexp.MustCompile(regexp.QuoteMeta("test_" + "path"))},
		{name: "previous Paper owner path", regex: regexp.MustCompile(regexp.QuoteMeta(previousOwnerPaperModulePath()))},
		{name: "legacy module path", regex: regexp.MustCompile(regexp.QuoteMeta("github.com/johnfercher/" + legacy))},
		{name: "legacy display name", regex: regexp.MustCompile(`(?i)\b` + legacy + `\b`)},
		{name: "legacy config file", regex: regexp.MustCompile(regexp.QuoteMeta("." + legacy + ".yml"))},
		{name: "legacy fixture path", regex: regexp.MustCompile(regexp.QuoteMeta("test/" + legacy))},
		{name: "legacy fixture prefix", regex: regexp.MustCompile(regexp.QuoteMeta(legacy + "_"))},
		{name: "stale provider path", regex: regexp.MustCompile(regexp.QuoteMeta("internal/providers/" + "gofpdf"))},
		{name: "stale provider phrase", regex: regexp.MustCompile(`(?i)gofpdf ` + `provider`)},
	}
}

func removedVersionedPaths() []string {
	return []string{
		"docs/" + "v1",
		"docs/" + "v2",
	}
}

func removedVersionedGlobs(root string) []string {
	return []string{
		filepath.Join(root, "docs", "assets", "examples", "*", "v2"),
		filepath.Join(root, "docs", "assets", "pdf", "*"+"v2"+".pdf"),
		filepath.Join(root, "docs", "assets", "pdf", "v2"+".pdf"),
		filepath.Join(root, "docs", "assets", "text", "*"+"v2"+".txt"),
		filepath.Join(root, "docs", "assets", "text", "v2"+".txt"),
	}
}

func removedDeadCodePaths() []string {
	return []string{
		filepath.Join("pkg", "pkg.go"),
		filepath.Join("pkg", "components", "components.go"),
		filepath.Join("pkg", "consts", "consts.go"),
		filepath.Join("pkg", "components", "checkbox", "example_test.go"),
		filepath.Join("pkg", "components", "code", "example_test.go"),
		filepath.Join("pkg", "components", "col", "example_test.go"),
		filepath.Join("pkg", "components", "image", "example_test.go"),
		filepath.Join("pkg", "components", "line", "example_test.go"),
		filepath.Join("pkg", "components", "list", "example_test.go"),
		filepath.Join("pkg", "components", "row", "example_test.go"),
		filepath.Join("pkg", "components", "signature", "example_test.go"),
		filepath.Join("pkg", "components", "text", "example_test.go"),
		filepath.Join("pkg", "config", "example_test.go"),
	}
}

func removedDeadCodeSourceLiterals() []string {
	return []string{
		"wrapRow" + "AnchorSource",
		"newAnchor" + "Source",
		"row" + "Component",
		"func (tr *translator) " + "flexRow(",
		"func " + "hrRow(",
		"func (m *PaperTest) " + "Save(",
		"func " + "Green(",
	}
}

func maintenanceReorganizationFiles() []string {
	return []string{
		"pkg/core/provider_services.go",
		"paper_page_builder.go",
		"internal/providers/paper/richtext_layout.go",
		"internal/providers/paper/richtext_render.go",
		"internal/providers/paper/provider_issues.go",
		"pkg/html/css/computed_font.go",
		"pkg/html/css/computed_box.go",
		"pkg/html/css/computed_border.go",
		"pkg/html/css/computed_flex.go",
		"pkg/html/css/computed_effects.go",
		"pkg/html/css/computed_typography.go",
		"internal/paperpdf/OWNERSHIP.md",
	}
}

func legacyName() string {
	return "mar" + "oto"
}

func previousOwner() string {
	return "john" + "fercher"
}

func assertMissing(t *testing.T, output string, values ...string) {
	t.Helper()

	for _, value := range values {
		if strings.Contains(output, value) {
			t.Fatalf("unexpected dependency reference %q in command output", value)
		}
	}
}

func runGo(t *testing.T, root string, args ...string) string {
	t.Helper()

	cmd := exec.Command("go", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}
}

func shouldSkipDir(path string) bool {
	if strings.HasPrefix(path, ".") && path != ".github" {
		return true
	}

	switch path {
	case ".git", ".worktrees", "docs/plans", "docs/assets/pdf", "internal/paperpdf":
		return true
	default:
		return false
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("expected %s to be removed", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("could not inspect %s: %v", path, err)
	}
}

func assertNoGlobMatches(t *testing.T, pattern string) {
	t.Helper()

	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("invalid glob %s: %v", pattern, err)
	}
	if len(matches) > 0 {
		t.Fatalf("expected no matches for %s, found %v", pattern, matches)
	}
}

func shouldScanFile(path string) bool {
	switch filepath.Base(path) {
	case "go.mod", "go.sum", "README.md", "CODE_OF_CONDUCT.md", "pull_request_template.md", "CNAME", ".mockery.yaml", ".golangci.yml":
		return true
	}

	switch filepath.Ext(path) {
	case ".go", ".md", ".html", ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
