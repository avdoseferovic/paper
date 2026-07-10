package html_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html"
)

func TestLimitErrorFromMaxElements(t *testing.T) {
	t.Parallel()

	_, err := html.FromString(context.Background(), repeatedSpans(20), html.WithMaxElements(10))

	var le *html.LimitError
	require.True(t, errors.As(err, &le), "expected *html.LimitError")
	assert.Equal(t, html.LimitElements, le.Kind)
	assert.Equal(t, 10, le.Limit)
	require.ErrorIs(t, err, html.ErrDOMTooLarge)
}

func TestLimitErrorFromMaxDepth(t *testing.T) {
	t.Parallel()

	_, err := html.FromString(context.Background(), nestedDivs(20), html.WithMaxDepth(5))

	var le *html.LimitError
	require.True(t, errors.As(err, &le), "expected *html.LimitError")
	assert.Equal(t, html.LimitDepth, le.Kind)
	assert.Equal(t, 5, le.Limit)
	require.ErrorIs(t, err, html.ErrDOMTooDeep)
}

func TestLimitErrorFromStyleRuleLimit(t *testing.T) {
	t.Parallel()

	_, err := html.FromString(context.Background(), manyStyleRules(20), html.WithLimits(html.Limits{MaxStyleRules: 5}))

	var le *html.LimitError
	require.True(t, errors.As(err, &le), "expected *html.LimitError")
	assert.Equal(t, html.LimitStyleRules, le.Kind)
	assert.Equal(t, 5, le.Limit)
	require.ErrorIs(t, err, html.ErrStyleRulesTooLarge)
}

func TestLimitErrorKindString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "elements", html.LimitElements.String())
	assert.Equal(t, "depth", html.LimitDepth.String())
	assert.Equal(t, "style-rules", html.LimitStyleRules.String())
}

func repeatedSpans(n int) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	for range n {
		b.WriteString("<span>x</span>")
	}
	b.WriteString("</body></html>")
	return b.String()
}

func nestedDivs(depth int) string {
	return "<html><body>" + strings.Repeat("<div>", depth) + "x" + strings.Repeat("</div>", depth) + "</body></html>"
}

func manyStyleRules(n int) string {
	var b strings.Builder
	b.WriteString("<html><head><style>")
	for i := range n {
		b.WriteString(".x")
		b.WriteString(string(rune('a' + i)))
		b.WriteString("{color:red}")
	}
	b.WriteString("</style></head><body><p>ok</p></body></html>")
	return b.String()
}
