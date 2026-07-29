// Package tmpl renders Go html/templates through Paper's HTML-to-PDF
// pipeline.
//
// It provides a template input mode alongside Paper's component API and raw
// HTML conversion: html/template -> pkg/html rows -> generated PDF.
package tmpl

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	htmltpl "html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	paperhtml "github.com/avdoseferovic/paper/pkg/html"
)

var (
	errDictOddArguments = errors.New("tmpl: dict: odd number of arguments")
	errDictKeyNotString = errors.New("tmpl: dict: key is not a string")
)

// Options configures template rendering. All fields are optional.
type Options struct {
	// Funcs registers custom helpers for html/template. Custom helpers
	// override built-in helpers with the same name.
	Funcs htmltpl.FuncMap

	// BaseTemplate is an optional pre-parsed template tree. Template strings
	// are parsed into a clone of this tree so shared definitions are available
	// without mutating the caller's template.
	BaseTemplate *htmltpl.Template

	// HTMLOptions are passed to pkg/html after template execution.
	HTMLOptions []paperhtml.Option

	// Config is the Paper document configuration used by RenderDocument,
	// RenderTo, RenderFile, and RenderFileTo.
	Config *entity.Config
}

// Render executes templateStr with data and converts the resulting HTML into
// Paper rows.
func Render(ctx context.Context, templateStr string, data any, opts *Options) ([]core.Row, error) {
	htmlStr, err := execute(templateStr, "", data, opts)
	if err != nil {
		return nil, err
	}
	rows, err := paperhtml.FromString(ctx, htmlStr, htmlOptions(opts)...)
	if err != nil {
		return nil, fmt.Errorf("tmpl: html conversion failed: %w", err)
	}
	return rows, nil
}

// RenderDocument executes templateStr with data and returns a generated PDF.
func RenderDocument(ctx context.Context, templateStr string, data any, opts *Options) (*core.Pdf, error) {
	htmlStr, err := execute(templateStr, "", data, opts)
	if err != nil {
		return nil, err
	}
	return renderHTML(ctx, htmlStr, opts, nil)
}

// RenderTo executes templateStr with data and writes the generated PDF to w.
func RenderTo(ctx context.Context, w io.Writer, templateStr string, data any, opts *Options) error {
	doc, err := RenderDocument(ctx, templateStr, data, opts)
	if err != nil {
		return err
	}
	_, err = doc.Write(w)
	if err != nil {
		return fmt.Errorf("tmpl: write pdf: %w", err)
	}
	return nil
}

// RenderFile reads a template from disk, executes it, and writes the PDF to
// outPath. Relative image and stylesheet references resolve against the
// template's directory unless HTMLOptions overrides those base directories.
func RenderFile(ctx context.Context, templatePath string, data any, opts *Options, outPath string) error {
	doc, err := renderFileDocument(ctx, templatePath, data, opts)
	if err != nil {
		return err
	}
	err = doc.Save(outPath)
	if err != nil {
		return fmt.Errorf("tmpl: save %q: %w", outPath, err)
	}
	return nil
}

// RenderFileTo reads a template from disk, executes it, and writes the PDF to
// w. It is useful for HTTP handlers that stream generated PDFs directly.
func RenderFileTo(ctx context.Context, w io.Writer, templatePath string, data any, opts *Options) error {
	doc, err := renderFileDocument(ctx, templatePath, data, opts)
	if err != nil {
		return err
	}
	_, err = doc.Write(w)
	if err != nil {
		return fmt.Errorf("tmpl: write pdf: %w", err)
	}
	return nil
}

func renderFileDocument(ctx context.Context, templatePath string, data any, opts *Options) (*core.Pdf, error) {
	tmplBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("tmpl: read template %q: %w", templatePath, err)
	}
	htmlStr, err := execute(string(tmplBytes), filepath.Base(templatePath), data, opts)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(templatePath)
	return renderHTML(ctx, htmlStr, opts, []paperhtml.Option{
		paperhtml.WithImageBaseDir(baseDir),
		paperhtml.WithStylesheetBaseDir(baseDir),
	})
}

func renderHTML(
	ctx context.Context,
	htmlStr string,
	opts *Options,
	baseOptions []paperhtml.Option,
) (*core.Pdf, error) {
	doc, err := paper.FromHTMLWithOptions(ctx, htmlStr, htmlOptions(opts, baseOptions...), documentConfig(opts)...)
	if err != nil {
		return nil, fmt.Errorf("tmpl: html conversion failed: %w", err)
	}
	return doc, nil
}

func execute(templateStr, name string, data any, opts *Options) (string, error) {
	name = cmp.Or(name, "paper")

	t, err := newTemplate(name, opts)
	if err != nil {
		return "", err
	}
	_, err = t.Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("tmpl: parse template %q: %w", name, err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("tmpl: execute template %q: %w", name, err)
	}
	return buf.String(), nil
}

func newTemplate(name string, opts *Options) (*htmltpl.Template, error) {
	if opts != nil && opts.BaseTemplate != nil {
		clone, err := opts.BaseTemplate.Clone()
		if err != nil {
			return nil, fmt.Errorf("tmpl: clone base template: %w", err)
		}
		registerFuncs(clone, opts)
		return clone.New(name), nil
	}

	t := htmltpl.New(name)
	registerFuncs(t, opts)
	return t, nil
}

func registerFuncs(t *htmltpl.Template, opts *Options) {
	t.Funcs(defaultFuncs())
	if opts != nil && opts.Funcs != nil {
		t.Funcs(opts.Funcs)
	}
}

func htmlOptions(opts *Options, baseOptions ...paperhtml.Option) []paperhtml.Option {
	out := make([]paperhtml.Option, 0, len(baseOptions)+len(optionsHTML(opts)))
	out = append(out, baseOptions...)
	out = append(out, optionsHTML(opts)...)
	return out
}

func optionsHTML(opts *Options) []paperhtml.Option {
	if opts == nil || len(opts.HTMLOptions) == 0 {
		return nil
	}
	out := make([]paperhtml.Option, len(opts.HTMLOptions))
	copy(out, opts.HTMLOptions)
	return out
}

func documentConfig(opts *Options) []*entity.Config {
	if opts == nil || opts.Config == nil {
		return nil
	}
	return []*entity.Config{opts.Config}
}

func defaultFuncs() htmltpl.FuncMap {
	return htmltpl.FuncMap{
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("%w: %d", errDictOddArguments, len(pairs))
			}
			m := make(map[string]any, len(pairs)/2)
			for i := 0; i+1 < len(pairs); i += 2 {
				key, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("%w: position %d has %T", errDictKeyNotString, i, pairs[i])
				}
				m[key] = pairs[i+1]
			}
			return m, nil
		},
	}
}
