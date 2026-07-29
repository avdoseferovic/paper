// Package reader parses existing PDF files and exposes basic page metadata,
// content streams, text extraction, and merge conveniences.
package reader

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/avdoseferovic/paper/internal/pdfscan"
	"github.com/avdoseferovic/paper/pkg/merge"
)

var (
	// ErrCannotReadFile is returned when Load cannot read a PDF from disk.
	ErrCannotReadFile = errors.New("reader: cannot read file")
	// ErrEncryptedUnsupported is returned for encrypted PDFs.
	ErrEncryptedUnsupported = errors.New("reader: encrypted PDFs are not supported")
	// ErrPageOutOfRange is returned when Page receives an invalid index.
	ErrPageOutOfRange = errors.New("reader: page index out of range")
	// ErrUnsupportedPDF is returned for PDF constructs not supported by this foundation parser.
	ErrUnsupportedPDF = errors.New("reader: unsupported PDF")
)

// Strictness controls how the reader handles unsupported stream filters.
type Strictness int

const (
	// StrictnessTolerant preserves undecodable streams as raw data where possible.
	StrictnessTolerant Strictness = iota
	// StrictnessStrict fails when a page content stream uses an unsupported filter.
	StrictnessStrict
)

const (
	defaultPageWidthPt  = 612.0
	defaultPageHeightPt = 792.0
)

// ReadOptions configures PDF parsing.
type ReadOptions struct {
	Strictness Strictness
}

// Box represents a PDF rectangle in points: lower-left (X1,Y1) to upper-right (X2,Y2).
type Box struct {
	X1, Y1, X2, Y2 float64
}

// Width returns the box width.
func (b Box) Width() float64 { return b.X2 - b.X1 }

// Height returns the box height.
func (b Box) Height() float64 { return b.Y2 - b.Y1 }

// IsZero reports whether the box is unset.
func (b Box) IsZero() bool { return b == Box{} }

// PdfReader holds parsed state for an existing PDF.
type PdfReader struct {
	data        []byte
	version     string
	objects     map[int]pdfObject
	rootID      int
	pagesRootID int
	pages       []*PageInfo
	strictness  Strictness
}

// PageInfo holds parsed metadata for one PDF page.
type PageInfo struct {
	Number int
	Width  float64
	Height float64
	Rotate int

	MediaBox Box
	CropBox  Box
	BleedBox Box
	TrimBox  Box
	ArtBox   Box

	reader   *PdfReader
	objectID int
}

type pdfObject struct {
	number       int
	generation   int
	content      []byte
	contentStart int
	contentEnd   int
}

type inheritedPageState struct {
	mediaBox Box
	cropBox  Box
	bleedBox Box
	trimBox  Box
	artBox   Box
	rotate   int
}

var (
	headerRe      = regexp.MustCompile(`%PDF-(\d+\.\d+)`)
	objectRe      = regexp.MustCompile(`(?s)(\d+)\s+(\d+)\s+obj\s*(.*?)\s*endobj`)
	rootRefRe     = regexp.MustCompile(`/Root\s+(\d+)\s+\d+\s+R`)
	indirectRefRe = regexp.MustCompile(`(\d+)\s+\d+\s+R`)
	kidsArrayRe   = regexp.MustCompile(`(?s)/Kids\s*\[(.*?)\]`)
	pageTypeRe    = regexp.MustCompile(`/Type\s*/Page\b`)
	pagesTypeRe   = regexp.MustCompile(`/Type\s*/Pages\b`)
	contentsRe    = regexp.MustCompile(`(?s)/Contents\s*(\[(.*?)\]|(\d+)\s+\d+\s+R)`)
	filterRe      = regexp.MustCompile(`(?s)/Filter\s*(/\w+|\[.*?\])`)

	// keyedPatterns caches the dictionary-key expressions built at call time.
	// Parsing one page compiles six of them, so a large document recompiled the
	// same handful of expressions thousands of times.
	keyedPatterns sync.Map
)

// keyedPattern returns the compiled form of format with key substituted. Keys
// are library-internal literals, quoted defensively.
func keyedPattern(format, key string) *regexp.Regexp {
	cacheKey := format + "\x00" + key
	cached, ok := keyedPatterns.Load(cacheKey)
	if ok {
		compiled, _ := cached.(*regexp.Regexp)
		return compiled
	}
	compiled := regexp.MustCompile(fmt.Sprintf(format, regexp.QuoteMeta(key)))
	keyedPatterns.Store(cacheKey, compiled)
	return compiled
}

