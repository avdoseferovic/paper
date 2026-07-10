package config

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// WithAcroForm attaches interactive form fields to the generated PDF. The
// form is cloned so later mutations of the input do not leak into the config.
func (b *CfgBuilder) WithAcroForm(form *entity.AcroForm) Builder {
	b.acroForm = entity.CloneAcroForm(form)
	return b
}

// WithAnnotations adds page annotations to the generated PDF.
func (b *CfgBuilder) WithAnnotations(annotations ...entity.PageAnnotation) Builder {
	b.annotations = entity.CloneAnnotations(annotations)
	return b
}

// WithPageGeometries configures per-page rotation and geometry boxes.
func (b *CfgBuilder) WithPageGeometries(geometries ...entity.PageGeometry) Builder {
	b.pageGeometries = entity.ClonePageGeometries(geometries)
	return b
}

// WithPdfA configures PDF/A conformance metadata and output intent.
// Level-A conformance profiles require a tagged PDF, so they enable
// WithTaggedPDF implicitly.
func (b *CfgBuilder) WithPdfA(config entity.PdfAConfig) Builder {
	b.pdfA = entity.ClonePdfAConfig(&config)
	if config.Level == entity.PdfA1A || config.Level == entity.PdfA2A || config.Level == entity.PdfA3A {
		b.taggedPDF = true
	}
	return b
}

// WithTaggedPDF enables tagged (accessible) PDF output.
func (b *CfgBuilder) WithTaggedPDF(enabled bool) Builder {
	b.taggedPDF = enabled
	return b
}

// WithLanguage sets the document language written to the catalog /Lang entry.
func (b *CfgBuilder) WithLanguage(language string) Builder {
	b.language = language
	return b
}

// WithViewerPreferences configures catalog viewer hints.
func (b *CfgBuilder) WithViewerPreferences(prefs entity.ViewerPreferences) Builder {
	b.viewerPreferences = &prefs
	return b
}

// WithPageLabels defines the labels viewers show instead of physical page
// numbers.
func (b *CfgBuilder) WithPageLabels(labels ...entity.PageLabelRange) Builder {
	b.pageLabels = append([]entity.PageLabelRange(nil), labels...)
	return b
}

// WithFileAttachments embeds files in the generated PDF.
func (b *CfgBuilder) WithFileAttachments(attachments ...entity.FileAttachment) Builder {
	b.attachments = entity.CloneAttachments(attachments)
	return b
}

// WithNamedDestinations defines named destinations in the generated PDF.
func (b *CfgBuilder) WithNamedDestinations(destinations ...entity.NamedDestination) Builder {
	b.namedDestinations = append([]entity.NamedDestination(nil), destinations...)
	return b
}

// WithFileID sets an explicit trailer /ID for the generated PDF.
func (b *CfgBuilder) WithFileID(id []byte) Builder {
	b.fileID = append([]byte(nil), id...)
	return b
}

// WithDeterministic makes repeated builds of the same document byte-identical
// (fixed implicit dates, sorted resources, content-derived trailer /ID).
func (b *CfgBuilder) WithDeterministic(deterministic bool) Builder {
	b.deterministic = deterministic
	return b
}
