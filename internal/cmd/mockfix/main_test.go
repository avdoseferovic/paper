package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteDir_WhenMocksImportTestify_ShouldRewriteToMocktest(t *testing.T) {
	dir := t.TempDir()
	src := `package mocks

import mock "github.com/stretchr/testify/mock"

var _ = mock.Anything
`
	path := filepath.Join(dir, "Provider.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	changed, err := rewriteDir(dir)
	if err != nil {
		t.Fatalf("rewriteDir returned error: %v", err)
	}
	if changed != 1 {
		t.Fatalf("expected 1 changed file, got %d", changed)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten file: %v", err)
	}
	if strings.Contains(string(got), "stretchr/testify") {
		t.Fatalf("testify import still present:\n%s", got)
	}
	if !strings.Contains(string(got), "github.com/avdoseferovic/paper/internal/mocktest") {
		t.Fatalf("mocktest import missing:\n%s", got)
	}
}

func TestRewriteDir_WhenFileAlreadyRewritten_ShouldLeaveUntouched(t *testing.T) {
	dir := t.TempDir()
	src := `package mocks

import mock "github.com/avdoseferovic/paper/internal/mocktest"

var _ = mock.Anything
`
	path := filepath.Join(dir, "Cache.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	changed, err := rewriteDir(dir)
	if err != nil {
		t.Fatalf("rewriteDir returned error: %v", err)
	}
	if changed != 0 {
		t.Fatalf("expected 0 changed files, got %d", changed)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != src {
		t.Fatalf("file content changed unexpectedly:\n%s", got)
	}
}

func TestRewriteDir_WhenDirMissing_ShouldReturnError(t *testing.T) {
	_, err := rewriteDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
}
