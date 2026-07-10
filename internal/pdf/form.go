package pdf

import (
	"fmt"
	"strconv"
	"strings"
)

type preparedAcroForm struct {
	ref         int
	fields      []preparedFormField
	pageWidgets map[int][]int
}

type preparedFormField struct {
	field    FormField
	ref      int
	checkbox preparedCheckboxAppearance
	widgets  []preparedFormWidget
}

type preparedFormWidget struct {
	field      FormField
	ref        int
	appearance preparedRadioAppearance
}

type preparedCheckboxAppearance struct {
	yesRef int
	offRef int
}

type preparedRadioAppearance struct {
	exportValue string
	onRef       int
	offRef      int
}

// SetAcroFormFields sets generated interactive PDF AcroForm fields.
func (f *PDF) SetAcroFormFields(fields ...FormField) {
	f.acroFormFields = clonePDFFormFields(fields)
	f.acroForm = nil
}

func clonePDFFormFields(fields []FormField) []FormField {
	if fields == nil {
		return nil
	}
	clones := make([]FormField, len(fields))
	for i, field := range fields {
		clones[i] = clonePDFFormField(field)
	}
	return clones
}

func clonePDFFormField(field FormField) FormField {
	field.BGColor = clonePDFColorTriple(field.BGColor)
	field.BorderColor = clonePDFColorTriple(field.BorderColor)
	field.Options = append([]string(nil), field.Options...)
	field.Children = clonePDFFormFields(field.Children)
	return field
}

func clonePDFColorTriple(color *[3]float64) *[3]float64 {
	if color == nil {
		return nil
	}
	clone := *color
	return &clone
}

func (f *PDF) prepareAcroForm(pageCount int) {
	f.acroForm = nil
	if len(f.acroFormFields) == 0 {
		return
	}

	nextRef := f.n + pageCount*2 + 1
	prepared := &preparedAcroForm{
		pageWidgets: make(map[int][]int),
	}
	for _, field := range f.acroFormFields {
		if strings.TrimSpace(field.Name) == "" {
			continue
		}
		preparedField := preparedFormField{
			field: field,
			ref:   nextRef,
		}
		nextRef++
		if field.Type == FormFieldRadio && len(field.Children) > 0 {
			for _, child := range field.Children {
				widget := preparedFormWidget{
					field: child,
					ref:   nextRef,
					appearance: preparedRadioAppearance{
						exportValue: formExportValue(child),
						onRef:       nextRef + 1,
						offRef:      nextRef + 2,
					},
				}
				nextRef += 3
				preparedField.widgets = append(preparedField.widgets, widget)
				if validFormPage(child.PageIndex, pageCount) {
					prepared.pageWidgets[child.PageIndex] = append(prepared.pageWidgets[child.PageIndex], widget.ref)
				}
			}
		} else {
			if validFormPage(field.PageIndex, pageCount) {
				prepared.pageWidgets[field.PageIndex] = append(prepared.pageWidgets[field.PageIndex], preparedField.ref)
			}
			if field.Type == FormFieldCheckbox {
				preparedField.checkbox = preparedCheckboxAppearance{
					yesRef: nextRef,
					offRef: nextRef + 1,
				}
				nextRef += 2
			}
		}
		prepared.fields = append(prepared.fields, preparedField)
	}
	if len(prepared.fields) == 0 {
		return
	}
	prepared.ref = nextRef
	f.acroForm = prepared
}

func validFormPage(pageIndex, pageCount int) bool {
	return pageIndex >= 0 && pageIndex < pageCount
}

func (f *PDF) putAcroFormObjects() {
	if f.acroForm == nil {
		return
	}
	for _, field := range f.acroForm.fields {
		f.putPreparedFormField(field)
		if f.err != nil {
			return
		}
	}
	f.putAcroFormDictionary()
}

func (f *PDF) putPreparedFormField(field preparedFormField) {
	f.newobjExpected(field.ref)
	if f.err != nil {
		return
	}
	f.putFormFieldDictionary(field)
	f.out("endobj")

	if field.field.Type == FormFieldCheckbox {
		f.putCheckboxAppearanceStreams(field.field, field.checkbox)
		return
	}
	for _, widget := range field.widgets {
		f.putRadioWidgetObject(field.ref, widget)
		if f.err != nil {
			return
		}
		f.putRadioAppearanceStreams(widget.field, widget.appearance)
		if f.err != nil {
			return
		}
	}
}

func (f *PDF) newobjExpected(ref int) {
	f.newobj()
	if f.n != ref {
		f.err = staticErrorf(errAcroFormObjectNumber, "got %d, want %d", f.n, ref)
	}
}