// Load reads and parses a PDF file from disk.
func Load(path string) (*PdfReader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotReadFile, err)
	}
	return Parse(data)
}

// Parse reads and parses a PDF byte slice.
func Parse(data []byte) (*PdfReader, error) {
	return ParseWithOptions(data, ReadOptions{})
}

// ParseWithOptions reads and parses a PDF byte slice with custom options.
func ParseWithOptions(data []byte, opts ReadOptions) (*PdfReader, error) {
	trimmed, err := trimToPDFHeader(data)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(trimmed, []byte("/Encrypt")) {
		return nil, ErrEncryptedUnsupported
	}

	objects, err := parseObjects(trimmed)
	if err != nil {
		return nil, err
	}
	rootID, err := parseRootID(trimmed, objects)
	if err != nil {
		return nil, err
	}
	root := objects[rootID]
	pagesRootID, err := parseReferenceForKey(root.content, "Pages")
	if err != nil {
		return nil, fmt.Errorf("%w: catalog has no /Pages reference", ErrUnsupportedPDF)
	}

	r := &PdfReader{
		data:        slices.Clone(trimmed),
		version:     parseVersion(trimmed),
		objects:     objects,
		rootID:      rootID,
		pagesRootID: pagesRootID,
		strictness:  opts.Strictness,
	}
	err = r.parsePageTree()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// RawBytes returns a copy of the parsed PDF bytes.
func (r *PdfReader) RawBytes() []byte {
	if r == nil {
		return nil
	}
	return slices.Clone(r.data)
}

// Version returns the PDF version from the header.
func (r *PdfReader) Version() string {
	if r == nil {
		return ""
	}
	return r.version
}

// PageCount returns the number of pages in the document.
func (r *PdfReader) PageCount() int {
	if r == nil {
		return 0
	}
	return len(r.pages)
}

// Page returns page metadata by zero-based index.
func (r *PdfReader) Page(index int) (*PageInfo, error) {
	if r == nil || index < 0 || index >= len(r.pages) {
		return nil, fmt.Errorf("%w: %d", ErrPageOutOfRange, index)
	}
	return r.pages[index], nil
}

// VisibleBox returns the effective visible page box.
func (p *PageInfo) VisibleBox() Box {
	if p == nil {
		return Box{}
	}
	if !p.CropBox.IsZero() {
		return p.CropBox
	}
	return p.MediaBox
}

// ContentStream returns the decoded page content stream bytes.
func (p *PageInfo) ContentStream() ([]byte, error) {
	if p == nil || p.reader == nil {
		return nil, fmt.Errorf("%w: nil page", ErrPageOutOfRange)
	}
	pageObject, ok := p.reader.objects[p.objectID]
	if !ok {
		return nil, fmt.Errorf("%w: page object %d missing", ErrUnsupportedPDF, p.objectID)
	}

	refs := parseContentReferences(pageObject.content)
	if len(refs) == 0 {
		if bytes.Contains(pageObject.content, []byte("stream")) {
			return p.reader.decodeStream(pageObject.content)
		}
		return nil, nil
	}

	var out bytes.Buffer
	for _, ref := range refs {
		object, ok := p.reader.objects[ref]
		if !ok {
			return nil, fmt.Errorf("%w: content object %d missing", ErrUnsupportedPDF, ref)
		}
		data, err := p.reader.decodeStream(object.content)
		if err != nil {
			return nil, err
		}
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.Write(data)
	}
	return out.Bytes(), nil
}

// ExtractText extracts simple text operators from decoded page content streams.
func (p *PageInfo) ExtractText() (string, error) {
	content, err := p.ContentStream()
	if err != nil {
		return "", err
	}
	return extractTextFromContent(content), nil
}

// Modifier contains the bytes produced by reader operations such as Merge.
type Modifier struct {
	data      []byte
	pageCount int
}

// Bytes returns a copy of the modified PDF bytes.
func (m *Modifier) Bytes() []byte {
	if m == nil {
		return nil
	}
	return slices.Clone(m.data)
}

// PageCount returns the expected page count for the modified PDF.
func (m *Modifier) PageCount() int {
	if m == nil {
		return 0
	}
	return m.pageCount
}

// SaveTo writes the modified PDF to path.
func (m *Modifier) SaveTo(path string) error {
	if m == nil {
		return fmt.Errorf("%w: nil modifier", ErrUnsupportedPDF)
	}
	err := os.WriteFile(path, m.data, 0o600)
	if err != nil {
		return fmt.Errorf("reader: save: %w", err)
	}
	return nil
}

// Merge concatenates parsed PDFs using Paper's existing merge engine.
func Merge(readers ...*PdfReader) (*Modifier, error) {
	if len(readers) == 0 {
		return nil, fmt.Errorf("%w: no PDFs provided", merge.ErrCannotMergePDFs)
	}
	pdfs := make([][]byte, 0, len(readers))
	pageCount := 0
	for _, r := range readers {
		if r == nil {
			return nil, fmt.Errorf("%w: nil reader", merge.ErrCannotMergePDFs)
		}
		pdfs = append(pdfs, r.RawBytes())
		pageCount += r.PageCount()
	}
	merged, err := merge.Bytes(context.Background(), pdfs...)
	if err != nil {
		return nil, err
	}
	return &Modifier{data: merged, pageCount: pageCount}, nil
}

// MergeFiles loads and merges PDF files from disk.
func MergeFiles(paths ...string) (*Modifier, error) {
	readers := make([]*PdfReader, 0, len(paths))
	for _, path := range paths {
		r, err := Load(path)
		if err != nil {
			return nil, err
		}
		readers = append(readers, r)
	}
	return Merge(readers...)
}

// ExtractPages copies selected zero-based pages from r into a new PDF.
// The output order follows pageIndexes.
func ExtractPages(r *PdfReader, pageIndexes ...int) (*Modifier, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: nil reader", ErrUnsupportedPDF)
	}
	if len(pageIndexes) == 0 {
		return nil, fmt.Errorf("%w: no pages selected", ErrPageOutOfRange)
	}
	for _, pageIndex := range pageIndexes {
		if pageIndex < 0 || pageIndex >= r.PageCount() {
			return nil, fmt.Errorf("%w: %d", ErrPageOutOfRange, pageIndex)
		}
	}
	out, err := merge.BytesSelected(context.Background(), merge.PageSelection{
		PDF:   r.RawBytes(),
		Pages: slices.Clone(pageIndexes),
	})
	if err != nil {
		return nil, err
	}
	return &Modifier{data: out, pageCount: len(pageIndexes)}, nil
}

