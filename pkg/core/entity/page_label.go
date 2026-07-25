package entity

import "fmt"

// LabelStyle is how a page label numbers its pages.
type LabelStyle string

// The page numbering styles: decimal, upper/lower roman, upper/lower letters,
// or no number at all.
const (
	LabelDecimal    LabelStyle = "D"
	LabelRomanUpper LabelStyle = "R"
	LabelRomanLower LabelStyle = "r"
	LabelAlphaUpper LabelStyle = "A"
	LabelAlphaLower LabelStyle = "a"
	LabelNone       LabelStyle = ""
)

// PageLabelRange labels the run of pages starting at PageIndex. Start is the
// number the run counts from, and Prefix is put in front of each number.
type PageLabelRange struct {
	PageIndex int
	Style     LabelStyle
	Prefix    string
	Start     int
}

func appendPageLabelsMap(labels []PageLabelRange, m map[string]any) map[string]any {
	for i, label := range labels {
		prefix := fmt.Sprintf("config_page_label_%d_", i)
		if label.PageIndex != 0 {
			m[prefix+"page_index"] = label.PageIndex
		}
		if label.Style != "" {
			m[prefix+"style"] = label.Style
		}
		if label.Prefix != "" {
			m[prefix+"prefix"] = label.Prefix
		}
		if label.Start != 0 {
			m[prefix+"start"] = label.Start
		}
	}
	if len(labels) > 0 {
		m["config_page_label_count"] = len(labels)
	}
	return m
}
