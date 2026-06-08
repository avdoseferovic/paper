package translate

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
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
	inlineCSS, _ := doc.StyleSources()
	tr := &translator{sheet: parseStylesheet(string(inlineCSS))}
	style := computeNodeStyle(tr.sheet, target, nil)
	return tr.inlineRunsStyled(target, blockInlineStyle(style))
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

func TestTypography_TextShadow_PropagatesMultipleShadows(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<p style="text-shadow:1mm 1mm red, 2mm 2mm blue">shadowed</p>`)
	require.NotEmpty(t, runs)
	require.Len(t, runs[0].TextShadows, 2)
	require.NotNil(t, runs[0].TextShadow)
	assert.Equal(t, runs[0].TextShadows[0], *runs[0].TextShadow)
	assert.InDelta(t, 2.0, runs[0].TextShadows[1].OffsetX, 0.001)
}

func TestTypography_DisplayNoneInlineElementSkipped(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<style>.hidden{display:none}</style><p>A<span class="hidden">hidden</span>B<span hidden>gone</span></p>`)
	require.NotEmpty(t, runs)

	var text string
	for _, run := range runs {
		text += run.Text
	}
	assert.Equal(t, "AB", text)
}

func TestTypography_VisibilityHiddenMarksRunsWithoutDroppingLayout(t *testing.T) {
	t.Parallel()

	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(minimalPNG(t))
	runs := runsFromHTML(t, `<p>A<span style="visibility:hidden">hidden<img src="`+uri+`" width="2mm" height="2mm" alt="icon"></span><span style="visibility:visible">shown</span></p>`)
	require.NotEmpty(t, runs)

	var hiddenText, hiddenImage, visibleText bool
	for _, run := range runs {
		switch {
		case run.Text == "hidden":
			hiddenText = true
			assert.True(t, run.Hidden, "visibility:hidden text should preserve the run but skip painting")
		case run.Image != nil:
			hiddenImage = true
			assert.True(t, run.Hidden, "visibility:hidden inline image should preserve dimensions but skip painting")
			assert.InDelta(t, 2.0, run.Image.Width, 0.001)
		case run.Text == "shown":
			visibleText = true
			assert.False(t, run.Hidden)
		}
	}
	assert.True(t, hiddenText, "expected hidden text run")
	assert.True(t, hiddenImage, "expected hidden inline image run")
	assert.True(t, visibleText, "expected visible override run")
}

func TestTypography_VisibilityInheritedAndOverridable(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><div style="visibility:hidden"><p>secret</p><p style="visibility:visible">shown</p></div></body></html>`)
	require.NoError(t, err)

	div := findFirstNode(t, doc, "div")
	var paragraphs []*dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			paragraphs = append(paragraphs, n)
		}
		return true
	})
	require.Len(t, paragraphs, 2)

	divStyle := computeNodeStyle(nil, div, nil)
	hiddenStyle := computeNodeStyle(nil, paragraphs[0], divStyle)
	visibleStyle := computeNodeStyle(nil, paragraphs[1], divStyle)
	assert.Equal(t, "hidden", hiddenStyle.Visibility)
	assert.Equal(t, "visible", visibleStyle.Visibility)

	tr := &translator{}
	hiddenRuns := tr.inlineRunsStyled(paragraphs[0], blockInlineStyle(hiddenStyle))
	visibleRuns := tr.inlineRunsStyled(paragraphs[1], blockInlineStyle(visibleStyle))
	require.NotEmpty(t, hiddenRuns)
	require.NotEmpty(t, visibleRuns)
	assert.True(t, hiddenRuns[0].Hidden)
	assert.False(t, visibleRuns[0].Hidden)
}

func TestTypography_InlineCSSVerticalAlignMappedToRuns(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<p>H<span style="vertical-align:sub;font-size:9pt">2</span>O x<span style="vertical-align:super;font-size:9pt">2</span></p>`)
	require.NotEmpty(t, runs)

	var foundSub, foundSuper bool
	for _, run := range runs {
		switch run.Text {
		case "2":
			switch run.VerticalAlign {
			case "sub":
				assert.InDelta(t, 9.0, run.Size, 0.01)
				foundSub = true
			case "super":
				assert.InDelta(t, 9.0, run.Size, 0.01)
				foundSuper = true
			}
		}
	}
	assert.True(t, foundSub, "expected CSS vertical-align:sub run")
	assert.True(t, foundSuper, "expected CSS vertical-align:super run")
}

