package forms

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/avdoseferovic/paper/pkg/reader"
)

// FormFiller reads AcroForm fields from an existing PDF and writes filled or
// flattened PDF bytes.
type FormFiller struct {
	data []byte
}

type formFieldObject struct {
	objNum     int
	content    []byte
	name       string
	fieldType  string
	value      string
	exportName string
}

type formObject struct {
	content []byte
}

type formPageObject struct {
	content         []byte
	resourcesObjNum int
}

type formPDFInfo struct {
	data            []byte
	objects         map[int]formObject
	rootObjNum      int
	prevXref        int64
	size            int
	maxObjectNumber int
	infoRef         string
	idValue         string
}

var (
	errNilFormReader       = errors.New("forms: reader is nil")
	errFormFieldNotFound   = errors.New("forms: field not found")
	errFormNoObjects       = errors.New("forms: no indirect objects found")
	errFormNoStartXref     = errors.New("forms: startxref not found")
	errFormXrefOutOfBounds = errors.New("forms: xref offset out of bounds")
	errFormNoTrailer       = errors.New("forms: trailer not found")
	errFormNoRoot          = errors.New("forms: trailer has no /Root")
	errFormBadDictionary   = errors.New("forms: field dictionary is malformed")

	formObjectRe = regexp.MustCompile(`(?s)(\d+)\s+\d+\s+obj\s*(.*?)\s*endobj`)
	formRootRe   = regexp.MustCompile(`/Root\s+(\d+)\s+\d+\s+R`)
	formSizeRe   = regexp.MustCompile(`/Size\s+(\d+)`)
	formInfoRe   = regexp.MustCompile(`/Info\s+(\d+\s+\d+\s+R)`)
	formIDRe     = regexp.MustCompile(`(?s)/ID\s*(\[[^\]]+\])`)
	formAPNRe    = regexp.MustCompile(`(?s)/AP\s*<<\s*/N\s*<<(.*?)>>`)
	formNameRe   = regexp.MustCompile(`/([A-Za-z0-9_.#-]+)`)
)

const (
	defaultCheckboxExportName = "Yes"
	formOffState              = "Off"
)

// NewFormFiller creates a filler from a parsed PDF.
func NewFormFiller(r *reader.PdfReader) *FormFiller {
	if r == nil {
		return &FormFiller{}
	}
	return &FormFiller{data: r.RawBytes()}
}

// Bytes returns the current PDF bytes after any appended form updates.
func (ff *FormFiller) Bytes() []byte {
	if ff == nil {
		return nil
	}
	return append([]byte(nil), ff.data...)
}

// SaveTo writes the current PDF bytes to path.
func (ff *FormFiller) SaveTo(path string) error {
	if ff == nil || ff.data == nil {
		return errNilFormReader
	}
	err := os.WriteFile(path, ff.data, 0o600)
	if err != nil {
		return fmt.Errorf("forms: save: %w", err)
	}
	return nil
}

// FieldNames returns form field names in object order.
func (ff *FormFiller) FieldNames() ([]string, error) {
	fields, err := ff.fields()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.name)
	}
	return names, nil
}

// GetValue returns the current value of fieldName.
func (ff *FormFiller) GetValue(fieldName string) (string, error) {
	field, err := ff.field(fieldName)
	if err != nil {
		return "", err
	}
	return field.value, nil
}

// SetValue sets a text or choice field value by appending a replacement field object.
func (ff *FormFiller) SetValue(fieldName, value string) error {
	field, err := ff.field(fieldName)
	if err != nil {
		return err
	}
	updated, err := setFormDictionaryEntry(field.content, "V", pdfLiteralString(value))
	if err != nil {
		return err
	}
	return ff.appendFieldUpdate(field.objNum, updated)
}

// SetCheckbox sets a checkbox field's /V and /AS values.
func (ff *FormFiller) SetCheckbox(fieldName string, checked bool) error {
	field, err := ff.field(fieldName)
	if err != nil {
		return err
	}
	state := formOffState
	if checked {
		state = field.exportName
		if state == "" {
			state = defaultCheckboxExportName
		}
	}
	updated, err := setFormDictionaryEntry(field.content, "V", "/"+state)
	if err != nil {
		return err
	}
	updated, err = setFormDictionaryEntry(updated, "AS", "/"+state)
	if err != nil {
		return err
	}
	return ff.appendFieldUpdate(field.objNum, updated)
}

