package sign

import (
	"crypto"
	"errors"
	"fmt"

	"github.com/avdoseferovic/paper/pkg/reader"
)

var errDocumentTimestampTSARequired = errors.New("sign: TSAClient is required for document timestamp")

// AddDocumentTimestamp appends a PAdES document timestamp signature.
func AddDocumentTimestamp(pdfBytes []byte, tsaClient *TSAClient, hashFunc crypto.Hash) ([]byte, error) {
	if tsaClient == nil {
		return nil, errDocumentTimestampTSARequired
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
	tsDictObjNum := nextObjNum
	tsFieldObjNum := nextObjNum + 1
	acroFormObjNum := nextObjNum + 2

	updatedCatalog, err := catalogWithAcroForm(info.rootContent, acroFormObjNum)
	if err != nil {
		return nil, err
	}
	objects := []incrementalObject{
		{number: tsDictObjNum, content: buildDocTimestampDict()},
		{number: tsFieldObjNum, content: buildDocTimestampField(tsFieldObjNum, tsDictObjNum)},
		{number: acroFormObjNum, content: buildAcroForm(info, tsFieldObjNum)},
		{number: info.rootObjNum, content: updatedCatalog},
	}

	result := writeIncrementalUpdate(info, objects)
	ph, err := locatePlaceholders(result, tsDictObjNum)
	if err != nil {
		return nil, err
	}
	patchByteRange(result, ph)
	digest, err := computeByteRangeDigest(result, ph, hashFunc)
	if err != nil {
		return nil, err
	}
	token, err := tsaClient.Timestamp(digest, hashFunc)
	if err != nil {
		return nil, fmt.Errorf("sign: TSA timestamp: %w", err)
	}
	err = patchContents(result, ph, token)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func buildDocTimestampDict() []byte {
	return []byte("<< /Type /DocTimeStamp /Filter /Adobe.PPKLite /SubFilter /ETSI.RFC3161 /ByteRange " +
		byteRangePlaceholder + " /Contents " + contentsPlaceholder + " >>")
}

func buildDocTimestampField(objNum, tsDictObjNum int) []byte {
	return fmt.Appendf(
		nil,
		"<< /Type /Annot /Subtype /Widget /FT /Sig /T (%s) /V %d 0 R /F 132 /Rect [0 0 0 0] >>",
		escapePDFString(fmt.Sprintf("DocTimeStamp%d", objNum)),
		tsDictObjNum,
	)
}