func TestTypography_StylesheetInlineSelectorMappedToRuns(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<style>.chem{vertical-align:sub}.power{vertical-align:super}</style><p>H<span class="chem">2</span>O x<span class="power">2</span></p>`)
	require.NotEmpty(t, runs)

	var foundSub, foundSuper bool
	for _, run := range runs {
		if run.Text != "2" {
			continue
		}
		foundSub = foundSub || run.VerticalAlign == "sub"
		foundSuper = foundSuper || run.VerticalAlign == "super"
	}
	assert.True(t, foundSub, "expected stylesheet subscript run")
	assert.True(t, foundSuper, "expected stylesheet superscript run")
}

func TestTypography_InlineCSSMappedToRichRun(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<p><span style="font-family:'Courier New', monospace;font-weight:bold;font-style:italic;text-decoration:underline line-through;background-color:#eee;color:red">styled</span></p>`)
	require.Len(t, runs, 1)

	run := runs[0]
	assert.Equal(t, "Courier New", run.Family)
	assert.Equal(t, fontstyle.BoldItalic, run.Style)
	assert.True(t, run.Underline)
	assert.True(t, run.Strikethrough)
	require.NotNil(t, run.Background)
	require.NotNil(t, run.Color)
	assert.Equal(t, 255, run.Color.Red)
}

func TestTypography_PseudoElementsGenerateStyledContent(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<style>
p::before { content:"Note: "; font-weight:bold; color:red }
p::after { content:"." }
</style><p>hello</p>`)
	require.Len(t, runs, 3)

	assert.Equal(t, "Note: ", runs[0].Text)
	assert.Equal(t, fontstyle.Bold, runs[0].Style)
	require.NotNil(t, runs[0].Color)
	assert.Equal(t, 255, runs[0].Color.Red)
	assert.Equal(t, "hello", runs[1].Text)
	assert.Equal(t, ".", runs[2].Text)
}

func TestTypography_PseudoElementContentSupportsAttr(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<style>
.field::after { content:" (" attr(data-unit) ")" }
</style><p><span class="field" data-unit="kg">Weight</span></p>`)
	require.NotEmpty(t, runs)

	var text string
	for _, run := range runs {
		text += run.Text
	}
	assert.Equal(t, "Weight (kg)", text)
}

func TestTypography_PseudoElementContentSupportsURLImage(t *testing.T) {
	t.Parallel()

	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(minimalPNG(t))
	runs := runsFromHTML(t, `<style>
p::before { content:url("`+uri+`") " "; width:2mm; height:3mm }
</style><p>Label</p>`)
	require.Len(t, runs, 3)
	require.NotNil(t, runs[0].Image)
	assert.Equal(t, extension.Png, runs[0].Image.Extension)
	assert.InDelta(t, 2.0, runs[0].Image.Width, 0.001)
	assert.InDelta(t, 3.0, runs[0].Image.Height, 0.001)
	assert.Equal(t, " ", runs[1].Text)
	assert.Equal(t, "Label", runs[2].Text)
}

func TestTypography_PseudoElementContentSupportsURLSVG(t *testing.T) {
	t.Parallel()

	uri := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(minimalSVG))
	runs := runsFromHTML(t, `<style>
p::before { content:url("`+uri+`") " "; width:3mm; height:3mm }
</style><p>Vector</p>`)
	require.Len(t, runs, 3)
	require.NotNil(t, runs[0].Image)
	assert.Equal(t, extension.Png, runs[0].Image.Extension)
	assert.InDelta(t, 3.0, runs[0].Image.Width, 0.001)
	assert.InDelta(t, 3.0, runs[0].Image.Height, 0.001)
	assert.Equal(t, "Vector", runs[2].Text)
}