// Flatten renders supported AcroForm field values into page content and
// rewrites the PDF without active AcroForm/widget objects.
func (ff *FormFiller) Flatten() error {
	if ff == nil || len(ff.data) == 0 {
		return errNilFormReader
	}
	info, err := parseFormPDFInfo(ff.data)
	if err != nil {
		return err
	}
	fields, err := ff.fields()
	if err != nil {
		return err
	}
	objects := cloneFormObjects(info.objects)
	pages := parseFormPageObjects(objects)
	renderedByPage, textPages := collectFlattenedFieldContent(fields, pages)

	omitObjects := flattenOmittedObjects(objects, info.rootObjNum)
	err = removeFormCatalogAcroForm(objects, info.rootObjNum)
	if err != nil {
		return err
	}
	err = appendFlattenedPageStreams(objects, pages, renderedByPage, textPages)
	if err != nil {
		return err
	}
	err = removeFlattenedWidgetAnnotations(objects, pages, omitObjects)
	if err != nil {
		return err
	}

	ff.data = writeFormFullRewrite(info, objects, omitObjects)
	return nil
}

func collectFlattenedFieldContent(fields []formFieldObject, pages map[int]formPageObject) (map[int][]string, map[int]bool) {
	renderedByPage := make(map[int][]string)
	textPages := make(map[int]bool)
	for _, field := range fields {
		pageObjNum, ok := formRefForKey(field.content, "P")
		if !ok {
			continue
		}
		if _, ok := pages[pageObjNum]; !ok {
			continue
		}
		rect, ok := formRectForKey(field.content, "Rect")
		if !ok {
			continue
		}
		content, usesText := flattenFieldContent(field, rect)
		if strings.TrimSpace(content) == "" {
			continue
		}
		renderedByPage[pageObjNum] = append(renderedByPage[pageObjNum], content)
		if usesText {
			textPages[pageObjNum] = true
		}
	}
	return renderedByPage, textPages
}

func removeFormCatalogAcroForm(objects map[int]formObject, rootObjNum int) error {
	catalog, ok := objects[rootObjNum]
	if !ok {
		return fmt.Errorf("%w: catalog object %d missing", errFormNoRoot, rootObjNum)
	}
	objects[rootObjNum] = formObject{
		content: removeFormDictionaryEntry(catalog.content, "AcroForm"),
	}
	return nil
}

func appendFlattenedPageStreams(
	objects map[int]formObject,
	pages map[int]formPageObject,
	renderedByPage map[int][]string,
	textPages map[int]bool,
) error {
	nextObjNum := maxFormObjectNumber(objects) + 1
	for pageObjNum, streams := range renderedByPage {
		page, ok := pages[pageObjNum]
		if !ok {
			continue
		}
		stream := strings.Join(streams, "\n")
		streamObjNum := nextObjNum
		nextObjNum++
		objects[streamObjNum] = formObject{content: formStreamObject([]byte(stream))}

		updatedPage, err := appendFormPageContent(page.content, streamObjNum)
		if err != nil {
			return err
		}
		objects[pageObjNum] = formObject{content: updatedPage}

		if textPages[pageObjNum] && page.resourcesObjNum > 0 {
			resources, ok := objects[page.resourcesObjNum]
			if ok {
				updatedResources, err := ensureFormHelvResource(resources.content)
				if err != nil {
					return err
				}
				objects[page.resourcesObjNum] = formObject{content: updatedResources}
			}
		}
	}
	return nil
}

func removeFlattenedWidgetAnnotations(
	objects map[int]formObject,
	pages map[int]formPageObject,
	omitObjects map[int]bool,
) error {
	for pageObjNum, page := range pages {
		content := page.content
		if updated, ok := objects[pageObjNum]; ok {
			content = updated.content
		}
		updatedPage, err := removeFormPageWidgetAnnots(content, omitObjects)
		if err != nil {
			return err
		}
		objects[pageObjNum] = formObject{content: updatedPage}
	}
	return nil
}

