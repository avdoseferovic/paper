package translate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avdoseferovic/paper/v2/pkg/html/dom"
)

func parseInlineRuns(t *testing.T, htmlStr string) []runEntry {
	t.Helper()
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	var target *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			if target == nil {
				target = n
			}
		}
		return true
	})
	require.NotNil(t, target)
	runs := inlineRuns(target)
	out := make([]runEntry, len(runs))
	for i, r := range runs {
		out[i] = runEntry{
			text:        r.Text,
			family:      r.Family,
			style:       string(r.Style),
			underline:   r.Underline,
			bg:          r.Background,
			hasBg:       r.Background != nil,
			hasAnchor:   r.LocalAnchor != "",
			localAnchor: r.LocalAnchor,
		}
	}
	return out
}

type runEntry struct {
	text        string
	family      string
	style       string
	underline   bool
	bg          interface{}
	hasBg       bool
	hasAnchor   bool
	localAnchor string
}

func TestInlineTag_Mark_YellowBackground(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p>before <mark>highlight</mark> after</p>`)
	// Find the "highlight" run.
	var found bool
	for _, r := range runs {
		if strings.TrimSpace(r.text) == "highlight" {
			assert.True(t, r.hasBg, "<mark> run should have background")
			found = true
		}
	}
	assert.True(t, found)
}

func TestInlineTag_Small_NoError(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p>normal <small>tiny</small></p>`)
	assert.NotEmpty(t, runs)
}

func TestInlineTag_Code_MonospaceFamily(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p>see <code>foo()</code></p>`)
	var found bool
	for _, r := range runs {
		if strings.Contains(r.text, "foo") {
			assert.Equal(t, "courier", r.family, "<code> should pick courier")
			assert.True(t, r.hasBg, "<code> should have light bg")
			found = true
		}
	}
	assert.True(t, found)
}

func TestInlineTag_Kbd_BoxedWithBg(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p>press <kbd>Ctrl+C</kbd></p>`)
	var found bool
	for _, r := range runs {
		if strings.Contains(r.text, "Ctrl") {
			assert.Equal(t, "courier", r.family)
			assert.True(t, r.hasBg)
			found = true
		}
	}
	assert.True(t, found)
}

func TestInlineTag_Samp_MonospaceNoBg(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><samp>output</samp></p>`)
	require.NotEmpty(t, runs)
	var found bool
	for _, r := range runs {
		if strings.Contains(r.text, "output") {
			assert.Equal(t, "courier", r.family)
			assert.False(t, r.hasBg, "<samp> should NOT have background")
			found = true
		}
	}
	assert.True(t, found)
}

func TestInlineTag_Var_Italic(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><var>x</var></p>`)
	require.NotEmpty(t, runs)
	for _, r := range runs {
		if r.text == "x" {
			assert.Equal(t, "I", r.style)
		}
	}
}

func TestInlineTag_Cite_Italic(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><cite>The Book</cite></p>`)
	require.NotEmpty(t, runs)
	for _, r := range runs {
		if strings.Contains(r.text, "Book") {
			assert.Equal(t, "I", r.style)
		}
	}
}

func TestInlineTag_Q_AsciiQuotes(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><q>quoted</q></p>`)
	require.NotEmpty(t, runs)
	// First and last runs should be quote chars.
	var concatenated strings.Builder
	for _, r := range runs {
		concatenated.WriteString(r.text)
	}
	full := concatenated.String()
	assert.Contains(t, full, `"quoted"`)
}

func TestInlineTag_Abbr_TitleSurfacedViaHandler(t *testing.T) {
	t.Parallel()
	var calls []string
	doc, err := dom.Parse(`<p><abbr title="HyperText Markup Language">HTML</abbr></p>`)
	require.NoError(t, err)
	var p *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			p = n
		}
		return true
	})
	require.NotNil(t, p)
	_ = inlineRunsWithHandler(p, func(thing, value string) {
		calls = append(calls, thing+":"+value)
	})
	found := false
	for _, c := range calls {
		if c == "abbr.title:HyperText Markup Language" {
			found = true
		}
	}
	assert.True(t, found, "<abbr title> should surface via unsupportedHandler")
}

func TestInlineTag_Time_DatetimeSurfacedViaHandler(t *testing.T) {
	t.Parallel()
	var calls []string
	doc, err := dom.Parse(`<p><time datetime="2026-05-20">May 20</time></p>`)
	require.NoError(t, err)
	var p *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			p = n
		}
		return true
	})
	require.NotNil(t, p)
	_ = inlineRunsWithHandler(p, func(thing, value string) {
		calls = append(calls, thing+":"+value)
	})
	found := false
	for _, c := range calls {
		if c == "time.datetime:2026-05-20" {
			found = true
		}
	}
	assert.True(t, found, "<time datetime> should surface via unsupportedHandler")
}

func TestInlineTag_Abbr_Underline(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><abbr title="HyperText Markup Language">HTML</abbr></p>`)
	require.NotEmpty(t, runs)
	for _, r := range runs {
		if r.text == "HTML" {
			assert.True(t, r.underline, "<abbr> should be underlined")
		}
	}
}

func TestInlineTag_Time_RendersText(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><time datetime="2026-05-20">May 20, 2026</time></p>`)
	require.NotEmpty(t, runs)
	var found bool
	for _, r := range runs {
		if strings.Contains(r.text, "May 20") {
			found = true
		}
	}
	assert.True(t, found)
}

func TestInlineTag_AnchorHrefHash_SetsLocalAnchor(t *testing.T) {
	t.Parallel()
	runs := parseInlineRuns(t, `<p><a href="#section1">jump</a></p>`)
	require.NotEmpty(t, runs)
	for _, r := range runs {
		if r.text == "jump" {
			assert.True(t, r.hasAnchor)
			assert.Equal(t, "section1", r.localAnchor)
		}
	}
}
