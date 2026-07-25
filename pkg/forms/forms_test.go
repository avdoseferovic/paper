package forms_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/forms"
	"github.com/avdoseferovic/paper/pkg/reader"
)

func TestAcroFormTextFieldGeneration(t *testing.T) {
	t.Parallel()

	form := forms.NewAcroForm().
		Add(forms.NewTextField("name", [4]float64{72, 700, 300, 720}, 0).SetValue("John Doe"))

	pdfBytes := generateFormPDF(t, form)

	assert.True(t, bytes.Contains(pdfBytes, []byte("/AcroForm")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/FT /Tx")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Subtype /Widget")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Annots [")))
	assert.True(t, bytes.Contains(pdfBytes, []byte(" 0 R ]")), "page annots should reference widget objects")
	assert.True(t, bytes.Contains(pdfBytes, []byte("John Doe")))
}

func TestAcroFormCheckboxGeneration(t *testing.T) {
	t.Parallel()

	form := forms.NewAcroForm().
		Add(forms.NewCheckbox("agree", [4]float64{72, 680, 92, 700}, 0, true)).
		Add(forms.NewCheckbox("newsletter", [4]float64{72, 650, 92, 670}, 0, false))

	pdfBytes := generateFormPDF(t, form)

	assert.True(t, bytes.Contains(pdfBytes, []byte("/FT /Btn")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/AS /Yes")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/AS /Off")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/AP")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("re S")), "checkbox appearance should draw a border")
}

func TestAcroFormChoiceRadioAndSignatureGeneration(t *testing.T) {
	t.Parallel()

	form := forms.NewAcroForm().
		Add(forms.NewDropdown("country", [4]float64{72, 620, 250, 640}, 0,
			[]string{"USA", "Canada", "Mexico"}).SetValue("Canada")).
		Add(forms.NewListBox("lang", [4]float64{72, 500, 250, 580}, 0,
			[]string{"Go", "Rust"})).
		Add(forms.NewRadioGroup("size", []forms.RadioOption{
			{Value: "S", Rect: [4]float64{72, 460, 92, 480}, PageIndex: 0},
			{Value: "M", Rect: [4]float64{102, 460, 122, 480}, PageIndex: 0},
		})).
		Add(forms.NewSignatureField("sig", [4]float64{72, 100, 250, 150}, 0))

	pdfBytes := generateFormPDF(t, form)
	pdfText := string(pdfBytes)

	assert.True(t, strings.Contains(pdfText, "/FT /Ch"))
	assert.True(t, strings.Contains(pdfText, "/Opt"))
	assert.True(t, strings.Contains(pdfText, "Canada"))
	assert.True(t, strings.Contains(pdfText, "Rust"))
	assert.True(t, strings.Contains(pdfText, "/Ff 131072"), "dropdown should carry combo-box flag")
	assert.True(t, strings.Contains(pdfText, "/Kids"))
	assert.True(t, strings.Contains(pdfText, "/S "))
	assert.True(t, strings.Contains(pdfText, "/M "))
	assert.GreaterOrEqual(t, strings.Count(pdfText, "/AS /Off"), 2)
	assert.True(t, strings.Contains(pdfText, "/FT /Sig"))
}

func TestAcroFormRuntimeSetterClonesInput(t *testing.T) {
	t.Parallel()

	field := forms.NewTextField("name", [4]float64{72, 700, 300, 720}, 0).SetValue("Ada")
	form := forms.NewAcroForm().Add(field)
	cfg := config.NewBuilder().WithCompression(false).Build()
	doc := paper.New(cfg)
	doc.SetAcroForm(form)
	field.Name = "mutated"
	field.Value = "Grace"
	doc.AddAutoRow(col.New(12).Add(text.New("Runtime form")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)
	pdfBytes := pdf.GetBytes()

	assert.True(t, bytes.Contains(pdfBytes, []byte("(name)")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("Ada")))
	assert.False(t, bytes.Contains(pdfBytes, []byte("mutated")))
	assert.False(t, bytes.Contains(pdfBytes, []byte("Grace")))
}

func TestAcroFormForcesWholeDocumentGeneration(t *testing.T) {
	t.Parallel()

	form := forms.NewAcroForm().
		Add(forms.NewTextField("name", [4]float64{72, 700, 300, 720}, 0))
	cfg := config.NewBuilder().
		WithCompression(false).
		WithParallelPagesMode(2).
		WithAcroForm(form).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("Concurrent mode should be bypassed")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)

	assert.True(t, bytes.Contains(pdf.GetBytes(), []byte("/AcroForm")))
}

func TestFormFillerReadsAndUpdatesGeneratedFields(t *testing.T) {
	t.Parallel()

	form := forms.NewAcroForm().
		Add(forms.NewTextField("name", [4]float64{72, 700, 300, 720}, 0).SetValue("Ada")).
		Add(forms.NewCheckbox("agree", [4]float64{72, 680, 92, 700}, 0, true)).
		Add(forms.NewDropdown("role", [4]float64{72, 640, 250, 660}, 0,
			[]string{"Dev", "QA", "PM"}).SetValue("QA"))
	r, err := reader.Parse(generateFormPDF(t, form))
	require.NoError(t, err)

	filler := forms.NewFormFiller(r)
	names, err := filler.FieldNames()
	require.NoError(t, err)
	assert.Equal(t, []string{"name", "agree", "role"}, names)

	value, err := filler.GetValue("agree")
	require.NoError(t, err)
	assert.Equal(t, "Yes", value)

	require.NoError(t, filler.SetValue("name", "Grace"))
	require.NoError(t, filler.SetValue("role", "PM"))
	require.NoError(t, filler.SetCheckbox("agree", false))

	value, err = filler.GetValue("name")
	require.NoError(t, err)
	assert.Equal(t, "Grace", value)
	value, err = filler.GetValue("role")
	require.NoError(t, err)
	assert.Equal(t, "PM", value)
	value, err = filler.GetValue("agree")
	require.NoError(t, err)
	assert.Equal(t, "Off", value)

	out := filler.Bytes()
	assert.True(t, bytes.Contains(out, []byte("Grace")))
	assert.True(t, bytes.Contains(out, []byte("/AS /Off")))
	_, err = reader.Parse(out)
	require.NoError(t, err)
}

func TestFormFillerFlattenRendersValuesAndRemovesFormObjects(t *testing.T) {
	t.Parallel()

	form := forms.NewAcroForm().
		Add(forms.NewTextField("name", [4]float64{72, 700, 300, 720}, 0).SetValue("Ada")).
		Add(forms.NewCheckbox("agree", [4]float64{72, 680, 92, 700}, 0, true)).
		Add(forms.NewDropdown("role", [4]float64{72, 640, 250, 660}, 0,
			[]string{"Dev", "QA", "PM"}).SetValue("QA"))
	r, err := reader.Parse(generateFormPDF(t, form))
	require.NoError(t, err)

	filler := forms.NewFormFiller(r)
	require.NoError(t, filler.SetValue("name", "Grace Hopper"))
	require.NoError(t, filler.SetValue("role", "PM"))
	require.NoError(t, filler.SetCheckbox("agree", false))

	require.NoError(t, filler.Flatten())
	out := filler.Bytes()

	assert.False(t, bytes.Contains(out, []byte("/AcroForm")), "flattened bytes should not carry an AcroForm")
	assert.False(t, bytes.Contains(out, []byte("/Subtype /Widget")), "flattened bytes should not carry widget annotations")
	assert.False(t, bytes.Contains(out, []byte("/NeedAppearances")), "flattened bytes should not request viewer appearances")
	assert.True(t, bytes.Contains(out, []byte("Grace Hopper")), "flattened page content should include filled text")
	assert.True(t, bytes.Contains(out, []byte("(PM)")), "flattened page content should include filled choice")

	parsed, err := reader.Parse(out)
	require.NoError(t, err)
	page, err := parsed.Page(0)
	require.NoError(t, err)
	content, err := page.ContentStream()
	require.NoError(t, err)
	assert.True(t, bytes.Contains(content, []byte("Grace Hopper")))
	assert.True(t, bytes.Contains(content, []byte("(PM)")))

	flattenedFiller := forms.NewFormFiller(parsed)
	names, err := flattenedFiller.FieldNames()
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestFormFillerMissingFieldReturnsError(t *testing.T) {
	t.Parallel()

	form := forms.NewAcroForm().
		Add(forms.NewTextField("name", [4]float64{72, 700, 300, 720}, 0))
	r, err := reader.Parse(generateFormPDF(t, form))
	require.NoError(t, err)
	filler := forms.NewFormFiller(r)

	_, err = filler.GetValue("missing")
	assert.Error(t, err)
	assert.Error(t, filler.SetValue("missing", "value"))
	assert.Error(t, filler.SetCheckbox("missing", true))
}

func generateFormPDF(t *testing.T, form *forms.AcroForm) []byte {
	t.Helper()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithAcroForm(form).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("Form document")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)
	return pdf.GetBytes()
}
