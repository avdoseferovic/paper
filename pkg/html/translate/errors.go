package translate

import (
	"errors"
	"fmt"

	"github.com/avdoseferovic/paper/internal/htmllimits"
)

// ErrURLPolicyDenied wraps a URLPolicy rejection so callers can distinguish
// intentional policy blocks from network or filesystem failures.
var ErrURLPolicyDenied = errors.New("html: URL fetch blocked by URLPolicy")

// ParseError indicates the input HTML could not be parsed into a document.
type ParseError struct {
	Err error
}

func (e *ParseError) Error() string {
	if e == nil || e.Err == nil {
		return "html: parse error"
	}
	return "html: parsing input: " + e.Err.Error()
}

func (e *ParseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// AssetError indicates a document-referenced asset failed to load while
// translating HTML with strict asset checking enabled.
type AssetError struct {
	Category string
	Ref      string
	Err      error
}

// NewAssetError builds an AssetError for a reference that could not be loaded.
// category names the kind of asset, and ref is the value that failed.
func NewAssetError(category, ref string, err error) *AssetError {
	return &AssetError{Category: category, Ref: ref, Err: err}
}

func (e *AssetError) Error() string {
	if e == nil {
		return "html: asset error"
	}
	msg := fmt.Sprintf("html: %s asset", e.Category)
	if e.Ref != "" {
		msg += " " + fmt.Sprintf("%q", e.Ref)
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *AssetError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// LimitKind identifies which HTML translation resource ceiling was exceeded.
type LimitKind int

const (
	// LimitElements reports that the configured element/node budget was exceeded.
	LimitElements LimitKind = iota
	// LimitDepth reports that the configured DOM nesting depth was exceeded.
	LimitDepth
	// LimitStyleRules reports that the configured stylesheet rule budget was exceeded.
	LimitStyleRules
)

func (k LimitKind) String() string {
	switch k {
	case LimitElements:
		return "elements"
	case LimitDepth:
		return "depth"
	case LimitStyleRules:
		return "style-rules"
	default:
		return "elements"
	}
}

// LimitError indicates HTML translation stopped because a configured resource
// ceiling was crossed.
type LimitError struct {
	Kind  LimitKind
	Limit int
	Err   error
}

func (e *LimitError) Error() string {
	if e == nil {
		return "html: limit exceeded"
	}
	switch e.Kind {
	case LimitElements:
		return fmt.Sprintf("html: element count exceeded limit of %d", e.Limit)
	case LimitDepth:
		return fmt.Sprintf("html: nesting depth exceeded limit of %d", e.Limit)
	case LimitStyleRules:
		return fmt.Sprintf("html: stylesheet rule count exceeded limit of %d", e.Limit)
	default:
		return fmt.Sprintf("html: element count exceeded limit of %d", e.Limit)
	}
}

func (e *LimitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func limitErrorFrom(err error, limits htmllimits.Limits) error {
	switch {
	case errors.Is(err, htmllimits.ErrDOMTooDeep):
		return &LimitError{Kind: LimitDepth, Limit: limits.MaxDOMDepth, Err: err}
	case errors.Is(err, htmllimits.ErrDOMTooLarge):
		return &LimitError{Kind: LimitElements, Limit: limits.MaxDOMNodes, Err: err}
	case errors.Is(err, htmllimits.ErrStyleRulesTooLarge):
		return &LimitError{Kind: LimitStyleRules, Limit: limits.MaxStyleRules, Err: err}
	default:
		return err
	}
}
