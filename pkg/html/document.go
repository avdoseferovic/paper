package html

import (
	"context"
	"fmt"
	"io"

	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/html/translate"
)

type (
	// Document is the full result of converting an HTML document: content
	// rows plus document-level options parsed from `@page` rules.
	Document = translate.Document
	// PageOptions captures the supported subset of the CSS `@page` rule.
	PageOptions = translate.PageOptions
)

// DocumentFromString parses an HTML string and returns the full Document
// result. Unlike FromString, it surfaces `@page` size/margin options so
// callers (notably paper.FromHTML) can configure the page before rendering.
// It observes ctx before and after DOM parsing and during translation.
func DocumentFromString(ctx context.Context, htmlStr string, opts ...Option) (*Document, error) {
	if htmlStr == "" {
		return &Document{}, nil
	}
	err := conversionCanceled(ctx)
	if err != nil {
		return nil, err
	}
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	doc, err := dom.Parse(htmlStr)
	if err != nil {
		return nil, err
	}
	err = conversionCanceled(ctx)
	if err != nil {
		return nil, err
	}
	return translate.TranslateDocument(ctx, doc, cfg.translateOptions()...)
}

// DocumentFromReader parses HTML from an io.Reader and returns the full
// Document result. It observes ctx before and after reading and during
// translation.
func DocumentFromReader(ctx context.Context, r io.Reader, opts ...Option) (*Document, error) {
	err := conversionCanceled(ctx)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("html: reading input: %w", err)
	}
	err = conversionCanceled(ctx)
	if err != nil {
		return nil, err
	}
	return DocumentFromString(ctx, string(data), opts...)
}

// translateOptions converts the package-level config into translate options.
func (c *config) translateOptions() []translate.Option {
	var tOpts []translate.Option
	if c.gridSize > 0 {
		tOpts = append(tOpts, translate.WithGridSize(c.gridSize))
	}
	if c.contentWidthMM > 0 {
		tOpts = append(tOpts, translate.WithContentWidth(c.contentWidthMM))
	}
	if c.imageBaseDir != "" {
		tOpts = append(tOpts, translate.WithImageBaseDir(c.imageBaseDir))
	}
	if c.stylesheetBaseDir != "" {
		tOpts = append(tOpts, translate.WithStylesheetBaseDir(c.stylesheetBaseDir))
	}
	if c.limitsSet {
		tOpts = append(tOpts, translate.WithLimits(c.limits))
	} else {
		tOpts = append(tOpts, translate.WithLimits(htmllimits.Default()))
	}
	if c.unsupportedHandler != nil {
		tOpts = append(tOpts, translate.WithUnsupportedHandler(c.unsupportedHandler))
	}
	if c.outlineFromHeadings {
		tOpts = append(tOpts, translate.WithOutlineFromHeadings())
	}
	return tOpts
}
