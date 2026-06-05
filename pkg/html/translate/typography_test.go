package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

func runsFromHTML(t *testing.T, htmlStr string) []props.RichRun {
	t.Helper()
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	// Find the first block element of interest and inline its runs.
	var target *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" || n.Tag() == "span" {
			if target == nil {
				target = n
			}
		}
		return true
	})
	require.NotNil(t, target, "expected to find a <p> or <span>")
	style := computeNodeStyle(nil, target, nil)
	for prop, val := range parseInlineStyle(target.InlineStyle()) {
		style.Apply(prop, val, nil)
	}
	runs := inlineRuns(target)
	applyInlineStyleToRuns(style, runs)
	return runs
}

func TestTypography_TextTransform_Uppercase(t *testing.T) {
	t.Parallel()
	runs := runsFromHTML(t, `<p style="text-transform:uppercase">hello world</p>`)
	require.NotEmpty(t, runs)
	assert.Equal(t, "HELLO WORLD", runs[0].Text)
}

func TestTypography_TextTransform_Lowercase(t *testing.T) {
	t.Parallel()
	runs := runsFromHTML(t, `<p style="text-transform:lowercase">HELLO World</p>`)
	require.NotEmpty(t, runs)
	assert.Equal(t, "hello world", runs[0].Text)
}

func TestTypography_TextTransform_Capitalize(t *testing.T) {
	t.Parallel()
	runs := runsFromHTML(t, `<p style="text-transform:capitalize">hello world from go</p>`)
	require.NotEmpty(t, runs)
	assert.Equal(t, "Hello World From Go", runs[0].Text)
}

func TestTypography_LetterSpacing_Propagated(t *testing.T) {
	t.Parallel()
	// 0.5pt = 0.176389mm
	runs := runsFromHTML(t, `<p style="letter-spacing:0.5pt">spaced</p>`)
	require.NotEmpty(t, runs)
	assert.InDelta(t, 0.176389, runs[0].LetterSpacing, 0.001)
}

func TestTypography_TextIndent_Stored(t *testing.T) {
	t.Parallel()
	// text-indent value is stored on ComputedStyle (validated via the
	// translateToRows path applying it as paragraph Left).
	doc, err := dom.Parse(`<p style="text-indent:5mm">indented</p>`)
	require.NoError(t, err)
	var p *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			p = n
		}
		return true
	})
	require.NotNil(t, p)
	style := css.NewComputedStyle()
	for prop, val := range parseInlineStyle(p.InlineStyle()) {
		style.Apply(prop, val, nil)
	}
	assert.InDelta(t, 5.0, style.TextIndent, 0.001)
}

func TestTypography_WhiteSpace_Stored(t *testing.T) {
	t.Parallel()
	style := css.NewComputedStyle()
	style.Apply("white-space", "nowrap", nil)
	assert.Equal(t, "nowrap", style.WhiteSpace)
}

func TestTypography_RichTextLayoutPropsMappedFromCSS(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><p style="white-space:pre-line;text-indent:5mm;text-align:right">one
two</p></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	var details map[string]any
	walkStructure(rows[0].GetStructure(), func(s core.Structure) {
		if s.Type == "richtext" {
			details = s.Details
		}
	})
	require.NotNil(t, details)
	assert.Equal(t, "pre-line", details["white_space"])
	assert.Equal(t, 5.0, details["first_line_indent"])
	assert.Equal(t, align.Right, details["align"])
	assert.Zero(t, details["left"], "text-indent should not shift every line through left padding")
}
