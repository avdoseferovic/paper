package entity

import "fmt"

// FieldType identifies the kind of interactive AcroForm field.
type FieldType int

const (
	FieldText FieldType = iota
	FieldCheckbox
	FieldRadio
	FieldDropdown
	FieldListBox
	FieldPushButton
	FieldSignature
)

// FieldFlags are AcroForm field flags from ISO 32000 section 12.7.3.
type FieldFlags uint32

const (
	FlagReadOnly FieldFlags = 1 << 0
	FlagRequired FieldFlags = 1 << 1
	FlagNoExport FieldFlags = 1 << 2

	FlagMultiline       FieldFlags = 1 << 12
	FlagPassword        FieldFlags = 1 << 13
	FlagFileSelect      FieldFlags = 1 << 20
	FlagDoNotSpellCheck FieldFlags = 1 << 22
	FlagDoNotScroll     FieldFlags = 1 << 23
	FlagComb            FieldFlags = 1 << 24
	FlagRichText        FieldFlags = 1 << 25

	FlagCombo       FieldFlags = 1 << 17
	FlagEdit        FieldFlags = 1 << 18
	FlagSort        FieldFlags = 1 << 19
	FlagMultiSelect FieldFlags = 1 << 21

	FlagNoToggleToOff  FieldFlags = 1 << 14
	FlagRadioFlag      FieldFlags = 1 << 15
	FlagPushButtonFlag FieldFlags = 1 << 16
)

// RadioOption defines one button in a radio group.
type RadioOption struct {
	Value     string
	Rect      [4]float64
	PageIndex int
}

// Field represents a generated PDF AcroForm field.
type Field struct {
	Name      string
	Type      FieldType
	Value     string
	Default   string
	Flags     FieldFlags
	Rect      [4]float64
	PageIndex int

	FontSize    float64
	FontName    string
	TextColor   [3]float64
	BGColor     *[3]float64
	BorderColor *[3]float64
	BorderWidth float64

	Options     []string
	ExportValue string

	children []*Field
}

// AcroForm manages generated PDF interactive form fields.
type AcroForm struct {
	fields []*Field
}

// NewAcroForm creates an empty AcroForm.
func NewAcroForm() *AcroForm {
	return &AcroForm{}
}

// Add appends a field and returns the form for chaining.
func (af *AcroForm) Add(f *Field) *AcroForm {
	if af == nil {
		return af
	}
	if f != nil {
		af.fields = append(af.fields, f)
	}
	return af
}

// Fields returns the top-level fields in insertion order.
func (af *AcroForm) Fields() []*Field {
	if af == nil {
		return nil
	}
	return af.fields
}

// NewTextField creates a single-line text input field.
func NewTextField(name string, rect [4]float64, pageIndex int) *Field {
	return &Field{
		Name:        name,
		Type:        FieldText,
		Rect:        rect,
		PageIndex:   pageIndex,
		FontSize:    12,
		FontName:    "Helv",
		BorderWidth: 1,
	}
}

// NewMultilineTextField creates a multi-line text area field.
func NewMultilineTextField(name string, rect [4]float64, pageIndex int) *Field {
	f := NewTextField(name, rect, pageIndex)
	f.Flags |= FlagMultiline
	return f
}

// NewPasswordField creates a password text field.
func NewPasswordField(name string, rect [4]float64, pageIndex int) *Field {
	f := NewTextField(name, rect, pageIndex)
	f.Flags |= FlagPassword
	return f
}

// NewCheckbox creates a checkbox field.
func NewCheckbox(name string, rect [4]float64, pageIndex int, checked bool) *Field {
	f := &Field{
		Name:        name,
		Type:        FieldCheckbox,
		Rect:        rect,
		PageIndex:   pageIndex,
		ExportValue: "Yes",
		BorderWidth: 1,
	}
	if checked {
		f.Value = "Yes"
	} else {
		f.Value = "Off"
	}
	return f
}

