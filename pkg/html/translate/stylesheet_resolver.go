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
// to load a non-data: URI without an explicit WithStylesheetBaseDir.
var ErrStylesheetResolverRefused = errors.New("html: default stylesheet resolver refuses local file reads; configure WithStylesheetBaseDir")

var (
	errStylesheetBaseDirEmpty   = errors.New("html: stylesheet base dir is empty; refusing all reads")
	errStylesheetBaseDirInvalid = errors.New("html: stylesheet base dir is invalid")
)

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
// dir, rejecting any path that would escape via "../" or absolute prefix.
// Mirrors the image baseDirResolver safety model exactly.
//
// Security: when dir is empty or filepath.Abs fails, the returned resolver
// errors on every call to prevent collapsing the prefix guard to "" (which
// would allow CWD-relative reads).
func stylesheetBaseDirResolver(dir string) StylesheetResolver {
	if dir == "" {
		return func(string) ([]byte, error) {
			return nil, errStylesheetBaseDirEmpty
		}
	}
	cleanBase, err := filepath.Abs(filepath.Clean(dir))
	if err != nil || cleanBase == "" {
		return func(string) ([]byte, error) {
			if err != nil {
				return nil, fmt.Errorf("%w: %q: %w", errStylesheetBaseDirInvalid, dir, err)
			}
			return nil, fmt.Errorf("%w: %q", errStylesheetBaseDirInvalid, dir)
		}
	}
	return func(href string) ([]byte, error) {
		if strings.HasPrefix(href, "data:") {
			return decodeCSSDataURI(href)
		}
		if filepath.IsAbs(href) {
			return nil, fmt.Errorf("%w: %q", errAbsolutePathRefused, href)
		}
		full, err := filepath.Abs(filepath.Clean(filepath.Join(cleanBase, href)))
		if err != nil {
			return nil, fmt.Errorf("html: resolving stylesheet path: %w", err)
		}
		if !strings.HasPrefix(full, cleanBase+string(filepath.Separator)) && full != cleanBase {
			return nil, fmt.Errorf("%w: %q", errPathEscapesBaseDir, href)
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("html: reading stylesheet: %w", err)
		}
		return data, nil
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
