package pdf

import (
	"bytes"
	"math"
	"strings"
	"unicode"
)

// GetStringWidth returns the length of a string in user units. A font must be
// currently selected.
func (f *PDF) GetStringWidth(s string) float64 {
	if f.err != nil {
		return 0
	}
	w := f.GetStringSymbolWidth(s)
	return float64(w) * f.fontSize / 1000
}

// GetStringSymbolWidth returns the length of a string in glyf units. A font must be
// currently selected.
func (f *PDF) GetStringSymbolWidth(s string) int {
	if f.err != nil {
		return 0
	}
	w := 0
	if f.isCurrentUTF8 {
		for _, char := range s {
			w += f.currentRuneWidth(char)
		}
	} else {
		for _, ch := range []byte(s) {
			if ch == 0 {
				break
			}
			w += f.currentFont.Cw[ch]
		}
	}
	return w
}

func (f *PDF) currentRuneWidth(r rune) int {
	char := int(r)
	if char >= 0 && char < len(f.currentFont.Cw) {
		width := f.currentFont.Cw[char]
		if width > 0 {
			if width == 65535 {
				return 0
			}
			return width
		}
	}
	if width := f.currentFont.CwExtra[char]; width > 0 {
		if width == 65535 {
			return 0
		}
		return width
	}
	if f.currentFont.Desc.MissingWidth != 0 {
		return f.currentFont.Desc.MissingWidth
	}
	return 500
}

// Text prints a character string. The origin (x, y) is on the left of the
// first character at the baseline. This method permits a string to be placed
// precisely on the page, but it is usually easier to use Cell(), MultiCell()
// or Write() which are the standard methods to print text.
func (f *PDF) Text(x, y float64, txtStr string) {
	if f.isCurrentUTF8 && f.HasColorEmoji() && f.textContainsColorEmoji(txtStr) {
		f.textWithColorEmoji(x, y, txtStr)
		return
	}
	if f.isCurrentUTF8 && f.currentFont.Tp == "UTF8Bitmap" {
		return
	}

	var txt2 string
	if f.isCurrentUTF8 {
		if f.isRTL {
			txtStr = reverseText(txtStr)
			x -= f.GetStringWidth(txtStr)
		}
		txt2 = f.escape(f.stringToCIDs(txtStr))
	} else {
		txt2 = f.escape(txtStr)
	}
	s := sprintf("BT %.2f %.2f Td (%s) Tj ET", x*f.k, (f.h-y)*f.k, txt2)
	if f.underline && txtStr != "" {
		s += " " + f.dounderline(x, y, txtStr)
	}
	if f.strikeout && txtStr != "" {
		s += " " + f.dostrikeout(x, y, txtStr)
	}
	if f.colorFlag {
		s = sprintf("q %s %s Q", f.color.text.str, s)
	}
	f.out(s)
}

// SetWordSpacing sets spacing between words of following text. See the
// WriteAligned() example for a demonstration of its use.
func (f *PDF) SetWordSpacing(space float64) {
	f.out(sprintf("%.5f Tw", space*f.k))
}

// SetTextRenderingMode sets the rendering mode of following text.
// The mode can be as follows:
// 0: Fill text
// 1: Stroke text
// 2: Fill, then stroke text
// 3: Neither fill nor stroke text (invisible)
// 4: Fill text and add to path for clipping
// 5: Stroke text and add to path for clipping
// 6: Fills then stroke text and add to path for clipping
// 7: Add text to path for clipping
// This method is demonstrated in the SetTextRenderingMode example.
func (f *PDF) SetTextRenderingMode(mode int) {
	if mode >= 0 && mode <= 7 {
		f.out(sprintf("%d Tr", mode))
	}
}

