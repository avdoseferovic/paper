package paper

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/breakline"
	"github.com/avdoseferovic/paper/pkg/consts/fontfamily"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type Text struct {
	pdf                   gofpdfwrapper.Fpdf
	math                  core.Math
	font                  core.Font
	layoutCache           map[textLayoutKey][]string
	defaultCodeTranslator func(string) string
}

type textLayoutKey struct {
	text     string
	family   string
	style    string
	size     float64
	width    float64
	strategy string
}

// NewText create a Text.
func NewText(pdf gofpdfwrapper.Fpdf, math core.Math, font core.Font) *Text {
	return &Text{
		pdf:         pdf,
		math:        math,
		font:        font,
		layoutCache: make(map[textLayoutKey][]string),
	}
}

// Add a text inside a cell.
func (s *Text) Add(text string, cell *entity.Cell, textProp *props.Text) {
	s.font.SetFont(textProp.Family, textProp.Style, textProp.Size)
	fontHeight := s.font.GetHeight(textProp.Family, textProp.Style, textProp.Size)

	if textProp.Top > cell.Height {
		textProp.Top = cell.Height
	}

	if textProp.Left > cell.Width {
		textProp.Left = cell.Width
	}

	if textProp.Right > cell.Width {
		textProp.Right = cell.Width
	}

	width := cell.Width - textProp.Left - textProp.Right
	if width < 0 {
		width = 0
	}

	x := cell.X + textProp.Left
	y := cell.Y + textProp.Top

	originalColor := s.font.GetColor()
	if textProp.Color != nil {
		s.font.SetColor(textProp.Color)
	}

	// override style if hyperlink is set
	if textProp.Hyperlink != nil {
		blue := props.Blue()
		s.font.SetColor(&blue)
	}

	y += fontHeight

	// Apply Unicode before calc spaces
	unicodeText := s.textToUnicode(text, textProp)
	stringWidth := s.pdf.GetStringWidth(unicodeText)

	// If should add one line
	if stringWidth <= width {
		s.addLine(textProp, x, width, y, stringWidth, unicodeText)
		s.font.SetColor(originalColor)
		return
	}

	lines := s.getCachedLines(unicodeText, textProp, width)

	accumulateOffsetY := 0.0

	for index, line := range lines {
		lineWidth := s.pdf.GetStringWidth(line)

		s.addLine(textProp, x, width, y+float64(index)*fontHeight+accumulateOffsetY, lineWidth, line)
		accumulateOffsetY += textProp.VerticalPadding
	}

	s.font.SetColor(originalColor)
}

// GetLinesQuantity retrieve the quantity of lines which a text will occupy to avoid that text to extrapolate a cell.
func (s *Text) GetLinesQuantity(text string, textProp *props.Text, colWidth float64) int {
	s.font.SetFont(textProp.Family, textProp.Style, textProp.Size)

	textTranslated := s.textToUnicode(text, textProp)

	if textProp.BreakLineStrategy == breakline.DashStrategy {
		lines := s.getLinesBreakingLineWithDash(text, colWidth)
		s.setCachedLines(textTranslated, textProp, colWidth, lines)
		return len(lines)
	}

	lines := s.getLinesBreakingLineFromSpace(strings.Split(textTranslated, " "), colWidth)
	s.setCachedLines(textTranslated, textProp, colWidth, lines)
	return len(lines)
}

func (s *Text) getCachedLines(text string, textProp *props.Text, width float64) []string {
	key := s.textLayoutKey(text, textProp, width)
	if lines, ok := s.layoutCache[key]; ok {
		return lines
	}

	var lines []string
	if textProp.BreakLineStrategy == breakline.DashStrategy {
		lines = s.getLinesBreakingLineWithDash(text, width)
	} else {
		lines = s.getLinesBreakingLineFromSpace(strings.Split(text, " "), width)
	}
	s.layoutCache[key] = lines
	return lines
}

func (s *Text) setCachedLines(text string, textProp *props.Text, width float64, lines []string) {
	s.layoutCache[s.textLayoutKey(text, textProp, width)] = lines
}

func (s *Text) textLayoutKey(text string, textProp *props.Text, width float64) textLayoutKey {
	return textLayoutKey{
		text:     text,
		family:   textProp.Family,
		style:    string(textProp.Style),
		size:     textProp.Size,
		width:    width,
		strategy: string(textProp.BreakLineStrategy),
	}
}

