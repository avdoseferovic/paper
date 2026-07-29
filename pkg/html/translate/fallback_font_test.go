package translate

import (
	"errors"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func TestLoadFallbackFont_WhenBodyHasNonWinAnsiText_LoadsThroughResolver(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><p>Hello 中</p></body></html>`)
	require.NoError(t, err)
	body := findBody(doc)
	require.NotNil(t, body)

	var requested string
	tr := &translator{fallbackFontPath: "fonts/fallback.ttf"}
	tr.loadFallbackFontIfNeeded(t.Context(), body, func(path string) ([]byte, error) {
		requested = path
		return []byte("font bytes"), nil
	})

	assert.Equal(t, "fonts/fallback.ttf", requested)
	require.NotEmpty(t, tr.loadedFonts)
	assert.Equal(t, fallbackFontFamily, tr.loadedFonts[0].family)
	assert.Equal(t, []byte("font bytes"), tr.loadedFonts[0].bytes)
	assert.True(t, tr.fallbackFontReady)
}

func TestLoadFallbackFont_WhenTextIsWinAnsi_DoesNotLoad(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><p>Hello cafe</p></body></html>`)
	require.NoError(t, err)
	body := findBody(doc)
	require.NotNil(t, body)

	called := false
	tr := &translator{fallbackFontPath: "fonts/fallback.ttf"}
	tr.loadFallbackFontIfNeeded(t.Context(), body, func(string) ([]byte, error) {
		called = true
		return []byte("font bytes"), nil
	})

	assert.False(t, called)
	assert.Empty(t, tr.loadedFonts)
	assert.False(t, tr.fallbackFontReady)
}

func TestLoadFallbackFont_WhenResolverFails_ReportsStrictAssetError(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><p>中</p></body></html>`)
	require.NoError(t, err)
	body := findBody(doc)
	require.NotNil(t, body)

	wantErr := errors.New("missing font")
	tr := &translator{
		fallbackFontPath: "missing.ttf",
		strictAssets:     true,
	}
	tr.loadFallbackFontIfNeeded(t.Context(), body, func(string) ([]byte, error) {
		return nil, wantErr
	})

	assert.Empty(t, tr.loadedFonts)
	assert.False(t, tr.fallbackFontReady)
	require.Error(t, tr.strictAssetError())
	assert.ErrorIs(t, tr.strictAssetError(), wantErr)
}

func TestFallbackFontSplitsMixedWinAnsiAndNonWinAnsiRuns(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<p>Hello 中 world</p>`)
	require.NoError(t, err)
	var target *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			target = n
			return false
		}
		return true
	})
	require.NotNil(t, target)

	tr := &translator{fallbackFontReady: true}
	runs := tr.inlineRunsStyled(target, nil)

	require.Len(t, runs, 3)
	assert.Equal(t, "Hello ", runs[0].Text)
	assert.Equal(t, "", runs[0].Family)
	assert.Equal(t, "中", runs[1].Text)
	assert.Equal(t, fallbackFontFamily, runs[1].Family)
	assert.Equal(t, " world", runs[2].Text)
	assert.Equal(t, "", runs[2].Family)
}

func TestFallbackFontDoesNotOverrideAuthorFontFamily(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<p style='font-family:"Custom"'>中</p>`)
	require.NoError(t, err)
	var target *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			target = n
			return false
		}
		return true
	})
	require.NotNil(t, target)

	tr := &translator{fallbackFontReady: true}
	style := computeNodeStyle(nil, target, nil)
	runs := tr.inlineRunsStyled(target, blockInlineStyle(style))

	require.Len(t, runs, 1)
	assert.Equal(t, "中", runs[0].Text)
	assert.Equal(t, "Custom", runs[0].Family)
}

func TestCanEncodeWinAnsiRune(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		rune rune
		want bool
	}{
		{rune: 'A', want: true},
		{rune: 'é', want: true},
		{rune: '€', want: true},
		{rune: 'Œ', want: true},
		{rune: '中', want: false},
		{rune: '😀', want: false},
	} {
		if got := canEncodeWinAnsiRune(tt.rune); got != tt.want {
			t.Errorf("canEncodeWinAnsiRune(%q) = %t, want %t", tt.rune, got, tt.want)
		}
	}
}
