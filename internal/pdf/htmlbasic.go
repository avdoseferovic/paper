package pdf

import (
	"regexp"
	"strings"
)

var (
	htmlBasicTagRe  = regexp.MustCompile(`(?U)<.*>`)
	htmlBasicAttrRe = regexp.MustCompile(`([^=]+)=["']?([^"']+)`)
)

// HTMLBasicSegmentType defines a segment of literal text in which the current
// attributes do not vary, or an open tag or a close tag.
type HTMLBasicSegmentType struct {
	Cat  byte              // 'O' open tag, 'C' close tag, 'T' text
	Str  string            // Literal text unchanged, tags are lower case
	Attr map[string]string // Attribute keys are lower case
}

// HTMLBasicTokenize returns a list of HTML tags and literal elements. This is
// done with regular expressions, so the result is only marginally better than
// useless.
func HTMLBasicTokenize(htmlStr string) []HTMLBasicSegmentType {
	// This routine is adapted from http://www.fpdf.org/
	htmlStr = strings.ReplaceAll(htmlStr, "\n", " ")
	htmlStr = strings.ReplaceAll(htmlStr, "\r", "")
	capList := htmlBasicTagRe.FindAllStringIndex(htmlStr, -1)
	if capList == nil {
		return []HTMLBasicSegmentType{htmlBasicTextSegment(htmlStr)}
	}

	list := make([]HTMLBasicSegmentType, 0, len(capList)*2+1)
	pos := 0
	for _, cap := range capList {
		if pos < cap[0] {
			list = append(list, htmlBasicTextSegment(htmlStr[pos:cap[0]]))
		}
		list = append(list, htmlBasicTagSegment(htmlStr[cap[0]+1:cap[1]-1]))
		pos = cap[1]
	}
	if len(htmlStr) > pos {
		list = append(list, htmlBasicTextSegment(htmlStr[pos:]))
	}
	return list
}

func htmlBasicTextSegment(text string) HTMLBasicSegmentType {
	return HTMLBasicSegmentType{Cat: 'T', Str: text, Attr: nil}
}

func htmlBasicTagSegment(tag string) HTMLBasicSegmentType {
	if strings.HasPrefix(tag, "/") {
		return HTMLBasicSegmentType{Cat: 'C', Str: strings.ToLower(tag[1:]), Attr: nil}
	}

	parts := strings.Split(tag, " ")
	if len(parts) == 0 {
		return HTMLBasicSegmentType{Cat: 'O', Str: "", Attr: nil}
	}
	return HTMLBasicSegmentType{
		Cat:  'O',
		Str:  strings.ToLower(parts[0]),
		Attr: htmlBasicAttrs(parts[1:]),
	}
}

func htmlBasicAttrs(parts []string) map[string]string {
	attrs := make(map[string]string)
	for _, part := range parts {
		attrList := htmlBasicAttrRe.FindAllStringSubmatch(part, -1)
		for _, attr := range attrList {
			attrs[strings.ToLower(attr[1])] = attr[2]
		}
	}
	return attrs
}

// HTMLBasicType is used for rendering a very basic subset of HTML. It supports
// only hyperlinks and bold, italic and underscore attributes. In the Link
// structure, the ClrR, ClrG and ClrB fields (0 through 255) define the color
// of hyperlinks. The Bold, Italic and Underscore values define the hyperlink
// style.
type HTMLBasicType struct {
	pdf  *PDF
	Link struct {
		ClrR, ClrG, ClrB         int
		Bold, Italic, Underscore bool
	}
}

// HTMLBasicNew returns an instance that facilitates writing basic HTML in the
// specified PDF file.
func (f *PDF) HTMLBasicNew() HTMLBasicType {
	var html HTMLBasicType
	html.pdf = f
	html.Link.ClrR, html.Link.ClrG, html.Link.ClrB = 0, 0, 128
	html.Link.Bold, html.Link.Italic, html.Link.Underscore = false, false, true
	return html
}

