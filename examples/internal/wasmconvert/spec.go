package wasmconvert

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// ErrEmptySpec is returned by SpecToBase64 when the supplied spec is empty or
// only whitespace.
var ErrEmptySpec = errors.New("wasmconvert: spec is empty")

// errSpecPanic wraps a recovered panic from the render tree.
var errSpecPanic = errors.New("wasmconvert: spec render panicked")

// Spec is the JSON layout document accepted by SpecToBase64. It mirrors the
// schema used by the Paper Playground's component-grid editor.
type Spec struct {
	Rows []SpecRow `json:"rows"`
}

// SpecRow is one row of the 12-column grid. Gap and Mt are layout hints from the
// editor; they are accepted but not applied by Paper's grid engine.
type SpecRow struct {
	Cols []SpecComp `json:"cols"`
	Gap  *float64   `json:"gap,omitempty"`
	Mt   *float64   `json:"mt,omitempty"`
}

// SpecComp is one cell: a component plus its column span and presentation hints.
type SpecComp struct {
	Type     string     `json:"type"`
	Style    string     `json:"style,omitempty"`
	Align    string     `json:"align,omitempty"`
	Value    string     `json:"value,omitempty"`
	Label    string     `json:"label,omitempty"`
	Span     int        `json:"span,omitempty"`
	Checked  bool       `json:"checked,omitempty"`
	Soft     bool       `json:"soft,omitempty"`
	Size     float64    `json:"size,omitempty"`
	Width    float64    `json:"width,omitempty"`
	Height   float64    `json:"height,omitempty"`
	Head     []string   `json:"head,omitempty"`
	Rows     [][]string `json:"rows,omitempty"`
	ColAlign []string   `json:"colAlign,omitempty"`
}

// rowHeightForCodes is the explicit height (mm) used for rows that contain a
// code or signature. Auto-height rows can collapse such components to zero, so
// a fixed cell gives them stable space and room for an optional caption.
const rowHeightForCodes = 30.0

// SpecToBase64 renders a JSON layout spec into a PDF and returns it as base64.
// It returns ErrEmptySpec for empty input, a wrapped error for invalid JSON or
// generation failures, and recovers from any panic in the render tree so the
// caller (and the wasm goroutine) always survives.
func SpecToBase64(ctx context.Context, jsonSpec, pageSize string) (string, error) {
	var b64 string
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				b64 = ""
				err = fmt.Errorf("%w: %v", errSpecPanic, r)
			}
		}()
		b64, err = renderSpec(ctx, jsonSpec, pageSize)
	}()

	return b64, err
}

func renderSpec(ctx context.Context, jsonSpec, pageSize string) (string, error) {
	if strings.TrimSpace(jsonSpec) == "" {
		return "", ErrEmptySpec
	}

	spec, err := parseSpec(jsonSpec)
	if err != nil {
		return "", err
	}

	doc := paper.New(configForPageSize(pageSize))
	doc.AddRows(buildRows(spec)...)

	pdf, err := doc.Generate(ctx)
	if err != nil {
		return "", fmt.Errorf("wasmconvert: generate pdf: %w", err)
	}

	return pdf.GetBase64(), nil
}

func configForPageSize(pageSize string) *entity.Config {
	size := pagesize.A4
	if strings.EqualFold(pageSize, "Letter") {
		size = pagesize.Letter
	}
	return config.NewBuilder().WithPageSize(size).Build()
}

// buildRows converts each spec row into a Paper row. Rows containing a code or
// signature component get an explicit height; the rest auto-size.
func buildRows(spec Spec) []core.Row {
	rows := make([]core.Row, 0, len(spec.Rows))
	for _, r := range spec.Rows {
		cols := make([]core.Col, 0, len(r.Cols))
		needsHeight := false
		for _, c := range r.Cols {
			cols = append(cols, col.New(clampSpan(c.Span)).Add(buildComponent(c)...))
			if isSizedComponent(c.Type) {
				needsHeight = true
			}
		}
		if needsHeight {
			rows = append(rows, row.New(rowHeightForCodes).Add(cols...))
		} else {
			rows = append(rows, row.New().Add(cols...))
		}
	}
	return rows
}

func clampSpan(span int) int {
	if span < 1 {
		return 12
	}
	if span > 12 {
		return 12
	}
	return span
}

func isSizedComponent(t string) bool {
	switch t {
	case "qrcode", "barcode", "signature":
		return true
	default:
		return false
	}
}