// CellFormat prints a rectangular cell with optional borders, background color
// and character string. The upper-left corner of the cell corresponds to the
// current position. The text can be aligned or centered. After the call, the
// current position moves to the right or to the next line. It is possible to
// put a link on the text.
//
// An error will be returned if a call to SetFont() has not already taken
// place before this method is called.
//
// If automatic page breaking is enabled and the cell goes beyond the limit, a
// page break is done before outputting.
//
// w and h specify the width and height of the cell. If w is 0, the cell
// extends up to the right margin. Specifying 0 for h will result in no output,
// but the current position will be advanced by w.
//
// txtStr specifies the text to display.
//
// borderStr specifies how the cell border will be drawn. An empty string
// indicates no border, "1" indicates a full border, and one or more of "L",
// "T", "R" and "B" indicate the left, top, right and bottom sides of the
// border.
//
// ln indicates where the current position should go after the call. Possible
// values are 0 (to the right), 1 (to the beginning of the next line), and 2
// (below). Putting 1 is equivalent to putting 0 and calling Ln() just after.
//
// alignStr specifies how the text is to be positioned within the cell.
// Horizontal alignment is controlled by including "L", "C" or "R" (left,
// center, right) in alignStr. Vertical alignment is controlled by including
// "T", "M", "B" or "A" (top, middle, bottom, baseline) in alignStr. The default
// alignment is left middle.
//
// fill is true to paint the cell background or false to leave it transparent.
//
// link is the identifier returned by AddLink() or 0 for no internal link.
//
// linkStr is a target URL or empty for no external link. A non--zero value for
// link takes precedence over linkStr.
func (f *PDF) CellFormat(w, h float64, txtStr, borderStr string, ln int,
	alignStr string, fill bool, link int, linkStr string,
) {
	if f.err != nil {
		return
	}

	if f.currentFont.Name == "" {
		f.err = errFontNotSet
		return
	}

	borderStr = strings.ToUpper(borderStr)
	f.cellPageBreak(h)
	if f.err != nil {
		return
	}
	if w == 0 {
		w = f.w - f.rMargin - f.x
	}
	var s fmtBuffer
	f.appendCellFill(&s, w, h, borderStr, fill)
	f.appendCellBorders(&s, w, h, borderStr)
	if len(txtStr) > 0 {
		f.appendCellText(&s, w, h, txtStr, alignStr, link, linkStr)
	}
	str := s.String()
	if len(str) > 0 {
		f.out(str)
	}
	f.lasth = h
	if ln > 0 {
		f.y += h
		if ln == 1 {
			f.x = f.lMargin
		}
	} else {
		f.x += w
	}
}

func (f *PDF) cellPageBreak(h float64) {
	if f.y+h <= f.pageBreakTrigger || f.inHeader || f.inFooter || !f.acceptPageBreak() {
		return
	}
	x := f.x
	ws := f.ws
	if ws > 0 {
		f.ws = 0
		f.out("0 Tw")
	}
	f.AddPageFormat(f.curOrientation, f.curPageSize)
	if f.err != nil {
		return
	}
	f.x = x
	if ws > 0 {
		f.ws = ws
		f.outf("%.3f Tw", ws*f.k)
	}
}

func (f *PDF) appendCellFill(s *fmtBuffer, w, h float64, borderStr string, fill bool) {
	op := cellFillOp(fill, borderStr)
	if op == "" {
		return
	}
	k := f.k
	s.printf("%.2f %.2f %.2f %.2f re %s ", f.x*k, (f.h-f.y)*k, w*k, -h*k, op)
}

func cellFillOp(fill bool, borderStr string) string {
	if fill && borderStr == "1" {
		return "B"
	}
	if fill {
		return "f"
	}
	if borderStr == "1" {
		return "S"
	}
	return ""
}

func (f *PDF) appendCellBorders(s *fmtBuffer, w, h float64, borderStr string) {
	if len(borderStr) == 0 || borderStr == "1" {
		return
	}
	k := f.k
	left := f.x * k
	top := (f.h - f.y) * k
	right := (f.x + w) * k
	bottom := (f.h - (f.y + h)) * k
	if strings.Contains(borderStr, "L") {
		s.printf("%.2f %.2f m %.2f %.2f l S ", left, top, left, bottom)
	}
	if strings.Contains(borderStr, "T") {
		s.printf("%.2f %.2f m %.2f %.2f l S ", left, top, right, top)
	}
	if strings.Contains(borderStr, "R") {
		s.printf("%.2f %.2f m %.2f %.2f l S ", right, top, right, bottom)
	}
	if strings.Contains(borderStr, "B") {
		s.printf("%.2f %.2f m %.2f %.2f l S ", left, bottom, right, bottom)
	}
}

func (f *PDF) appendCellText(s *fmtBuffer, w, h float64, txtStr, alignStr string, link int, linkStr string) {
	dx := f.cellTextDX(w, txtStr, alignStr)
	dy := f.cellTextDY(h, alignStr)
	if f.colorFlag {
		s.printf("q %s ", f.color.text.str)
	}
	renderedText := f.appendCellTextOperation(s, w, h, txtStr, alignStr, dx, dy)
	f.appendCellTextDecorations(s, h, renderedText, dx, dy)
	if f.colorFlag {
		s.printf(" Q")
	}
	if link > 0 || len(linkStr) > 0 {
		f.newLink(f.x+dx, f.y+dy+.5*h-.5*f.fontSize, f.GetStringWidth(renderedText), f.fontSize, link, linkStr)
	}
}

