package merge

import (
	"bytes"
	"regexp"
)

// inheritablePageAttrNames are the page-tree attributes that PDF 32000-1
// §7.7.3.4 lets leaf pages inherit from ancestor /Pages nodes. The merged
// output re-parents every page to a fresh /Pages node, so inherited values
// must be materialized onto each page dictionary or they are lost.
var inheritablePageAttrNames = []string{"Resources", "MediaBox", "CropBox", "Rotate"}

type pageAttr struct {
	name  string
	value []byte
}

var indirectRefValueRe = regexp.MustCompile(`^\d+\s+\d+\s+R`)

// dictHasName reports whether the dictionary content defines /name itself.
func dictHasName(content []byte, name string) bool {
	needle := []byte("/" + name)
	for offset := 0; ; {
		idx := bytes.Index(content[offset:], needle)
		if idx < 0 {
			return false
		}
		end := offset + idx + len(needle)
		if end >= len(content) || !isPDFNameChar(content[end]) {
			return true
		}
		offset = end
	}
}

func isPDFNameChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '#':
		return true
	default:
		return false
	}
}

// dictNameValue extracts the raw value bytes of /name from dictionary
// content: an array, an inline dictionary, an indirect reference, or a
// single token.
func dictNameValue(content []byte, name string) ([]byte, bool) {
	needle := []byte("/" + name)
	offset := 0
	for {
		idx := bytes.Index(content[offset:], needle)
		if idx < 0 {
			return nil, false
		}
		start := offset + idx + len(needle)
		if start < len(content) && isPDFNameChar(content[start]) {
			offset = start
			continue
		}
		value := content[start:]
		value = bytes.TrimLeft(value, " \t\r\n")
		return scanDictValue(value)
	}
}

func scanDictValue(value []byte) ([]byte, bool) {
	if len(value) == 0 {
		return nil, false
	}
	switch {
	case value[0] == '[':
		return scanBalanced(value, '[', ']')
	case bytes.HasPrefix(value, []byte("<<")):
		return scanBalancedDict(value)
	default:
		if match := indirectRefValueRe.Find(value); match != nil {
			return match, true
		}
		end := bytes.IndexAny(value, " \t\r\n/[]<>()")
		if end < 0 {
			end = len(value)
		}
		if end == 0 {
			return nil, false
		}
		return value[:end], true
	}
}

func scanBalanced(value []byte, open, closing byte) ([]byte, bool) {
	depth := 0
	for i, c := range value {
		switch c {
		case open:
			depth++
		case closing:
			depth--
			if depth == 0 {
				return value[:i+1], true
			}
		}
	}
	return nil, false
}

func scanBalancedDict(value []byte) ([]byte, bool) {
	depth := 0
	for i := 0; i+1 < len(value); i++ {
		switch {
		case value[i] == '<' && value[i+1] == '<':
			depth++
			i++
		case value[i] == '>' && value[i+1] == '>':
			depth--
			i++
			if depth == 0 {
				return value[:i+1], true
			}
		}
	}
	return nil, false
}

// mergeInheritedAttrs layers a /Pages node's inheritable attributes over the
// ones collected from farther ancestors; nearer nodes win.
func mergeInheritedAttrs(inherited []pageAttr, pagesContent []byte) []pageAttr {
	merged := append([]pageAttr(nil), inherited...)
	for _, name := range inheritablePageAttrNames {
		value, ok := dictNameValue(pagesContent, name)
		if !ok {
			continue
		}
		replaced := false
		for i := range merged {
			if merged[i].name == name {
				merged[i].value = value
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, pageAttr{name: name, value: value})
		}
	}
	return merged
}

// injectInheritedAttrs inserts every inherited attribute the page dictionary
// does not define itself, right after the opening << of the page dictionary.
func injectInheritedAttrs(content []byte, inherited []pageAttr) []byte {
	if len(inherited) == 0 {
		return content
	}
	open := bytes.Index(content, []byte("<<"))
	if open < 0 {
		return content
	}
	var injected bytes.Buffer
	for _, attr := range inherited {
		if dictHasName(content, attr.name) {
			continue
		}
		injected.WriteString("/" + attr.name + " ")
		injected.Write(attr.value)
		injected.WriteString("\n")
	}
	if injected.Len() == 0 {
		return content
	}
	var out bytes.Buffer
	out.Write(content[:open+2])
	out.WriteString("\n")
	out.Write(injected.Bytes())
	out.Write(content[open+2:])
	return out.Bytes()
}
