package sign

import (
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// OCSPClient fetches OCSP responses for certificate revocation checking.
type OCSPClient struct {
	HTTPClient *http.Client

	// Timeout bounds one responder round trip. Zero uses a 30s default: the
	// responder URL comes from the certificate being validated, so it is not
	// necessarily a cooperative endpoint.
	Timeout time.Duration
}

var (
	errNoOCSPResponder  = errors.New("sign: certificate has no OCSP responder URL")
	errOCSPStatus       = errors.New("sign: OCSP responder returned non-OK status")
	errOCSPUnsuccessful = errors.New(
		"sign: OCSP response status is not successful",
	)
)

// NewOCSPClient creates an OCSP client.
func NewOCSPClient() *OCSPClient {
	return &OCSPClient{}
}

// FetchResponse fetches a DER-encoded OCSP response for cert from its first
// responder URL. issuer is required to build the request.
func (c *OCSPClient) FetchResponse(cert, issuer *x509.Certificate) ([]byte, error) {
	if len(cert.OCSPServer) == 0 {
		return nil, errNoOCSPResponder
	}
	reqDER, err := buildOCSPRequest(cert, issuer)
	if err != nil {
		return nil, fmt.Errorf("sign: build OCSP request: %w", err)
	}
	var client *http.Client
	var timeout time.Duration
	if c != nil {
		client, timeout = c.HTTPClient, c.Timeout
	}
	body, status, err := postDER(client, cert.OCSPServer[0], "application/ocsp-request", reqDER, timeout)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errOCSPStatus, status)
	}
	err = validateOCSPResponse(body)
	if err != nil {
		return nil, fmt.Errorf("sign: invalid OCSP response: %w", err)
	}
	return body, nil
}

// FetchChainResponses fetches OCSP responses for non-root certificates in chain.
func (c *OCSPClient) FetchChainResponses(chain []*x509.Certificate) ([][]byte, error) {
	responses := make([][]byte, 0, max(0, len(chain)-1))
	for i := range max(0, len(chain)-1) {
		resp, err := c.FetchResponse(chain[i], chain[i+1])
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

type ocspRequest struct {
	TBSRequest tbsRequest
}

type tbsRequest struct {
	RequestList []request
}

type request struct {
	ReqCert certID
}

type certID struct {
	HashAlgorithm  algorithmIdentifier
	IssuerNameHash []byte
	IssuerKeyHash  []byte
	SerialNumber   asn1.RawValue
}

type ocspResponse struct {
	ResponseStatus asn1.Enumerated
}

func buildOCSPRequest(cert, issuer *x509.Certificate) ([]byte, error) {
	serialDER, err := asn1.Marshal(cert.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal OCSP serial: %w", err)
	}
	req := ocspRequest{
		TBSRequest: tbsRequest{
			RequestList: []request{{
				ReqCert: certID{
					HashAlgorithm:  algorithmIdentifier{Algorithm: oidSHA256},
					IssuerNameHash: hashBytes(crypto.SHA256, issuer.RawSubject),
					IssuerKeyHash:  hashBytes(crypto.SHA256, issuer.RawSubjectPublicKeyInfo),
					SerialNumber:   asn1.RawValue{FullBytes: serialDER},
				},
			}},
		},
	}
	out, err := asn1.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal OCSP request: %w", err)
	}
	return out, nil
}

func validateOCSPResponse(data []byte) error {
	var resp ocspResponse
	_, err := asn1.Unmarshal(data, &resp)
	if err != nil {
		return fmt.Errorf("sign: unmarshal OCSP response: %w", err)
	}
	if resp.ResponseStatus != 0 {
		return fmt.Errorf("%w: %d", errOCSPUnsuccessful, resp.ResponseStatus)
	}
	return nil
}
