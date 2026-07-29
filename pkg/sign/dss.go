package sign

import (
	"bytes"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/avdoseferovic/paper/pkg/reader"
)

// DSS holds Document Security Store validation data for PAdES B-LT/B-LTA.
type DSS struct {
	Certs [][]byte
	OCSPs [][]byte
	CRLs  [][]byte
	VRI   map[string]*VRIEntry
}

// VRIEntry holds validation data associated with one signature.
type VRIEntry struct {
	Certs [][]byte
	OCSPs [][]byte
	CRLs  [][]byte
}

var errNilDSS = errors.New("sign: DSS is nil")

// NewDSS creates an empty Document Security Store.
func NewDSS() *DSS {
	return &DSS{VRI: make(map[string]*VRIEntry)}
}

// AddSignatureValidation adds certificate and revocation data for sigContents.
func (d *DSS) AddSignatureValidation(sigContents []byte, chain []*x509.Certificate, ocspResponses, crls [][]byte) {
	if d == nil {
		return
	}
	if d.VRI == nil {
		d.VRI = make(map[string]*VRIEntry)
	}
	entry := &VRIEntry{}
	for _, cert := range chain {
		if cert == nil {
			continue
		}
		d.addCert(cert.Raw)
		entry.Certs = append(entry.Certs, cert.Raw)
	}
	for _, ocsp := range ocspResponses {
		d.addOCSP(ocsp)
		entry.OCSPs = append(entry.OCSPs, ocsp)
	}
	for _, crl := range crls {
		d.addCRL(crl)
		entry.CRLs = append(entry.CRLs, crl)
	}
	d.VRI[computeVRIKey(sigContents)] = entry
}

// AddDSS appends a DSS dictionary to pdfBytes using an incremental update.
func AddDSS(pdfBytes []byte, dss *DSS) ([]byte, error) {
	if dss == nil {
		return nil, errNilDSS
	}
	_, err := reader.Parse(pdfBytes)
	if err != nil {
		return nil, fmt.Errorf("sign: parse PDF: %w", err)
	}
	info, err := parsePDFInfo(pdfBytes)
	if err != nil {
		return nil, err
	}

	nextObjNum := max(info.maxObjectNumber+1, info.size)
	objects := make([]incrementalObject, 0)
	addObject := func(content []byte) string {
		objNum := nextObjNum
		nextObjNum++
		objects = append(objects, incrementalObject{number: objNum, content: content})
		return fmt.Sprintf("%d 0 R", objNum)
	}

	dssDict := dss.build(addObject)
	dssObjNum := nextObjNum
	nextObjNum++
	objects = append(objects, incrementalObject{number: dssObjNum, content: dssDict})

	updatedCatalog, err := catalogWithDSS(info.rootContent, dssObjNum)
	if err != nil {
		return nil, err
	}
	objects = append(objects, incrementalObject{number: info.rootObjNum, content: updatedCatalog})
	return writeIncrementalUpdate(info, objects), nil
}

// CollectValidationData fetches OCSP responses for chain when ocspClient is set.
func CollectValidationData(chain []*x509.Certificate, ocspClient *OCSPClient) ([][]byte, error) {
	if ocspClient == nil {
		return nil, nil
	}
	return ocspClient.FetchChainResponses(chain)
}

func (d *DSS) build(addObject func([]byte) string) []byte {
	parts := make([]string, 0, 4)
	if len(d.Certs) > 0 {
		parts = append(parts, "/Certs "+buildStreamArray(d.Certs, addObject))
	}
	if len(d.OCSPs) > 0 {
		parts = append(parts, "/OCSPs "+buildStreamArray(d.OCSPs, addObject))
	}
	if len(d.CRLs) > 0 {
		parts = append(parts, "/CRLs "+buildStreamArray(d.CRLs, addObject))
	}
	if len(d.VRI) > 0 {
		vriRef := addObject(d.buildVRIDictionary(addObject))
		parts = append(parts, "/VRI "+vriRef)
	}
	return []byte("<< " + strings.Join(parts, " ") + " >>")
}

func (d *DSS) buildVRIDictionary(addObject func([]byte) string) []byte {
	keys := make([]string, 0, len(d.VRI))
	for key := range d.VRI {
		keys = append(keys, key)
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		entry := d.VRI[key]
		entryRef := addObject(entry.build(addObject))
		parts = append(parts, fmt.Sprintf("/%s %s", key, entryRef))
	}
	return []byte("<< " + strings.Join(parts, " ") + " >>")
}

func (e *VRIEntry) build(addObject func([]byte) string) []byte {
	parts := make([]string, 0, 3)
	if len(e.Certs) > 0 {
		parts = append(parts, "/Cert "+buildStreamArray(e.Certs, addObject))
	}
	if len(e.OCSPs) > 0 {
		parts = append(parts, "/OCSP "+buildStreamArray(e.OCSPs, addObject))
	}
	if len(e.CRLs) > 0 {
		parts = append(parts, "/CRL "+buildStreamArray(e.CRLs, addObject))
	}
	return []byte("<< " + strings.Join(parts, " ") + " >>")
}

func buildStreamArray(items [][]byte, addObject func([]byte) string) string {
	refs := make([]string, 0, len(items))
	for _, item := range items {
		refs = append(refs, addObject(buildStream(item)))
	}
	return "[" + strings.Join(refs, " ") + "]"
}

func buildStream(data []byte) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "<< /Length %d >>\nstream\n", len(data))
	out.Write(data)
	out.WriteString("\nendstream")
	return out.Bytes()
}

func (d *DSS) addCert(der []byte) {
	if len(der) > 0 && !containsBytes(d.Certs, der) {
		d.Certs = append(d.Certs, slices.Clone(der))
	}
}

func (d *DSS) addOCSP(der []byte) {
	if len(der) > 0 && !containsBytes(d.OCSPs, der) {
		d.OCSPs = append(d.OCSPs, slices.Clone(der))
	}
}

func (d *DSS) addCRL(der []byte) {
	if len(der) > 0 && !containsBytes(d.CRLs, der) {
		d.CRLs = append(d.CRLs, slices.Clone(der))
	}
}

func containsBytes(slice [][]byte, item []byte) bool {
	return slices.ContainsFunc(slice, func(existing []byte) bool {
		return bytes.Equal(existing, item)
	})
}

func computeVRIKey(sigContents []byte) string {
	sum := sha1.Sum(sigContents)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}
