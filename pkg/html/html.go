// Package html converts HTML strings into Maroto rows so they can be added to
// a Maroto document. No browser, no external binary, no JavaScript.
//
// Supported tags and CSS properties are documented in docs/v2/html-support.md.
package html

import (
	"fmt"
	"io"

	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/html/translate"
)

// Option configures FromString / FromReader behaviour.
type Option func(*config)

type config struct {
	unsupportedHandler func(thing, value string)
	gridSize           int
	contentWidthMM     float64
}

// WithUnsupportedHandler registers a callback invoked for unsupported HTML tags
// or CSS properties. Use it to log diagnostics during development.
func WithUnsupportedHandler(fn func(thing, value string)) Option {
	return func(c *config) {
		c.unsupportedHandler = fn
	}
}

// WithGridSize overrides the default 12-column grid for flex quantization.
// Use this when the maroto document was built with config.WithMaxGridSize(n).
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

// FromString parses an HTML string and returns the corresponding Maroto rows.
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