func trimToPDFHeader(data []byte) ([]byte, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("%w: file too small", ErrUnsupportedPDF)
	}
	searchLen := min(len(data), 1024)
	offset := bytes.Index(data[:searchLen], []byte("%PDF-"))
	if offset < 0 {
		return nil, fmt.Errorf("%w: missing PDF header", ErrUnsupportedPDF)
	}
	return data[offset:], nil
}

func parseVersion(data []byte) string {
	match := headerRe.FindSubmatch(data)
	if len(match) != 2 {
		return "1.3"
	}
	return string(match[1])
}

func parseObjects(data []byte) (map[int]pdfObject, error) {
	matches := objectRe.FindAllSubmatchIndex(data, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: no indirect objects found", ErrUnsupportedPDF)
	}
	objects := make(map[int]pdfObject, len(matches))
	for _, match := range matches {
		number, err := strconv.Atoi(string(data[match[2]:match[3]]))
		if err != nil {
			return nil, fmt.Errorf("reader: invalid object number: %w", err)
		}
		generation, err := strconv.Atoi(string(data[match[4]:match[5]]))
		if err != nil {
			return nil, fmt.Errorf("reader: invalid object generation: %w", err)
		}
		contentStart := match[6]
		contentEnd := match[7]
		objects[number] = pdfObject{
			number:       number,
			generation:   generation,
			content:      slices.Clone(data[contentStart:contentEnd]),
			contentStart: contentStart,
			contentEnd:   contentEnd,
		}
	}
	return objects, nil
}

func parseRootID(data []byte, objects map[int]pdfObject) (int, error) {
	matches := rootRefRe.FindAllSubmatch(data, -1)
	if len(matches) > 0 {
		id, err := strconv.Atoi(string(matches[len(matches)-1][1]))
		if err != nil {
			return 0, fmt.Errorf("reader: invalid root reference: %w", err)
		}
		if _, ok := objects[id]; ok {
			return id, nil
		}
		return 0, fmt.Errorf("%w: root object %d missing", ErrUnsupportedPDF, id)
	}
	for id, object := range objects {
		if bytes.Contains(object.content, []byte("/Type /Catalog")) {
			return id, nil
		}
	}
	return 0, fmt.Errorf("%w: trailer has no /Root reference", ErrUnsupportedPDF)
}