func (ff *FormFiller) field(name string) (formFieldObject, error) {
	fields, err := ff.fields()
	if err != nil {
		return formFieldObject{}, err
	}
	for _, field := range fields {
		if field.name == name {
			return field, nil
		}
	}
	return formFieldObject{}, fmt.Errorf("%w: %q", errFormFieldNotFound, name)
}

func (ff *FormFiller) fields() ([]formFieldObject, error) {
	if ff == nil || len(ff.data) == 0 {
		return nil, errNilFormReader
	}
	objects, _, err := parseFormObjects(ff.data)
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(objects))
	for id := range objects {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	fields := make([]formFieldObject, 0)
	for _, id := range ids {
		content := objects[id].content
		name, ok := formLiteralForKey(content, "T")
		if !ok {
			continue
		}
		fieldType := formNameForKey(content, "FT")
		if fieldType == "" {
			continue
		}
		fields = append(fields, formFieldObject{
			objNum:     id,
			content:    content,
			name:       name,
			fieldType:  fieldType,
			value:      formValueForKey(content, "V"),
			exportName: checkboxExportName(content),
		})
	}
	return fields, nil
}

func (ff *FormFiller) appendFieldUpdate(objNum int, content []byte) error {
	info, err := parseFormPDFInfo(ff.data)
	if err != nil {
		return err
	}
	ff.data = writeFormIncrementalUpdate(info, objNum, content)
	return nil
}

func parseFormPDFInfo(data []byte) (formPDFInfo, error) {
	prevXref, err := findFormStartXref(data)
	if err != nil {
		return formPDFInfo{}, err
	}
	trailer, err := formTrailerBytes(data, prevXref)
	if err != nil {
		return formPDFInfo{}, err
	}
	objects, maxObjNum, err := parseFormObjects(data)
	if err != nil {
		return formPDFInfo{}, err
	}
	rootObjNum, err := parseFormRootObjectNumber(trailer)
	if err != nil {
		return formPDFInfo{}, err
	}
	return formPDFInfo{
		data:            append([]byte(nil), data...),
		objects:         objects,
		rootObjNum:      rootObjNum,
		prevXref:        prevXref,
		size:            parseFormTrailerSize(trailer, maxObjNum+1),
		maxObjectNumber: maxObjNum,
		infoRef:         parseFormTrailerRef(trailer, formInfoRe),
		idValue:         parseFormTrailerRef(trailer, formIDRe),
	}, nil
}

func parseFormObjects(data []byte) (map[int]formObject, int, error) {
	matches := formObjectRe.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil, 0, errFormNoObjects
	}
	objects := make(map[int]formObject, len(matches))
	maxObjNum := 0
	for _, match := range matches {
		number, err := strconv.Atoi(string(match[1]))
		if err != nil {
			return nil, 0, fmt.Errorf("forms: invalid object number: %w", err)
		}
		if number > maxObjNum {
			maxObjNum = number
		}
		objects[number] = formObject{content: append([]byte(nil), match[2]...)}
	}
	return objects, maxObjNum, nil
}

func cloneFormObjects(objects map[int]formObject) map[int]formObject {
	cloned := make(map[int]formObject, len(objects))
	for id, object := range objects {
		cloned[id] = formObject{content: append([]byte(nil), object.content...)}
	}
	return cloned
}

func parseFormPageObjects(objects map[int]formObject) map[int]formPageObject {
	pages := make(map[int]formPageObject)
	for id, object := range objects {
		if !isFormPageObject(object.content) {
			continue
		}
		resourcesObjNum, _ := formRefForKey(object.content, "Resources")
		pages[id] = formPageObject{
			content:         object.content,
			resourcesObjNum: resourcesObjNum,
		}
	}
	return pages
}

func isFormPageObject(content []byte) bool {
	return regexp.MustCompile(`/Type\s*/Page\b`).Match(content)
}

func flattenFieldContent(field formFieldObject, rect [4]float64) (string, bool) {
	switch field.fieldType {
	case "Tx", "Ch":
		if field.value == "" {
			return "", false
		}
		return flattenTextFieldContent(field.value, rect), true
	case "Btn":
		return flattenCheckboxFieldContent(field.value != "" && field.value != formOffState, rect), false
	default:
		return "", false
	}
}