func (f *PDF) cellTextDX(w float64, txtStr, alignStr string) float64 {
	switch {
	case strings.Contains(alignStr, "R"):
		return w - f.cMargin - f.GetStringWidth(txtStr)
	case strings.Contains(alignStr, "C"):
		return (w - f.GetStringWidth(txtStr)) / 2
	default:
		return f.cMargin
	}
}

func (f *PDF) cellTextDY(h float64, alignStr string) float64 {
	switch {
	case strings.Contains(alignStr, "T"):
		return (f.fontSize - h) / 2.0
	case strings.Contains(alignStr, "B"):
		return (h - f.fontSize) / 2.0
	case strings.Contains(alignStr, "A"):
		return (h-f.fontSize)/2.0 - f.fontDescent()
	default:
		return 0
	}
}

func (f *PDF) fontDescent() float64 {
	d := f.currentFont.Desc
	if d.Descent == 0 {
		return -0.19 * f.fontSize
	}
	return float64(d.Descent) * f.fontSize / float64(d.Ascent-d.Descent)
}

func (f *PDF) appendCellTextOperation(
	s *fmtBuffer,
	w, h float64,
	txtStr, alignStr string,
	dx, dy float64,
) string {
	if (f.ws != 0 || alignStr == "J") && f.isCurrentUTF8 {
		return f.appendJustifiedUTF8CellText(s, w, h, txtStr, dx)
	}
	renderedText, escapedText := f.cellEscapedText(txtStr)
	bt := (f.x + dx) * f.k
	td := (f.h - (f.y + dy + .5*h + .3*f.fontSize)) * f.k
	s.printf("BT %.2f %.2f Td (%s)Tj ET", bt, td, escapedText)
	return renderedText
}

func (f *PDF) appendJustifiedUTF8CellText(s *fmtBuffer, w, h float64, txtStr string, dx float64) string {
	if f.isRTL {
		txtStr = reverseText(txtStr)
	}
	wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
	space := f.escape(f.stringToCIDs(" "))
	strSize := f.GetStringSymbolWidth(txtStr)
	s.printf("BT 0 Tw %.2f %.2f Td [", (f.x+dx)*f.k, (f.h-(f.y+.5*h+.3*f.fontSize))*f.k)
	parts := strings.Split(txtStr, " ")
	shift := float64(wmax-strSize) / float64(len(parts)-1)
	for i, tx := range parts {
		s.printf("%s ", "("+f.escape(f.stringToCIDs(tx))+")")
		if i+1 < len(parts) {
			s.printf("%.3f(%s) ", -shift, space)
		}
	}
	s.printf("] TJ ET")
	return txtStr
}

func (f *PDF) cellEscapedText(txtStr string) (string, string) {
	if f.isCurrentUTF8 {
		if f.isRTL {
			txtStr = reverseText(txtStr)
		}
		return txtStr, f.escape(f.stringToCIDs(txtStr))
	}
	txt2 := strings.ReplaceAll(txtStr, "\\", "\\\\")
	txt2 = strings.ReplaceAll(txt2, "(", "\\(")
	txt2 = strings.ReplaceAll(txt2, ")", "\\)")
	return txtStr, txt2
}

func (f *PDF) appendCellTextDecorations(s *fmtBuffer, h float64, txtStr string, dx, dy float64) {
	y := f.y + dy + .5*h + .3*f.fontSize
	if f.underline {
		s.printf(" %s", f.dounderline(f.x+dx, y, txtStr))
	}
	if f.strikeout {
		s.printf(" %s", f.dostrikeout(f.x+dx, y, txtStr))
	}
}

// Revert string to use in RTL languages
func reverseText(text string) string {
	oldText := []rune(text)
	newText := make([]rune, len(oldText))
	length := len(oldText) - 1
	for i, r := range oldText {
		newText[length-i] = r
	}
	return string(newText)
}

// Cell is a simpler version of CellFormat with no fill, border, links or
// special alignment. The Cell_strikeout() example demonstrates this method.
func (f *PDF) Cell(w, h float64, txtStr string) {
	f.CellFormat(w, h, txtStr, "", 0, "L", false, 0, "")
}

