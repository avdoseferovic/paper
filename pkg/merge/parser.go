package merge

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	headerRe      = regexp.MustCompile(`%PDF-(\d+\.\d+)`)
	objectHeadRe  = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s+obj\b`)
	rootRefRe     = regexp.MustCompile(`/Root\s+(\d+)\s+\d+\s+R`)
	pagesRefRe    = regexp.MustCompile(`/Pages\s+(\d+)\s+\d+\s+R`)
	kidsArrayRe   = regexp.MustCompile(`(?s)/Kids\s*\[(.*?)\]`)
	indirectRefRe = regexp.MustCompile(`(\d+)\s+\d+\s+R`)
	pageTypeRe    = regexp.MustCompile(`/Type\s*/Page\b`)
	pagesTypeRe   = regexp.MustCompile(`/Type\s*/Pages\b`)
)

var errUnsupportedPDF = errors.New("unsupported PDF")

type pdfDocument struct {
	data        []byte
	version     string
	xrefOffset  int
	rootID      int
	pagesRootID int
	objects     map[int]pdfObject
	pageTreeIDs map[int]struct{}
	pageIDs     []int
}

type pdfObject struct {
	number     int
	generation int
	content    []byte
}

type xrefEntry struct {
	number     int
	generation int
	offset     int
}

func parsePDF(data []byte) (*pdfDocument, error) {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, fmt.Errorf("%w: missing PDF header", errUnsupportedPDF)
	}
	if bytes.Contains(data, []byte("/Encrypt")) {
		return nil, fmt.Errorf("%w: encrypted PDFs are not supported", errUnsupportedPDF)
	}

	xrefOffset, err := parseStartXref(data)
	if err != nil {
		return nil, err
	}
	entries, err := parseXrefTable(data, xrefOffset)
	if err != nil {
		return nil, err
	}
	objects, err := parseObjects(data, entries, xrefOffset)
	if err != nil {
		return nil, err
	}
	rootID, err := parseRootID(data, xrefOffset)
	if err != nil {
		return nil, err
	}
	root, ok := objects[rootID]
	if !ok {
		return nil, fmt.Errorf("%w: root object %d not found", errUnsupportedPDF, rootID)
	}
	pagesRootID, err := parsePagesRootID(root.content)
	if err != nil {
		return nil, err
	}

	document := &pdfDocument{
		data:        data,
		version:     parseVersion(data),
		xrefOffset:  xrefOffset,
		rootID:      rootID,
		pagesRootID: pagesRootID,
		objects:     objects,
		pageTreeIDs: make(map[int]struct{}),
	}
	document.pageIDs, err = collectPages(document, pagesRootID, map[int]bool{})
	if err != nil {
		return nil, err
	}
	if len(document.pageIDs) == 0 {
		return nil, fmt.Errorf("%w: PDF has no pages", errUnsupportedPDF)
	}

	return document, nil
}

func parseVersion(data []byte) string {
	match := headerRe.FindSubmatch(data)
	if len(match) != 2 {
		return "1.3"
	}
	return string(match[1])
}

func parseStartXref(data []byte) (int, error) {
	idx := bytes.LastIndex(data, []byte("startxref"))
	if idx < 0 {
		return 0, fmt.Errorf("%w: startxref not found", errUnsupportedPDF)
	}

	fields := strings.Fields(string(data[idx+len("startxref"):]))
	if len(fields) == 0 {
		return 0, fmt.Errorf("%w: startxref offset not found", errUnsupportedPDF)
	}
	offset, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, fmt.Errorf("invalid startxref offset: %w", err)
	}
	if offset < 0 || offset >= len(data) {
		return 0, fmt.Errorf("%w: startxref offset out of bounds", errUnsupportedPDF)
	}
	return offset, nil
}

//nolint:gocognit // Classic xref parsing is clearer as one stateful scan.
func parseXrefTable(data []byte, xrefOffset int) ([]xrefEntry, error) {
	segment := data[xrefOffset:]
	if !bytes.HasPrefix(segment, []byte("xref")) {
		return nil, fmt.Errorf("%w: xref streams are not supported", errUnsupportedPDF)
	}

	scanner := bufio.NewScanner(bytes.NewReader(segment))
	scanner.Buffer(make([]byte, 1024), len(segment))
	if !scanner.Scan() {
		return nil, fmt.Errorf("%w: xref table is empty", errUnsupportedPDF)
	}

	var entries []xrefEntry
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "trailer" {
			break
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: invalid xref subsection header %q", errUnsupportedPDF, line)
		}
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid xref subsection start: %w", err)
		}
		count, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid xref subsection count: %w", err)
		}

		for i := range count {
			if !scanner.Scan() {
				return nil, fmt.Errorf("%w: xref subsection ended early", errUnsupportedPDF)
			}
			entryParts := strings.Fields(scanner.Text())
			if len(entryParts) < 3 {
				return nil, fmt.Errorf("%w: invalid xref entry", errUnsupportedPDF)
			}
			if entryParts[2] != "n" {
				continue
			}
			offset, err := strconv.Atoi(entryParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid xref object offset: %w", err)
			}
			generation, err := strconv.Atoi(entryParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid xref object generation: %w", err)
			}
			if offset <= 0 || offset >= len(data) {
				return nil, fmt.Errorf("%w: xref object offset out of bounds", errUnsupportedPDF)
			}
			entries = append(entries, xrefEntry{
				number:     start + i,
				generation: generation,
				offset:     offset,
			})
		}
	}
	err := scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("scan xref table: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: xref table has no in-use objects", errUnsupportedPDF)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].offset < entries[j].offset
	})
	return entries, nil
}

func parseObjects(data []byte, entries []xrefEntry, xrefOffset int) (map[int]pdfObject, error) {
	objects := make(map[int]pdfObject, len(entries))
	for i, entry := range entries {
		end := xrefOffset
		if i+1 < len(entries) {
			end = entries[i+1].offset
		}
		object, err := parseObjectAt(data, entry, end)
		if err != nil {
			return nil, err
		}
		objects[object.number] = object
	}
	return objects, nil
}

func parseObjectAt(data []byte, entry xrefEntry, end int) (pdfObject, error) {
	if end <= entry.offset || end > len(data) {
		return pdfObject{}, fmt.Errorf("%w: object %d has invalid bounds", errUnsupportedPDF, entry.number)
	}

	segment := data[entry.offset:end]
	match := objectHeadRe.FindSubmatchIndex(segment)
	if len(match) == 0 {
		return pdfObject{}, fmt.Errorf("%w: object %d header not found", errUnsupportedPDF, entry.number)
	}
	number, err := strconv.Atoi(string(segment[match[2]:match[3]]))
	if err != nil {
		return pdfObject{}, fmt.Errorf("invalid object number: %w", err)
	}
	generation, err := strconv.Atoi(string(segment[match[4]:match[5]]))
	if err != nil {
		return pdfObject{}, fmt.Errorf("invalid object generation: %w", err)
	}
	if number != entry.number {
		return pdfObject{}, fmt.Errorf("%w: xref object %d points to object %d", errUnsupportedPDF, entry.number, number)
	}

	body := segment[match[1]:]
	endObject := bytes.LastIndex(body, []byte("endobj"))
	if endObject < 0 {
		return pdfObject{}, fmt.Errorf("%w: object %d end marker not found", errUnsupportedPDF, number)
	}

	return pdfObject{
		number:     number,
		generation: generation,
		content:    bytes.TrimSpace(body[:endObject]),
	}, nil
}

func parseRootID(data []byte, xrefOffset int) (int, error) {
	trailer := trailerBytes(data, xrefOffset)
	match := rootRefRe.FindSubmatch(trailer)
	if len(match) != 2 {
		return 0, fmt.Errorf("%w: trailer root reference not found", errUnsupportedPDF)
	}
	rootID, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, fmt.Errorf("invalid root reference: %w", err)
	}
	return rootID, nil
}

func trailerBytes(data []byte, xrefOffset int) []byte {
	segment := data[xrefOffset:]
	trailer := bytes.Index(segment, []byte("trailer"))
	if trailer < 0 {
		return nil
	}
	start := trailer + len("trailer")
	end := bytes.Index(segment[start:], []byte("startxref"))
	if end < 0 {
		return segment[start:]
	}
	return segment[start : start+end]
}

func parsePagesRootID(root []byte) (int, error) {
	match := pagesRefRe.FindSubmatch(root)
	if len(match) != 2 {
		return 0, fmt.Errorf("%w: catalog pages reference not found", errUnsupportedPDF)
	}
	pagesRootID, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, fmt.Errorf("invalid pages reference: %w", err)
	}
	return pagesRootID, nil
}

func collectPages(document *pdfDocument, objectID int, visited map[int]bool) ([]int, error) {
	if visited[objectID] {
		return nil, fmt.Errorf("%w: page tree cycle at object %d", errUnsupportedPDF, objectID)
	}
	visited[objectID] = true

	object, ok := document.objects[objectID]
	if !ok {
		return nil, fmt.Errorf("%w: page tree object %d not found", errUnsupportedPDF, objectID)
	}
	if pageTypeRe.Match(object.content) {
		return []int{objectID}, nil
	}
	if !pagesTypeRe.Match(object.content) {
		return nil, fmt.Errorf("%w: page tree object %d is not /Pages", errUnsupportedPDF, objectID)
	}

	document.pageTreeIDs[objectID] = struct{}{}
	kids := kidsForPagesObject(object.content)
	if len(kids) == 0 {
		return nil, fmt.Errorf("%w: pages object %d has no kids", errUnsupportedPDF, objectID)
	}

	var pages []int
	for _, kid := range kids {
		kidPages, err := collectPages(document, kid, visited)
		if err != nil {
			return nil, err
		}
		pages = append(pages, kidPages...)
	}
	return pages, nil
}

func kidsForPagesObject(content []byte) []int {
	match := kidsArrayRe.FindSubmatch(content)
	if len(match) != 2 {
		return nil
	}
	return refsInBytes(match[1])
}

func refsInBytes(data []byte) []int {
	matches := indirectRefRe.FindAllSubmatch(data, -1)
	refs := make([]int, 0, len(matches))
	for _, match := range matches {
		ref, err := strconv.Atoi(string(match[1]))
		if err == nil {
			refs = append(refs, ref)
		}
	}
	return refs
}
