package pdf

import (
	"bytes"
	"regexp"
	"testing"
)

func TestAcroFormTextFieldEmitted(t *testing.T) {
	f := readyPDF(t)
	f.SetAcroFormFields(FormField{
		Name:      "name",
		Type:      FormFieldText,
		Value:     "Ada",
		Rect:      [4]float64{50, 700, 250, 720},
		PageIndex: 0,
	})
	out := mustOutput(t, f)

	acroForm := regexp.MustCompile(`/AcroForm (\d+) 0 R`).FindSubmatch(out)
	if acroForm == nil {
		t.Fatal("catalog missing /AcroForm reference")
	}
	for _, want := range []string{
		"/T (name)",
		"/FT /Tx",
		"/V (Ada)",
		"/Subtype /Widget",
		"/NeedAppearances true",
		"/Annots [",
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("expected AcroForm output to contain %q", want)
		}
	}
}

func TestAcroFormCheckboxAndRadioObjectNumbering(t *testing.T) {
	f := readyPDF(t)
	f.AddPage()
	f.SetAcroFormFields(
		FormField{
			Name:      "agree",
			Type:      FormFieldCheckbox,
			Value:     "Yes",
			Rect:      [4]float64{50, 650, 65, 665},
			PageIndex: 0,
		},
		FormField{
			Name: "choice",
			Type: FormFieldRadio,
			Children: []FormField{
				{Name: "a", Type: FormFieldRadio, ExportValue: "A", Rect: [4]float64{50, 600, 65, 615}, PageIndex: 0},
				{Name: "b", Type: FormFieldRadio, ExportValue: "B", Rect: [4]float64{80, 600, 95, 615}, PageIndex: 1},
			},
		},
	)
	// newobjExpected inside the form putters errors out if the reserved
	// object numbers do not line up with the actual emission order.
	out := mustOutput(t, f)
	for _, want := range []string{
		"/FT /Btn",
		"/AP << /N << /Yes",
		"/AP << /N << /A",
		"/AP << /N << /B",
		"/AcroForm",
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("expected AcroForm output to contain %q", want)
		}
	}
}

func TestNoAcroFormWithoutFields(t *testing.T) {
	f := readyPDF(t)
	out := mustOutput(t, f)
	if bytes.Contains(out, []byte("/AcroForm")) {
		t.Fatal("unexpected /AcroForm entry without form fields")
	}
}