// Cellf is a simpler printf-style version of CellFormat with no fill, border,
// links or special alignment. See documentation for the fmt package for
// details on fmtStr and args.
func (f *PDF) Cellf(w, h float64, fmtStr string, args ...any) {
	f.CellFormat(w, h, sprintf(fmtStr, args...), "", 0, "L", false, 0, "")
}

// SplitLines splits text into several lines using the current font. Each line
// has its length limited to a maximum width given by w. This function can be
// used to determine the total height of wrapped text for vertical placement
// purposes.
//
// This method is useful for codepage-based fonts only. For UTF-8 encoded text,
// use SplitText().
//
// You can use MultiCell if you want to print a text on several lines in a
// simple way.
func (f *PDF) SplitLines(txt []byte, w float64) [][]byte {
	lines := [][]byte{}
	cw := f.currentFont.Cw
	wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
	s := bytes.ReplaceAll(txt, []byte("\r"), []byte{})
	nb := len(s)
	for nb > 0 && s[nb-1] == '\n' {
		nb--
	}
	s = s[0:nb]
	sep := -1
	i := 0
	j := 0
	l := 0
	for i < nb {
		c := s[i]
		l += cw[c]
		if c == ' ' || c == '\t' || c == '\n' {
			sep = i
		}
		if c == '\n' || l > wmax {
			lineEnd, nextI := splitLineBreak(i, j, sep)
			lines = append(lines, s[j:lineEnd])
			i = nextI
			sep = -1
			j = i
			l = 0
		} else {
			i++
		}
	}
	if i != j {
		lines = append(lines, s[j:i])
	}
	return lines
}

// MultiCell supports printing text with line breaks. They can be automatic (as
// soon as the text reaches the right border of the cell) or explicit (via the
// \n character). As many cells as necessary are output, one below the other.
//
// Text can be aligned, centered or justified. The cell block can be framed and
// the background painted. See CellFormat() for more details.
//
// The current position after calling MultiCell() is the beginning of the next
// line, equivalent to calling CellFormat with ln equal to 1.
//
// w is the width of the cells. A value of zero indicates cells that reach to
// the right margin.
//
// h indicates the line height of each cell in the unit of measure specified in New().
//
// Note: this method has a known bug that treats UTF-8 fonts differently than
// non-UTF-8 fonts. With UTF-8 fonts, all trailing newlines in txtStr are
// removed. With a non-UTF-8 font, if txtStr has one or more trailing newlines,
// only the last is removed. In the next major module version, the UTF-8 logic
// will be changed to match the non-UTF-8 logic. To prepare for that change,
// applications that use UTF-8 fonts and depend on having all trailing newlines
// removed should call strings.TrimRight(txtStr, "\r\n") before calling this
// method.
func (f *PDF) MultiCell(w, h float64, txtStr, borderStr, alignStr string, fill bool) {
	if f.err != nil {
		return
	}

	state := f.newMultiCellState(w, h, txtStr, borderStr, alignStr, fill)
	for state.i < state.nb {
		c := state.currentRune()
		if c == '\n' {
			state.handleNewline()
			continue
		}
		state.trackSeparator(c)
		state.l += f.currentRuneWidth(c)
		if state.l > state.wmax {
			state.handleLineOverflow()
			continue
		}
		state.i++
	}
	state.finish()
	f.x = f.lMargin
}

type multiCellState struct {
	pdf       *PDF
	w         float64
	h         float64
	s         string
	srune     []rune
	borderStr string
	alignStr  string
	fill      bool
	wmax      int
	nb        int
	b         string
	b2        string
	sep       int
	i         int
	j         int
	l         int
	ls        int
	ns        int
	nl        int
}

func (f *PDF) newMultiCellState(w, h float64, txtStr, borderStr, alignStr string, fill bool) multiCellState {
	if alignStr == "" {
		alignStr = "J"
	}
	if w == 0 {
		w = f.w - f.rMargin - f.x
	}
	s, srune, nb := f.normalizedMultiCellText(txtStr)
	borderStr, b, b2 := multiCellBorders(borderStr)
	return multiCellState{
		pdf:       f,
		w:         w,
		h:         h,
		s:         s,
		srune:     srune,
		borderStr: borderStr,
		alignStr:  alignStr,
		fill:      fill,
		wmax:      int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize)),
		nb:        nb,
		b:         b,
		b2:        b2,
		sep:       -1,
		nl:        1,
	}
}

