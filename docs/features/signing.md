# Digital Signing

`pkg/sign` signs existing PDFs by appending an incremental update. The original
document bytes remain intact and the new revision adds a signature dictionary,
signature field, AcroForm entry, byte range, and detached CMS signature:

```go
signer, err := sign.NewLocalSigner(privateKey, []*x509.Certificate{cert})
if err != nil {
	return err
}

signed, err := sign.SignPDF(pdfBytes, sign.Options{
	Signer:      signer,
	Level:       sign.LevelBB,
	Name:        "Ada Lovelace",
	Reason:      "Approved",
	Location:    "London",
	SigningTime: time.Now(),
})
if err != nil {
	return err
}
```

For PAdES B-T, provide an RFC 3161 timestamp authority client:

```go
signed, err := sign.SignPDF(pdfBytes, sign.Options{
	Signer:    signer,
	Level:     sign.LevelBT,
	TSAClient: sign.NewTSAClient("https://tsa.example.com"),
})
```

For PAdES B-LT and B-LTA, Paper embeds a Document Security Store (DSS) with
the signer's certificate chain, optional OCSP responses, CRLs, and extra
certificates. B-LTA appends a document timestamp after the DSS update:

```go
signed, err := sign.SignPDF(pdfBytes, sign.Options{
	Signer:     signer,
	Level:      sign.LevelBLTA,
	TSAClient:  sign.NewTSAClient("https://tsa.example.com"),
	OCSPClient: sign.NewOCSPClient(),
	CRLs:       [][]byte{crlDER},
	ExtraCerts: [][]byte{intermediateDER},
})
```

Applications that keep private keys outside the process can provide an
external signing callback:

```go
signer, err := sign.NewExternalSigner(
	func(digest []byte) ([]byte, error) {
		return hsm.Sign(digest)
	},
	[]*x509.Certificate{cert},
	sign.SHA256WithRSA,
)
```

Current scope:

- `Signer`, `LocalSigner`, and `ExternalSigner`
- RSA and ECDSA SHA-2 algorithms: SHA-256, SHA-384, and SHA-512
- algorithm hash, digest OID, and signature OID metadata
- `BuildDetachedCMS` for detached CMS SignedData generation with signing-time
  and message-digest signed attributes
- `SignPDF` incremental signing for classic-xref PDFs supported by Paper's
  reader foundation
- PAdES B-B signatures using `/SubFilter /ETSI.CAdES.detached`
- PAdES B-T timestamp token embedding through `TSAClient`
- PAdES B-LT DSS/VRI validation data through `NewDSS`, `AddDSS`,
  `AddSignatureValidation`, OCSP responses, CRLs, and extra certificates
- PAdES B-LTA document timestamps through `AddDocumentTimestamp`

Limitations:

- xref streams, object streams, and encrypted PDFs are not supported by this
  signing path yet
- visible signature appearances are not implemented
- PKCS#12 parsing is still future work; use `NewLocalSigner` with a pre-parsed
  key and certificate chain