func TestTypography_PseudoElementContentSupportsQuoteTokens(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<style>
p { quotes:"<<" ">>" "<" ">" }
.quote::before { content:open-quote }
.quote::after { content:close-quote }
</style><p><span class="quote">Outer <span class="quote">Inner</span></span></p>`)
	require.NotEmpty(t, runs)

	var text string
	for _, run := range runs {
		text += run.Text
	}
	assert.Equal(t, "<<Outer <Inner>>>", text)
}

func TestTypography_QElementUsesCSSQuotes(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<style>
p { quotes:"<<" ">>" "<" ">" }
</style><p><q>Outer <q>Inner</q></q></p>`)
	require.NotEmpty(t, runs)

	var text string
	for _, run := range runs {
		text += run.Text
	}
	assert.Equal(t, "<<Outer <Inner>>>", text)
}

func TestTypography_PseudoElementContentSupportsCounter(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><head><style>
body { counter-reset: section }
h2 { counter-increment: section }
h2::before { content:"Section " counter(section) ": " }
</style></head><body><h2>Intro</h2><h2>Usage</h2></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)

	assert.Equal(t, []string{"Section 1: Intro", "Section 2: Usage"}, richTextValues(rows))
}

func TestTypography_PseudoElementCanIncrementCounter(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><head><style>
body { counter-reset: note }
p::before { counter-increment: note; content: counter(note, decimal-leading-zero) ". " }
</style></head><body><p>First</p><p>Second</p></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)

	assert.Equal(t, []string{"01. First", "02. Second"}, richTextValues(rows))
}

func TestTypography_CountersAreScopedToResetElement(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><head><style>
body { counter-reset: section }
section { counter-increment: section; counter-reset: item }
h2::before { content: counter(section, upper-roman) ". " }
p.item { counter-increment: item }
p.item::before { content: counter(section) "." counter(item) " " }
</style></head><body>
<section><h2>First</h2><p class="item">One</p><p class="item">Two</p></section>
<section><h2>Second</h2><p class="item">One</p></section>
</body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"I. First",
		"1.1 One",
		"1.2 Two",
		"II. Second",
		"2.1 One",
	}, richTextValues(rows))
}

func TestTypography_PseudoElementSelectorSpecificity(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<style>
p::before { content:"base " }
p.note::before { content:"specific " }
</style><p class="note">body</p>`)
	require.NotEmpty(t, runs)

	assert.Equal(t, "specific ", runs[0].Text)
}

func richTextValues(rows []core.Row) []string {
	var values []string
	for _, r := range rows {
		walkStructure(r.GetStructure(), func(s core.Structure) {
			if s.Type == "richtext" {
				if value, ok := s.Value.(string); ok {
					values = append(values, value)
				}
			}
		})
	}
	return values
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

func TestTypography_PreDefaultsPreserveWhitespaceAndUseMonospace(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><pre>one
  two</pre></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	var details map[string]any
	var value string
	walkStructure(rows[0].GetStructure(), func(s core.Structure) {
		if s.Type == "richtext" {
			details = s.Details
			value, _ = s.Value.(string)
		}
	})
	require.NotNil(t, details)
	assert.Equal(t, "pre", details["white_space"])
	assert.Equal(t, "one\n  two", value)

	tr, _ := parseTranslator(t, `<html><body><pre>one
  two</pre></body></html>`)
	pre := findNode(doc, "pre")
	require.NotNil(t, pre)
	style := computeNodeStyle(tr.sheet, pre, nil)
	runs := tr.inlineRunsStyled(pre, blockInlineStyle(style))
	require.NotEmpty(t, runs)
	assert.Equal(t, "courier", runs[0].Family)
}

func TestTypography_TextAlignJustifyMappedFromCSS(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><p style="text-align:justify">one two three</p></body></html>`)
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
	assert.Equal(t, align.Type(align.Justify), details["align"])
}