func (f *PDF) normalizedMultiCellText(txtStr string) (string, []rune, int) {
	s := strings.ReplaceAll(txtStr, "\r", "")
	srune := []rune(s)
	if f.isCurrentUTF8 {
		nb := len(srune)
		for nb > 0 && srune[nb-1] == '\n' {
			nb--
		}
		return s, srune[:nb], nb
	}
	nb := len(s)
	bytes2 := []byte(s)
	if nb > 0 && bytes2[nb-1] == '\n' {
		nb--
	}
	return s[:nb], srune, nb
}

func multiCellBorders(borderStr string) (string, string, string) {
	if len(borderStr) == 0 {
		return borderStr, "0", ""
	}
	if borderStr == "1" {
		return "LTRB", "LRT", "LR"
	}
	b2 := ""
	if strings.Contains(borderStr, "L") {
		b2 += "L"
	}
	if strings.Contains(borderStr, "R") {
		b2 += "R"
	}
	if strings.Contains(borderStr, "T") {
		return borderStr, b2 + "T", b2
	}
	return borderStr, b2, b2
}

func (state *multiCellState) currentRune() rune {
	if state.pdf.isCurrentUTF8 {
		return state.srune[state.i]
	}
	return rune(state.s[state.i])
}

func (state *multiCellState) handleNewline() {
	state.pdf.clearWordSpacing()
	state.cell(state.j, state.i, state.newlineAlign())
	state.i++
	state.nextLine()
}

func (state *multiCellState) newlineAlign() string {
	if !state.pdf.isCurrentUTF8 || state.alignStr != "J" {
		return state.alignStr
	}
	if state.pdf.isRTL {
		return "R"
	}
	return "L"
}

func (state *multiCellState) trackSeparator(c rune) {
	if c != ' ' && !isChinese(c) {
		return
	}
	state.sep = state.i
	state.ls = state.l
	state.ns++
}

func (state *multiCellState) handleLineOverflow() {
	if state.sep == -1 {
		state.handleUnseparatedOverflow()
	} else {
		state.handleSeparatedOverflow()
	}
	state.nextLine()
}

func (state *multiCellState) handleUnseparatedOverflow() {
	if state.i == state.j {
		state.i++
	}
	state.pdf.clearWordSpacing()
	state.cell(state.j, state.i, state.alignStr)
}

func (state *multiCellState) handleSeparatedOverflow() {
	if state.alignStr == "J" {
		state.pdf.setJustifiedWordSpacing(state.wmax, state.ls, state.ns)
	}
	state.cell(state.j, state.sep, state.alignStr)
	state.i = state.sep + 1
}

func (state *multiCellState) nextLine() {
	state.sep = -1
	state.j = state.i
	state.l = 0
	state.ns = 0
	state.nl++
	if len(state.borderStr) > 0 && state.nl == 2 {
		state.b = state.b2
	}
}

func (state *multiCellState) finish() {
	state.pdf.clearWordSpacing()
	if len(state.borderStr) > 0 && strings.Contains(state.borderStr, "B") {
		state.b += "B"
	}
	state.cell(state.j, state.i, state.finalAlign())
}

func (state *multiCellState) finalAlign() string {
	if !state.pdf.isCurrentUTF8 || state.alignStr != "J" {
		return state.alignStr
	}
	if state.pdf.isRTL {
		return "R"
	}
	return ""
}

func (state *multiCellState) cell(start, end int, alignStr string) {
	if state.pdf.isCurrentUTF8 {
		state.pdf.CellFormat(state.w, state.h, string(state.srune[start:end]), state.b, 2, alignStr, state.fill, 0, "")
		return
	}
	state.pdf.CellFormat(state.w, state.h, state.s[start:end], state.b, 2, alignStr, state.fill, 0, "")
}

func (f *PDF) clearWordSpacing() {
	if f.ws <= 0 {
		return
	}
	f.ws = 0
	f.out("0 Tw")
}

func (f *PDF) setJustifiedWordSpacing(wmax, lineWidth, spaces int) {
	if spaces > 1 {
		f.ws = float64((wmax-lineWidth)/1000) * f.fontSize / float64(spaces-1)
	} else {
		f.ws = 0
	}
	f.outf("%.3f Tw", f.ws*f.k)
}

func blankCount(str string) int {
	count := 0
	l := len(str)
	for j := range l {
		if byte(' ') == str[j] {
			count++
		}
	}
	return count
}

