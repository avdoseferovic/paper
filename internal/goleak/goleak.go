// Package goleak provides a small goroutine leak check for this repository's
// tests.
package goleak

import (
	"fmt"
	"maps"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	baselineMu sync.Mutex
	baseline   stackSet
)

type stackSet map[string]int

// VerifyTestMain runs m and fails the test binary if extra goroutines remain.
func VerifyTestMain(m *testing.M) {
	baselineMu.Lock()
	baseline = snapshot()
	baselineMu.Unlock()

	code := m.Run()
	if isFuzzRun(os.Args) {
		os.Exit(code)
	}
	if extras := waitForNoExtras(currentBaseline()); len(extras) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "goleak: found unexpected goroutines after tests:\n%s\n", strings.Join(extras, "\n\n"))
		code = 1
	}
	os.Exit(code)
}

// VerifyNone fails t if goroutines outside the suite baseline remain.
func VerifyNone(t *testing.T) {
	t.Helper()
	if extras := waitForNoExtras(currentBaseline()); len(extras) > 0 {
		t.Errorf("goleak: found unexpected goroutines:\n%s", strings.Join(extras, "\n\n"))
	}
}

func currentBaseline() stackSet {
	baselineMu.Lock()
	defer baselineMu.Unlock()
	cp := make(stackSet, len(baseline))
	maps.Copy(cp, baseline)
	return cp
}

func waitForNoExtras(base stackSet) []string {
	var extras []string
	for range 20 {
		extras = diff(snapshot(), base)
		if len(extras) == 0 {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return extras
}

func snapshot() stackSet {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	blocks := strings.Split(string(buf[:n]), "\n\n")

	stacks := make(stackSet, len(blocks))
	for _, block := range blocks {
		stack := normalize(block)
		if stack == "" || isIgnored(stack) {
			continue
		}
		stacks[stack]++
	}
	return stacks
}

func normalize(block string) string {
	lines := strings.Split(strings.TrimSpace(block), "\n")
	if len(lines) <= 1 {
		return ""
	}
	return strings.Join(lines[1:], "\n")
}

func isIgnored(stack string) bool {
	ignored := []string{
		"github.com/avdoseferovic/paper/internal/goleak.snapshot",
		"testing.(*M).Run(",
		"testing.(*T).Run(",
		"testing.(*T).Parallel(",
		"testing.runTests",
	}
	for _, pattern := range ignored {
		if strings.Contains(stack, pattern) {
			return true
		}
	}
	return false
}

func isFuzzRun(args []string) bool {
	for i, arg := range args {
		for _, prefix := range []string{"-test.fuzz=", "--test.fuzz="} {
			target, ok := strings.CutPrefix(arg, prefix)
			if ok {
				return target != ""
			}
		}
		if arg == "-test.fuzz" || arg == "--test.fuzz" {
			return i+1 < len(args) && args[i+1] != ""
		}
	}
	return false
}

func diff(current, base stackSet) []string {
	var extras []string
	for stack, count := range current {
		if extra := count - base[stack]; extra > 0 {
			for range extra {
				extras = append(extras, stack)
			}
		}
	}
	sort.Strings(extras)
	return extras
}
