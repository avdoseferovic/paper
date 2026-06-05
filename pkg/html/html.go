// Package html converts HTML strings into Paper rows so they can be added to
// a Paper document. No browser, no external binary, no JavaScript.
//
// Supported tags and CSS properties are documented in docs/v2/html-support.md.
package html

import (
	"fmt"
	"io"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/html/translate"
)

// Option configures FromString / FromReader behaviour.
type Option func(*config)

type config struct {
	unsupportedHandler func(thing, value string)
	gridSize           int
	contentWidthMM     float64
	imageResolver      translate.ImageResolver
	imageBaseDir       string
	stylesheetResolver translate.StylesheetResolver
	stylesheetBaseDir  string
}

// WithUnsupportedHandler registers a callback invoked for unsupported HTML tags
// or CSS properties. Use it to log diagnostics during development.
func WithUnsupportedHandler(fn func(thing, value string)) Option {
	return func(c *config) {
		c.unsupportedHandler = fn
	}
}

// WithGridSize overrides the default 12-column grid for flex quantization.
// Use this when the paper document was built with config.WithMaxGridSize(n).
func WithGridSize(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.gridSize = n
		}
	}
}

// WithContentWidth sets the page content width in mm for accurate gap-to-col
// approximation. Default is 170mm (A4 with 20mm left+right margins).
func WithContentWidth(mm float64) Option {
	return func(c *config) {
		if mm > 0 {
			c.contentWidthMM = mm
		}
	}
}

// WithImageResolver lets callers plug in a custom resolver for <img src=...>.
// The default resolver only accepts data: URIs to prevent path traversal on
// user-controlled HTML.
func WithImageResolver(fn translate.ImageResolver) Option {
	return func(c *config) {
		c.imageResolver = fn
	}
}

// WithImageBaseDir scopes <img src=...> local-file reads to a single directory.
// Paths that escape via ".." or absolute prefix are refused.
func WithImageBaseDir(dir string) Option {
	return func(c *config) {
		c.imageBaseDir = dir
	}
}

// WithStylesheetResolver registers a custom resolver for <link rel="stylesheet">.
// The default (no resolver) only accepts data: URIs.
func WithStylesheetResolver(fn translate.StylesheetResolver) Option {
	return func(c *config) {
		c.stylesheetResolver = fn
	}
}

// WithStylesheetBaseDir scopes <link href=...> local-file reads to a single
// directory. Paths that escape via ".." or absolute prefix are refused.
func WithStylesheetBaseDir(dir string) Option {
	return func(c *config) {
		c.stylesheetBaseDir = dir
	}
}

// FromString parses an HTML string and returns the corresponding Paper rows.
func FromString(htmlStr string, opts ...Option) ([]core.Row, error) {
	if htmlStr == "" {
		return nil, nil
	}
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	doc, err := dom.Parse(htmlStr)
	if err != nil {
		return nil, err
	}
	var tOpts []translate.Option
	if cfg.gridSize > 0 {
		tOpts = append(tOpts, translate.WithGridSize(cfg.gridSize))
	}
	if cfg.contentWidthMM > 0 {
		tOpts = append(tOpts, translate.WithContentWidth(cfg.contentWidthMM))
	}
	if cfg.imageResolver != nil {
		tOpts = append(tOpts, translate.WithImageResolver(cfg.imageResolver))
	} else if cfg.imageBaseDir != "" {
		tOpts = append(tOpts, translate.WithImageBaseDir(cfg.imageBaseDir))
	}
	if cfg.stylesheetResolver != nil {
		tOpts = append(tOpts, translate.WithStylesheetResolver(cfg.stylesheetResolver))
	} else if cfg.stylesheetBaseDir != "" {
		tOpts = append(tOpts, translate.WithStylesheetBaseDir(cfg.stylesheetBaseDir))
	}
	if cfg.unsupportedHandler != nil {
		tOpts = append(tOpts, translate.WithUnsupportedHandler(cfg.unsupportedHandler))
	}
	return translate.Translate(doc, tOpts...)
}

// FromReader parses HTML from an io.Reader and returns the corresponding rows.
func FromReader(r io.Reader, opts ...Option) ([]core.Row, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("html: reading input: %w", err)
	}
	return FromString(string(data), opts...)
}
