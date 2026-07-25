// Package exampleregistry maps a documented example's name to the builder that
// produces it, so the wasm binary can render any of them on demand for the docs
// site instead of the site shipping a committed PDF per page.
//
// Entries hold the same GetPaper builders the docs pages display and the
// structure tests assert, so a preview cannot drift from the code beside it.
//
// Assets lists every file the builder reads. Those paths are repo-root relative
// because that is exactly the string the library hands to os.ReadFile — on the
// host it resolves against the working directory, and in the browser it is the
// key the in-memory filesystem (docs/assets/js/paper-fs.js) is populated under.
package exampleregistry

import (
	"errors"
	"fmt"
	"slices"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper/examples/internal/examplepath"

	"github.com/avdoseferovic/paper/docs/assets/examples/addpage"
	"github.com/avdoseferovic/paper/docs/assets/examples/autorow"
	"github.com/avdoseferovic/paper/docs/assets/examples/barcodegrid"
	"github.com/avdoseferovic/paper/docs/assets/examples/billing"
	"github.com/avdoseferovic/paper/docs/assets/examples/bookmark"
	"github.com/avdoseferovic/paper/docs/assets/examples/cellstyle"
	"github.com/avdoseferovic/paper/docs/assets/examples/checkbox"
	"github.com/avdoseferovic/paper/docs/assets/examples/compression"
	"github.com/avdoseferovic/paper/docs/assets/examples/customdimensions"
	"github.com/avdoseferovic/paper/docs/assets/examples/custompage"
	"github.com/avdoseferovic/paper/docs/assets/examples/datamatrixgrid"
	"github.com/avdoseferovic/paper/docs/assets/examples/footer"
	"github.com/avdoseferovic/paper/docs/assets/examples/header"
	"github.com/avdoseferovic/paper/docs/assets/examples/imagegrid"
	"github.com/avdoseferovic/paper/docs/assets/examples/line"
	"github.com/avdoseferovic/paper/docs/assets/examples/list"
	"github.com/avdoseferovic/paper/docs/assets/examples/lowmemory"
	"github.com/avdoseferovic/paper/docs/assets/examples/margins"
	"github.com/avdoseferovic/paper/docs/assets/examples/maxgridsum"
	"github.com/avdoseferovic/paper/docs/assets/examples/metadatas"
	"github.com/avdoseferovic/paper/docs/assets/examples/orientation"
	"github.com/avdoseferovic/paper/docs/assets/examples/pagenumber"
	"github.com/avdoseferovic/paper/docs/assets/examples/parallelism"
	"github.com/avdoseferovic/paper/docs/assets/examples/protection"
	"github.com/avdoseferovic/paper/docs/assets/examples/qrgrid"
	"github.com/avdoseferovic/paper/docs/assets/examples/signaturegrid"
	"github.com/avdoseferovic/paper/docs/assets/examples/simplest"
	"github.com/avdoseferovic/paper/docs/assets/examples/textgrid"
	"github.com/avdoseferovic/paper/docs/assets/examples/watermark"
)

// ErrUnknownExample is returned by Lookup for a name that is not registered.
var ErrUnknownExample = errors.New("exampleregistry: unknown example")

// Image assets shared by several examples.
const (
	biplane   = "docs/assets/images/biplane.jpg"
	frontpage = "docs/assets/images/frontpage.png"
	gopherbw  = "docs/assets/images/gopherbw.png"
)

// Example is one documented example that can be rendered on demand.
type Example struct {
	// Assets are repo-root-relative paths the builder reads. They must be
	// readable before Build runs.
	Assets []string
	// Build produces the document. It never returns nil.
	Build func() core.Paper
}

// registry holds every example whose preview is generated in the browser.
//
// Four documented examples are deliberately absent because their inputs are too
// large or too awkward to ship to a browser, so their pages keep a committed
// PDF: background and disablepagebreak (a 792KB PNG), customfont (a 23MB TTF)
// and mergepdf (needs an existing PDF to merge). unittests has no builder.
var registry = map[string]Example{
	"addpage":          {Build: addpage.GetPaper},
	"autorow":          {Assets: []string{biplane}, Build: autorow.GetPaper},
	"barcodegrid":      {Build: barcodegrid.GetPaper},
	"billing":          {Assets: []string{biplane}, Build: billing.GetPaper},
	"bookmark":         {Build: bookmark.GetPaper},
	"cellstyle":        {Build: cellstyle.GetPaper},
	"checkbox":         {Build: checkbox.GetPaper},
	"compression":      {Assets: []string{biplane, frontpage}, Build: compressionPaper},
	"customdimensions": {Assets: []string{biplane}, Build: customdimensions.GetPaper},
	"custompage":       {Assets: []string{biplane}, Build: custompage.GetPaper},
	"datamatrixgrid":   {Build: datamatrixgrid.GetPaper},
	"footer":           {Build: footer.GetPaper},
	"header":           {Build: header.GetPaper},
	"imagegrid":        {Assets: []string{biplane, frontpage}, Build: imagegrid.GetPaper},
	"line":             {Build: line.GetPaper},
	"list":             {Build: list.GetPaper},
	"lowmemory":        {Build: lowmemory.GetPaper},
	"margins":          {Assets: []string{gopherbw}, Build: margins.GetPaper},
	"maxgridsum":       {Build: maxgridsum.GetPaper},
	"metadatas":        {Build: metadatas.GetPaper},
	"orientation":      {Build: orientation.GetPaper},
	"pagenumber":       {Build: pagenumber.GetPaper},
	"parallelism":      {Build: parallelism.GetPaper},
	"protection":       {Build: protection.GetPaper},
	"qrgrid":           {Build: qrgrid.GetPaper},
	"signaturegrid":    {Build: signaturegrid.GetPaper},
	"simplest":         {Assets: []string{biplane}, Build: simplest.GetPaper},
	"textgrid":         {Build: textgrid.GetPaper},
	"watermark":        {Build: watermark.GetPaper},
}

// compressionPaper adapts the one builder that takes an argument and reports an
// error. Unlike the others it reads its image eagerly, at build time rather than
// at render time, so the path has to resolve right now.
//
// examplepath.Repo makes that work on both sides: on the host it anchors the
// path to the repository root, which a test running in this package's directory
// would otherwise miss, and under wasm os.Getwd fails so it hands back the bare
// relative path — precisely the key the in-memory filesystem is populated under.
//
// A failure here means a declared asset was not made readable first, which is a
// wiring bug rather than something a caller can act on, so it panics and lets
// the wasm boundary turn it into an error result.
func compressionPaper() core.Paper {
	doc, err := compression.GetPaper(examplepath.Repo(frontpage))
	if err != nil {
		panic(fmt.Sprintf("exampleregistry: compression: %v", err))
	}
	return doc
}

// Names returns every registered example name, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Lookup returns the example registered under name, or ErrUnknownExample.
func Lookup(name string) (Example, error) {
	entry, ok := registry[name]
	if !ok {
		return Example{}, fmt.Errorf("%w: %q", ErrUnknownExample, name)
	}
	return entry, nil
}
