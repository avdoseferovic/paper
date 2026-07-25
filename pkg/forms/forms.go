// Package forms provides AcroForm support for generated Paper PDFs.
package forms

import "github.com/avdoseferovic/paper/pkg/core/entity"

// These are aliases for the form types in pkg/core/entity, so callers can build
// forms without importing the entity package directly.
type (
	// FieldType is the kind of a form field.
	FieldType = entity.FieldType
	// FieldFlags is the set of behaviour flags on a field.
	FieldFlags = entity.FieldFlags
	// RadioOption is one button in a radio group, with its own page and box.
	RadioOption = entity.RadioOption
	// Field is a single form field.
	Field = entity.Field
	// AcroForm is the set of fields belonging to a document.
	AcroForm = entity.AcroForm
)

// The kinds of form field a Field can be.
const (
	FieldText       = entity.FieldText
	FieldCheckbox   = entity.FieldCheckbox
	FieldRadio      = entity.FieldRadio
	FieldDropdown   = entity.FieldDropdown
	FieldListBox    = entity.FieldListBox
	FieldPushButton = entity.FieldPushButton
	FieldSignature  = entity.FieldSignature
)

// Field flags, grouped by the field kind they apply to: any field, text fields,
// choice fields, then button fields. Combine them with bitwise OR.
const (
	FlagReadOnly = entity.FlagReadOnly
	FlagRequired = entity.FlagRequired
	FlagNoExport = entity.FlagNoExport

	FlagMultiline       = entity.FlagMultiline
	FlagPassword        = entity.FlagPassword
	FlagFileSelect      = entity.FlagFileSelect
	FlagDoNotSpellCheck = entity.FlagDoNotSpellCheck
	FlagDoNotScroll     = entity.FlagDoNotScroll
	FlagComb            = entity.FlagComb
	FlagRichText        = entity.FlagRichText

	FlagCombo       = entity.FlagCombo
	FlagEdit        = entity.FlagEdit
	FlagSort        = entity.FlagSort
	FlagMultiSelect = entity.FlagMultiSelect

	FlagNoToggleToOff  = entity.FlagNoToggleToOff
	FlagRadioFlag      = entity.FlagRadioFlag
	FlagPushButtonFlag = entity.FlagPushButtonFlag
)

// NewAcroForm creates an empty form to add fields to.
func NewAcroForm() *AcroForm {
	return entity.NewAcroForm()
}

// NewTextField creates a single-line text input on the given page. rect is the
// field box as x1, y1, x2, y2.
func NewTextField(name string, rect [4]float64, pageIndex int) *Field {
	return entity.NewTextField(name, rect, pageIndex)
}

// NewMultilineTextField creates a text area that wraps across several lines.
func NewMultilineTextField(name string, rect [4]float64, pageIndex int) *Field {
	return entity.NewMultilineTextField(name, rect, pageIndex)
}

// NewPasswordField creates a text input that masks what is typed into it.
func NewPasswordField(name string, rect [4]float64, pageIndex int) *Field {
	return entity.NewPasswordField(name, rect, pageIndex)
}

// NewCheckbox creates a checkbox, initially ticked when checked is true.
func NewCheckbox(name string, rect [4]float64, pageIndex int, checked bool) *Field {
	return entity.NewCheckbox(name, rect, pageIndex, checked)
}

// NewRadioGroup creates a set of radio buttons that share one name, so only one
// option can be selected. Each option carries its own page and box.
func NewRadioGroup(name string, options []RadioOption) *Field {
	return entity.NewRadioGroup(name, options)
}

// NewDropdown creates a combo box that shows one option at a time.
func NewDropdown(name string, rect [4]float64, pageIndex int, options []string) *Field {
	return entity.NewDropdown(name, rect, pageIndex, options)
}

// NewListBox creates a scrollable list showing several options at once.
func NewListBox(name string, rect [4]float64, pageIndex int, options []string) *Field {
	return entity.NewListBox(name, rect, pageIndex, options)
}

// NewSignatureField creates an empty signature field for a reader to sign.
func NewSignatureField(name string, rect [4]float64, pageIndex int) *Field {
	return entity.NewSignatureField(name, rect, pageIndex)
}
