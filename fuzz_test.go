package paper_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
)

const (
	maxHTMLFuzzSeedBytes = 16 << 10
	maxHTMLFuzzInputSize = 64 << 10
)

func FuzzFromHTML(f *testing.F) {
	seeds := []string{
		`<html><body><h1>Paper</h1><p>hello</p></body></html>`,
		`<html><head><style>p{color:red;padding:1mm}</style></head><body><p>styled</p></body></html>`,
		`<div style="display:flex;gap:2mm"><p>A</p><p>B</p></div>`,
		`<img src="data:image/png;base64,not-a-png" alt="fallback">`,
	}
	seeds = append(seeds, collectHTMLFuzzSeeds(
		"examples/cmd/html-demo/assets",
		"examples/cmd/survey-report/assets",
		"docs/assets/examples",
		"test/paper",
	)...)
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, htmlStr string) {
		if len(htmlStr) > maxHTMLFuzzInputSize {
			t.Skip("input too large for smoke fuzzing")
		}
		doc, err := paper.FromHTML(htmlStr)
		if err != nil {
			return
		}
		_ = doc.GetBytes()
	})
}

func collectHTMLFuzzSeeds(dirs ...string) []string {
	var files []string
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			collectHTMLFuzzSeedFiles(filepath.Join(dir, entry.Name()), entry, &files)
		}
	}
	slices.Sort(files)

	seeds := make([]string, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil || len(data) == 0 || len(data) > maxHTMLFuzzSeedBytes {
			continue
		}
		seeds = append(seeds, string(data))
		if len(seeds) >= 8 {
			return seeds
		}
	}
	return seeds
}

func collectHTMLFuzzSeedFiles(path string, entry os.DirEntry, files *[]string) {
	if entry.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		for _, child := range entries {
			collectHTMLFuzzSeedFiles(filepath.Join(path, child.Name()), child, files)
		}
		return
	}
	if isHTMLFuzzSeedFile(path) {
		*files = append(*files, path)
	}
}

func isHTMLFuzzSeedFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm", ".json", ".txt", ".svg":
		return true
	default:
		return false
	}
}
