package html_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html"
)

const remoteCSS = "p { color: rgb(200, 0, 0); }"

func newAssetServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/style.css", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(remoteCSS))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestWithRemoteAssetsLoadsRemoteStylesheet(t *testing.T) {
	t.Parallel()

	server := newAssetServer(t)
	rows, err := html.FromString(context.Background(),
		`<html><head><link rel="stylesheet" href="`+server.URL+`/style.css"></head><body><p>styled</p></body></html>`,
		html.WithRemoteAssets(),
	)

	require.NoError(t, err)
	assert.NotEmpty(t, rows)
}

func TestWithURLPolicyBlocksFetches(t *testing.T) {
	t.Parallel()

	server := newAssetServer(t)
	denied := errors.New("denied by policy")
	rows, err := html.FromString(context.Background(),
		`<html><head><link rel="stylesheet" href="`+server.URL+`/style.css"></head><body><p>ok</p></body></html>`,
		html.WithRemoteAssets(),
		html.WithURLPolicy(func(string) error { return denied }),
	)

	// Policy denials warn-and-continue; the document still renders.
	require.NoError(t, err)
	assert.NotEmpty(t, rows)
}

func TestWithHTTPClientIsUsedForFetches(t *testing.T) {
	t.Parallel()

	server := newAssetServer(t)
	used := false
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		used = true
		return http.DefaultTransport.RoundTrip(req)
	})}
	_, err := html.FromString(context.Background(),
		`<html><head><link rel="stylesheet" href="`+server.URL+`/style.css"></head><body><p>ok</p></body></html>`,
		html.WithRemoteAssets(),
		html.WithHTTPClient(client),
	)

	require.NoError(t, err)
	assert.True(t, used, "custom HTTP client must serve remote asset fetches")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestWithFallbackFontPathRendersNonWinAnsiText(t *testing.T) {
	t.Parallel()

	const fontPath = "../../docs/assets/fonts/arial-unicode-ms.ttf"
	_, statErr := os.Stat(fontPath)
	require.NoError(t, statErr)

	rows, err := html.FromString(context.Background(),
		`<html><body><p>Zażółć gęślą jaźń</p></body></html>`,
		html.WithFallbackFontPath(fontPath),
	)

	require.NoError(t, err)
	assert.NotEmpty(t, rows)
}