func (s *Text) getLinesBreakingLineFromSpace(words []string, colWidth float64) []string {
	currentlySize := 0.0
	lines := []string{}

	for _, word := range words {
		if word == "" {
			continue
		}
		var piece, separator string
		if len(lines) == 0 || lines[len(lines)-1] == "" {
			piece = word
			separator = ""
		} else {
			piece = " " + word
			separator = " "
		}

		width := s.pdf.GetStringWidth(piece)
		if currentlySize+width <= colWidth {
			if len(lines) == 0 {
				lines = append(lines, "")
			}
			lines[len(lines)-1] += separator + word
			currentlySize += width
		} else {
			lines = append(lines, word)
			currentlySize = s.pdf.GetStringWidth(word)
		}
	}

	return lines
}

func (s *Text) getLinesBreakingLineWithDash(words string, colWidth float64) []string {
	currentlySize := 0.0

	lines := []string{}

	dashSize := s.pdf.GetStringWidth(" - ")

	var content string
	for _, letter := range words {
		if currentlySize+dashSize > colWidth-dashSize {
			content += "-"
			lines = append(lines, content)
			content = ""
			currentlySize = 0
		}

		letterString := fmt.Sprintf("%c", letter)
		width := s.pdf.GetStringWidth(letterString)
		content += letterString
		currentlySize += width
	}

	if content != "" {
		lines = append(lines, content)
	}

	return lines
}

func (s *Text) addLine(textProp *props.Text, xColOffset, colWidth, yColOffset, textWidth float64, text string) {
	left, top, _, _ := s.pdf.GetMargins()

	fontHeight := s.font.GetHeight(textProp.Family, textProp.Style, textProp.Size)

	if textProp.Align == align.Left {
		s.pdf.Text(xColOffset+left, yColOffset+top, text)

		if textProp.Hyperlink != nil {
			s.pdf.LinkString(xColOffset+left, yColOffset+top-fontHeight, textWidth, fontHeight, *textProp.Hyperlink)
		}

		return
	}

	if textProp.Align == align.Justify {
		const spaceString = " "
		const emptyString = ""

		text = strings.TrimRight(text, spaceString)
		textNotSpaces := strings.ReplaceAll(text, spaceString, emptyString)
		textWidth = s.pdf.GetStringWidth(textNotSpaces)
		defaultSpaceWidth := s.pdf.GetStringWidth(spaceString)
		words := strings.Fields(text)

		numSpaces := max(len(words)-1, 1)
		spaceWidth := (colWidth - textWidth) / float64(numSpaces)
		x := xColOffset + left

		if isIncorrectSpaceWidth(textWidth, spaceWidth, defaultSpaceWidth, textNotSpaces) {
			spaceWidth = defaultSpaceWidth
		}
		initX := x
		var finishX float64
		for _, word := range words {
			s.pdf.Text(x, yColOffset+top, word)
			finishX = x + s.pdf.GetStringWidth(word)
			x = finishX + spaceWidth
		}

		if textProp.Hyperlink != nil {
			s.pdf.LinkString(initX, yColOffset+top-fontHeight, finishX-initX, fontHeight, *textProp.Hyperlink)
		}

		return
	}

	var modifier float64 = 2

	if textProp.Align == align.Right {
		modifier = 1
	}

	dx := (colWidth - textWidth) / modifier

	if textProp.Hyperlink != nil {
		s.pdf.LinkString(dx+xColOffset+left, yColOffset+top-fontHeight, textWidth, fontHeight, *textProp.Hyperlink)
	}

	s.pdf.Text(dx+xColOffset+left, yColOffset+top, text)
}

func (s *Text) textToUnicode(txt string, props *props.Text) string {
	if props.Family == fontfamily.Arial ||
		props.Family == fontfamily.Helvetica ||
		props.Family == fontfamily.Symbol ||
		props.Family == fontfamily.ZapBats ||
		props.Family == fontfamily.Courier {
		return s.translateDefaultCodePage(txt)
	}

	return txt
}

func (s *Text) translateDefaultCodePage(txt string) string {
	if s.defaultCodeTranslator == nil {
		s.defaultCodeTranslator = s.pdf.UnicodeTranslatorFromDescriptor("")
	}
	return s.defaultCodeTranslator(txt)
}

func isIncorrectSpaceWidth(textWidth, spaceWidth, defaultSpaceWidth float64, text string) bool {
	if textWidth <= 0 || spaceWidth <= defaultSpaceWidth*10 {
		return false
	}

	r, _ := utf8.DecodeLastRuneInString(text)
	lastChar := r
	return !unicode.IsLetter(lastChar) && !unicode.IsNumber(lastChar)
}
