package wasmconvert_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/examples/internal/examplepath"
	"github.com/avdoseferovic/paper/examples/internal/exampleregistry"
	"github.com/avdoseferovic/paper/examples/internal/wasmconvert"
)

// TestExampleToBase64RendersEveryExample generates all 29 registered examples on
// the host. It is the cheapest regression net for the previews that no longer
// ship as committed PDFs.
//
// It cannot run in parallel: the examples reference their images by repo-root
// relative path (unchanged, because the browser resolves those same strings
// through the in-memory filesystem), so the test has to run from the repo root,
// and t.Chdir is process-wide.
func TestExampleToBase64RendersEveryExample(t *testing.T) {
	t.Chdir(examplepath.Repo("."))

	for _, name := range exampleregistry.Names() {
		t.Run(name, func(t *testing.T) {
			b64, err := wasmconvert.ExampleToBase64(context.Background(), name)
			if err != nil {
				t.Fatalf("ExampleToBase64(%q): %v", name, err)
			}

			raw, decodeErr := base64.StdEncoding.DecodeString(b64)
			if decodeErr != nil {
				t.Fatalf("result is not valid base64: %v", decodeErr)
			}
			if !strings.HasPrefix(string(raw), "%PDF-") {
				t.Errorf("decoded output is not a PDF, starts with %q", firstBytes(raw))
			}
		})
	}
}

// TestExampleToBase64ReportsUnreadableAsset pins the behaviour that makes the
// suite above meaningful. A missing image does not fail generation: the provider
// records a render issue, draws an error box and returns a perfectly valid PDF.
// Checking only for a %PDF- prefix would therefore pass even if no image ever
// loaded, so ExampleToBase64 turns that recorded issue into an error.
func TestExampleToBase64ReportsUnreadableAsset(t *testing.T) {
	t.Chdir(t.TempDir()) // nothing readable here, so every image lookup fails

	_, err := wasmconvert.ExampleToBase64(context.Background(), "imagegrid")
	if err == nil {
		t.Fatal("expected an error when the example's images are unreadable, got nil")
	}
	if !strings.Contains(err.Error(), "image.load") {
		t.Errorf("error should name the failing operation, got: %v", err)
	}
}

func TestExampleToBase64UnknownName(t *testing.T) {
	t.Parallel()

	_, err := wasmconvert.ExampleToBase64(context.Background(), "nope")
	if !errors.Is(err, wasmconvert.ErrUnknownExample) {
		t.Errorf("expected ErrUnknownExample, got %v", err)
	}
	// The registry's own sentinel stays in the chain, so a caller can match
	// either one and a future non-not-found failure cannot be mislabelled.
	if !errors.Is(err, exampleregistry.ErrUnknownExample) {
		t.Errorf("registry sentinel should remain in the chain, got %v", err)
	}
}

// TestExampleToBase64RecoversPanic covers the deferred recover(). A builder can
// genuinely panic — compressionPaper does so by design when its image was not
// made readable first — and a panic crossing the js.FuncOf boundary would abort
// the program and permanently disable every export on the page.
func TestExampleToBase64RecoversPanic(t *testing.T) {
	t.Chdir(t.TempDir()) // compression reads its image eagerly, so this panics

	b64, err := wasmconvert.ExampleToBase64(context.Background(), "compression")

	if err == nil {
		t.Fatal("expected an error from the panicking builder, got nil")
	}
	if b64 != "" {
		t.Errorf("expected no output alongside the error, got %d bytes", len(b64))
	}
	if !strings.Contains(err.Error(), "panicked") {
		t.Errorf("error should identify the panic, got: %v", err)
	}

	// The process is still usable afterwards — that is the whole point.
	if _, err := wasmconvert.ExampleAssets("textgrid"); err != nil {
		t.Errorf("package unusable after recovering a panic: %v", err)
	}
}

// TestExampleAssetsDoesNotAliasRegistry pins that callers get a copy. The slice
// would otherwise share its backing array with the package-level registry map.
func TestExampleAssetsDoesNotAliasRegistry(t *testing.T) {
	t.Parallel()

	first, err := wasmconvert.ExampleAssets("imagegrid")
	if err != nil {
		t.Fatalf("ExampleAssets: %v", err)
	}
	first[0] = "clobbered"

	second, err := wasmconvert.ExampleAssets("imagegrid")
	if err != nil {
		t.Fatalf("ExampleAssets: %v", err)
	}
	if second[0] == "clobbered" {
		t.Error("mutating the returned slice corrupted the registry")
	}
}

func TestExampleAssets(t *testing.T) {
	t.Parallel()

	assets, err := wasmconvert.ExampleAssets("imagegrid")
	if err != nil {
		t.Fatalf("ExampleAssets: %v", err)
	}
	if len(assets) != 2 {
		t.Errorf("imagegrid should declare 2 images, got %v", assets)
	}

	empty, err := wasmconvert.ExampleAssets("textgrid")
	if err != nil {
		t.Fatalf("ExampleAssets: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("textgrid needs no assets, got %v", empty)
	}

	if _, err := wasmconvert.ExampleAssets("nope"); !errors.Is(err, wasmconvert.ErrUnknownExample) {
		t.Errorf("expected ErrUnknownExample, got %v", err)
	}
}

func firstBytes(b []byte) string {
	if len(b) > 8 {
		b = b[:8]
	}
	return string(b)
}