func flattenTextFieldContent(value string, rect [4]float64) string {
	x := rect[0] + 2
	y := rect[1] + 4
	fontSize := 10.0
	height := rect[3] - rect[1]
	if height > 0 {
		fontSize = min(10.0, max(6.0, height-4))
		y = rect[1] + max(2.0, (height-fontSize)/2)
	}
	return fmt.Sprintf("q\nBT\n/Helv %s Tf\n0 0 0 rg\n1 0 0 1 %s %s Tm\n%s Tj\nET\nQ",
		formatFormFloat(fontSize),
		formatFormFloat(x),
		formatFormFloat(y),
		pdfLiteralString(value))
}

func flattenCheckboxFieldContent(checked bool, rect [4]float64) string {
	x := rect[0]
	y := rect[1]
	width := max(1.0, rect[2]-rect[0])
	height := max(1.0, rect[3]-rect[1])
	content := fmt.Sprintf("q\n0 0 0 RG\n0.7 w\n%s %s %s %s re S",
		formatFormFloat(x), formatFormFloat(y), formatFormFloat(width), formatFormFloat(height))
	if checked {
		inset := min(width, height) * 0.2
		content += fmt.Sprintf("\n%s %s m %s %s l %s %s m %s %s l S",
			formatFormFloat(x+inset), formatFormFloat(y+inset),
			formatFormFloat(x+width-inset), formatFormFloat(y+height-inset),
			formatFormFloat(x+width-inset), formatFormFloat(y+inset),
			formatFormFloat(x+inset), formatFormFloat(y+height-inset))
	}
	return content + "\nQ"
}

func flattenOmittedObjects(objects map[int]formObject, rootObjNum int) map[int]bool {
	omit := seedFlattenOmittedObjects(objects, rootObjNum)
	expandFlattenOmittedObjects(objects, omit)
	return omit
}

func seedFlattenOmittedObjects(objects map[int]formObject, rootObjNum int) map[int]bool {
	omit := make(map[int]bool)
	if root, ok := objects[rootObjNum]; ok {
		if acroFormObjNum, ok := formRefForKey(root.content, "AcroForm"); ok {
			omit[acroFormObjNum] = true
		}
	}
	for id, object := range objects {
		if isFormFieldOrWidgetObject(object.content) {
			omit[id] = true
		}
	}
	return omit
}

func expandFlattenOmittedObjects(objects map[int]formObject, omit map[int]bool) {
	changed := true
	for changed {
		changed = expandFlattenOmittedObjectsOnce(objects, omit)
	}
}

func expandFlattenOmittedObjectsOnce(objects map[int]formObject, omit map[int]bool) bool {
	changed := false
	for id := range omit {
		object, ok := objects[id]
		if !ok {
			continue
		}
		if addFlattenOmittedRefs(objects, omit, object.content) {
			changed = true
		}
	}
	return changed
}

func addFlattenOmittedRefs(objects map[int]formObject, omit map[int]bool, content []byte) bool {
	changed := false
	for _, key := range []string{"Fields", "Kids", "AP"} {
		if addFlattenOmittedRefsForKey(objects, omit, content, key) {
			changed = true
		}
	}
	return changed
}

func addFlattenOmittedRefsForKey(objects map[int]formObject, omit map[int]bool, content []byte, key string) bool {
	changed := false
	for _, ref := range formRefsForDictionaryKey(content, key) {
		if omit[ref] || !shouldOmitFormReference(objects, ref, key) {
			continue
		}
		omit[ref] = true
		changed = true
	}
	return changed
}

func shouldOmitFormReference(objects map[int]formObject, ref int, key string) bool {
	if key == "AP" {
		return true
	}
	refObj, ok := objects[ref]
	return ok && isFormFieldOrWidgetObject(refObj.content)
}

func isFormFieldOrWidgetObject(content []byte) bool {
	return bytes.Contains(content, []byte("/FT /")) ||
		bytes.Contains(content, []byte("/Subtype /Widget"))
}

func formStreamObject(content []byte) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "<< /Length %d >>\nstream\n", len(content))
	out.Write(content)
	out.WriteString("\nendstream")
	return out.Bytes()
}