// Write prints text from the current position using the currently selected
// font. See HTMLBasicNew() to create a receiver that is associated with the
// PDF document instance. The text can be encoded with a basic subset of HTML
// that includes hyperlinks and tags for italic (I), bold (B), underscore
// (U) and center (CENTER) attributes. When the right margin is reached a line
// break occurs and text continues from the left margin. Upon method exit, the
// current position is left at the end of the text.
//
// lineHt indicates the line height in the unit of measure specified in New().
func (html *HTMLBasicType) Write(lineHt float64, htmlStr string) {
	textR, textG, textB := html.pdf.GetTextColor()
	state := htmlBasicWriteState{
		html:       html,
		lineHt:     lineHt,
		textR:      textR,
		textG:      textG,
		textB:      textB,
		alignStr:   "L",
		linkBold:   boolInt(html.Link.Bold),
		linkItalic: boolInt(html.Link.Italic),
		linkUnder:  boolInt(html.Link.Underscore),
	}
	for _, el := range HTMLBasicTokenize(htmlStr) {
		switch el.Cat {
		case 'T':
			state.writeText(el.Str)
		case 'O':
			state.openTag(el)
		case 'C':
			state.closeTag(el.Str)
		}
	}
}

type htmlBasicWriteState struct {
	html       *HTMLBasicType
	lineHt     float64
	textR      int
	textG      int
	textB      int
	boldLvl    int
	italicLvl  int
	underLvl   int
	linkBold   int
	linkItalic int
	linkUnder  int
	hrefStr    string
	alignStr   string
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (state *htmlBasicWriteState) setStyle(boldAdj, italicAdj, underscoreAdj int) {
	styleStr := ""
	state.boldLvl += boldAdj
	if state.boldLvl > 0 {
		styleStr += "B"
	}
	state.italicLvl += italicAdj
	if state.italicLvl > 0 {
		styleStr += "I"
	}
	state.underLvl += underscoreAdj
	if state.underLvl > 0 {
		styleStr += "U"
	}
	state.html.pdf.SetFont("", styleStr, 0)
}

func (state *htmlBasicWriteState) writeText(text string) {
	if len(state.hrefStr) > 0 {
		state.putLink(state.hrefStr, text)
		state.hrefStr = ""
		return
	}
	if state.alignStr == "C" || state.alignStr == "R" {
		state.html.pdf.WriteAligned(0, state.lineHt, text, state.alignStr)
		return
	}
	state.html.pdf.Write(state.lineHt, text)
}

func (state *htmlBasicWriteState) putLink(urlStr, txtStr string) {
	state.html.pdf.SetTextColor(state.html.Link.ClrR, state.html.Link.ClrG, state.html.Link.ClrB)
	state.setStyle(state.linkBold, state.linkItalic, state.linkUnder)
	state.html.pdf.WriteLinkString(state.lineHt, txtStr, urlStr)
	state.setStyle(-state.linkBold, -state.linkItalic, -state.linkUnder)
	state.html.pdf.SetTextColor(state.textR, state.textG, state.textB)
}

func (state *htmlBasicWriteState) openTag(el HTMLBasicSegmentType) {
	switch el.Str {
	case "b":
		state.setStyle(1, 0, 0)
	case "i":
		state.setStyle(0, 1, 0)
	case "u":
		state.setStyle(0, 0, 1)
	case "br":
		state.html.pdf.Ln(state.lineHt)
	case "center":
		state.setAlignment("C")
	case "right":
		state.setAlignment("R")
	case "left":
		state.setAlignment("L")
	case "a":
		state.hrefStr = el.Attr["href"]
	}
}

func (state *htmlBasicWriteState) closeTag(tag string) {
	switch tag {
	case "b":
		state.setStyle(-1, 0, 0)
	case "i":
		state.setStyle(0, -1, 0)
	case "u":
		state.setStyle(0, 0, -1)
	case "center", "right":
		state.setAlignment("L")
	}
}

func (state *htmlBasicWriteState) setAlignment(alignStr string) {
	state.html.pdf.Ln(state.lineHt)
	state.alignStr = alignStr
}