func parseReferenceForKey(content []byte, key string) (int, error) {
	re := keyedPattern(`/%s\s+(\d+)\s+\d+\s+R`, key)
	match := re.FindSubmatch(content)
	if len(match) != 2 {
		return 0, fmt.Errorf("%w: /%s reference not found", ErrUnsupportedPDF, key)
	}
	id, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, fmt.Errorf("reader: invalid /%s reference: %w", key, err)
	}
	return id, nil
}

func (r *PdfReader) parsePageTree() error {
	pages, err := r.collectPages(r.pagesRootID, inheritedPageState{}, make(map[int]bool))
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("%w: no pages found", ErrUnsupportedPDF)
	}
	for i, page := range pages {
		page.Number = i + 1
	}
	r.pages = pages
	return nil
}

func (r *PdfReader) collectPages(objectID int, inherited inheritedPageState, visited map[int]bool) ([]*PageInfo, error) {
	if visited[objectID] {
		return nil, fmt.Errorf("%w: cyclic page tree at object %d", ErrUnsupportedPDF, objectID)
	}
	visited[objectID] = true
	defer delete(visited, objectID)

	object, ok := r.objects[objectID]
	if !ok {
		return nil, fmt.Errorf("%w: page tree object %d missing", ErrUnsupportedPDF, objectID)
	}
	state := inherited.withObject(object.content)

	switch {
	case pagesTypeRe.Match(object.content):
		kids := parseKids(object.content)
		if len(kids) == 0 {
			return nil, fmt.Errorf("%w: /Pages object %d has no /Kids", ErrUnsupportedPDF, objectID)
		}
		var pages []*PageInfo
		for _, kid := range kids {
			childPages, err := r.collectPages(kid, state, visited)
			if err != nil {
				return nil, err
			}
			pages = append(pages, childPages...)
		}
		return pages, nil

	case pageTypeRe.Match(object.content):
		if state.mediaBox.IsZero() {
			if r.strictness == StrictnessStrict {
				return nil, fmt.Errorf("%w: page %d has no /MediaBox", ErrUnsupportedPDF, objectID)
			}
			state.mediaBox = Box{X2: defaultPageWidthPt, Y2: defaultPageHeightPt}
		}
		visible := state.mediaBox
		if !state.cropBox.IsZero() {
			visible = state.cropBox
		}
		return []*PageInfo{{
			Width:    visible.Width(),
			Height:   visible.Height(),
			Rotate:   state.rotate,
			MediaBox: state.mediaBox,
			CropBox:  state.cropBox,
			BleedBox: state.bleedBox,
			TrimBox:  state.trimBox,
			ArtBox:   state.artBox,
			reader:   r,
			objectID: objectID,
		}}, nil
	default:
		return nil, fmt.Errorf("%w: object %d is not a page node", ErrUnsupportedPDF, objectID)
	}
}

func (s inheritedPageState) withObject(content []byte) inheritedPageState {
	if box, ok := parseBox(content, "MediaBox"); ok {
		s.mediaBox = box
	}
	if box, ok := parseBox(content, "CropBox"); ok {
		s.cropBox = box
	}
	if box, ok := parseBox(content, "BleedBox"); ok {
		s.bleedBox = box
	}
	if box, ok := parseBox(content, "TrimBox"); ok {
		s.trimBox = box
	}
	if box, ok := parseBox(content, "ArtBox"); ok {
		s.artBox = box
	}
	if rotate, ok := parseIntegerForKey(content, "Rotate"); ok {
		s.rotate = rotate
	}
	return s
}

func parseKids(content []byte) []int {
	match := kidsArrayRe.FindSubmatch(content)
	if len(match) < 2 {
		return nil
	}
	refMatches := indirectRefRe.FindAllSubmatch(match[1], -1)
	kids := make([]int, 0, len(refMatches))
	for _, refMatch := range refMatches {
		id, err := strconv.Atoi(string(refMatch[1]))
		if err == nil {
			kids = append(kids, id)
		}
	}
	return kids
}

func parseBox(content []byte, key string) (Box, bool) {
	re := keyedPattern(`/%s\s*\[([^\]]+)\]`, key)
	match := re.FindSubmatch(content)
	if len(match) != 2 {
		return Box{}, false
	}
	fields := strings.Fields(string(match[1]))
	if len(fields) < 4 {
		return Box{}, false
	}
	values := make([]float64, 4)
	for i := range 4 {
		value, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return Box{}, false
		}
		values[i] = value
	}
	return Box{X1: values[0], Y1: values[1], X2: values[2], Y2: values[3]}, true
}

