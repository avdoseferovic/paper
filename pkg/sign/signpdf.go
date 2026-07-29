package sign

import (
	"bytes"
	"crypto"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/avdoseferovic/paper/internal/pdfscan"
	"github.com/avdoseferovic/paper/pkg/reader"
)

// PAdESLevel specifies the requested PAdES conformance level.
type PAdESLevel int

const (
	// LevelBB is PAdES B-B: a basic CMS detached signature.
	LevelBB PAdESLevel = iota
	// LevelBT is PAdES B-T: B-B plus a timestamp token from an RFC 3161 TSA.
	LevelBT
	// LevelBLT is reserved for PAdES B-LT validation data.
	LevelBLT
	// LevelBLTA is reserved for PAdES B-LTA archival timestamps.
	LevelBLTA
)

// Options configures PDF.
type Options struct {
	Signer      Signer
	Level       PAdESLevel
	Name        string
	Reason      string
	Location    string
	ContactInfo string
	SigningTime time.Time
	TSAClient   *TSAClient
	OCSPClient  *OCSPClient
	CRLs        [][]byte
	ExtraCerts  [][]byte
}

type parsedPDFInfo struct {
	data            []byte
	objects         map[int]rawPDFObject
	rootObjNum      int
	rootContent     []byte
	prevXref        int64
	size            int
	maxObjectNumber int
	infoRef         string
	idValue         string
}

type rawPDFObject struct {
	content []byte
}

var (
	signObjectRe = regexp.MustCompile(`(?s)(\d+)\s+\d+\s+obj\s*(.*?)\s*endobj`)
	signRootRe   = regexp.MustCompile(`/Root\s+(\d+)\s+\d+\s+R`)
	signSizeRe   = regexp.MustCompile(`/Size\s+(\d+)`)
	signInfoRe   = regexp.MustCompile(`/Info\s+(\d+\s+\d+\s+R)`)
	signIDRe     = regexp.MustCompile(`(?s)/ID\s*(\[[^\]]+\])`)
	fieldRefsRe  = regexp.MustCompile(`(?s)/Fields\s*\[(.*?)\]`)
	refTokenRe   = regexp.MustCompile(`\d+\s+\d+\s+R`)
)

var (
	errSignerRequired       = errors.New("sign: Signer is required")
	errTSAClientRequired    = errors.New("sign: TSAClient is required for PAdES B-T and above")
	errRootObjectMissing    = errors.New("sign: root object missing")
	errNoIndirectObjects    = errors.New("sign: no indirect objects found")
	errTrailerNoRoot        = errors.New("sign: trailer has no /Root")
	errCatalogNotDictionary = errors.New("sign: catalog is not a dictionary")
	errInvalidPlaceholder   = errors.New("sign: invalid signature placeholder bounds")
)

// PDF applies a detached CMS/PAdES signature to an existing PDF using an
// incremental update.
func PDF(pdfBytes []byte, opts Options) ([]byte, error) {
	if opts.Signer == nil {
		return nil, errSignerRequired
	}
	if opts.Level >= LevelBT && opts.TSAClient == nil {
		return nil, errTSAClientRequired
	}
	_, err := reader.Parse(pdfBytes)
	if err != nil {
		return nil, fmt.Errorf("sign: parse PDF: %w", err)
	}

	signingTime := opts.SigningTime
	if signingTime.IsZero() {
		signingTime = time.Now()
	}

	info, err := parsePDFInfo(pdfBytes)
	if err != nil {
		return nil, err
	}
	nextObjNum := max(info.maxObjectNumber+1, info.size)
	if nextObjNum <= 0 {
		nextObjNum = 1
	}

	sigDictObjNum := nextObjNum
	sigFieldObjNum := nextObjNum + 1
	acroFormObjNum := nextObjNum + 2

	updatedCatalog, err := catalogWithAcroForm(info.rootContent, acroFormObjNum)
	if err != nil {
		return nil, err
	}

	objects := []incrementalObject{
		{number: sigDictObjNum, content: buildSigDict(opts.Name, opts.Location, opts.Reason, opts.ContactInfo)},
		{number: sigFieldObjNum, content: buildSigField(sigFieldObjNum, sigDictObjNum)},
		{number: acroFormObjNum, content: buildAcroForm(info, sigFieldObjNum)},
		{number: info.rootObjNum, content: updatedCatalog},
	}

	signedPDF := writeIncrementalUpdate(info, objects)
	ph, err := locatePlaceholders(signedPDF, sigDictObjNum)
	if err != nil {
		return nil, err
	}
	patchByteRange(signedPDF, ph)

	digest, err := computeByteRangeDigest(signedPDF, ph, opts.Signer.Algorithm().HashFunc())
	if err != nil {
		return nil, err
	}
	var tsaToken []byte
	if opts.Level >= LevelBT {
		tsaToken, err = opts.TSAClient.Timestamp(digest, opts.Signer.Algorithm().HashFunc())
		if err != nil {
			return nil, fmt.Errorf("sign: TSA timestamp: %w", err)
		}
	}
	cmsSig, err := BuildDetachedCMS(digest, opts.Signer, signingTime, tsaToken)
	if err != nil {
		return nil, fmt.Errorf("sign: build CMS: %w", err)
	}
	err = patchContents(signedPDF, ph, cmsSig)
	if err != nil {
		return nil, err
	}
	if opts.Level >= LevelBLT {
		signedPDF, err = addValidationData(signedPDF, opts, cmsSig)
		if err != nil {
			return nil, err
		}
	}
	if opts.Level >= LevelBLTA {
		signedPDF, err = AddDocumentTimestamp(signedPDF, opts.TSAClient, opts.Signer.Algorithm().HashFunc())
		if err != nil {
			return nil, fmt.Errorf("sign: document timestamp: %w", err)
		}
	}
	return signedPDF, nil
}

