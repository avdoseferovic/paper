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
	return "github.com/johnfercher/" + legacyName() + "/v2"
}

func previousOwnerPaperModulePath() string {
	return "github.com/" + previousOwner() + "/paper"
}

func forbiddenTextPatterns() []forbiddenPattern {
	legacy := legacyName()
	return []forbiddenPattern{
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
	case ".git", ".worktrees", "docs/plans", "docs/v1", "docs/assets/pdf", "internal/paperpdf":
		return true
	default:
		return false
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
