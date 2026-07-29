package html_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html"
)

func TestWithStrictAssets_DefaultFalsePreservesWarnAndContinue(t *testing.T) {
	t.Parallel()

	rows, err := html.FromString(t.Context(),
		`<html><body><img src="missing.png" alt="fallback"><p>ok</p></body></html>`,
		html.WithImageBaseDir(t.TempDir()),
	)

	require.NoError(t, err)
	assert.NotEmpty(t, rows)
}

func TestWithStrictAssets_BadStylesheetReturnsAssetError(t *testing.T) {
	t.Parallel()

	rows, err := html.FromString(t.Context(),
		`<html><head><link rel="stylesheet" href="missing.css"></head><body><p>ok</p></body></html>`,
		html.WithStylesheetBaseDir(t.TempDir()),
		html.WithStrictAssets(),
	)

	require.Error(t, err)
	assert.NotEmpty(t, rows)
	assertAssetError(t, err, "stylesheet", "missing.css")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestWithStrictAssets_BadFontFaceReturnsAssetError(t *testing.T) {
	t.Parallel()

	rows, err := html.FromString(t.Context(),
		`<html><head><style>@font-face{font-family:"Missing";src:url("missing.ttf") format("truetype")}</style></head><body><p>ok</p></body></html>`,
		html.WithStylesheetBaseDir(t.TempDir()),
		html.WithStrictAssets(),
	)

	require.Error(t, err)
	assert.NotEmpty(t, rows)
	assertAssetError(t, err, "@font-face", "missing.ttf")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestWithStrictAssets_BadImageReturnsAssetError(t *testing.T) {
	t.Parallel()

	rows, err := html.FromString(t.Context(),
		`<html><body><img src="missing.png" alt="fallback"><p>ok</p></body></html>`,
		html.WithImageBaseDir(t.TempDir()),
		html.WithStrictAssets(),
	)

	require.Error(t, err)
	assert.NotEmpty(t, rows)
	assertAssetError(t, err, "image", "missing.png")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestWithStrictAssets_BadBackgroundImageReturnsAssetError(t *testing.T) {
	t.Parallel()

	rows, err := html.FromString(t.Context(),
		`<html><body><div style="background-image:url('missing-bg.png')">ok</div></body></html>`,
		html.WithImageBaseDir(t.TempDir()),
		html.WithStrictAssets(),
	)

	require.Error(t, err)
	assert.NotEmpty(t, rows)
	assertAssetError(t, err, "background-image", "missing-bg.png")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestWithStrictAssets_CollectsMultipleAssetErrors(t *testing.T) {
	t.Parallel()

	_, err := html.FromString(t.Context(),
		`<html><head><link rel="stylesheet" href="missing.css"></head><body><img src="missing.png" alt="fallback"></body></html>`,
		html.WithStylesheetBaseDir(t.TempDir()),
		html.WithImageBaseDir(t.TempDir()),
		html.WithStrictAssets(),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing.css")
	assert.Contains(t, err.Error(), "missing.png")
	assertAssetError(t, err, "stylesheet", "missing.css")
	assertAssetError(t, err, "image", "missing.png")
}

func assertAssetError(t *testing.T, err error, category, ref string) {
	t.Helper()

	ae, ok := findAssetError(err, category, ref)
	require.True(t, ok, "expected *html.AssetError")
	assert.Equal(t, category, ae.Category)
	assert.Equal(t, ref, ae.Ref)
}

func findAssetError(err error, category, ref string) (*html.AssetError, bool) {
	if err == nil {
		return nil, false
	}
	var ae *html.AssetError
	if errors.As(err, &ae) && ae.Category == category && ae.Ref == ref {
		return ae, true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			if ae, ok := findAssetError(child, category, ref); ok {
				return ae, true
			}
		}
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return findAssetError(wrapped.Unwrap(), category, ref)
	}
	return nil, false
}