func appendFormPageContent(pageContent []byte, streamObjNum int) ([]byte, error) {
	idx := bytes.Index(pageContent, []byte("/Contents"))
	ref := fmt.Sprintf("%d 0 R", streamObjNum)
	if idx < 0 {
		return setFormDictionaryEntry(pageContent, "Contents", ref)
	}
	valueStart := formSkipSpaces(pageContent, idx+len("/Contents"))
	if valueStart >= len(pageContent) {
		return nil, errFormBadDictionary
	}
	var value string
	switch pageContent[valueStart] {
	case '[':
		valueEnd := formSkipArray(pageContent, valueStart)
		existing := strings.TrimSpace(string(pageContent[valueStart+1 : valueEnd-1]))
		if existing == "" {
			value = "[" + ref + "]"
		} else {
			value = "[" + existing + " " + ref + "]"
		}
	default:
		valueEnd, ok := formSkipIndirectRef(pageContent, valueStart)
		if !ok {
			return nil, errFormBadDictionary
		}
		existing := strings.TrimSpace(string(pageContent[valueStart:valueEnd]))
		value = "[" + existing + " " + ref + "]"
	}
	return setFormDictionaryEntry(pageContent, "Contents", value)
}

func removeFormPageWidgetAnnots(pageContent []byte, omit map[int]bool) ([]byte, error) {
	idx := bytes.Index(pageContent, []byte("/Annots"))
	if idx < 0 {
		return pageContent, nil
	}
	valueStart := formSkipSpaces(pageContent, idx+len("/Annots"))
	if valueStart >= len(pageContent) || pageContent[valueStart] != '[' {
		return pageContent, nil
	}
	valueEnd := formSkipArray(pageContent, valueStart)
	refs := formRefsInBytes(pageContent[valueStart:valueEnd])
	kept := make([]string, 0, len(refs))
	for _, ref := range refs {
		if !omit[ref] {
			kept = append(kept, fmt.Sprintf("%d 0 R", ref))
		}
	}
	if len(kept) == 0 {
		return removeFormDictionaryEntry(pageContent, "Annots"), nil
	}
	return setFormDictionaryEntry(pageContent, "Annots", "["+strings.Join(kept, " ")+"]")
}

func ensureFormHelvResource(resources []byte) ([]byte, error) {
	if bytes.Contains(resources, []byte("/Helv")) {
		return resources, nil
	}
	helv := []byte(" /Helv << /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >> ")
	fontIdx := bytes.Index(resources, []byte("/Font"))
	if fontIdx >= 0 {
		valueStart := formSkipSpaces(resources, fontIdx+len("/Font"))
		if valueStart < len(resources) && bytes.HasPrefix(resources[valueStart:], []byte("<<")) {
			valueEnd := formSkipBalanced(resources, valueStart, []byte("<<"), []byte(">>"))
			if valueEnd >= valueStart+4 {
				insertAt := valueEnd - len(">>")
				out := make([]byte, 0, len(resources)+len(helv))
				out = append(out, resources[:insertAt]...)
				out = append(out, helv...)
				out = append(out, resources[insertAt:]...)
				return out, nil
			}
		}
	}
	end := bytes.LastIndex(resources, []byte(">>"))
	if end < 0 {
		return nil, errFormBadDictionary
	}
	fontDict := []byte(" /Font <<")
	fontDict = append(fontDict, helv...)
	fontDict = append(fontDict, []byte(">> ")...)
	out := make([]byte, 0, len(resources)+len(fontDict))
	out = append(out, resources[:end]...)
	out = append(out, fontDict...)
	out = append(out, resources[end:]...)
	return out, nil
}

