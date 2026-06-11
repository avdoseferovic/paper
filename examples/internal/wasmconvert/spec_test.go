package wasmconvert_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/examples/internal/wasmconvert"
)

func decodeSpec(t *testing.T, b64 string) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("result is not valid base64: %v", err)
	}
	return raw
}

func assertPDF(t *testing.T, b64 string) {
	t.Helper()
	if b64 == "" {
		t.Fatal("expected non-empty base64 output")
	}
	if !strings.HasPrefix(string(decodeSpec(t, b64)), "%PDF-") {
		t.Fatal("decoded output is not a PDF document")
	}
}

func TestSpecToBase64_MinimalText_ReturnsPDF(t *testing.T) {
	spec := `{"rows":[{"cols":[{"span":12,"type":"text","style":"h1","value":"Hi"}]}]}`
	b64, err := wasmconvert.SpecToBase64(context.Background(), spec, "A4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPDF(t, b64)
}

func TestSpecToBase64_Empty_ReturnsErrEmptySpec(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t "} {
		_, err := wasmconvert.SpecToBase64(context.Background(), in, "A4")
		if !errors.Is(err, wasmconvert.ErrEmptySpec) {
			t.Fatalf("input %q: expected ErrEmptySpec, got %v", in, err)
		}
	}
}

func TestSpecToBase64_InvalidJSON_ReturnsError(t *testing.T) {
	_, err := wasmconvert.SpecToBase64(context.Background(), `{not json`, "A4")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if errors.Is(err, wasmconvert.ErrEmptySpec) {
		t.Fatal("invalid JSON should not report ErrEmptySpec")
	}
}

