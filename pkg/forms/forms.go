// Package forms provides AcroForm support for generated Paper PDFs.
package forms

import "github.com/avdoseferovic/paper/pkg/core/entity"

type (
	FieldType   = entity.FieldType
	FieldFlags  = entity.FieldFlags
	RadioOption = entity.RadioOption
	Field       = entity.Field
	AcroForm    = entity.AcroForm
)

const (
	FieldText       = entity.FieldText
	FieldCheckbox   = entity.FieldCheckbox
	FieldRadio      = entity.FieldRadio
	FieldDropdown   = entity.FieldDropdown
	FieldListBox    = entity.FieldListBox
	FieldPushButton = entity.FieldPushButton
	FieldSignature  = entity.FieldSignature
)

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

func NewAcroForm() *AcroForm {
	return entity.NewAcroForm()
}

func NewTextField(name string, rect [4]float64, pageIndex int) *Field {
	return entity.NewTextField(name, rect, pageIndex)
}

func NewMultilineTextField(name string, rect [4]float64, pageIndex int) *Field {
	return entity.NewMultilineTextField(name, rect, pageIndex)
}

func NewPasswordField(name string, rect [4]float64, pageIndex int) *Field {
	return entity.NewPasswordField(name, rect, pageIndex)
}

func NewCheckbox(name string, rect [4]float64, pageIndex int, checked bool) *Field {
	return entity.NewCheckbox(name, rect, pageIndex, checked)
}

func NewRadioGroup(name string, options []RadioOption) *Field {
	return entity.NewRadioGroup(name, options)
}

func NewDropdown(name string, rect [4]float64, pageIndex int, options []string) *Field {
	return entity.NewDropdown(name, rect, pageIndex, options)
}

func NewListBox(name string, rect [4]float64, pageIndex int, options []string) *Field {
	return entity.NewListBox(name, rect, pageIndex, options)
}

func NewSignatureField(name string, rect [4]float64, pageIndex int) *Field {
	return entity.NewSignatureField(name, rect, pageIndex)
}
