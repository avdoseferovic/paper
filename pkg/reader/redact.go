package reader

import (
	"fmt"
	"regexp"
	"strings"
)

// RedactOptions configures redaction behavior.
//
// This foundation currently performs byte-preserving text removal only. Visual
// overlay fields are kept for API compatibility with richer redaction flows.
type RedactOptions struct {
	FillColor       [3]float64
	OverlayText     string
	OverlayFontSize float64
	OverlayColor    [3]float64
	StripMetadata   bool
}

// RedactText permanently removes literal text targets from uncompressed page
// content streams by replacing matched bytes with spaces.
func RedactText(r *PdfReader, targets []string, opts *RedactOptions) (*Modifier, error) {
	if len(targets) == 0 {
		return modifierFromReader(r)
	}
	patterns := make([]*regexp.Regexp, 0, len(targets))
	for _, target := range targets {
		if target == "" {
			continue
		}
		patterns = append(patterns, regexp.MustCompile(`(?i)`+regexp.QuoteMeta(target)))
	}
	return redactPatterns(r, patterns, opts)
}

// RedactPattern permanently removes regex matches from uncompressed page
// content streams by replacing matched bytes with spaces.
func RedactPattern(r *PdfReader, pattern *regexp.Regexp, opts *RedactOptions) (*Modifier, error) {
	if pattern == nil {
		return modifierFromReader(r)
	}
	return redactPatterns(r, []*regexp.Regexp{pattern}, opts)
}

func redactPatterns(r *PdfReader, patterns []*regexp.Regexp, _ *RedactOptions) (*Modifier, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: nil reader", ErrUnsupportedPDF)
	}
	out := r.RawBytes()
	for _, page := range r.pages {
		err := redactPageStreams(out, r, page, patterns)
		if err != nil {
			return nil, err
		}
	}
	return &Modifier{data: out, pageCount: r.PageCount()}, nil
}

func modifierFromReader(r *PdfReader) (*Modifier, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: nil reader", ErrUnsupportedPDF)
	}
	return &Modifier{data: r.RawBytes(), pageCount: r.PageCount()}, nil
}

func redactPageStreams(out []byte, r *PdfReader, page *PageInfo, patterns []*regexp.Regexp) error {
	pageObject, ok := r.objects[page.objectID]
	if !ok {
		return fmt.Errorf("%w: page object %d missing", ErrUnsupportedPDF, page.objectID)
	}
	for _, ref := range parseContentReferences(pageObject.content) {
		object, ok := r.objects[ref]
		if !ok {
			return fmt.Errorf("%w: content object %d missing", ErrUnsupportedPDF, ref)
		}
		err := redactObjectStream(out, object, patterns)
		if err != nil {
			return err
		}
	}
	return nil
}

func redactObjectStream(out []byte, object pdfObject, patterns []*regexp.Regexp) error {
	start, end, err := streamDataBounds(object.content)
	if err != nil {
		return err
	}
	filter := parseFilter(object.content[:start])
	if strings.Contains(filter, "FlateDecode") {
		return fmt.Errorf("%w: compressed content streams cannot be redacted by the byte-preserving foundation", ErrUnsupportedPDF)
	}
	if filter != "" {
		return fmt.Errorf("%w: stream filter %s cannot be redacted by the byte-preserving foundation", ErrUnsupportedPDF, filter)
	}

	absoluteStart := object.contentStart + start
	absoluteEnd := object.contentStart + end
	redactBytesInPlace(out[absoluteStart:absoluteEnd], patterns)
	return nil
}

func redactBytesInPlace(data []byte, patterns []*regexp.Regexp) {
	for _, pattern := range patterns {
		if pattern == nil {
			continue
		}
		matches := pattern.FindAllIndex(data, -1)
		for _, match := range matches {
			for i := match[0]; i < match[1]; i++ {
				data[i] = ' '
			}
		}
	}
}
