// Package pdfscan holds a minimal byte scanner for the subset of PDF syntax
// that paper's reader, form filler and signer need in order to walk object
// dictionaries in place, without building a full object model.
//
// Every Skip* function takes the byte offset to start at and returns the offset
// just past the construct it consumed. Malformed input never panics: scanning
// runs to the end of data and returns len(data).
package pdfscan

import (
	"bytes"
	"strconv"
	"strings"
)

// IsSpace reports whether c is PDF whitespace (PDF 32000-1 §7.2.2).
func IsSpace(c byte) bool {
	return c == 0 || c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == ' '
}

// IsDelimiter reports whether c is a PDF delimiter (PDF 32000-1 §7.2.2).
func IsDelimiter(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	default:
		return false
	}
}

// SkipSpaces returns the offset of the first non-whitespace byte at or after start.
func SkipSpaces(data []byte, start int) int {
	for start < len(data) && IsSpace(data[start]) {
		start++
	}
	return start
}

// SkipToken consumes one regular token (a name, number or keyword). A single
// delimiter byte counts as a token so callers always make progress.
func SkipToken(data []byte, start int) int {
	i := start
	for i < len(data) && !IsSpace(data[i]) && !IsDelimiter(data[i]) {
		i++
	}
	if i == start && i < len(data) {
		return i + 1
	}
	return i
}

// SkipLiteral consumes a literal string "(…)", honouring nested parentheses and
// backslash escapes.
func SkipLiteral(data []byte, start int) int {
	_, end := scanLiteral(data, start, false)
	return end
}

// ParseLiteral consumes a literal string "(…)" and returns its contents with
// the outer parentheses removed and each backslash escape reduced to the byte
// that follows it.
func ParseLiteral(data []byte, start int) (string, int) {
	return scanLiteral(data, start, true)
}

func scanLiteral(data []byte, start int, capture bool) (string, int) {
	var out strings.Builder
	depth := 0
	for i := start; i < len(data); i++ {
		switch data[i] {
		case '\\':
			if i+1 < len(data) {
				if capture {
					out.WriteByte(data[i+1])
				}
				i++
			}
		case '(':
			depth++
			if capture && depth > 1 {
				out.WriteByte('(')
			}
		case ')':
			depth--
			if depth == 0 {
				return out.String(), i + 1
			}
			if capture {
				out.WriteByte(')')
			}
		default:
			if capture {
				out.WriteByte(data[i])
			}
		}
	}
	return out.String(), len(data)
}

// SkipArray consumes an array "[…]", honouring nesting and literal strings. It
// returns the index just past the array and reports whether the closing bracket
// was found. Callers that slice the array's interior must check: an unterminated
// array ends at len(data), and the interior bounds [start+1 : end-1] are then
// invalid for a trailing "[".
func SkipArray(data []byte, start int) (int, bool) {
	depth := 0
	for i := start; i < len(data); i++ {
		switch data[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i + 1, true
			}
		case '(':
			i = SkipLiteral(data, i) - 1
		}
	}
	return len(data), false
}

// SkipBalanced consumes a construct delimited by the multi-byte open/closing
// markers (typically "<<" and ">>"), honouring nesting and literal strings.
func SkipBalanced(data []byte, start int, open, closing []byte) int {
	depth := 0
	for i := start; i < len(data); {
		switch {
		case bytes.HasPrefix(data[i:], open):
			depth++
			i += len(open)
		case bytes.HasPrefix(data[i:], closing):
			depth--
			i += len(closing)
			if depth == 0 {
				return i
			}
		case data[i] == '(':
			i = SkipLiteral(data, i)
		default:
			i++
		}
	}
	return len(data)
}

// SkipIndirectRef consumes an indirect reference ("12 0 R"). It reports false
// and leaves the offset untouched when the bytes at start are something else.
func SkipIndirectRef(data []byte, start int) (int, bool) {
	start = SkipSpaces(data, start)
	numberEnd := func(from int) (int, bool) {
		end := SkipToken(data, from)
		if end <= from {
			return 0, false
		}
		if _, err := strconv.Atoi(string(data[from:end])); err != nil {
			return 0, false
		}
		return end, true
	}
	firstEnd, ok := numberEnd(start)
	if !ok {
		return start, false
	}
	secondEnd, ok := numberEnd(SkipSpaces(data, firstEnd))
	if !ok {
		return start, false
	}
	refStart := SkipSpaces(data, secondEnd)
	if refStart >= len(data) || data[refStart] != 'R' {
		return start, false
	}
	return refStart + 1, true
}

// SkipValue consumes one PDF value: a dictionary, indirect reference, array,
// literal string or bare token.
func SkipValue(data []byte, start int) int {
	start = SkipSpaces(data, start)
	if start >= len(data) {
		return start
	}
	if bytes.HasPrefix(data[start:], []byte("<<")) {
		return SkipBalanced(data, start, []byte("<<"), []byte(">>"))
	}
	if end, ok := SkipIndirectRef(data, start); ok {
		return end
	}
	switch data[start] {
	case '[':
		end, _ := SkipArray(data, start)
		return end
	case '(':
		return SkipLiteral(data, start)
	default:
		return SkipToken(data, start)
	}
}

// IsNameChar reports whether c can appear in a PDF name after the leading '/'.
func IsNameChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '-' || c == '.' || c == '#'
}

// KeyIndex returns the offset of the "/key" entry in a dictionary, or -1 when
// absent. Longer names that merely start with key (e.g. "/Type" for "/Ty") are
// skipped.
func KeyIndex(content []byte, key string) int {
	marker := []byte("/" + key)
	searchFrom := 0
	for searchFrom < len(content) {
		idx := bytes.Index(content[searchFrom:], marker)
		if idx < 0 {
			return -1
		}
		idx += searchFrom
		after := idx + len(marker)
		if after >= len(content) || !IsNameChar(content[after]) {
			return idx
		}
		searchFrom = after
	}
	return -1
}

// RemoveDictEntry drops the "/key value" pair from a dictionary body, returning
// the dictionary unchanged when the key is absent.
func RemoveDictEntry(dictionary []byte, key string) []byte {
	idx := KeyIndex(dictionary, key)
	if idx < 0 {
		return dictionary
	}
	valueStart := SkipSpaces(dictionary, idx+len(key)+1)
	valueEnd := SkipValue(dictionary, valueStart)
	if valueEnd <= valueStart {
		return dictionary
	}
	out := make([]byte, 0, len(dictionary)-(valueEnd-idx))
	out = append(out, bytes.TrimRight(dictionary[:idx], " \t\r\n")...)
	out = append(out, ' ')
	out = append(out, bytes.TrimLeft(dictionary[valueEnd:], " \t\r\n")...)
	return out
}