// write outputs text in flowing mode
func (f *PDF) write(h float64, txtStr string, link int, linkStr string) {
	state := f.newWriteFlowState(h, txtStr, link, linkStr)
	if state.done {
		return
	}
	for state.i < state.nb {
		c := state.currentRune()
		if c == '\n' {
			state.handleNewline()
			continue
		}
		state.trackSeparator(c)
		state.l += float64(f.currentRuneWidth(c))
		if state.l > state.wmax {
			state.handleOverflow()
			continue
		}
		state.i++
	}
	state.finish()
}

type writeFlowState struct {
	pdf     *PDF
	h       float64
	link    int
	linkStr string
	w       float64
	wmax    float64
	s       string
	srune   []rune
	nb      int
	sep     int
	i       int
	j       int
	l       float64
	nl      int
	done    bool
}

func (f *PDF) newWriteFlowState(h float64, txtStr string, link int, linkStr string) writeFlowState {
	s := strings.ReplaceAll(txtStr, "\r", "")
	state := writeFlowState{
		pdf:     f,
		h:       h,
		link:    link,
		linkStr: linkStr,
		s:       s,
		srune:   []rune(s),
		sep:     -1,
		nl:      1,
	}
	state.resetWidth()
	if f.isCurrentUTF8 {
		state.nb = len(state.srune)
		state.done = state.nb == 1 && s == " "
		if state.done {
			f.x += f.GetStringWidth(s)
		}
	} else {
		state.nb = len(s)
	}
	return state
}

func (state *writeFlowState) resetWidth() {
	state.w = state.pdf.w - state.pdf.rMargin - state.pdf.x
	state.wmax = (state.w - 2*state.pdf.cMargin) * 1000 / state.pdf.fontSize
}

func (state *writeFlowState) currentRune() rune {
	if state.pdf.isCurrentUTF8 {
		return state.srune[state.i]
	}
	return rune(state.s[state.i])
}

func (state *writeFlowState) handleNewline() {
	state.cell(state.j, state.i, state.w, 2)
	state.i++
	state.nextLine()
}

func (state *writeFlowState) trackSeparator(c rune) {
	if c == ' ' {
		state.sep = state.i
	}
}

func (state *writeFlowState) handleOverflow() {
	if state.sep == -1 {
		if state.handleUnseparatedOverflow() {
			return
		}
		state.nextLine()
		return
	}

	state.cell(state.j, state.sep, state.w, 2)
	state.i = state.sep + 1
	state.nextLine()
}

func (state *writeFlowState) handleUnseparatedOverflow() bool {
	if state.pdf.x > state.pdf.lMargin {
		state.pdf.x = state.pdf.lMargin
		state.pdf.y += state.h
		state.resetWidth()
		state.i++
		state.nl++
		return true
	}
	if state.i == state.j {
		state.i++
	}
	state.cell(state.j, state.i, state.w, 2)
	return false
}

func (state *writeFlowState) nextLine() {
	state.sep = -1
	state.j = state.i
	state.l = 0
	if state.nl == 1 {
		state.pdf.x = state.pdf.lMargin
		state.resetWidth()
	}
	state.nl++
}

func (state *writeFlowState) finish() {
	if state.i == state.j {
		return
	}
	state.cell(state.j, state.i, state.l/1000*state.pdf.fontSize, 0)
}

func (state *writeFlowState) cell(start, end int, w float64, ln int) {
	if state.pdf.isCurrentUTF8 {
		state.pdf.CellFormat(w, state.h, string(state.srune[start:end]), "", ln, "", false, state.link, state.linkStr)
		return
	}
	state.pdf.CellFormat(w, state.h, state.s[start:end], "", ln, "", false, state.link, state.linkStr)
}

// Write prints text from the current position. When the right margin is
// reached (or the \n character is met) a line break occurs and text continues
// from the left margin. Upon method exit, the current position is left just at
// the end of the text.
//
// It is possible to put a link on the text.
//
// h indicates the line height in the unit of measure specified in New().
func (f *PDF) Write(h float64, txtStr string) {
	f.write(h, txtStr, 0, "")
}

// Writef is like Write but uses printf-style formatting. See the documentation
// for package fmt for more details on fmtStr and args.
func (f *PDF) Writef(h float64, fmtStr string, args ...any) {
	f.write(h, sprintf(fmtStr, args...), 0, "")
}

// WriteLinkString writes text that when clicked launches an external URL. See
// Write() for argument details.
func (f *PDF) WriteLinkString(h float64, displayStr, targetStr string) {
	f.write(h, displayStr, 0, targetStr)
}

// WriteLinkID writes text that when clicked jumps to another location in the
// PDF. linkID is an identifier returned by AddLink(). See Write() for argument
// details.
func (f *PDF) WriteLinkID(h float64, displayStr string, linkID int) {
	f.write(h, displayStr, linkID, "")
}

