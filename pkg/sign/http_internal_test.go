package sign

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestPostDER_RefusesOversizedBody covers a responder that streams more than the
// cap. Timestamp tokens and OCSP responses are kilobytes, and the endpoint can be
// chosen by the certificate under validation, so an unbounded read was a way to
// exhaust memory during signing.
func TestPostDER_RefusesOversizedBody(t *testing.T) {
	t.Parallel()

	chunk := strings.Repeat("A", 1<<20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for range (maxResponseBytes / len(chunk)) + 2 {
			_, _ = w.Write([]byte(chunk))
		}
	}))
	defer server.Close()

	_, _, err := postDER(server.Client(), server.URL, "application/ocsp-request", []byte{0x30, 0x00}, time.Minute)

	if !errors.Is(err, errResponseTooLarge) {
		t.Fatalf("postDER() error = %v, want %v", err, errResponseTooLarge)
	}
}

func TestPostDER_ReturnsBodyAndStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/timestamp-query" {
			t.Errorf("Content-Type = %q, want application/timestamp-query", got)
		}
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("token"))
	}))
	defer server.Close()

	body, status, err := postDER(server.Client(), server.URL, "application/timestamp-query", []byte{0x30, 0x00}, time.Minute)
	if err != nil {
		t.Fatalf("postDER() error = %v", err)
	}
	if status != http.StatusTeapot || string(body) != "token" {
		t.Fatalf("postDER() = (%q, %d), want (\"token\", %d)", body, status, http.StatusTeapot)
	}
}

// TestPostDER_TimesOut proves a stalled responder cannot hang signing: neither
// http.DefaultClient nor a caller-supplied client necessarily sets a timeout.
func TestPostDER_TimesOut(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)

	_, _, err := postDER(server.Client(), server.URL, "application/ocsp-request", []byte{0x30, 0x00}, 50*time.Millisecond)

	if err == nil {
		t.Fatal("postDER() error = nil, want a timeout")
	}
}