func parseIntegerForKey(content []byte, key string) (int, bool) {
	re := keyedPattern(`/%s\s+(-?\d+)`, key)
	match := re.FindSubmatch(content)
	if len(match) != 2 {
		return 0, false
	}
	value, err := strconv.Atoi(string(match[1]))
	return value, err == nil
}

func parseContentReferences(pageContent []byte) []int {
	match := contentsRe.FindSubmatch(pageContent)
	if len(match) == 0 {
		return nil
	}
	source := match[1]
	if len(match[2]) > 0 {
		source = match[2]
	}
	refMatches := indirectRefRe.FindAllSubmatch(source, -1)
	refs := make([]int, 0, len(refMatches))
	for _, refMatch := range refMatches {
		id, err := strconv.Atoi(string(refMatch[1]))
		if err == nil {
			refs = append(refs, id)
		}
	}
	return refs
}

func (r *PdfReader) decodeStream(objectContent []byte) ([]byte, error) {
	start, end, err := streamDataBounds(objectContent)
	if err != nil {
		return nil, err
	}
	data := objectContent[start:end]

	filter := parseFilter(objectContent[:start])
	if filter == "" {
		return slices.Clone(data), nil
	}
	if strings.Contains(filter, "FlateDecode") {
		decoded, err := flateDecode(data)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	}
	if r.strictness == StrictnessStrict {
		return nil, fmt.Errorf("%w: stream filter %s", ErrUnsupportedPDF, filter)
	}
	return slices.Clone(data), nil
}

func streamDataBounds(objectContent []byte) (int, int, error) {
	streamStart := bytes.Index(objectContent, []byte("stream"))
	streamEnd := bytes.LastIndex(objectContent, []byte("endstream"))
	if streamStart < 0 || streamEnd < streamStart {
		return 0, 0, fmt.Errorf("%w: object has no stream", ErrUnsupportedPDF)
	}
	start := streamStart + len("stream")
	switch {
	case bytes.HasPrefix(objectContent[start:], []byte("\r\n")):
		start += 2
	case bytes.HasPrefix(objectContent[start:], []byte("\n")) || bytes.HasPrefix(objectContent[start:], []byte("\r")):
		start++
	}
	end := streamEnd
	for end > start && (objectContent[end-1] == '\r' || objectContent[end-1] == '\n') {
		end--
	}
	return start, end, nil
}

func parseFilter(dictionary []byte) string {
	match := filterRe.FindSubmatch(dictionary)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}

const (
	// maxDeflateExpansion is deflate's theoretical maximum expansion ratio
	// (1032:1) with slack. Bounding a decoded stream by this multiple of its
	// compressed size keeps memory use proportional to the PDF the caller handed
	// over, rather than to whatever a crafted stream expands into.
	maxDeflateExpansion = 1100

	// maxDecodedStreamBytes is the ceiling for a single decoded content stream.
	// Even a PDF that is mostly compressed payload cannot make one page's
	// operators exceed this.
	maxDecodedStreamBytes = 512 << 20
)

func flateDecode(data []byte) ([]byte, error) {
	return flateDecodeLimit(data, min(maxDeflateExpansion*int64(len(data)), maxDecodedStreamBytes))
}

// flateDecodeLimit inflates data, refusing streams that decode to more than
// limit bytes instead of allocating whatever they expand to.
func flateDecodeLimit(data []byte, limit int64) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: flate decode: %w", ErrUnsupportedPDF, err)
	}
	defer func() { _ = zr.Close() }()
	decoded, err := io.ReadAll(io.LimitReader(zr, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: flate read: %w", ErrUnsupportedPDF, err)
	}
	if int64(len(decoded)) > limit {
		return nil, fmt.Errorf(
			"%w: flate stream expands past the %d byte stream limit",
			ErrUnsupportedPDF,
			limit,
		)
	}
	return decoded, nil
}

type tokenKind int

const (
	tokenOther tokenKind = iota
	tokenString
)

type pdfToken struct {
	kind  tokenKind
	value string
}

func extractTextFromContent(content []byte) string {
	tokens := tokenizeContent(content)
	var out strings.Builder
	for i, token := range tokens {
		appendTextForOperator(&out, tokens, i, token.value)
	}
	return strings.TrimSpace(out.String())
}