// WriteAligned is an implementation of Write that makes it possible to align
// text.
//
// width indicates the width of the box the text will be drawn in. This is in
// the unit of measure specified in New(). If it is set to 0, the bounding box
// of the page will be taken (pageWidth - leftMargin - rightMargin).
//
// lineHeight indicates the line height in the unit of measure specified in
// New().
//
// alignStr sees to horizontal alignment of the given textStr. The options are
// "L", "C" and "R" (Left, Center, Right). The default is "L".
func (f *PDF) WriteAligned(width, lineHeight float64, textStr, alignStr string) {
	lMargin, _, rMargin, _ := f.GetMargins()

	pageWidth, _ := f.GetPageSize()
	if width == 0 {
		width = pageWidth - (lMargin + rMargin)
	}

	var lines []string

	if f.isCurrentUTF8 {
		lines = f.SplitText(textStr, width)
	} else {
		for _, line := range f.SplitLines([]byte(textStr), width) {
			lines = append(lines, string(line))
		}
	}

	for _, lineBt := range lines {
		lineStr := lineBt
		lineWidth := f.GetStringWidth(lineStr)

		switch alignStr {
		case "C":
			f.SetLeftMargin(lMargin + ((width - lineWidth) / 2))
			f.Write(lineHeight, lineStr)
			f.SetLeftMargin(lMargin)
		case "R":
			f.SetLeftMargin(lMargin + (width - lineWidth) - 2.01*f.cMargin)
			f.Write(lineHeight, lineStr)
			f.SetLeftMargin(lMargin)
		default:
			f.SetRightMargin(pageWidth - lMargin - width)
			f.Write(lineHeight, lineStr)
			f.SetRightMargin(rMargin)
		}
	}
}

// AddLink creates a new internal link and returns its identifier. An internal
// link is a clickable area which directs to another place within the document.
// The identifier can then be passed to Cell(), Write(), Image() or Link(). The
// destination is defined with SetLink().
func (f *PDF) AddLink() int {
	f.links = append(f.links, intLinkType{})
	return len(f.links) - 1
}

// SetLink defines the page and position a link points to. See AddLink().
func (f *PDF) SetLink(link int, y float64, page int) {
	if y == -1 {
		y = f.y
	}
	if page == -1 {
		page = f.page
	}
	f.links[link] = intLinkType{page, y}
}

// newLink adds a new clickable link on current page
func (f *PDF) newLink(x, y, w, h float64, link int, linkStr string) {
	f.pageLinks[f.page] = append(f.pageLinks[f.page],
		linkType{x * f.k, f.hPt - y*f.k, w * f.k, h * f.k, link, linkStr})
}

// Link puts a link on a rectangular area of the page. Text or image links are
// generally put via Cell(), Write() or Image(), but this method can be useful
// for instance to define a clickable area inside an image. link is the value
// returned by AddLink().
func (f *PDF) Link(x, y, w, h float64, link int) {
	f.newLink(x, y, w, h, link, "")
}

// LinkString puts a link on a rectangular area of the page. Text or image
// links are generally put via Cell(), Write() or Image(), but this method can
// be useful for instance to define a clickable area inside an image. linkStr
// is the target URL.
func (f *PDF) LinkString(x, y, w, h float64, linkStr string) {
	f.newLink(x, y, w, h, 0, linkStr)
}

// Bookmark sets a bookmark that will be displayed in a sidebar outline. txtStr
// is the title of the bookmark. level specifies the level of the bookmark in
// the outline; 0 is the top level, 1 is just below, and so on. y specifies the
// vertical position of the bookmark destination in the current page; -1
// indicates the current position.
func (f *PDF) Bookmark(txtStr string, level int, y float64) {
	if y == -1 {
		y = f.y
	}
	if f.isCurrentUTF8 {
		txtStr = utf8toutf16(txtStr)
	}
	f.outlines = append(f.outlines, outlineType{text: txtStr, level: level, y: y, p: f.PageNo(), prev: -1, last: -1, next: -1, first: -1})
}

func (f *PDF) putbookmarks() {
	nb := len(f.outlines)
	if nb == 0 {
		return
	}
	lru := f.prepareBookmarkOutlines(nb)
	n := f.n + 1
	f.putBookmarkObjects(n)
	f.putBookmarkRoot(n, lru[0])
}

