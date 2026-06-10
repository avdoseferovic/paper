package paper

import (
	"github.com/avdoseferovic/paper/internal/cache"
	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/metrics"
	"github.com/avdoseferovic/paper/pkg/props"
)

// compile-time assertion: *provider satisfies core.Provider.
var _ core.Provider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.RenderIssueProvider.
var _ core.RenderIssueProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.RichTextProvider.
var _ core.RichTextProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.RichTextMeasurer.
var _ core.RichTextMeasurer = (*provider)(nil)

// compile-time assertion: *provider satisfies core.ShapeProvider.
var _ core.ShapeProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.PositionProvider.
var _ core.PositionProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.PageProvider.
var _ core.PageProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.AlphaProvider.
var _ core.AlphaProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.LinkProvider.
var _ core.LinkProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.LateFontProvider.
var _ core.LateFontProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.CharSpacingProvider.
// Note: the underlying phpdave11/gofpdf fork does not expose SetCharSpacing,
// so WithCharSpacing is currently a no-op (documented limitation in
// docs/html-support.md). The capability interface is in place so a future
// fork swap can light up the feature without further wiring changes.
var _ core.CharSpacingProvider = (*provider)(nil)

type provider struct {
	fpdf        providerPDF
	documentPDF providerDocumentPDF
	metadataPDF providerMetadataPDF
	errorPDF    providerErrorPDF
	font        core.Font
	text        core.Text
	richText    *Text // typed pointer for RichTextProvider; nil-safe when text is a mock
	code        core.Code
	image       core.Image
	line        core.Line
	checkbox    core.Checkbox
	cache       cache.Cache
	cellWriter  cellwriter.CellWriter
	cfg         *entity.Config
	issues      []metrics.RenderIssue
}

// New is the constructor of provider for gofpdf.
func New(dep *Dependencies) core.Provider {
	richText, _ := dep.Text.(*Text)
	return &provider{
		fpdf:        asProviderPDF[providerPDF](dep.PDF),
		documentPDF: asProviderPDF[providerDocumentPDF](dep.PDF),
		metadataPDF: asProviderPDF[providerMetadataPDF](dep.PDF),
		errorPDF:    asProviderPDF[providerErrorPDF](dep.PDF),
		font:        dep.Font,
		text:        dep.Text,
		richText:    richText,
		code:        dep.Code,
		image:       dep.Image,
		line:        dep.Line,
		checkbox:    dep.Checkbox,
		cellWriter:  dep.CellWriter,
		cfg:         dep.Cfg,
		cache:       dep.Cache,
	}
}

func asProviderPDF[T any](pdf any) T {
	typed, _ := pdf.(T)
	return typed
}

func (g *provider) MeasureString(text string, prop *props.Text) float64 {
	if g.richText == nil {
		return 0
	}
	return g.richText.MeasureString(text, prop)
}

func (g *provider) AddTextAt(x, y float64, text string, prop *props.Text) {
	if g.richText == nil {
		return
	}
	g.richText.AddTextAt(x, y, text, prop)
}

func (g *provider) AddRichText(runs []props.RichRun, cell *entity.Cell, prop *props.RichText) {
	if g.richText == nil {
		return
	}
	g.richText.AddRichText(runs, cell, prop)
}

func (g *provider) MeasureRichText(runs []props.RichRun, cell *entity.Cell, prop *props.RichText) float64 {
	if g.richText == nil {
		return 0
	}
	return g.richText.MeasureRichText(runs, cell, prop)
}
