package pdf

import (
	"bytes"
	"encoding/gob"
	"strings"
	"testing"
)

func TestTemplateSerializeUseWritesFormXObjectResources(t *testing.T) {
	gob.Register(&PDFTpl{})

	source := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	source.SetCompression(false)

	child := source.CreateTemplateCustom(PointType{X: 0, Y: 0}, SizeType{Wd: 20, Ht: 12}, func(tpl *Tpl) {
		tpl.Rect(1, 1, 18, 10, "D")
	})
	parent := source.CreateTemplateCustom(PointType{X: 0, Y: 0}, SizeType{Wd: 40, Ht: 24}, func(tpl *Tpl) {
		tpl.UseTemplateScaled(child, PointType{X: 4, Y: 4}, SizeType{Wd: 20, Ht: 12})
		tpl.Rect(0, 0, 40, 24, "D")
	})

	serialized, err := parent.Serialize()
	if err != nil {
		t.Fatalf("serialize template: %v", err)
	}

	var restored PDFTpl
	if err := gob.NewDecoder(bytes.NewReader(serialized)).Decode(&restored); err != nil {
		t.Fatalf("deserialize template: %v", err)
	}

	pdf := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetCompression(false)
	pdf.AddPage()
	pdf.UseTemplate(&restored)

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		t.Fatalf("output pdf: %v", err)
	}

	body := out.String()
	for _, marker := range []string{
		"%PDF",
		"/Type /XObject",
		"/Subtype /Form",
		"/Resources",
		"/XObject <<",
		"/TPL",
		" Do Q",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected PDF output to contain %q", marker)
		}
	}

	if got := strings.Count(body, "/Subtype /Form"); got < 2 {
		t.Fatalf("expected parent and child form XObjects, got %d", got)
	}
}
