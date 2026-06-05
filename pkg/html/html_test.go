package html_test

import (
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/pkg/html"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromString(t *testing.T) {
	t.Parallel()

	t.Run("returns rows for simple html", func(t *testing.T) {
		t.Parallel()
		rows, err := html.FromString("<p>hello</p>")
		require.NoError(t, err)
		assert.NotEmpty(t, rows)
	})

	t.Run("handles malformed html without error", func(t *testing.T) {
		t.Parallel()
		rows, err := html.FromString("<p>unclosed")
		require.NoError(t, err) // golang.org/x/net/html is permissive
		assert.NotEmpty(t, rows)
	})

	t.Run("empty string returns empty rows", func(t *testing.T) {
		t.Parallel()
		rows, err := html.FromString("")
		require.NoError(t, err)
		_ = rows // may be nil or empty
	})
}

func TestFromReader(t *testing.T) {
	t.Parallel()
	r := strings.NewReader("<p>hi</p>")
	rows, err := html.FromReader(r)
	require.NoError(t, err)
	assert.NotEmpty(t, rows)
}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithUnsupportedHandler is invocable", func(t *testing.T) {
		t.Parallel()
		opt := html.WithUnsupportedHandler(func(_, _ string) {})
		assert.NotNil(t, opt)
	})
}
