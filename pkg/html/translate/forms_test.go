package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func TestTranslate_FormControlsRenderVisualRows(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><body>
		<input type="text" value="Hello">
		<input type="password" value="secret">
		<input type="checkbox" checked="checked">
		<input type="radio">
		<input type="submit">
		<button>Save</button>
		<select><option>Draft</option><option selected="selected">Final</option></select>
		<textarea>Notes</textarea>
	</body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"Hello",
		"••••••",
		"☑",
		"○",
		"Submit",
		"Save",
		"Final ▾",
		"Notes",
	}, richTextValues(rows))
}

func TestTranslate_HiddenInputsAreSkippedAndPlaceholdersRender(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><body>
		<input type="hidden" value="secret">
		<input placeholder="Email">
		<textarea placeholder="Message"></textarea>
	</body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)

	assert.Equal(t, []string{"Email", "Message"}, richTextValues(rows))
}

func TestTranslate_FormControlsRenderInsideParagraphs(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><body><p>Name <input value="Ada"> <button>Save</button></p></body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 3)

	assert.Equal(t, []string{"Name Ada Save"}, richTextValues(rows))
}

func TestTranslate_FormAndLabelRenderChildControls(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><body><form><label>Name <input value="Ada"></label> <button>Go</button></form></body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, []string{"Name Ada Go"}, richTextValues(rows))
}

func TestTranslate_FieldsetRendersBorderedContainerWithLegend(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><body><fieldset>
		<legend>Personal Info</legend>
		<p>Name: <input value="John"></p>
	</fieldset></body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	assertDefaultFieldsetMargins(t, rows)

	fieldset := requireFieldsetContainer(t, rows)
	container := fieldset.container
	require.NotNil(t, container)
	require.NotNil(t, container.style)
	assert.Equal(t, consts.LineStyleSolid, container.style.BorderTopStyle)
	assert.InDelta(t, 0.75*0.352778, container.style.BorderTopThickness, 0.001)
	assert.InDelta(t, 3*0.352778, container.style.BorderRadius, 0.001)
	assert.InDelta(t, (8+0.75)*0.352778, container.paddingLeft, 0.001)
	assert.Equal(t, []string{"Personal Info", "Name: John"}, richTextValues(rows))
}

func TestTranslate_FieldsetLegendDefaultsMatchFolio(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><body><fieldset><legend>Personal Info</legend></fieldset></body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	assertDefaultFieldsetMargins(t, rows)
	setBlockTagTestConfig(rows)

	provider := &blockTagRecordingProvider{}
	for _, r := range rows {
		r.Render(provider, entity.Cell{Width: 100, Height: 100})
	}

	legend := requireTextCall(t, provider.texts, "Personal Info")
	assert.Equal(t, fontstyle.Bold, legend.prop.Style)
	assert.InDelta(t, 4*0.352778, legend.prop.Bottom, 0.001)
}

func TestTranslate_FieldsetLegendCSSOverridesDefaults(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><head><style>
		legend { font-weight: normal; padding-bottom: 0 }
	</style></head><body><fieldset><legend>Login</legend></fieldset></body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	setBlockTagTestConfig(rows)

	provider := &blockTagRecordingProvider{}
	for _, r := range rows {
		r.Render(provider, entity.Cell{Width: 100, Height: 100})
	}

	legend := requireTextCall(t, provider.texts, "Login")
	assert.Equal(t, fontstyle.Normal, legend.prop.Style)
	assert.Equal(t, 0.0, legend.prop.Bottom)
}

func TestTranslate_FieldsetCSSOverridesDefaultChrome(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><head><style>
		fieldset { border: 2mm solid #ff0000; border-radius: 4mm; padding: 5mm; }
	</style></head><body><fieldset><legend>Login</legend></fieldset></body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)

	fieldset := requireFieldsetContainer(t, rows)
	container := fieldset.container
	require.NotNil(t, container)
	require.NotNil(t, container.style)
	assert.InDelta(t, 2.0, container.style.BorderTopThickness, 0.001)
	assert.InDelta(t, 4.0, container.style.BorderRadiusTopLeft, 0.001)
	assert.InDelta(t, 7.0, container.paddingLeft, 0.001)
	require.NotNil(t, container.style.BorderTopColor)
	assert.Equal(t, 255, container.style.BorderTopColor.Red)
}

func TestTranslate_SelectUsesSelectedOptionInsideOptgroup(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><body><select>
		<optgroup label="Primary">
			<option>One</option>
			<option selected="selected">Two</option>
		</optgroup>
	</select></body></html>`)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)

	assert.Equal(t, []string{"Two ▾"}, richTextValues(rows))
}

func TestFormControlRuns_UsesPlaceholderPseudoStyle(t *testing.T) {
	t.Parallel()

	doc := parseInternalDoc(t, `<html><body><input placeholder="Email"></body></html>`)
	input := firstTestNodeByTag(t, doc, "input")
	sheet := parseStylesheet(`input::placeholder { color: #ff0000; font-style: italic }`)
	tr := &translator{sheet: sheet}
	style := computeNodeStyle(sheet, input, css.NewComputedStyle())
	ctx := tr.styledRunContext(style)

	runs, handled := formControlRuns(input, ctx)
	require.True(t, handled)
	require.Len(t, runs, 1)

	assert.Equal(t, "Email", runs[0].Text)
	require.NotNil(t, runs[0].Color)
	assert.Equal(t, 255, runs[0].Color.Red)
	assert.Equal(t, 0, runs[0].Color.Green)
	assert.Equal(t, 0, runs[0].Color.Blue)
	assert.Equal(t, fontstyle.Italic, runs[0].Style)
}

func firstTestNodeByTag(t *testing.T, doc *dom.Document, tag string) *dom.Node {
	t.Helper()
	var found *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == tag {
			found = n
			return false
		}
		return true
	})
	require.NotNil(t, found)
	return found
}

func requireFieldsetContainer(t *testing.T, rows []core.Row) *splittableContainerRow {
	t.Helper()
	for _, r := range rows {
		if fieldset, ok := r.(*splittableContainerRow); ok {
			return fieldset
		}
	}
	require.FailNow(t, "expected fieldset container row")
	return nil
}

func assertDefaultFieldsetMargins(t *testing.T, rows []core.Row) {
	t.Helper()
	wantMargin := 9 * 0.352778
	var marginSpacers int
	for _, r := range rows {
		if spacer, ok := r.(marginSpacerRow); ok {
			marginSpacers++
			assert.InDelta(t, wantMargin, spacer.height, 0.001)
		}
	}
	assert.Equal(t, 2, marginSpacers)
}

func requireTextCall(t *testing.T, calls []blockTagTextCall, text string) blockTagTextCall {
	t.Helper()
	for _, call := range calls {
		if call.text == text {
			return call
		}
	}
	require.FailNow(t, "expected text call %q", text)
	return blockTagTextCall{}
}