// NewRadioGroup creates a radio button group.
func NewRadioGroup(name string, options []RadioOption) *Field {
	parent := &Field{
		Name:  name,
		Type:  FieldRadio,
		Flags: FlagRadioFlag | FlagNoToggleToOff,
	}
	for _, opt := range options {
		child := &Field{
			Name:        opt.Value,
			Type:        FieldRadio,
			Rect:        opt.Rect,
			PageIndex:   opt.PageIndex,
			ExportValue: opt.Value,
			BorderWidth: 1,
		}
		parent.children = append(parent.children, child)
	}
	return parent
}

// NewDropdown creates a combo-box choice field.
func NewDropdown(name string, rect [4]float64, pageIndex int, options []string) *Field {
	return &Field{
		Name:        name,
		Type:        FieldDropdown,
		Rect:        rect,
		PageIndex:   pageIndex,
		Options:     append([]string(nil), options...),
		Flags:       FlagCombo,
		FontSize:    12,
		FontName:    "Helv",
		BorderWidth: 1,
	}
}

// NewListBox creates a list-box choice field.
func NewListBox(name string, rect [4]float64, pageIndex int, options []string) *Field {
	return &Field{
		Name:        name,
		Type:        FieldListBox,
		Rect:        rect,
		PageIndex:   pageIndex,
		Options:     append([]string(nil), options...),
		FontSize:    12,
		FontName:    "Helv",
		BorderWidth: 1,
	}
}

// NewSignatureField creates an unsigned signature field.
func NewSignatureField(name string, rect [4]float64, pageIndex int) *Field {
	return &Field{
		Name:        name,
		Type:        FieldSignature,
		Rect:        rect,
		PageIndex:   pageIndex,
		BorderWidth: 1,
	}
}

// Children returns radio widget children for a radio group field.
func (f *Field) Children() []*Field {
	if f == nil {
		return nil
	}
	return f.children
}

// SetValue sets the field's current value.
func (f *Field) SetValue(v string) *Field {
	if f != nil {
		f.Value = v
	}
	return f
}

// SetReadOnly marks the field read-only.
func (f *Field) SetReadOnly() *Field {
	if f != nil {
		f.Flags |= FlagReadOnly
	}
	return f
}

// SetRequired marks the field required.
func (f *Field) SetRequired() *Field {
	if f != nil {
		f.Flags |= FlagRequired
	}
	return f
}

// SetBackgroundColor sets the field widget background color as RGB values in [0, 1].
func (f *Field) SetBackgroundColor(r, g, b float64) *Field {
	if f != nil {
		f.BGColor = &[3]float64{r, g, b}
	}
	return f
}

// SetBorderColor sets the field widget border color as RGB values in [0, 1].
func (f *Field) SetBorderColor(r, g, b float64) *Field {
	if f != nil {
		f.BorderColor = &[3]float64{r, g, b}
	}
	return f
}

// CloneAcroForm returns a deep copy of form.
func CloneAcroForm(form *AcroForm) *AcroForm {
	if form == nil {
		return nil
	}
	clone := &AcroForm{fields: make([]*Field, 0, len(form.fields))}
	for _, field := range form.fields {
		clone.fields = append(clone.fields, CloneField(field))
	}
	return clone
}

// CloneField returns a deep copy of field.
func CloneField(field *Field) *Field {
	if field == nil {
		return nil
	}
	clone := *field
	clone.BGColor = cloneColorTriple(field.BGColor)
	clone.BorderColor = cloneColorTriple(field.BorderColor)
	clone.Options = append([]string(nil), field.Options...)
	clone.children = make([]*Field, 0, len(field.children))
	for _, child := range field.children {
		clone.children = append(clone.children, CloneField(child))
	}
	return &clone
}

func cloneColorTriple(color *[3]float64) *[3]float64 {
	if color == nil {
		return nil
	}
	clone := *color
	return &clone
}

func appendAcroFormMap(form *AcroForm, m map[string]any) map[string]any {
	if form == nil || len(form.fields) == 0 {
		return m
	}
	m["config_acroform_fields"] = len(form.fields)
	for i, field := range form.fields {
		prefix := fmt.Sprintf("config_acroform_field_%d", i)
		m[prefix+"_name"] = field.Name
		m[prefix+"_type"] = field.Type
		m[prefix+"_page_index"] = field.PageIndex
	}
	return m
}