func (f *PDF) putFormFieldDictionary(field preparedFormField) {
	ff := field.field
	f.out("<<")
	f.outf("/T %s", f.textstring(ff.Name))
	f.outf("/FT /%s", formFieldPDFType(ff.Type))
	f.putFormCommonEntries(ff)
	if ff.Type == FormFieldRadio && len(field.widgets) > 0 {
		var kids fmtBuffer
		kids.printf("/Kids [")
		for _, widget := range field.widgets {
			kids.printf("%d 0 R ", widget.ref)
		}
		kids.printf("]")
		f.out(kids.String())
		f.out(">>")
		return
	}
	if validFormPage(ff.PageIndex, f.page) {
		f.putMergedWidgetEntries(ff)
	}
	if ff.Type == FormFieldCheckbox {
		f.outf("/AP << /N << /%s %d 0 R /Off %d 0 R >> >>",
			pdfNameEscape(formExportValue(ff)), field.checkbox.yesRef, field.checkbox.offRef)
		f.outf("/AS /%s", pdfNameEscape(checkboxAppearanceState(ff)))
	}
	f.out(">>")
}

func (f *PDF) putFormCommonEntries(field FormField) {
	if field.Flags != 0 {
		f.outf("/Ff %d", uint32(field.Flags))
	}
	if field.Value != "" {
		f.outf("/V %s", f.formValueObject(field))
	}
	if field.Default != "" {
		f.outf("/DV %s", f.formTextString(field.Default))
	}
	if len(field.Options) > 0 {
		var opt fmtBuffer
		opt.printf("/Opt [")
		for _, option := range field.Options {
			opt.printf("%s ", f.formTextString(option))
		}
		opt.printf("]")
		f.out(opt.String())
	}
}

func (f *PDF) formValueObject(field FormField) string {
	switch field.Type {
	case FormFieldCheckbox, FormFieldRadio:
		return "/" + pdfNameEscape(field.Value)
	case FormFieldText, FormFieldDropdown, FormFieldListBox, FormFieldPushButton, FormFieldSignature:
		return f.formTextString(field.Value)
	default:
		return f.formTextString(field.Value)
	}
}

func (f *PDF) formTextString(value string) string {
	return f.textstring(value)
}

func (f *PDF) putMergedWidgetEntries(field FormField) {
	f.out("/Type /Annot")
	f.out("/Subtype /Widget")
	f.outf("/Rect [%.2f %.2f %.2f %.2f]", field.Rect[0], field.Rect[1], field.Rect[2], field.Rect[3])
	f.outf("/P %d 0 R", formPageObjectNumber(field.PageIndex))
	if field.FontName != "" || field.FontSize != 0 || field.TextColor != [3]float64{} {
		f.outf("/DA %s", f.textstring(formDefaultAppearance(field)))
	}
	f.putWidgetAppearanceCharacteristics(field)
	f.putWidgetBorder(field)
}

func (f *PDF) putWidgetAppearanceCharacteristics(field FormField) {
	if field.BorderColor == nil && field.BGColor == nil {
		return
	}
	f.out("/MK <<")
	if field.BorderColor != nil {
		f.outf("/BC [%.3f %.3f %.3f]", field.BorderColor[0], field.BorderColor[1], field.BorderColor[2])
	}
	if field.BGColor != nil {
		f.outf("/BG [%.3f %.3f %.3f]", field.BGColor[0], field.BGColor[1], field.BGColor[2])
	}
	f.out(">>")
}

func (f *PDF) putWidgetBorder(field FormField) {
	if field.BorderWidth <= 0 {
		return
	}
	f.outf("/BS << /W %.2f /S /S >>", field.BorderWidth)
}

func (f *PDF) putRadioWidgetObject(parentRef int, widget preparedFormWidget) {
	f.newobjExpected(widget.ref)
	if f.err != nil {
		return
	}
	ff := widget.field
	f.out("<<")
	f.out("/Type /Annot")
	f.out("/Subtype /Widget")
	f.outf("/Rect [%.2f %.2f %.2f %.2f]", ff.Rect[0], ff.Rect[1], ff.Rect[2], ff.Rect[3])
	f.outf("/Parent %d 0 R", parentRef)
	if validFormPage(ff.PageIndex, f.page) {
		f.outf("/P %d 0 R", formPageObjectNumber(ff.PageIndex))
	}
	f.putWidgetBorder(ff)
	exportValue := formExportValue(ff)
	f.outf("/AP << /N << /%s %d 0 R /Off %d 0 R >> >>",
		pdfNameEscape(exportValue), widget.appearance.onRef, widget.appearance.offRef)
	if ff.Value == exportValue {
		f.outf("/AS /%s", pdfNameEscape(exportValue))
	} else {
		f.out("/AS /Off")
	}
	f.out(">>")
	f.out("endobj")
}