func addValidationData(pdfBytes []byte, opts Options, sigContents []byte) ([]byte, error) {
	dss := NewDSS()
	chain := opts.Signer.CertificateChain()

	var ocspResponses [][]byte
	if opts.OCSPClient != nil {
		var err error
		ocspResponses, err = opts.OCSPClient.FetchChainResponses(chain)
		if err != nil {
			return nil, fmt.Errorf("sign: collect OCSP data: %w", err)
		}
	}

	dss.AddSignatureValidation(sigContents, chain, ocspResponses, opts.CRLs)
	for _, certDER := range opts.ExtraCerts {
		dss.addCert(certDER)
	}
	return AddDSS(pdfBytes, dss)
}

func parsePDFInfo(pdfBytes []byte) (parsedPDFInfo, error) {
	prevXref, err := findStartXref(pdfBytes)
	if err != nil {
		return parsedPDFInfo{}, err
	}
	trailer, err := trailerBytes(pdfBytes, prevXref)
	if err != nil {
		return parsedPDFInfo{}, err
	}
	objects, maxObjNum, err := parseRawObjects(pdfBytes)
	if err != nil {
		return parsedPDFInfo{}, err
	}
	rootObjNum, err := parseRootObjectNumber(trailer)
	if err != nil {
		return parsedPDFInfo{}, err
	}
	root, ok := objects[rootObjNum]
	if !ok {
		return parsedPDFInfo{}, fmt.Errorf("%w: %d", errRootObjectMissing, rootObjNum)
	}
	return parsedPDFInfo{
		data:            slices.Clone(pdfBytes),
		objects:         objects,
		rootObjNum:      rootObjNum,
		rootContent:     root.content,
		prevXref:        prevXref,
		size:            parseTrailerSize(trailer, maxObjNum+1),
		maxObjectNumber: maxObjNum,
		infoRef:         parseTrailerRef(trailer, signInfoRe),
		idValue:         parseTrailerRef(trailer, signIDRe),
	}, nil
}

func parseRawObjects(data []byte) (map[int]rawPDFObject, int, error) {
	matches := signObjectRe.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil, 0, errNoIndirectObjects
	}
	objects := make(map[int]rawPDFObject, len(matches))
	maxObjNum := 0
	for _, match := range matches {
		number, err := strconv.Atoi(string(match[1]))
		if err != nil {
			return nil, 0, fmt.Errorf("sign: invalid object number: %w", err)
		}
		maxObjNum = max(maxObjNum, number)
		objects[number] = rawPDFObject{
			content: slices.Clone(match[2]),
		}
	}
	return objects, maxObjNum, nil
}

func parseRootObjectNumber(trailer []byte) (int, error) {
	match := signRootRe.FindSubmatch(trailer)
	if len(match) != 2 {
		return 0, errTrailerNoRoot
	}
	rootObjNum, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, fmt.Errorf("sign: invalid root reference: %w", err)
	}
	return rootObjNum, nil
}

func parseTrailerSize(trailer []byte, fallback int) int {
	match := signSizeRe.FindSubmatch(trailer)
	if len(match) != 2 {
		return fallback
	}
	size, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return fallback
	}
	return size
}

func parseTrailerRef(trailer []byte, re *regexp.Regexp) string {
	match := re.FindSubmatch(trailer)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}