func TestSpecToBase64_LetterPageSize_ReturnsPDF(t *testing.T) {
	spec := `{"rows":[{"cols":[{"span":12,"type":"text","value":"Letter sized"}]}]}`
	b64, err := wasmconvert.SpecToBase64(context.Background(), spec, "Letter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPDF(t, b64)
}

func TestSpecToBase64_LineAndUnknownType_NoPanic(t *testing.T) {
	spec := `{"rows":[
		{"cols":[{"span":12,"type":"line","soft":true}]},
		{"cols":[{"span":12,"type":"image","value":"placeholder"}]},
		{"cols":[{"span":12,"type":"text","value":"after"}]}
	]}`
	b64, err := wasmconvert.SpecToBase64(context.Background(), spec, "A4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPDF(t, b64)
}

// A degenerate spec (huge spans, negatives, deep nesting) must never panic the
// caller — SpecToBase64 recovers and returns a doc or an error.
func TestSpecToBase64_DegenerateSpec_DoesNotPanic(t *testing.T) {
	spec := `{"rows":[{"cols":[
		{"span":9999,"type":"text","value":"big span"},
		{"span":-3,"type":"line"}
	]}]}`
	_, _ = wasmconvert.SpecToBase64(context.Background(), spec, "A4")
	// Reaching here without a panic is the assertion.
}

func TestSpecToBase64_AllComponentTypes_ReturnsPDF(t *testing.T) {
	spec := `{"rows":[
		{"cols":[{"span":12,"type":"text","style":"h1","value":"Everything"}]},
		{"cols":[{"span":12,"type":"line"}]},
		{"cols":[
			{"span":4,"type":"qrcode","value":"q"},
			{"span":4,"type":"barcode","value":"b","label":"CODE"},
			{"span":4,"type":"signature","value":"A. S.","label":"SIGNED"}
		]},
		{"cols":[{"span":12,"type":"checkbox","checked":true,"value":"done"}]},
		{"cols":[{"span":12,"type":"table","head":["A","B"],"rows":[["1","2"]]}]},
		{"cols":[{"span":12,"type":"pagenumber","value":"Page {n} of {t}"}]},
		{"cols":[{"span":12,"type":"footer","value":"footer text"}]}
	]}`
	b64, err := wasmconvert.SpecToBase64(context.Background(), spec, "A4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPDF(t, b64)
}

func TestSpecToBase64_TableWithColAlign_ReturnsPDF(t *testing.T) {
	spec := `{"rows":[{"cols":[{"span":12,"type":"table",
		"head":["Description","Qty","Amount"],"colAlign":["","c","r"],
		"rows":[["Item one","1","$10"],["Item two","2","$20"],["Item three","3","$30"]]}]}]}`
	b64, err := wasmconvert.SpecToBase64(context.Background(), spec, "A4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPDF(t, b64)
}

func TestSpecToBase64_MalformedTable_DoesNotPanic(t *testing.T) {
	// Ragged rows (differing column counts) must not panic.
	spec := `{"rows":[{"cols":[{"span":12,"type":"table",
		"head":["A","B","C"],"rows":[["1"],["1","2","3","4","5"]]}]}]}`
	_, _ = wasmconvert.SpecToBase64(context.Background(), spec, "A4")
}

// The three component-grid presets shipped in the Paper Playground design must
// each generate a real PDF.
func TestSpecToBase64_DesignPresets_Generate(t *testing.T) {
	presets := map[string]string{
		"invoice": `{
  "rows": [
    { "cols": [
      { "span": 8, "type": "text", "style": "h1", "value": "Invoice" },
      { "span": 4, "type": "text", "style": "label", "align": "right", "value": "NO. INV-2026-0481" }
    ]},
    { "cols": [ { "span": 12, "type": "line" } ] },
    { "mt": 6, "cols": [
      { "span": 6, "type": "text", "style": "label", "value": "Billed to" },
      { "span": 6, "type": "text", "style": "label", "align": "right", "value": "Amount due" }
    ]},
    { "cols": [
      { "span": 6, "type": "text", "style": "body", "value": "Northwind Trading Co.<br>Seattle, WA" },
      { "span": 6, "type": "text", "style": "num", "align": "right", "value": "$961.91" }
    ]},
    { "mt": 10, "cols": [ { "span": 12, "type": "table",
      "head": ["Description", "Qty", "Amount"], "colAlign": ["", "c", "r"],
      "rows": [
        ["Document API — Team plan", "1", "$480.00"],
        ["Rendering credits", "12k", "$240.00"],
        ["Priority support", "1", "$120.00"]
      ] } ]},
    { "mt": 14, "cols": [
      { "span": 3, "type": "qrcode", "value": "pay.paperlabs.dev/inv-0481", "size": 96 },
      { "span": 5, "type": "text", "style": "body", "value": "Scan to pay online.<br>Net 30 · Wire or ACH." },
      { "span": 4, "type": "signature", "value": "Avdo S.", "label": "AUTHORIZED · PAPER LABS" }
    ]}
  ]
}`,
		"label": `{
  "rows": [
    { "cols": [
      { "span": 7, "type": "text", "style": "h2", "value": "SHIP TO" },
      { "span": 5, "type": "text", "style": "label", "align": "right", "value": "PKG 1 / 1" }
    ]},
    { "cols": [ { "span": 12, "type": "line" } ] },
    { "mt": 4, "cols": [
      { "span": 12, "type": "text", "style": "body",
        "value": "<strong>Northwind Trading Co.</strong><br>4400 Harbor Blvd<br>Seattle, WA 98101" }
    ]},
    { "mt": 12, "cols": [
      { "span": 6, "type": "qrcode", "value": "TRK-9920-4471-AX", "size": 120 },
      { "span": 6, "type": "barcode", "value": "TRK99204471AX", "width": 200, "height": 70,
        "label": "TRK 9920 4471 AX" }
    ]},
    { "mt": 10, "cols": [
      { "span": 12, "type": "checkbox", "checked": true, "value": "Signature on delivery required" }
    ]}
  ]
}`,
		"certificate": `{
  "rows": [
    { "cols": [ { "span": 12, "type": "text", "style": "label", "align": "center", "value": "CERTIFICATE OF COMPLETION" } ]},
    { "mt": 8, "cols": [ { "span": 12, "type": "text", "style": "h1", "align": "center", "value": "Document Automation 101" } ]},
    { "mt": 6, "cols": [ { "span": 12, "type": "text", "style": "body", "align": "center", "value": "awarded to" } ]},
    { "cols": [ { "span": 12, "type": "text", "style": "h2", "align": "center", "value": "Jordan Vega" } ]},
    { "mt": 18, "cols": [
      { "span": 4, "type": "signature", "value": "A. Sefer", "label": "INSTRUCTOR" },
      { "span": 4, "type": "qrcode", "value": "verify.paperlabs.dev/cert/8841", "size": 84, "align": "center" },
      { "span": 4, "type": "signature", "value": "M. Reyes", "label": "DIRECTOR" }
    ]},
    { "mt": 10, "cols": [ { "span": 12, "type": "pagenumber", "value": "Certificate ID 8841 · Issued June 2026" } ]}
  ]
}`,
	}
	for name, spec := range presets {
		t.Run(name, func(t *testing.T) {
			b64, err := wasmconvert.SpecToBase64(context.Background(), spec, "A4")
			if err != nil {
				t.Fatalf("preset %s: unexpected error: %v", name, err)
			}
			assertPDF(t, b64)
		})
	}
}

func TestPlainText_BrBecomesNewline(t *testing.T) {
	// <br> in a value should yield a multi-line string (2 lines).
	spec := `{"rows":[{"cols":[{"span":12,"type":"text","value":"line one<br>line two"}]}]}`
	b64, err := wasmconvert.SpecToBase64(context.Background(), spec, "A4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPDF(t, b64)
}
