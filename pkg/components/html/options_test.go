package html_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	htmlcomponent "github.com/avdoseferovic/paper/pkg/components/html"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// pngDataURI synthesises a tiny PNG and returns it as a data: URI so the safe
// default image resolver accepts it without any base dir.
func pngDataURI(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// firstColSize walks the component structure and returns the size of the first
// "col" node, which for an <img> row reflects the grid-quantised image width.
func firstColSize(t *testing.T, n *node.Node[core.Structure]) (int, bool) {
	t.Helper()
	var (
		size  int
		found bool
	)
	walkStructureNode(n, func(s core.Structure) {
		if found || s.Type != "col" {
			return
		}
		if v, ok := s.Value.(int); ok {
			size = v
			found = true
		}
	})
	return size, found
}

func walkStructureNode(n *node.Node[core.Structure], fn func(core.Structure)) {
	if n == nil {
		return
	}
	fn(n.GetData())
	for _, child := range n.GetNexts() {
		walkStructureNode(child, fn)
	}
}

func TestWithUnsupportedHandler_FiresForRefusedLocalImage(t *testing.T) {
	t.Parallel()

	// Arrange
	var things []string
	handler := func(thing, _ string) { things = append(things, thing) }

	// Act
	component, err := htmlcomponent.New(
		`<img src="local.png" alt="fallback">`,
		htmlcomponent.WithUnsupportedHandler(handler),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, component)
	assert.Contains(t, things, "img.src",
		"unsupported handler should report the refused local image source")
}

func TestWithImageBaseDir_RefusesPathEscapeViaHandler(t *testing.T) {
	t.Parallel()

	// Arrange
	dir := t.TempDir()
	var reports [][2]string
	handler := func(thing, value string) { reports = append(reports, [2]string{thing, value}) }

	// Act
	component, err := htmlcomponent.New(
		`<img src="../escape.png" alt="fallback">`,
		htmlcomponent.WithImageBaseDir(dir),
		htmlcomponent.WithUnsupportedHandler(handler),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, component)
	require.NotEmpty(t, reports, "path escape must surface through the unsupported handler")

	var escaped bool
	for _, report := range reports {
		if report[0] == "img.src" && strings.Contains(report[1], "escapes base dir") {
			escaped = true
		}
	}
	assert.True(t, escaped, "escaping image read must be refused with a path-escape error")

	// No image col is produced; the refused image falls back to its alt text.
	if size, ok := firstColSize(t, component.GetStructure()); ok {
		assert.NotEqual(t, 1, size,
			"refused image should not yield a loaded image column")
	}
}

func TestWithStylesheetBaseDir_RefusesPathEscapeViaHandler(t *testing.T) {
	t.Parallel()

	// Arrange
	dir := t.TempDir()
	var reports [][2]string
	handler := func(thing, value string) { reports = append(reports, [2]string{thing, value}) }

	// Act
	component, err := htmlcomponent.New(
		`<head><link rel="stylesheet" href="../x.css"></head><body><p>hi</p></body>`,
		htmlcomponent.WithStylesheetBaseDir(dir),
		htmlcomponent.WithUnsupportedHandler(handler),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, component)

	var skipped bool
	for _, report := range reports {
		if report[0] == "link.skipped" && report[1] == "../x.css" {
			skipped = true
		}
	}
	assert.True(t, skipped, "escaping stylesheet read must be skipped and reported")
}

func TestWithGridSize_QuantisesImageColumnToCustomGrid(t *testing.T) {
	t.Parallel()

	// Arrange: an 85mm image is half of the default 170mm content width.
	htmlStr := fmt.Sprintf(`<img src="%s" width="85mm" height="20mm" alt="x">`, pngDataURI(t))

	// Act
	defaultComponent, err := htmlcomponent.New(htmlStr)
	require.NoError(t, err)
	customComponent, err := htmlcomponent.New(htmlStr, htmlcomponent.WithGridSize(24))
	require.NoError(t, err)

	// Assert: half-width image spans half the grid in either grid size.
	defaultSize, ok := firstColSize(t, defaultComponent.GetStructure())
	require.True(t, ok)
	customSize, ok := firstColSize(t, customComponent.GetStructure())
	require.True(t, ok)

	assert.Equal(t, 6, defaultSize, "default 12-col grid: half width => 6 cols")
	assert.Equal(t, 12, customSize, "24-col grid: half width => 12 cols")
}

func TestWithContentWidth_ChangesImageColumnQuantisation(t *testing.T) {
	t.Parallel()

	// Arrange: an 85mm image is half of the default 170mm content width but
	// fills a custom 85mm content width entirely.
	htmlStr := fmt.Sprintf(`<img src="%s" width="85mm" height="20mm" alt="x">`, pngDataURI(t))

	// Act
	defaultComponent, err := htmlcomponent.New(htmlStr)
	require.NoError(t, err)
	narrowComponent, err := htmlcomponent.New(htmlStr, htmlcomponent.WithContentWidth(85))
	require.NoError(t, err)

	// Assert
	defaultSize, ok := firstColSize(t, defaultComponent.GetStructure())
	require.True(t, ok)
	narrowSize, ok := firstColSize(t, narrowComponent.GetStructure())
	require.True(t, ok)

	assert.Equal(t, 6, defaultSize, "85mm image in 170mm content => half the grid")
	assert.Equal(t, 12, narrowSize, "85mm image in 85mm content => full grid")
}
