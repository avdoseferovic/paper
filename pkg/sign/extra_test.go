package sign

import (
	"crypto/x509"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
)

func TestParsePKCS12IsUnsupported(t *testing.T) {
	t.Parallel()

	signer, err := ParsePKCS12([]byte("blob"), "password")

	assert.Nil(t, signer)
	assert.ErrorIs(t, err, ErrPKCS12Unsupported)
}

func TestExternalSignerCertificateChainReturnsCopy(t *testing.T) {
	t.Parallel()

	cert := &x509.Certificate{Raw: []byte{1, 2, 3}}
	signer, err := NewExternalSigner(
		func(digest []byte) ([]byte, error) { return digest, nil },
		[]*x509.Certificate{cert},
		SHA256WithRSA,
	)
	require.NoError(t, err)

	chain := signer.CertificateChain()
	require.Equal(t, 1, len(chain))
	chain[0] = nil
	assert.NotNil(t, signer.CertificateChain()[0], "mutation of the returned slice must not affect the signer")
}

func TestCollectValidationDataWithoutClient(t *testing.T) {
	t.Parallel()

	data, err := CollectValidationData([]*x509.Certificate{{Raw: []byte{1}}}, nil)

	require.NoError(t, err)
	assert.Nil(t, data)
}