func (f *PDF) putAcroFormDictionary() {
	f.newobjExpected(f.acroForm.ref)
	if f.err != nil {
		return
	}
	var fields fmtBuffer
	fields.printf("/Fields [")
	for _, field := range f.acroForm.fields {
		fields.printf("%d 0 R ", field.ref)
	}
	fields.printf("]")
	f.out("<<")
	f.out(fields.String())
	f.out("/DR << /Font <<")
	f.out("/Helv << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	f.out("/ZaDb << /Type /Font /Subtype /Type1 /BaseFont /ZapfDingbats >>")
	f.out(">> >>")
	f.out("/DA (/Helv 0 Tf 0 0 0 rg)")
	f.out("/NeedAppearances true")
	f.out(">>")
	f.out("endobj")
}

func (f *PDF) putCheckboxAppearanceStreams(field FormField, appearance preparedCheckboxAppearance) {
	width, height := formRectSize(field.Rect)
	f.putFormXObjectStream(appearance.yesRef, width, height, checkboxOnAppearanceContent(width, height), true)
	if f.err != nil {
		return
	}
	f.putFormXObjectStream(appearance.offRef, width, height, checkboxOffAppearanceContent(width, height), false)
}

func (f *PDF) putRadioAppearanceStreams(field FormField, appearance preparedRadioAppearance) {
	width, height := formRectSize(field.Rect)
	f.putFormXObjectStream(appearance.onRef, width, height, radioOnAppearanceContent(width, height), false)
	if f.err != nil {
		return
	}
	f.putFormXObjectStream(appearance.offRef, width, height, radioOffAppearanceContent(width, height), false)
}

func (f *PDF) putFormXObjectStream(ref int, width, height float64, content string, includeZapf bool) {
	f.newobjExpected(ref)
	if f.err != nil {
		return
	}
	stream := f.encryptedStream([]byte(content))
	if f.err != nil {
		return
	}
	f.out("<<")
	f.out("/Type /XObject")
	f.out("/Subtype /Form")
	f.outf("/BBox [0 0 %.2f %.2f]", width, height)
	if includeZapf {
		f.out("/Resources << /Font << /ZaDb << /Type /Font /Subtype /Type1 /BaseFont /ZapfDingbats >> >> >>")
	}
	f.outf("/Length %d", len(stream))
	f.out(">>")
	f.putstream(stream)
	f.out("endobj")
}

func formFieldPDFType(fieldType FormFieldType) string {
	switch fieldType {
	case FormFieldText:
		return "Tx"
	case FormFieldCheckbox, FormFieldRadio, FormFieldPushButton:
		return "Btn"
	case FormFieldDropdown, FormFieldListBox:
		return "Ch"
	case FormFieldSignature:
		return "Sig"
	default:
		return "Tx"
	}
}

func formDefaultAppearance(field FormField) string {
	fontName := strings.TrimSpace(field.FontName)
	if fontName == "" {
		fontName = "Helv"
	}
	return fmt.Sprintf("/%s %s Tf %s %s %s rg",
		pdfNameEscape(fontName),
		formatFormNumber(field.FontSize),
		formatFormNumber(field.TextColor[0]),
		formatFormNumber(field.TextColor[1]),
		formatFormNumber(field.TextColor[2]))
}

func formatFormNumber(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return fmt.Sprintf("%.2f", v)
}

func formExportValue(field FormField) string {
	if strings.TrimSpace(field.ExportValue) != "" {
		return field.ExportValue
	}
	return "Yes"
}

func checkboxAppearanceState(field FormField) string {
	if field.Value == formExportValue(field) {
		return formExportValue(field)
	}
	return "Off"
}

func formRectSize(rect [4]float64) (float64, float64) {
	width := rect[2] - rect[0]
	height := rect[3] - rect[1]
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	return width, height
}

func formPageObjectNumber(pageIndex int) int {
	return 3 + pageIndex*2
}

func checkboxOnAppearanceContent(width, height float64) string {
	return fmt.Sprintf("q\n0.6 0.6 0.6 RG\n0.5 w\n0 0 %s %s re S\n"+
		"0 0 0 rg\nBT\n/ZaDb %s Tf\n%s %s Td\n(4) Tj\nET\nQ",
		formatFormNumber(width), formatFormNumber(height),
		formatFormNumber(height*0.8), formatFormNumber(width*0.15), formatFormNumber(height*0.15))
}

func checkboxOffAppearanceContent(width, height float64) string {
	return fmt.Sprintf("q\n0.6 0.6 0.6 RG\n0.5 w\n0 0 %s %s re S\nQ",
		formatFormNumber(width), formatFormNumber(height))
}

func radioOnAppearanceContent(width, height float64) string {
	cx := width / 2
	cy := height / 2
	radius := min(width, height) / 2 * 0.8
	dot := radius * 0.5
	return fmt.Sprintf("q\n0.6 0.6 0.6 RG\n0.5 w\n%s %s %s %s re S\n"+
		"0 0 0 rg\n%s %s %s %s re f\nQ",
		formatFormNumber(cx-radius), formatFormNumber(cy-radius),
		formatFormNumber(radius*2), formatFormNumber(radius*2),
		formatFormNumber(cx-dot), formatFormNumber(cy-dot),
		formatFormNumber(dot*2), formatFormNumber(dot*2))
}

func radioOffAppearanceContent(width, height float64) string {
	cx := width / 2
	cy := height / 2
	radius := min(width, height) / 2 * 0.8
	return fmt.Sprintf("q\n0.6 0.6 0.6 RG\n0.5 w\n%s %s %s %s re S\nQ",
		formatFormNumber(cx-radius), formatFormNumber(cy-radius),
		formatFormNumber(radius*2), formatFormNumber(radius*2))
}