func appendTextForOperator(out *strings.Builder, tokens []pdfToken, index int, operator string) {
	switch operator {
	case "Tj", "'", "\"":
		appendPreviousString(out, tokens, index)
		if operator == "'" {
			out.WriteByte('\n')
		}
	case "TJ":
		appendTextArray(out, tokens, index)
	case "T*", "Td", "TD":
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
	}
}

func appendPreviousString(out *strings.Builder, tokens []pdfToken, index int) {
	if index > 0 && tokens[index-1].kind == tokenString {
		out.WriteString(tokens[index-1].value)
	}
}

func appendTextArray(out *strings.Builder, tokens []pdfToken, operatorIndex int) {
	start := -1
	for i := operatorIndex - 1; i >= 0; i-- {
		if tokens[i].value == "[" {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return
	}
	for _, part := range tokens[start:operatorIndex] {
		if part.kind == tokenString {
			out.WriteString(part.value)
		}
	}
}

func tokenizeContent(data []byte) []pdfToken {
	tokens := make([]pdfToken, 0)
	for i := 0; i < len(data); {
		c := data[i]
		switch {
		case pdfscan.IsSpace(c):
			i++
		case c == '%':
			for i < len(data) && data[i] != '\n' && data[i] != '\r' {
				i++
			}
		case c == '(':
			value, next := parseLiteralString(data, i+1)
			tokens = append(tokens, pdfToken{kind: tokenString, value: value})
			i = next
		case c == '<':
			if i+1 < len(data) && data[i+1] == '<' {
				tokens = append(tokens, pdfToken{value: "<<"})
				i += 2
				continue
			}
			value, next := parseHexString(data, i+1)
			tokens = append(tokens, pdfToken{kind: tokenString, value: value})
			i = next
		case c == '>' && i+1 < len(data) && data[i+1] == '>':
			tokens = append(tokens, pdfToken{value: ">>"})
			i += 2
		case c == '[' || c == ']':
			tokens = append(tokens, pdfToken{value: string(c)})
			i++
		default:
			start := i
			for i < len(data) && !pdfscan.IsSpace(data[i]) && !pdfscan.IsDelimiter(data[i]) {
				i++
			}
			if start == i {
				tokens = append(tokens, pdfToken{value: string(data[i])})
				i++
				continue
			}
			tokens = append(tokens, pdfToken{value: string(data[start:i])})
		}
	}
	return tokens
}

func parseLiteralString(data []byte, start int) (string, int) {
	var out strings.Builder
	depth := 1
	for i := start; i < len(data); i++ {
		c := data[i]
		switch c {
		case '\\':
			escaped, next := parseLiteralEscape(data, i+1)
			out.WriteString(escaped)
			i = next - 1
		case '(':
			depth++
			out.WriteByte(c)
		case ')':
			depth--
			if depth == 0 {
				return out.String(), i + 1
			}
			out.WriteByte(c)
		default:
			out.WriteByte(c)
		}
	}
	return out.String(), len(data)
}

func parseLiteralEscape(data []byte, start int) (string, int) {
	if start >= len(data) {
		return "", start
	}
	next := data[start]
	switch next {
	case 'n':
		return "\n", start + 1
	case 'r':
		return "\r", start + 1
	case 't':
		return "\t", start + 1
	case 'b':
		return "\b", start + 1
	case 'f':
		return "\f", start + 1
	case '(', ')', '\\':
		return string(next), start + 1
	case '\n':
		return "", start + 1
	case '\r':
		if start+1 < len(data) && data[start+1] == '\n' {
			return "", start + 2
		}
		return "", start + 1
	default:
		if isOctalDigit(next) {
			return parseOctalEscape(data, start)
		}
		return string(next), start + 1
	}
}

func parseOctalEscape(data []byte, start int) (string, int) {
	end := start
	for end < len(data) && end-start < 3 && isOctalDigit(data[end]) {
		end++
	}
	value, err := strconv.ParseInt(string(data[start:end]), 8, 32)
	if err != nil || value > 0xff {
		return "", end
	}
	return string([]byte{byte(value & 0xFF)}), end
}

func parseHexString(data []byte, start int) (string, int) {
	var hexDigits strings.Builder
	i := start
	for ; i < len(data); i++ {
		if data[i] == '>' {
			i++
			break
		}
		if pdfscan.IsSpace(data[i]) {
			continue
		}
		hexDigits.WriteByte(data[i])
	}
	raw := hexDigits.String()
	if len(raw)%2 == 1 {
		raw += "0"
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return "", i
	}
	return string(decoded), i
}

func isOctalDigit(c byte) bool {
	return c >= '0' && c <= '7'
}
