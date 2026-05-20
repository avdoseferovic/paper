package translate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StylesheetResolver loads CSS text for a given <link href="…"> value.
// Returns the raw bytes (UTF-8 CSS text) and any error.
type StylesheetResolver func(href string) ([]byte, error)

// ErrStylesheetResolverRefused is returned by the default resolver when asked
// to load a non-data: URI without an explicit WithStylesheetBaseDir or
// WithStylesheetResolver.
var ErrStylesheetResolverRefused = errors.New("html: default stylesheet resolver refuses local file reads; configure WithStylesheetBaseDir or WithStylesheetResolver")

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
	commaIdx := strings.IndexByte(prefix, ',')
	if commaIdx < 0 {
		return nil, fmt.Errorf("html: invalid data URI")
	}
	header := prefix[:commaIdx]
	payload := prefix[commaIdx+1:]
	if strings.Contains(header, "base64") {
		// Use the existing image data-URI decoder for consistency.
		dec, _, err := decodeDataURI(uri)
		if err != nil {
			return nil, err
		}
		return dec, nil
	}
	return []byte(payload), nil
}

// stylesheetBaseDirResolver returns a resolver that only loads files inside
// dir, rejecting any path that would escape via "../" or absolute prefix.
// Mirrors the image baseDirResolver safety model exactly.
//
// Security: when dir is empty or filepath.Abs fails, the returned resolver
// errors on every call to prevent collapsing the prefix guard to "" (which
// would allow CWD-relative reads).
func stylesheetBaseDirResolver(dir string) StylesheetResolver {
	if dir == "" {
		return func(string) ([]byte, error) {
			return nil, fmt.Errorf("html: stylesheet base dir is empty; refusing all reads")
		}
	}
	cleanBase, err := filepath.Abs(filepath.Clean(dir))
	if err != nil || cleanBase == "" {
		return func(string) ([]byte, error) {
			return nil, fmt.Errorf("html: stylesheet base dir %q is invalid: %v", dir, err)
		}
	}
	return func(href string) ([]byte, error) {
		if strings.HasPrefix(href, "data:") {
			return decodeCSSDataURI(href)
		}
		if filepath.IsAbs(href) {
			return nil, fmt.Errorf("html: absolute path %q refused outside base dir", href)
		}
		full, err := filepath.Abs(filepath.Clean(filepath.Join(cleanBase, href)))
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(full, cleanBase+string(filepath.Separator)) && full != cleanBase {
			return nil, fmt.Errorf("html: path %q escapes base dir", href)
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
}

// safeLoadStylesheet wraps a resolver call in defer/recover so a malformed
// URI or panicking resolver never crashes the caller. Returns the bytes
// (nil on failure) and a flag indicating whether the load succeeded.
func safeLoadStylesheet(resolver StylesheetResolver, href string) (data []byte, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			data = nil
			ok = false
		}
	}()
	d, err := resolver(href)
	if err != nil {
		return nil, false
	}
	return d, true
}
