package sign

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// defaultNetworkTimeout bounds one TSA or OCSP round trip. http.DefaultClient
	// sets no timeout, so a responder that accepts the connection and then stalls
	// would otherwise hang signing indefinitely.
	defaultNetworkTimeout = 30 * time.Second

	// maxResponseBytes caps a TSA or OCSP response body. Timestamp tokens and
	// OCSP responses run to a few kilobytes; the endpoint is caller configuration
	// or, for OCSP, a URL taken from the certificate being validated, so the body
	// is not necessarily friendly.
	maxResponseBytes = 4 << 20
)

var errResponseTooLarge = errors.New("sign: response body exceeds size limit")

// postDER posts a DER-encoded request and returns the response body along with
// its status code. The body is read under a size cap and the request under a
// timeout, so neither can be used to exhaust the caller.
func postDER(client *http.Client, url, contentType string, body []byte, timeout time.Duration) ([]byte, int, error) {
	if timeout <= 0 {
		timeout = defaultNetworkTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("sign: build HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req) // #nosec G704 -- endpoint is caller configuration or the validated certificate's responder.
	if err != nil {
		return nil, 0, fmt.Errorf("sign: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("sign: read response: %w", err)
	}
	if len(data) > maxResponseBytes {
		return nil, resp.StatusCode, fmt.Errorf("%w: %d bytes", errResponseTooLarge, maxResponseBytes)
	}
	return data, resp.StatusCode, nil
}
