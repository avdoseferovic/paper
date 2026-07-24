package reader

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
)

var (
	infoRefRe     = regexp.MustCompile(`/Info\s+(\d+)\s+\d+\s+R`)
	metadataRefRe = regexp.MustCompile(`/Metadata\s+(\d+)\s+\d+\s+R`)
)

// stripDocumentMetadata blanks the document information dictionary and any XMP
// metadata stream in place. Like the rest of redaction it is byte preserving:
// values are overwritten with spaces so xref offsets stay valid. Metadata this
// foundation cannot rewrite safely is reported as an error rather than left in
// the output, so a caller asking for a strip never gets a silent no-op.
func stripDocumentMetadata(out []byte, r *PdfReader) error {
	err := stripInfoDictionary(out, r)
	if err != nil {
		return err
	}
	return stripXMPMetadata(out, r)
}

func stripInfoDictionary(out []byte, r *PdfReader) error {
	objectID, ok := lastReferenceID(r.data, infoRefRe)
	if !ok {
		return nil
	}
	object, ok := r.objects[objectID]
	if !ok {
		return fmt.Errorf("%w: info object %d missing", ErrUnsupportedPDF, objectID)
	}
	// Emptying the whole dictionary drops the keys along with the values, which
	// leaves `<<   >>`: a legal empty /Info dictionary of unchanged length.
	open := bytes.Index(object.content, []byte("<<"))
	closing := bytes.LastIndex(object.content, []byte(">>"))
	if open < 0 || closing <= open {
		return fmt.Errorf("%w: info object %d is not a dictionary", ErrUnsupportedPDF, objectID)
	}
	blank(out[object.contentStart+open+2 : object.contentStart+closing])
	return nil
}

func stripXMPMetadata(out []byte, r *PdfReader) error {
	objectID, ok := lastReferenceID(r.data, metadataRefRe)
	if !ok {
		return nil
	}
	object, ok := r.objects[objectID]
	if !ok {
		return fmt.Errorf("%w: metadata object %d missing", ErrUnsupportedPDF, objectID)
	}
	start, end, err := streamDataBounds(object.content)
	if err != nil {
		return fmt.Errorf("%w: metadata object %d has no stream", ErrUnsupportedPDF, objectID)
	}
	if filter := parseFilter(object.content[:start]); filter != "" {
		// Overwriting compressed bytes would leave an undecodable stream, and
		// re-encoding would move every following xref offset.
		return fmt.Errorf(
			"%w: metadata stream filter %s cannot be stripped by the byte-preserving foundation",
			ErrUnsupportedPDF,
			filter,
		)
	}
	blank(out[object.contentStart+start : object.contentStart+end])
	return nil
}

// lastReferenceID returns the object number of the final match of re in data.
// The last match wins because incremental updates append newer trailers.
func lastReferenceID(data []byte, re *regexp.Regexp) (int, bool) {
	matches := re.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return 0, false
	}
	id, err := strconv.Atoi(string(matches[len(matches)-1][1]))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func blank(data []byte) {
	for i := range data {
		data[i] = ' '
	}
}
