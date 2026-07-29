package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func TestDiag_ChipRunBackground(t *testing.T) {
	t.Parallel()

	html := `<html><head><style>` +
		`.r2{--n100:#e9ecef;--rf:9999px}` +
		`.r2 .q-chip{background:var(--n100);border-radius:var(--rf);padding:3px 9px;color:#46535f}` +
		`</style></head>` +
		`<body class="r2"><div class="q-chips"><span class="q-chip">Barthel</span></div></body></html>`
	doc, err := dom.Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	sheet := parseStylesheet(doc.StyleText())
	body := findFirstNode(t, doc, "body")
	bs := computeNodeStyle(sheet, body, nil)
	div := findFirstNode(t, doc, "div")
	ds := computeNodeStyle(sheet, div, bs)

	tr := &translator{sheet: sheet}
	runs := tr.inlineRunsStyled(div, blockInlineStyle(ds))
	for i, r := range runs {
		t.Logf("run[%d] text=%q Background=%+v BgRadius=%v BgPadX=%v", i, r.Text, r.Background, r.BgRadius, r.BgPadX)
	}
	found := false
	for _, r := range runs {
		if r.Background != nil {
			found = true
		}
	}
	if !found {
		t.Error("chip run has NO Background — inline class background not applied to run")
	}
}
