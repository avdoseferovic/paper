// Package html converts HTML strings into Paper rows so they can be added to
// a Paper document. No browser, no external binary, no JavaScript.
//
// Supported tags and CSS properties are documented in docs/html-support.md.
package html

import (
	"context"
	"fmt"
	"io"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/html/translate"
)

// Option configures FromString / FromReader behaviour.
type Option func(*config)

type config struct {
	unsupportedHandler  func(thing, value string)
	gridSize            int
	contentWidthMM      float64
	imageBaseDir        string
	stylesheetBaseDir   string
	limits              Limits
	limitsSet           bool
	outlineFromHeadings bool
}

// WithOutlineFromHeadings adds h1-h6 headings to the PDF document outline:
// h1 becomes a level-0 entry, h2 a level-1 entry, and so on. Hidden headings
// are skipped.
func WithOutlineFromHeadings() Option {
	return func(c *config) {
		c.outlineFromHeadings = true
	}
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

// WithImageBaseDir scopes <img src=...> local-file reads to a single directory.
// Paths that escape via ".." or absolute prefix are refused.
func WithImageBaseDir(dir string) Option {
	return func(c *config) {
		c.imageBaseDir = dir
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
// It observes ctx before and after DOM parsing and during translation.
func FromString(ctx context.Context, htmlStr string, opts ...Option) ([]core.Row, error) {
	if htmlStr == "" {
		return nil, nil
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
	return translate.Translate(ctx, doc, cfg.translateOptions()...)
}

// FromReader parses HTML from an io.Reader and returns the corresponding
// rows. It observes ctx before and after reading and during translation.
func FromReader(ctx context.Context, r io.Reader, opts ...Option) ([]core.Row, error) {
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
	return FromString(ctx, string(data), opts...)
}

func conversionCanceled(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("html: conversion canceled: %w", err)
	}
	return nil
}
