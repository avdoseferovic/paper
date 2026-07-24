package translate

import "strings"

// safeLinkSchemes lists the URI schemes an <a href> may carry into a PDF link
// annotation. A viewer hands a /URI action straight to the operating system, so
// an unfiltered href lets untrusted HTML emit javascript: (script execution in
// some viewers), file: (local file access), or smb:// links (credential-leaking
// network requests) from a document that merely looks like text.
var safeLinkSchemes = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
	"tel":    true,
}

// sanitizedHyperlink returns the href to attach to a link annotation. Relative
// references pass through unchanged; absolute URLs must use a safe scheme.
// Reports false when the href must not become a link.
func sanitizedHyperlink(href string) (string, bool) {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return "", false
	}
	// A real URL percent-encodes control characters. Their presence means the
	// href is trying to look different to the parser than to the viewer, as in
	// "java\tscript:alert(1)".
	if strings.ContainsFunc(trimmed, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return "", false
	}
	scheme, ok := uriScheme(trimmed)
	if !ok {
		return trimmed, true
	}
	if !safeLinkSchemes[scheme] {
		return "", false
	}
	return trimmed, true
}

// uriScheme returns the lower-cased scheme of an absolute URI reference. It
// reports false for relative references, following RFC 3986: a colon only
// introduces a scheme when everything before it is a valid scheme token.
func uriScheme(reference string) (string, bool) {
	colon := strings.IndexByte(reference, ':')
	if colon <= 0 {
		return "", false
	}
	candidate := reference[:colon]
	for i := range len(candidate) {
		c := candidate[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9', c == '+', c == '-', c == '.':
			if i == 0 {
				// A scheme must start with a letter, so this is a path or host.
				return "", false
			}
		default:
			return "", false
		}
	}
	return strings.ToLower(candidate), true
}