func (f *PDF) prepareBookmarkOutlines(nb int) map[int]int {
	lru := make(map[int]int)
	level := 0
	for i, o := range f.outlines {
		f.setBookmarkParent(i, o, nb, lru, level)
		f.setBookmarkSiblings(i, o, lru, level)
		lru[o.level] = i
		level = o.level
	}
	return lru
}

func (f *PDF) setBookmarkParent(i int, o outlineType, nb int, lru map[int]int, level int) {
	if o.level == 0 {
		f.outlines[i].parent = nb
		return
	}
	parent := lru[o.level-1]
	f.outlines[i].parent = parent
	f.outlines[parent].last = i
	if o.level > level {
		f.outlines[parent].first = i
	}
}

func (f *PDF) setBookmarkSiblings(i int, o outlineType, lru map[int]int, level int) {
	if o.level > level || i == 0 {
		return
	}
	prev := lru[o.level]
	f.outlines[prev].next = i
	f.outlines[i].prev = prev
}

func (f *PDF) putBookmarkObjects(n int) {
	for _, o := range f.outlines {
		f.newobj()
		f.outf("<</Title %s", f.textstring(o.text))
		f.outf("/Parent %d 0 R", n+o.parent)
		f.putBookmarkObjectLinks(n, o)
		f.outf("/Dest [%d 0 R /XYZ 0 %.2f null]", 1+2*o.p, (f.h-o.y)*f.k)
		f.out("/Count 0>>")
		f.out("endobj")
	}
}

func (f *PDF) putBookmarkObjectLinks(n int, o outlineType) {
	if o.prev != -1 {
		f.outf("/Prev %d 0 R", n+o.prev)
	}
	if o.next != -1 {
		f.outf("/Next %d 0 R", n+o.next)
	}
	if o.first != -1 {
		f.outf("/First %d 0 R", n+o.first)
	}
	if o.last != -1 {
		f.outf("/Last %d 0 R", n+o.last)
	}
}

func (f *PDF) putBookmarkRoot(n, last int) {
	f.newobj()
	f.outlineRoot = f.n
	f.outf("<</Type /Outlines /First %d 0 R", n)
	f.outf("/Last %d 0 R>>", n+last)
	f.out("endobj")
}

// SplitText splits UTF-8 encoded text into several lines using the current
// font. Each line has its length limited to a maximum width given by w. This
// function can be used to determine the total height of wrapped text for
// vertical placement purposes.
func (f *PDF) SplitText(txt string, w float64) []string {
	lines := make([]string, 0)
	wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
	s := []rune(txt)
	nb := len(s)
	for nb > 0 && s[nb-1] == '\n' {
		nb--
	}
	s = s[0:nb]
	sep := -1
	i := 0
	j := 0
	l := 0
	for i < nb {
		c := s[i]
		l += f.currentRuneWidth(c)
		if unicode.IsSpace(c) || isChinese(c) {
			sep = i
		}
		if c == '\n' || l > wmax {
			lineEnd, nextI := splitLineBreak(i, j, sep)
			lines = append(lines, string(s[j:lineEnd]))
			i = nextI
			sep = -1
			j = i
			l = 0
		} else {
			i++
		}
	}
	if i != j {
		lines = append(lines, string(s[j:i]))
	}
	return lines
}

func splitLineBreak(i, j, sep int) (int, int) {
	if sep != -1 {
		return sep, sep + 1
	}
	if i == j {
		i++
	}
	return i, i
}

// SubWrite prints text from the current position in the same way as Write().
// ht is the line height in the unit of measure specified in New(). str
// specifies the text to write. subFontSize is the size of the font in points.
// subOffset is the vertical offset of the text in points; a positive value
// indicates a superscript, a negative value indicates a subscript. link is the
// identifier returned by AddLink() or 0 for no internal link. linkStr is a
// target URL or empty for no external link. A non--zero value for link takes
// precedence over linkStr.
//
// The SubWrite example demonstrates this method.
func (f *PDF) SubWrite(ht float64, str string, subFontSize, subOffset float64, link int, linkStr string) {
	if f.err != nil {
		return
	}

	subFontSizeOld := f.fontSizePt
	f.SetFontSize(subFontSize)

	subOffset = (((subFontSize - subFontSizeOld) / f.k) * 0.3) + (subOffset / f.k)
	subX := f.x
	subY := f.y
	f.SetXY(subX, subY-subOffset)

	f.write(ht, str, link, linkStr)

	subX = f.x
	subY = f.y
	f.SetXY(subX, subY+subOffset)

	f.SetFontSize(subFontSizeOld)
}