func buildSigField(objNum, sigDictObjNum int) []byte {
	return fmt.Appendf(
		nil,
		"<< /Type /Annot /Subtype /Widget /FT /Sig /T (%s) /V %d 0 R /F 132 /Rect [0 0 0 0] >>",
		escapePDFString(fmt.Sprintf("Signature%d", objNum)),
		sigDictObjNum,
	)
}

func buildAcroForm(info parsedPDFInfo, sigFieldObjNum int) []byte {
	existingFields := existingAcroFormFieldRefs(info)
	fields := make([]string, 0, 1+len(existingFields))
	fields = append(fields, fmt.Sprintf("%d 0 R", sigFieldObjNum))
	fields = append(fields, existingFields...)
	return fmt.Appendf(nil, "<< /Fields [%s] /SigFlags 3 >>", strings.Join(fields, " "))
}

func existingAcroFormFieldRefs(info parsedPDFInfo) []string {
	acroForm := acroFormContent(info)
	if len(acroForm) == 0 {
		return nil
	}
	match := fieldRefsRe.FindSubmatch(acroForm)
	if len(match) != 2 {
		return nil
	}
	refs := refTokenRe.FindAll(match[1], -1)
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, string(ref))
	}
	return out
}

func acroFormContent(info parsedPDFInfo) []byte {
	root := info.rootContent
	if ref, ok := indirectRefForKey(root, "AcroForm"); ok {
		if object, exists := info.objects[ref]; exists {
			return object.content
		}
		return nil
	}
	idx := bytes.Index(root, []byte("/AcroForm"))
	if idx < 0 {
		return nil
	}
	valueStart := pdfscan.SkipSpaces(root, idx+len("/AcroForm"))
	if !bytes.HasPrefix(root[valueStart:], []byte("<<")) {
		return nil
	}
	valueEnd := pdfscan.SkipBalanced(root, valueStart, []byte("<<"), []byte(">>"))
	if valueEnd <= valueStart {
		return nil
	}
	return root[valueStart:valueEnd]
}

func indirectRefForKey(content []byte, key string) (int, bool) {
	re := regexp.MustCompile(`/` + regexp.QuoteMeta(key) + `\s+(\d+)\s+\d+\s+R`)
	match := re.FindSubmatch(content)
	if len(match) != 2 {
		return 0, false
	}
	ref, err := strconv.Atoi(string(match[1]))
	return ref, err == nil
}

func catalogWithAcroForm(catalog []byte, acroFormObjNum int) ([]byte, error) {
	cleaned := removeDictionaryEntry(bytes.TrimSpace(catalog), "AcroForm")
	return catalogWithReference(cleaned, "AcroForm", acroFormObjNum)
}

func catalogWithDSS(catalog []byte, dssObjNum int) ([]byte, error) {
	cleaned := removeDictionaryEntry(bytes.TrimSpace(catalog), "DSS")
	return catalogWithReference(cleaned, "DSS", dssObjNum)
}

func catalogWithReference(catalog []byte, key string, objNum int) ([]byte, error) {
	end := bytes.LastIndex(catalog, []byte(">>"))
	if end < 0 {
		return nil, errCatalogNotDictionary
	}
	var out bytes.Buffer
	out.Write(bytes.TrimRight(catalog[:end], " \t\r\n"))
	fmt.Fprintf(&out, " /%s %d 0 R ", key, objNum)
	out.Write(bytes.TrimSpace(catalog[end:]))
	return out.Bytes(), nil
}

func removeDictionaryEntry(dictionary []byte, key string) []byte {
	marker := []byte("/" + key)
	idx := bytes.Index(dictionary, marker)
	if idx < 0 {
		return dictionary
	}
	valueStart := pdfscan.SkipSpaces(dictionary, idx+len(marker))
	valueEnd := pdfscan.SkipValue(dictionary, valueStart)
	if valueEnd <= valueStart {
		return dictionary
	}
	out := make([]byte, 0, len(dictionary)-(valueEnd-idx))
	out = append(out, bytes.TrimRight(dictionary[:idx], " \t\r\n")...)
	out = append(out, ' ')
	out = append(out, bytes.TrimLeft(dictionary[valueEnd:], " \t\r\n")...)
	return out
}

func computeByteRangeDigest(pdf []byte, ph signaturePlaceholder, hashFunc crypto.Hash) ([]byte, error) {
	contentsStart := ph.contentsOffset
	contentsEnd := ph.contentsOffset + ph.contentsLen
	if contentsStart < 0 || contentsEnd > len(pdf) || contentsStart > contentsEnd {
		return nil, errInvalidPlaceholder
	}
	h := hashFunc.New()
	_, _ = h.Write(pdf[:contentsStart])
	_, _ = h.Write(pdf[contentsEnd:])
	return h.Sum(nil), nil
}
