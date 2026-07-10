package sign

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	errStartXrefNotFound    = errors.New("sign: startxref not found")
	errStartXrefOutOfBounds = errors.New("sign: startxref offset out of bounds")
	errXrefOutOfBounds      = errors.New("sign: xref offset out of bounds")
	errTrailerNotFound      = errors.New("sign: trailer not found")
)

type incrementalObject struct {
	number  int
	content []byte
}

func writeIncrementalUpdate(info parsedPDFInfo, objects []incrementalObject) []byte {
	var buf bytes.Buffer
	buf.Write(info.data)
	if len(info.data) > 0 && info.data[len(info.data)-1] != '\n' {
		buf.WriteByte('\n')
	}

	type offsetEntry struct {
		objNum int
		offset int
	}
	offsets := make([]offsetEntry, 0, len(objects))
	maxObjectNumber := info.maxObjectNumber
	for _, object := range objects {
		if object.number > maxObjectNumber {
			maxObjectNumber = object.number
		}
		offsets = append(offsets, offsetEntry{objNum: object.number, offset: buf.Len()})
		fmt.Fprintf(&buf, "%d 0 obj\n", object.number)
		buf.Write(bytes.TrimSpace(object.content))
		buf.WriteString("\nendobj\n")
	}

	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	for _, entry := range offsets {
		fmt.Fprintf(&buf, "%d 1\n", entry.objNum)
		fmt.Fprintf(&buf, "%010d 00000 n \n", entry.offset)
	}
	trailerSize := max(info.size, maxObjectNumber+1)
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root %d 0 R", trailerSize, info.rootObjNum)
	if info.infoRef != "" {
		fmt.Fprintf(&buf, " /Info %s", info.infoRef)
	}
	if info.idValue != "" {
		fmt.Fprintf(&buf, " /ID %s", info.idValue)
	}
	fmt.Fprintf(&buf, " /Prev %d >>\nstartxref\n%d\n%%%%EOF\n", info.prevXref, xrefOffset)
	return buf.Bytes()
}

func findStartXref(data []byte) (int64, error) {
	searchLen := min(1024, len(data))
	tail := data[len(data)-searchLen:]

	idx := bytes.LastIndex(tail, []byte("startxref"))
	if idx < 0 {
		return 0, fmt.Errorf("%w: last %d bytes", errStartXrefNotFound, searchLen)
	}
	after := strings.TrimSpace(string(tail[idx+len("startxref"):]))
	if nl := strings.IndexAny(after, "\r\n"); nl > 0 {
		after = after[:nl]
	}
	after = strings.TrimSpace(after)

	offset, err := strconv.ParseInt(after, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("sign: invalid startxref offset %q: %w", after, err)
	}
	if offset < 0 || offset >= int64(len(data)) {
		return 0, fmt.Errorf("%w: %d", errStartXrefOutOfBounds, offset)
	}
	return offset, nil
}

func trailerBytes(data []byte, xrefOffset int64) ([]byte, error) {
	if xrefOffset < 0 || xrefOffset >= int64(len(data)) {
		return nil, fmt.Errorf("%w: %d", errXrefOutOfBounds, xrefOffset)
	}
	segment := data[xrefOffset:]
	trailerIdx := bytes.Index(segment, []byte("trailer"))
	if trailerIdx < 0 {
		return nil, errTrailerNotFound
	}
	start := trailerIdx + len("trailer")
	end := bytes.Index(segment[start:], []byte("startxref"))
	if end < 0 {
		end = len(segment) - start
	}
	return bytes.TrimSpace(segment[start : start+end]), nil
}
