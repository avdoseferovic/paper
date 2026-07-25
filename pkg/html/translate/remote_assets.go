package translate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultRemoteAssetBytes      = 32 << 20
	defaultRemoteStylesheetBytes = 10 << 20
)

var errHTTPStatus = errors.New("html: HTTP status error")

func (tr *translator) effectiveStylesheetResolver(ctx context.Context) StylesheetResolver {
	if tr == nil {
		return safeDefaultStylesheetResolver
	}
	resolver := tr.stylesheetResolver
	if resolver == nil {
		resolver = safeDefaultStylesheetResolver
	}
	if !tr.remoteAssets {
		return resolver
	}
	return func(href string) ([]byte, error) {
		if isHTTPURL(href) {
			return tr.fetchRemoteURL(ctx, href, defaultRemoteStylesheetBytes)
		}
		return resolver(href)
	}
}

func (tr *translator) fetchRemoteURL(ctx context.Context, rawURL string, maxBytes int64) ([]byte, error) {
	if tr != nil && tr.urlPolicy != nil {
		err := tr.urlPolicy(rawURL)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrURLPolicyDenied, err)
		}
	}
	if maxBytes <= 0 {
		maxBytes = defaultRemoteAssetBytes
	}
	var client *http.Client
	if tr != nil {
		client = tr.httpClient
	}
	return httpGetBytes(ctx, httpClientOrDefault(client), rawURL, maxBytes)
}

func httpClientOrDefault(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return http.DefaultClient
}

func httpGetBytes(ctx context.Context, client *http.Client, rawURL string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("html: building request for %s: %w", rawURL, err)
	}
	// Remote asset fetching is explicit opt-in (WithRemoteAssets); URLPolicy
	// lets callers gate the reachable targets.
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("html: fetch %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("html: fetch %s: %w: %s", rawURL, errHTTPStatus, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, fmt.Errorf("html: reading %s: %w", rawURL, err)
	}
	return data, nil
}

func isHTTPURL(value string) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func extFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil && u.Path != "" {
		return extFromFilename(u.Path)
	}
	return extFromFilename(rawURL)
}

// installRemoteImageResolver routes http(s) <img> sources through the remote
// fetcher when remote assets are enabled and no custom resolver is set. The
// returned resolver captures ctx so image fetches observe cancellation.
func (tr *translator) installRemoteImageResolver(ctx context.Context) {
	if tr == nil || !tr.remoteAssets || tr.imageResolver != nil {
		return
	}
	baseDir := tr.imageBaseDir
	limits := tr.limits
	tr.imageResolver = func(src string) ([]byte, string, error) {
		if isHTTPURL(src) {
			data, err := tr.fetchRemoteURL(ctx, src, defaultRemoteAssetBytes)
			if err != nil {
				return nil, "", err
			}
			return data, extFromURL(src), nil
		}
		if baseDir != "" {
			return baseDirResolverWithLimits(baseDir, limits)(src)
		}
		return safeDefaultResolverWithLimits(src, limits)
	}
}