func writeFormFullRewrite(info formPDFInfo, objects map[int]formObject, omit map[int]bool) []byte {
	ids := make([]int, 0, len(objects))
	maxObjNum := 0
	for id := range objects {
		if id > maxObjNum {
			maxObjNum = id
		}
		if !omit[id] {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%%PDF-%s\n", parseFormPDFVersion(info.data))
	offsets := make(map[int]int, len(ids))
	for _, id := range ids {
		offsets[id] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", id)
		buf.Write(bytes.TrimSpace(objects[id].content))
		buf.WriteString("\nendobj\n")
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", maxObjNum+1)
	buf.WriteString("0000000000 65535 f \n")
	for id := 1; id <= maxObjNum; id++ {
		offset, ok := offsets[id]
		if !ok {
			buf.WriteString("0000000000 65535 f \n")
			continue
		}
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root %d 0 R", maxObjNum+1, info.rootObjNum)
	if info.infoRef != "" {
		fmt.Fprintf(&buf, " /Info %s", info.infoRef)
	}
	if info.idValue != "" {
		fmt.Fprintf(&buf, " /ID %s", info.idValue)
	}
	fmt.Fprintf(&buf, " >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)
	return buf.Bytes()
}

func parseFormPDFVersion(data []byte) string {
	match := regexp.MustCompile(`%PDF-(\d+\.\d+)`).FindSubmatch(data)
	if len(match) != 2 {
		return "1.3"
	}
	return string(match[1])
}

func maxFormObjectNumber(objects map[int]formObject) int {
	maxObjNum := 0
	for id := range objects {
		if id > maxObjNum {
			maxObjNum = id
		}
	}
	return maxObjNum
}

func writeFormIncrementalUpdate(info formPDFInfo, objNum int, content []byte) []byte {
	var buf bytes.Buffer
	buf.Write(info.data)
	if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	offset := buf.Len()
	fmt.Fprintf(&buf, "%d 0 obj\n", objNum)
	buf.Write(bytes.TrimSpace(content))
	buf.WriteString("\nendobj\n")

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n%d 1\n%010d 00000 n \n", objNum, offset)
	trailerSize := max(info.size, max(info.maxObjectNumber, objNum)+1)
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

func findFormStartXref(data []byte) (int64, error) {
	searchLen := min(1024, len(data))
	tail := data[len(data)-searchLen:]
	idx := bytes.LastIndex(tail, []byte("startxref"))
	if idx < 0 {
		return 0, fmt.Errorf("%w: last %d bytes", errFormNoStartXref, searchLen)
	}
	after := strings.TrimSpace(string(tail[idx+len("startxref"):]))
	if nl := strings.IndexAny(after, "\r\n"); nl > 0 {
		after = after[:nl]
	}
	offset, err := strconv.ParseInt(strings.TrimSpace(after), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("forms: invalid startxref offset: %w", err)
	}
	if offset < 0 || offset >= int64(len(data)) {
		return 0, fmt.Errorf("%w: %d", errFormXrefOutOfBounds, offset)
	}
	return offset, nil
}

func formTrailerBytes(data []byte, xrefOffset int64) ([]byte, error) {
	if xrefOffset < 0 || xrefOffset >= int64(len(data)) {
		return nil, fmt.Errorf("%w: %d", errFormXrefOutOfBounds, xrefOffset)
	}
	segment := data[xrefOffset:]
	trailerIdx := bytes.Index(segment, []byte("trailer"))
	if trailerIdx < 0 {
		return nil, errFormNoTrailer
	}
	start := trailerIdx + len("trailer")
	end := bytes.Index(segment[start:], []byte("startxref"))
	if end < 0 {
		end = len(segment) - start
	}
	return bytes.TrimSpace(segment[start : start+end]), nil
}

func parseFormRootObjectNumber(trailer []byte) (int, error) {
	match := formRootRe.FindSubmatch(trailer)
	if len(match) != 2 {
		return 0, errFormNoRoot
	}
	rootObjNum, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, fmt.Errorf("forms: invalid root reference: %w", err)
	}
	return rootObjNum, nil
}

func parseFormTrailerSize(trailer []byte, fallback int) int {
	match := formSizeRe.FindSubmatch(trailer)
	if len(match) != 2 {
		return fallback
	}
	size, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return fallback
	}
	return size
}

func parseFormTrailerRef(trailer []byte, re *regexp.Regexp) string {
	match := re.FindSubmatch(trailer)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}

func setFormDictionaryEntry(content []byte, key, value string) ([]byte, error) {
	cleaned := removeFormDictionaryEntry(bytes.TrimSpace(content), key)
	end := bytes.LastIndex(cleaned, []byte(">>"))
	if end < 0 {
		return nil, errFormBadDictionary
	}
	var out bytes.Buffer
	out.Write(bytes.TrimRight(cleaned[:end], " \t\r\n"))
	fmt.Fprintf(&out, " /%s %s ", key, value)
	out.Write(bytes.TrimSpace(cleaned[end:]))
	return out.Bytes(), nil
}

func removeFormDictionaryEntry(dictionary []byte, key string) []byte {
	idx := formKeyIndex(dictionary, key)
	if idx < 0 {
		return dictionary
	}
	valueStart := formSkipSpaces(dictionary, idx+len(key)+1)
	valueEnd := formSkipPDFValue(dictionary, valueStart)
	if valueEnd <= valueStart {
		return dictionary
	}
	out := make([]byte, 0, len(dictionary)-(valueEnd-idx))
	out = append(out, bytes.TrimRight(dictionary[:idx], " \t\r\n")...)
	out = append(out, ' ')
	out = append(out, bytes.TrimLeft(dictionary[valueEnd:], " \t\r\n")...)
	return out
}

func formLiteralForKey(content []byte, key string) (string, bool) {
	idx := formKeyIndex(content, key)
	if idx < 0 {
		return "", false
	}
	start := formSkipSpaces(content, idx+len(key)+1)
	if start >= len(content) || content[start] != '(' {
		return "", false
	}
	value, _ := parseFormLiteral(content, start)
	return value, true
}

func formRefForKey(content []byte, key string) (int, bool) {
	idx := formKeyIndex(content, key)
	if idx < 0 {
		return 0, false
	}
	start := formSkipSpaces(content, idx+len(key)+1)
	firstEnd := formSkipToken(content, start)
	secondStart := formSkipSpaces(content, firstEnd)
	secondEnd := formSkipToken(content, secondStart)
	refStart := formSkipSpaces(content, secondEnd)
	if start >= len(content) || refStart >= len(content) || content[refStart] != 'R' {
		return 0, false
	}
	_, err := strconv.Atoi(string(content[secondStart:secondEnd]))
	if err != nil {
		return 0, false
	}
	ref, err := strconv.Atoi(string(content[start:firstEnd]))
	if err != nil {
		return 0, false
	}
	return ref, true
}

func formRectForKey(content []byte, key string) ([4]float64, bool) {
	idx := formKeyIndex(content, key)
	if idx < 0 {
		return [4]float64{}, false
	}
	start := formSkipSpaces(content, idx+len(key)+1)
	if start >= len(content) || content[start] != '[' {
		return [4]float64{}, false
	}
	end := formSkipArray(content, start)
	fields := strings.Fields(string(content[start+1 : end-1]))
	if len(fields) < 4 {
		return [4]float64{}, false
	}
	x1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return [4]float64{}, false
	}
	y1, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return [4]float64{}, false
	}
	x2, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return [4]float64{}, false
	}
	y2, err := strconv.ParseFloat(fields[3], 64)
	if err != nil {
		return [4]float64{}, false
	}
	return [4]float64{x1, y1, x2, y2}, true
}

func formNameForKey(content []byte, key string) string {
	re := regexp.MustCompile(`/` + regexp.QuoteMeta(key) + `\s*/([A-Za-z0-9_.#-]+)`)
	match := re.FindSubmatch(content)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}

func formValueForKey(content []byte, key string) string {
	idx := formKeyIndex(content, key)
	if idx < 0 {
		return ""
	}
	start := formSkipSpaces(content, idx+len(key)+1)
	if start >= len(content) {
		return ""
	}
	switch content[start] {
	case '(':
		value, _ := parseFormLiteral(content, start)
		return value
	case '/':
		end := formSkipToken(content, start+1)
		return string(content[start+1 : end])
	default:
		return ""
	}
}

func checkboxExportName(content []byte) string {
	match := formAPNRe.FindSubmatch(content)
	if len(match) != 2 {
		return defaultCheckboxExportName
	}
	names := formNameRe.FindAllSubmatch(match[1], -1)
	for _, name := range names {
		if len(name) == 2 && string(name[1]) != formOffState {
			return string(name[1])
		}
	}
	return defaultCheckboxExportName
}

func formRefsForDictionaryKey(content []byte, key string) []int {
	idx := formKeyIndex(content, key)
	if idx < 0 {
		return nil
	}
	start := formSkipSpaces(content, idx+len(key)+1)
	end := formSkipPDFValue(content, start)
	if end <= start {
		return nil
	}
	return formRefsInBytes(content[start:end])
}

func formRefsInBytes(content []byte) []int {
	re := regexp.MustCompile(`(\d+)\s+\d+\s+R`)
	matches := re.FindAllSubmatch(content, -1)
	refs := make([]int, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		ref, err := strconv.Atoi(string(match[1]))
		if err == nil {
			refs = append(refs, ref)
		}
	}
	return refs
}

func formKeyIndex(content []byte, key string) int {
	marker := []byte("/" + key)
	searchFrom := 0
	for searchFrom < len(content) {
		idx := bytes.Index(content[searchFrom:], marker)
		if idx < 0 {
			return -1
		}
		idx += searchFrom
		after := idx + len(marker)
		if after >= len(content) || !isFormNameChar(content[after]) {
			return idx
		}
		searchFrom = after
	}
	return -1
}

func isFormNameChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '-' || c == '.' || c == '#'
}

