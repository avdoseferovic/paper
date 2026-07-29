package paper

import (
	"slices"

	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// SetAcroForm attaches interactive form fields to the generated PDF. The
// form is cloned so later mutations of the input do not leak into the
// document.
func (m *Paper) SetAcroForm(form *entity.AcroForm) {
	m.config.AcroForm = entity.CloneAcroForm(form)
}

// SetPdfA configures PDF/A conformance metadata and output intent at runtime.
func (m *Paper) SetPdfA(config entity.PdfAConfig) {
	m.config.PdfA = entity.ClonePdfAConfig(&config)
}

// SetTagged enables or disables tagged (accessible) PDF output at runtime.
func (m *Paper) SetTagged(enabled bool) {
	m.config.TaggedPDF = enabled
}

// SetLanguage sets the document language written to the catalog /Lang entry.
func (m *Paper) SetLanguage(language string) {
	m.config.Language = language
}

// SetViewerPreferences configures catalog viewer hints at runtime.
func (m *Paper) SetViewerPreferences(prefs entity.ViewerPreferences) {
	clone := prefs
	m.config.ViewerPreferences = &clone
}

// SetPageLabels defines the labels viewers show instead of physical page
// numbers.
func (m *Paper) SetPageLabels(labels ...entity.PageLabelRange) {
	m.config.PageLabels = slices.Clone(labels)
}

// AddAttachment embeds a file in the generated PDF.
func (m *Paper) AddAttachment(attachment entity.FileAttachment) {
	m.config.Attachments = append(m.config.Attachments, entity.CloneAttachments([]entity.FileAttachment{attachment})...)
}

// AddNamedDestination defines a named location in the generated PDF.
func (m *Paper) AddNamedDestination(destination entity.NamedDestination) {
	m.config.NamedDestinations = append(m.config.NamedDestinations, destination)
}

// SetFileID sets an explicit trailer /ID for the generated PDF.
func (m *Paper) SetFileID(id []byte) {
	m.config.FileID = slices.Clone(id)
}

// SetDeterministic makes repeated builds of the same document byte-identical.
func (m *Paper) SetDeterministic(deterministic bool) {
	m.config.Deterministic = deterministic
}
