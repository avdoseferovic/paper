package translate

import (
	"errors"
	"fmt"
	"strings"
)

// StylesheetResolver loads CSS text for a given <link href="…"> value.
// Returns the raw bytes (UTF-8 CSS text) and any error.
type StylesheetResolver func(href string) ([]byte, error)

// ErrStylesheetResolverRefused is returned by the default resolver when asked
// to load a non-data: URI without an explicit WithStylesheetBaseDir.
var ErrStylesheetResolverRefused = errors.New("html: default stylesheet resolver refuses local file reads; configure WithStylesheetBaseDir")

// safeDefaultStylesheetResolver only accepts data: URIs.
func safeDefaultStylesheetResolver(href string) ([]byte, error) {
	if strings.HasPrefix(href, "data:") {
		return decodeCSSDataURI(href)
	}
	return nil, ErrStylesheetResolverRefused
}

// decodeCSSDataURI handles data:text/css,... data:text/css;base64,...
func decodeCSSDataURI(uri string) ([]byte, error) {
	prefix := strings.TrimPrefix(uri, "data:")
	header, payload, ok := strings.Cut(prefix, ",")
	if !ok {
		return nil, errDataURIInvalid
	}
	if strings.Contains(header, "base64") {
		// Use the existing image data-URI decoder for consistency.
		dec, _, err := decodeDataURI(uri)
		if err != nil {
			return nil, fmt.Errorf("html: decoding stylesheet data URI: %w", err)
		}
		return dec, nil
	}
	return []byte(payload), nil
}

// stylesheetBaseDirResolver returns a resolver that only loads files inside
// dir. Confinement (".." traversal, absolute paths, and out-of-root symlinks)
// is enforced by readFileInRoot via os.Root, mirroring the image resolver. An
// empty dir is refused outright (see readFileInRoot).
func stylesheetBaseDirResolver(dir string) StylesheetResolver {
	return func(href string) ([]byte, error) {
		if strings.HasPrefix(href, "data:") {
			return decodeCSSDataURI(href)
		}
		return readFileInRoot(dir, href)
	}
}

// safeLoadStylesheet wraps a resolver call in defer/recover so a malformed
// URI or panicking resolver never crashes the caller. Returns the bytes
// (nil on failure) and a flag indicating whether the load succeeded.
func safeLoadStylesheet(resolver StylesheetResolver, href string) ([]byte, bool) {
	var data []byte
	ok := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				data = nil
				ok = false
			}
		}()
		d, err := resolver(href)
		if err != nil {
			return
		}
		data = d
		ok = true
	}()
	return data, ok
}
