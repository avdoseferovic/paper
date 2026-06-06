/*
 * Copyright (c) 2013-2014 Kurt Jung (Gmail: kurt.w.jung)
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package paperpdf

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unicode"
)

// GetStringWidth returns the length of a string in user units. A font must be
// currently selected.
func (f *Fpdf) GetStringWidth(s string) float64 {
	if f.err != nil {
		return 0
	}
	w := f.GetStringSymbolWidth(s)
	return float64(w) * f.fontSize / 1000
}

// GetStringSymbolWidth returns the length of a string in glyf units. A font must be
// currently selected.
func (f *Fpdf) GetStringSymbolWidth(s string) int {
	if f.err != nil {
		return 0
	}
	w := 0
	if f.isCurrentUTF8 {
		unicode := []rune(s)
		for _, char := range unicode {
			intChar := int(char)
			if len(f.currentFont.Cw) >= intChar && f.currentFont.Cw[intChar] > 0 {
				if f.currentFont.Cw[intChar] != 65535 {
					w += f.currentFont.Cw[intChar]
				}
			} else if f.currentFont.Desc.MissingWidth != 0 {
				w += f.currentFont.Desc.MissingWidth
			} else {
				w += 500
			}
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

// Text prints a character string. The origin (x, y) is on the left of the
// first character at the baseline. This method permits a string to be placed
// precisely on the page, but it is usually easier to use Cell(), MultiCell()
// or Write() which are the standard methods to print text.
func (f *Fpdf) Text(x, y float64, txtStr string) {
	var txt2 string
	if f.isCurrentUTF8 {
		if f.isRTL {
			txtStr = reverseText(txtStr)
			x -= f.GetStringWidth(txtStr)
		}
		txt2 = f.escape(utf8toutf16(txtStr, false))
		for _, uni := range []rune(txtStr) {
			f.currentFont.usedRunes[int(uni)] = int(uni)
		}
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
func (f *Fpdf) SetWordSpacing(space float64) {
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
func (f *Fpdf) SetTextRenderingMode(mode int) {
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
func (f *Fpdf) CellFormat(w, h float64, txtStr, borderStr string, ln int,
	alignStr string, fill bool, link int, linkStr string) {

	if f.err != nil {
		return
	}

	if f.currentFont.Name == "" {
		f.err = fmt.Errorf("font has not been set; unable to render text")
		return
	}

	borderStr = strings.ToUpper(borderStr)
	k := f.k
	if f.y+h > f.pageBreakTrigger && !f.inHeader && !f.inFooter && f.acceptPageBreak() {

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
			f.outf("%.3f Tw", ws*k)
		}
	}
	if w == 0 {
		w = f.w - f.rMargin - f.x
	}
	var s fmtBuffer
	if fill || borderStr == "1" {
		var op string
		if fill {
			if borderStr == "1" {
				op = "B"

			} else {
				op = "f"

			}
		} else {

			op = "S"
		}

		s.printf("%.2f %.2f %.2f %.2f re %s ", f.x*k, (f.h-f.y)*k, w*k, -h*k, op)
	}
	if len(borderStr) > 0 && borderStr != "1" {
		x := f.x
		y := f.y
		left := x * k
		top := (f.h - y) * k
		right := (x + w) * k
		bottom := (f.h - (y + h)) * k
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
	if len(txtStr) > 0 {
		var dx, dy float64

		switch {
		case strings.Contains(alignStr, "R"):
			dx = w - f.cMargin - f.GetStringWidth(txtStr)
		case strings.Contains(alignStr, "C"):
			dx = (w - f.GetStringWidth(txtStr)) / 2
		default:
			dx = f.cMargin
		}

		switch {
		case strings.Contains(alignStr, "T"):
			dy = (f.fontSize - h) / 2.0
		case strings.Contains(alignStr, "B"):
			dy = (h - f.fontSize) / 2.0
		case strings.Contains(alignStr, "A"):
			var descent float64
			d := f.currentFont.Desc
			if d.Descent == 0 {

				descent = -0.19 * f.fontSize
			} else {
				descent = float64(d.Descent) * f.fontSize / float64(d.Ascent-d.Descent)
			}
			dy = (h-f.fontSize)/2.0 - descent
		default:
			dy = 0
		}
		if f.colorFlag {
			s.printf("q %s ", f.color.text.str)
		}

		if (f.ws != 0 || alignStr == "J") && f.isCurrentUTF8 {
			if f.isRTL {
				txtStr = reverseText(txtStr)
			}
			wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
			for _, uni := range []rune(txtStr) {
				f.currentFont.usedRunes[int(uni)] = int(uni)
			}
			space := f.escape(utf8toutf16(" ", false))
			strSize := f.GetStringSymbolWidth(txtStr)
			s.printf("BT 0 Tw %.2f %.2f Td [", (f.x+dx)*k, (f.h-(f.y+.5*h+.3*f.fontSize))*k)
			t := strings.Split(txtStr, " ")
			shift := float64((wmax - strSize)) / float64(len(t)-1)
			numt := len(t)
			for i := 0; i < numt; i++ {
				tx := t[i]
				tx = "(" + f.escape(utf8toutf16(tx, false)) + ")"
				s.printf("%s ", tx)
				if (i + 1) < numt {
					s.printf("%.3f(%s) ", -shift, space)
				}
			}
			s.printf("] TJ ET")
		} else {
			var txt2 string
			if f.isCurrentUTF8 {
				if f.isRTL {
					txtStr = reverseText(txtStr)
				}
				txt2 = f.escape(utf8toutf16(txtStr, false))
				for _, uni := range []rune(txtStr) {
					f.currentFont.usedRunes[int(uni)] = int(uni)
				}
			} else {

				txt2 = strings.Replace(txtStr, "\\", "\\\\", -1)
				txt2 = strings.Replace(txt2, "(", "\\(", -1)
				txt2 = strings.Replace(txt2, ")", "\\)", -1)
			}
			bt := (f.x + dx) * k
			td := (f.h - (f.y + dy + .5*h + .3*f.fontSize)) * k
			s.printf("BT %.2f %.2f Td (%s)Tj ET", bt, td, txt2)

		}

		if f.underline {
			s.printf(" %s", f.dounderline(f.x+dx, f.y+dy+.5*h+.3*f.fontSize, txtStr))
		}
		if f.strikeout {
			s.printf(" %s", f.dostrikeout(f.x+dx, f.y+dy+.5*h+.3*f.fontSize, txtStr))
		}
		if f.colorFlag {
			s.printf(" Q")
		}
		if link > 0 || len(linkStr) > 0 {
			f.newLink(f.x+dx, f.y+dy+.5*h-.5*f.fontSize, f.GetStringWidth(txtStr), f.fontSize, link, linkStr)
		}
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
	return
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
func (f *Fpdf) Cell(w, h float64, txtStr string) {
	f.CellFormat(w, h, txtStr, "", 0, "L", false, 0, "")
}

// Cellf is a simpler printf-style version of CellFormat with no fill, border,
// links or special alignment. See documentation for the fmt package for
// details on fmtStr and args.
func (f *Fpdf) Cellf(w, h float64, fmtStr string, args ...any) {
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
func (f *Fpdf) SplitLines(txt []byte, w float64) [][]byte {

	lines := [][]byte{}
	cw := f.currentFont.Cw
	wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
	s := bytes.Replace(txt, []byte("\r"), []byte{}, -1)
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
			if sep == -1 {
				if i == j {
					i++
				}
				sep = i
			} else {
				i = sep + 1
			}
			lines = append(lines, s[j:sep])
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
func (f *Fpdf) MultiCell(w, h float64, txtStr, borderStr, alignStr string, fill bool) {
	if f.err != nil {
		return
	}

	if alignStr == "" {
		alignStr = "J"
	}
	cw := f.currentFont.Cw
	if w == 0 {
		w = f.w - f.rMargin - f.x
	}
	wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
	s := strings.Replace(txtStr, "\r", "", -1)
	srune := []rune(s)

	// remove extra line breaks
	var nb int
	if f.isCurrentUTF8 {
		nb = len(srune)
		for nb > 0 && srune[nb-1] == '\n' {
			nb--
		}
		srune = srune[0:nb]
	} else {
		nb = len(s)
		bytes2 := []byte(s)

		if nb > 0 && bytes2[nb-1] == '\n' {
			nb--
		}
		s = s[0:nb]
	}
	// dbg("[%s]\n", s)
	var b, b2 string
	b = "0"
	if len(borderStr) > 0 {
		if borderStr == "1" {
			borderStr = "LTRB"
			b = "LRT"
			b2 = "LR"
		} else {
			b2 = ""
			if strings.Contains(borderStr, "L") {
				b2 += "L"
			}
			if strings.Contains(borderStr, "R") {
				b2 += "R"
			}
			if strings.Contains(borderStr, "T") {
				b = b2 + "T"
			} else {
				b = b2
			}
		}
	}
	sep := -1
	i := 0
	j := 0
	l := 0
	ls := 0
	ns := 0
	nl := 1
	for i < nb {
		// Get next character
		var c rune
		if f.isCurrentUTF8 {
			c = srune[i]
		} else {
			c = rune(s[i])
		}
		if c == '\n' {

			if f.ws > 0 {
				f.ws = 0
				f.out("0 Tw")
			}

			if f.isCurrentUTF8 {
				newAlignStr := alignStr
				if newAlignStr == "J" {
					if f.isRTL {
						newAlignStr = "R"
					} else {
						newAlignStr = "L"
					}
				}
				f.CellFormat(w, h, string(srune[j:i]), b, 2, newAlignStr, fill, 0, "")
			} else {
				f.CellFormat(w, h, s[j:i], b, 2, alignStr, fill, 0, "")
			}
			i++
			sep = -1
			j = i
			l = 0
			ns = 0
			nl++
			if len(borderStr) > 0 && nl == 2 {
				b = b2
			}
			continue
		}
		if c == ' ' || isChinese(c) {
			sep = i
			ls = l
			ns++
		}
		if int(c) >= len(cw) {
			f.err = fmt.Errorf("character outside the supported range: %s", string(c))
			return
		}
		if cw[int(c)] == 0 {
			l += f.currentFont.Desc.MissingWidth
		} else if cw[int(c)] != 65535 {
			l += cw[int(c)]
		}
		if l > wmax {

			if sep == -1 {
				if i == j {
					i++
				}
				if f.ws > 0 {
					f.ws = 0
					f.out("0 Tw")
				}
				if f.isCurrentUTF8 {
					f.CellFormat(w, h, string(srune[j:i]), b, 2, alignStr, fill, 0, "")
				} else {
					f.CellFormat(w, h, s[j:i], b, 2, alignStr, fill, 0, "")
				}
			} else {
				if alignStr == "J" {
					if ns > 1 {
						f.ws = float64((wmax-ls)/1000) * f.fontSize / float64(ns-1)
					} else {
						f.ws = 0
					}
					f.outf("%.3f Tw", f.ws*f.k)
				}
				if f.isCurrentUTF8 {
					f.CellFormat(w, h, string(srune[j:sep]), b, 2, alignStr, fill, 0, "")
				} else {
					f.CellFormat(w, h, s[j:sep], b, 2, alignStr, fill, 0, "")
				}
				i = sep + 1
			}
			sep = -1
			j = i
			l = 0
			ns = 0
			nl++
			if len(borderStr) > 0 && nl == 2 {
				b = b2
			}
		} else {
			i++
		}
	}

	if f.ws > 0 {
		f.ws = 0
		f.out("0 Tw")
	}
	if len(borderStr) > 0 && strings.Contains(borderStr, "B") {
		b += "B"
	}
	if f.isCurrentUTF8 {
		if alignStr == "J" {
			if f.isRTL {
				alignStr = "R"
			} else {
				alignStr = ""
			}
		}
		f.CellFormat(w, h, string(srune[j:i]), b, 2, alignStr, fill, 0, "")
	} else {
		f.CellFormat(w, h, s[j:i], b, 2, alignStr, fill, 0, "")
	}
	f.x = f.lMargin
}

func blankCount(str string) (count int) {
	l := len(str)
	for j := 0; j < l; j++ {
		if byte(' ') == str[j] {
			count++
		}
	}
	return
}

// write outputs text in flowing mode
func (f *Fpdf) write(h float64, txtStr string, link int, linkStr string) {

	cw := f.currentFont.Cw
	w := f.w - f.rMargin - f.x
	wmax := (w - 2*f.cMargin) * 1000 / f.fontSize
	s := strings.Replace(txtStr, "\r", "", -1)
	var nb int
	if f.isCurrentUTF8 {
		nb = len([]rune(s))
		if nb == 1 && s == " " {
			f.x += f.GetStringWidth(s)
			return
		}
	} else {
		nb = len(s)
	}
	sep := -1
	i := 0
	j := 0
	l := 0.0
	nl := 1
	for i < nb {
		// Get next character
		var c rune
		if f.isCurrentUTF8 {
			c = []rune(s)[i]
		} else {
			c = rune(byte(s[i]))
		}
		if c == '\n' {

			if f.isCurrentUTF8 {
				f.CellFormat(w, h, string([]rune(s)[j:i]), "", 2, "", false, link, linkStr)
			} else {
				f.CellFormat(w, h, s[j:i], "", 2, "", false, link, linkStr)
			}
			i++
			sep = -1
			j = i
			l = 0.0
			if nl == 1 {
				f.x = f.lMargin
				w = f.w - f.rMargin - f.x
				wmax = (w - 2*f.cMargin) * 1000 / f.fontSize
			}
			nl++
			continue
		}
		if c == ' ' {
			sep = i
		}
		l += float64(cw[int(c)])
		if l > wmax {

			if sep == -1 {
				if f.x > f.lMargin {

					f.x = f.lMargin
					f.y += h
					w = f.w - f.rMargin - f.x
					wmax = (w - 2*f.cMargin) * 1000 / f.fontSize
					i++
					nl++
					continue
				}
				if i == j {
					i++
				}
				if f.isCurrentUTF8 {
					f.CellFormat(w, h, string([]rune(s)[j:i]), "", 2, "", false, link, linkStr)
				} else {
					f.CellFormat(w, h, s[j:i], "", 2, "", false, link, linkStr)
				}
			} else {
				if f.isCurrentUTF8 {
					f.CellFormat(w, h, string([]rune(s)[j:sep]), "", 2, "", false, link, linkStr)
				} else {
					f.CellFormat(w, h, s[j:sep], "", 2, "", false, link, linkStr)
				}
				i = sep + 1
			}
			sep = -1
			j = i
			l = 0.0
			if nl == 1 {
				f.x = f.lMargin
				w = f.w - f.rMargin - f.x
				wmax = (w - 2*f.cMargin) * 1000 / f.fontSize
			}
			nl++
		} else {
			i++
		}
	}

	if i != j {
		if f.isCurrentUTF8 {
			f.CellFormat(l/1000*f.fontSize, h, string([]rune(s)[j:]), "", 0, "", false, link, linkStr)
		} else {
			f.CellFormat(l/1000*f.fontSize, h, s[j:], "", 0, "", false, link, linkStr)
		}
	}
}

// Write prints text from the current position. When the right margin is
// reached (or the \n character is met) a line break occurs and text continues
// from the left margin. Upon method exit, the current position is left just at
// the end of the text.
//
// It is possible to put a link on the text.
//
// h indicates the line height in the unit of measure specified in New().
func (f *Fpdf) Write(h float64, txtStr string) {
	f.write(h, txtStr, 0, "")
}

// Writef is like Write but uses printf-style formatting. See the documentation
// for package fmt for more details on fmtStr and args.
func (f *Fpdf) Writef(h float64, fmtStr string, args ...any) {
	f.write(h, sprintf(fmtStr, args...), 0, "")
}

// WriteLinkString writes text that when clicked launches an external URL. See
// Write() for argument details.
func (f *Fpdf) WriteLinkString(h float64, displayStr, targetStr string) {
	f.write(h, displayStr, 0, targetStr)
}

// WriteLinkID writes text that when clicked jumps to another location in the
// PDF. linkID is an identifier returned by AddLink(). See Write() for argument
// details.
func (f *Fpdf) WriteLinkID(h float64, displayStr string, linkID int) {
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
func (f *Fpdf) WriteAligned(width, lineHeight float64, textStr, alignStr string) {
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
		lineStr := string(lineBt)
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
func (f *Fpdf) AddLink() int {
	f.links = append(f.links, intLinkType{})
	return len(f.links) - 1
}

// SetLink defines the page and position a link points to. See AddLink().
func (f *Fpdf) SetLink(link int, y float64, page int) {
	if y == -1 {
		y = f.y
	}
	if page == -1 {
		page = f.page
	}
	f.links[link] = intLinkType{page, y}
}

// newLink adds a new clickable link on current page
func (f *Fpdf) newLink(x, y, w, h float64, link int, linkStr string) {

	f.pageLinks[f.page] = append(f.pageLinks[f.page],
		linkType{x * f.k, f.hPt - y*f.k, w * f.k, h * f.k, link, linkStr})
}

// Link puts a link on a rectangular area of the page. Text or image links are
// generally put via Cell(), Write() or Image(), but this method can be useful
// for instance to define a clickable area inside an image. link is the value
// returned by AddLink().
func (f *Fpdf) Link(x, y, w, h float64, link int) {
	f.newLink(x, y, w, h, link, "")
}

// LinkString puts a link on a rectangular area of the page. Text or image
// links are generally put via Cell(), Write() or Image(), but this method can
// be useful for instance to define a clickable area inside an image. linkStr
// is the target URL.
func (f *Fpdf) LinkString(x, y, w, h float64, linkStr string) {
	f.newLink(x, y, w, h, 0, linkStr)
}

// Bookmark sets a bookmark that will be displayed in a sidebar outline. txtStr
// is the title of the bookmark. level specifies the level of the bookmark in
// the outline; 0 is the top level, 1 is just below, and so on. y specifies the
// vertical position of the bookmark destination in the current page; -1
// indicates the current position.
func (f *Fpdf) Bookmark(txtStr string, level int, y float64) {
	if y == -1 {
		y = f.y
	}
	if f.isCurrentUTF8 {
		txtStr = utf8toutf16(txtStr)
	}
	f.outlines = append(f.outlines, outlineType{text: txtStr, level: level, y: y, p: f.PageNo(), prev: -1, last: -1, next: -1, first: -1})
}

func (f *Fpdf) putbookmarks() {
	nb := len(f.outlines)
	if nb > 0 {
		lru := make(map[int]int)
		level := 0
		for i, o := range f.outlines {
			if o.level > 0 {
				parent := lru[o.level-1]
				f.outlines[i].parent = parent
				f.outlines[parent].last = i
				if o.level > level {
					f.outlines[parent].first = i
				}
			} else {
				f.outlines[i].parent = nb
			}
			if o.level <= level && i > 0 {
				prev := lru[o.level]
				f.outlines[prev].next = i
				f.outlines[i].prev = prev
			}
			lru[o.level] = i
			level = o.level
		}
		n := f.n + 1
		for _, o := range f.outlines {
			f.newobj()
			f.outf("<</Title %s", f.textstring(o.text))
			f.outf("/Parent %d 0 R", n+o.parent)
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
			f.outf("/Dest [%d 0 R /XYZ 0 %.2f null]", 1+2*o.p, (f.h-o.y)*f.k)
			f.out("/Count 0>>")
			f.out("endobj")
		}
		f.newobj()
		f.outlineRoot = f.n
		f.outf("<</Type /Outlines /First %d 0 R", n)
		f.outf("/Last %d 0 R>>", n+lru[0])
		f.out("endobj")
	}
}

// SplitText splits UTF-8 encoded text into several lines using the current
// font. Each line has its length limited to a maximum width given by w. This
// function can be used to determine the total height of wrapped text for
// vertical placement purposes.
func (f *Fpdf) SplitText(txt string, w float64) (lines []string) {
	cw := f.currentFont.Cw
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
		l += cw[c]
		if unicode.IsSpace(c) || isChinese(c) {
			sep = i
		}
		if c == '\n' || l > wmax {
			if sep == -1 {
				if i == j {
					i++
				}
				sep = i
			} else {
				i = sep + 1
			}
			lines = append(lines, string(s[j:sep]))
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
func (f *Fpdf) SubWrite(ht float64, str string, subFontSize, subOffset float64, link int, linkStr string) {
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
