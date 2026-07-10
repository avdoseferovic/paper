package sign

import (
	"bytes"
	"context"
	"crypto"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	errTSAURLEmpty       = errors.New("sign: TSA URL is empty")
	errTSAStatus         = errors.New("sign: TSA returned non-OK status")
	errUnsupportedHash   = errors.New("sign: unsupported hash function")
	errTSARejected       = errors.New("sign: TSA rejected request")
	errTSATokenMissing   = errors.New("sign: TSA response contains no timestamp token")
	digestOIDsByHashFunc = map[crypto.Hash]asn1.ObjectIdentifier{
		crypto.SHA256: oidSHA256,
		crypto.SHA384: oidSHA384,
		crypto.SHA512: oidSHA512,
	}
)

// TSAClient is an RFC 3161 Time-Stamp Authority client.
type TSAClient struct {
	URL        string
	HTTPClient *http.Client
}

// NewTSAClient creates a TSA client for url.
func NewTSAClient(url string) *TSAClient {
	return &TSAClient{URL: url}
}

// Timestamp sends an RFC 3161 timestamp request and returns the timestamp token.
func (c *TSAClient) Timestamp(digest []byte, hashFunc crypto.Hash) ([]byte, error) {
	if c == nil || c.URL == "" {
		return nil, errTSAURLEmpty
	}
	reqDER, err := buildTimestampReq(digest, hashFunc)
	if err != nil {
		return nil, fmt.Errorf("sign: build TSA request: %w", err)
	}
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		c.URL,
		bytes.NewReader(reqDER),
	)
	if err != nil {
		return nil, fmt.Errorf("sign: build TSA HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/timestamp-query")

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req) // #nosec G704 -- TSA endpoint is caller configuration, not PDF input.
	if err != nil {
		return nil, fmt.Errorf("sign: TSA request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errTSAStatus, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sign: read TSA response: %w", err)
	}
	return parseTimestampResp(body)
}

type timeStampReq struct {
	Version        int
	MessageImprint messageImprint
	CertReq        bool `asn1:"optional"`
}

type messageImprint struct {
	HashAlgorithm algorithmIdentifier
	HashedMessage []byte
}

type timeStampResp struct {
	Status         pkiStatusInfo
	TimeStampToken asn1.RawValue `asn1:"optional"`
}

type pkiStatusInfo struct {
	Status int
}

func buildTimestampReq(digest []byte, hashFunc crypto.Hash) ([]byte, error) {
	hashOID, err := digestOIDForHash(hashFunc)
	if err != nil {
		return nil, err
	}
	req := timeStampReq{
		Version: 1,
		MessageImprint: messageImprint{
			HashAlgorithm: algorithmIdentifier{Algorithm: hashOID},
			HashedMessage: digest,
		},
		CertReq: true,
	}
	out, err := asn1.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal TSA request: %w", err)
	}
	return out, nil
}

func digestOIDForHash(hashFunc crypto.Hash) (asn1.ObjectIdentifier, error) {
	if oid, ok := digestOIDsByHashFunc[hashFunc]; ok {
		return oid, nil
	}
	return nil, fmt.Errorf("%w: %v", errUnsupportedHash, hashFunc)
}

func parseTimestampResp(data []byte) ([]byte, error) {
	var resp timeStampResp
	_, err := asn1.Unmarshal(data, &resp)
	if err != nil {
		return nil, fmt.Errorf("sign: parse TSA response: %w", err)
	}
	if resp.Status.Status > 1 {
		return nil, fmt.Errorf("%w: status %d", errTSARejected, resp.Status.Status)
	}
	if len(resp.TimeStampToken.FullBytes) == 0 {
		return nil, errTSATokenMissing
	}
	return resp.TimeStampToken.FullBytes, nil
}
