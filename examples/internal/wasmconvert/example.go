package wasmconvert

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/avdoseferovic/paper/pkg/metrics"

	"github.com/avdoseferovic/paper/examples/internal/exampleregistry"
)

// ErrUnknownExample is returned when the requested example is not registered.
var ErrUnknownExample = errors.New("wasmconvert: unknown example")

// errExamplePanic wraps a recovered panic from an example's render tree.
var errExamplePanic = errors.New("wasmconvert: example render panicked")

// ExampleAssets returns the repo-relative paths the named example reads. The
// caller is expected to make each one readable — in the browser, by fetching it
// and handing the bytes to paperFS — before calling ExampleToBase64.
func ExampleAssets(name string) ([]string, error) {
	entry, err := exampleregistry.Lookup(name)
	if err != nil {
		return nil, unknownExample(name, err)
	}
	// Clone: the slice shares its backing array with the package-level registry,
	// so handing it out directly would let a caller rewrite global state.
	return slices.Clone(entry.Assets), nil
}

// unknownExample translates a registry lookup failure into this package's
// sentinel while keeping the original error in the chain, so errors.Is matches
// either sentinel and a future non-not-found error is not mislabelled.
func unknownExample(name string, err error) error {
	if errors.Is(err, exampleregistry.ErrUnknownExample) {
		return fmt.Errorf("%w: %q: %w", ErrUnknownExample, name, err)
	}
	return fmt.Errorf("wasmconvert: look up %q: %w", name, err)
}

// ExampleToBase64 renders the named documented example and returns the PDF as
// base64. It recovers from any panic in the render tree so the caller — and the
// wasm goroutine — always survives.
func ExampleToBase64(ctx context.Context, name string) (b64 string, err error) {
	defer func() {
		if r := recover(); r != nil {
			b64 = ""
			err = fmt.Errorf("%w: %v", errExamplePanic, r)
		}
	}()

	entry, lookupErr := exampleregistry.Lookup(name)
	if lookupErr != nil {
		return "", unknownExample(name, lookupErr)
	}

	pdf, genErr := entry.Build().Generate(ctx)
	if genErr != nil {
		return "", fmt.Errorf("wasmconvert: generate %s: %w", name, genErr)
	}

	// A file the renderer could not read is not a generation error: the provider
	// records a render issue, draws an error box in the cell and carries on, so a
	// perfectly valid PDF comes back with a placeholder where the image should
	// be. Surfacing it here is what keeps a broken asset from shipping silently.
	if issue, ok := firstAssetIssue(pdf); ok {
		return "", fmt.Errorf("wasmconvert: %s: %s", name, issue)
	}

	return pdf.GetBase64(), nil
}

// firstAssetIssue reports the first recorded issue that means an input file was
// not readable, along with whether one was found.
func firstAssetIssue(pdf interface{ GetReport() *metrics.Report }) (string, bool) {
	report := pdf.GetReport()
	if report == nil {
		return "", false
	}
	for _, issue := range report.RenderIssues {
		if issue.Operation != "image.load" {
			continue
		}
		if issue.Error != "" {
			return fmt.Sprintf("%s: %s (%s)", issue.Operation, issue.Message, issue.Error), true
		}
		return fmt.Sprintf("%s: %s", issue.Operation, issue.Message), true
	}
	return "", false
}