func pdfLiteralString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "(", "\\(")
	value = strings.ReplaceAll(value, ")", "\\)")
	return "(" + value + ")"
}

func formatFormFloat(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func formSkipPDFValue(data []byte, start int) int {
	start = formSkipSpaces(data, start)
	if start >= len(data) {
		return start
	}
	if bytes.HasPrefix(data[start:], []byte("<<")) {
		return formSkipBalanced(data, start, []byte("<<"), []byte(">>"))
	}
	if end, ok := formSkipIndirectRef(data, start); ok {
		return end
	}
	switch data[start] {
	case '[':
		return formSkipArray(data, start)
	case '(':
		_, end := parseFormLiteral(data, start)
		return end
	}
	return formSkipToken(data, start)
}

func formSkipIndirectRef(data []byte, start int) (int, bool) {
	start = formSkipSpaces(data, start)
	firstEnd := formSkipToken(data, start)
	if firstEnd <= start {
		return start, false
	}
	_, err := strconv.Atoi(string(data[start:firstEnd]))
	if err != nil {
		return start, false
	}
	secondStart := formSkipSpaces(data, firstEnd)
	secondEnd := formSkipToken(data, secondStart)
	if secondEnd <= secondStart {
		return start, false
	}
	_, err = strconv.Atoi(string(data[secondStart:secondEnd]))
	if err != nil {
		return start, false
	}
	refStart := formSkipSpaces(data, secondEnd)
	if refStart >= len(data) || data[refStart] != 'R' {
		return start, false
	}
	return refStart + 1, true
}

func formSkipBalanced(data []byte, start int, open, closing []byte) int {
	depth := 0
	for i := start; i < len(data); {
		switch {
		case bytes.HasPrefix(data[i:], open):
			depth++
			i += len(open)
		case bytes.HasPrefix(data[i:], closing):
			depth--
			i += len(closing)
			if depth == 0 {
				return i
			}
		case data[i] == '(':
			_, i = parseFormLiteral(data, i)
		default:
			i++
		}
	}
	return len(data)
}

func formSkipArray(data []byte, start int) int {
	depth := 0
	for i := start; i < len(data); i++ {
		switch data[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i + 1
			}
		case '(':
			_, i = parseFormLiteral(data, i)
			i--
		}
	}
	return len(data)
}

func parseFormLiteral(data []byte, start int) (string, int) {
	var out strings.Builder
	depth := 0
	for i := start; i < len(data); i++ {
		switch data[i] {
		case '\\':
			if i+1 < len(data) {
				out.WriteByte(data[i+1])
				i++
			}
		case '(':
			depth++
			if depth > 1 {
				out.WriteByte('(')
			}
		case ')':
			depth--
			if depth == 0 {
				return out.String(), i + 1
			}
			out.WriteByte(')')
		default:
			out.WriteByte(data[i])
		}
	}
	return out.String(), len(data)
}

func formSkipSpaces(data []byte, start int) int {
	for start < len(data) && isFormPDFSpace(data[start]) {
		start++
	}
	return start
}

func formSkipToken(data []byte, start int) int {
	i := start
	for i < len(data) && !isFormPDFSpace(data[i]) && !isFormPDFDelimiter(data[i]) {
		i++
	}
	if i == start && i < len(data) {
		return i + 1
	}
	return i
}

func isFormPDFSpace(c byte) bool {
	return c == 0 || c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == ' '
}

func isFormPDFDelimiter(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	default:
		return false
	}
}
