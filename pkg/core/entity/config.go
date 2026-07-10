package entity

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/props"
)

// Config is the configuration of a paper instance.
type Config struct {
	ProviderType                   consts.ProviderType
	Dimensions                     *Dimensions
	Margins                        *Margins
	DefaultFont                    *props.Font
	CustomFonts                    []CustomFont
	GenerationMode                 consts.GenerationMode
	ChunkWorkers                   int
	Debug                          bool
	MaxGridSize                    int
	PageNumber                     *props.PageNumber
	Protection                     *Protection
	Compression                    bool
	Metadata                       *Metadata
	BackgroundImage                *Image
	FirstPageBackgroundImage       *Image
	FirstPageForegroundImage       *Image
	FirstPageForegroundImages      []*Image
	FirstPageFinalForegroundImages []*Image
	DisableAutoPageBreak           bool
	HTMLLimits                     HTMLLimits
	OutlineFromHeadings            bool
	Watermark                      *props.Watermark
	Annotations                    []PageAnnotation
	PageGeometries                 []PageGeometry
	AcroForm                       *AcroForm
	PdfA                           *PdfAConfig
	TaggedPDF                      bool
	Language                       string
	ViewerPreferences              *ViewerPreferences
	PageLabels                     []PageLabelRange
	Attachments                    []FileAttachment
	NamedDestinations              []NamedDestination
	FileID                         []byte
	Deterministic                  bool
}

// HasDocumentCatalog reports whether any document-catalog feature is
// configured. These features are written once per document, so generation
// falls back to sequential mode when any of them is present.
func (c *Config) HasDocumentCatalog() bool {
	if c == nil {
		return false
	}
	return c.AcroForm != nil ||
		len(c.Annotations) > 0 ||
		len(c.PageGeometries) > 0 ||
		c.PdfA != nil ||
		c.TaggedPDF ||
		c.Language != "" ||
		c.ViewerPreferences != nil ||
		len(c.PageLabels) > 0 ||
		len(c.Attachments) > 0 ||
		len(c.NamedDestinations) > 0 ||
		len(c.FileID) > 0 ||
		c.Deterministic
}

// ToMap converts Config to a map[string]any .
func (c *Config) ToMap() map[string]any {
	m := make(map[string]any)

	if c.ProviderType != "" {
		m["config_provider_type"] = c.ProviderType
	}

	if c.Dimensions != nil {
		m = c.Dimensions.AppendMap("paper", m)
	}

	if c.Margins != nil {
		m = c.Margins.AppendMap(m)
	}

	if c.DefaultFont != nil {
		m = c.DefaultFont.AppendMap(m)
	}

	m["generation_mode"] = c.GenerationMode
	m["chunk_workers"] = c.ChunkWorkers

	if c.Debug {
		m["config_debug"] = c.Debug
	}

	if c.MaxGridSize != 0 {
		m["config_max_grid_sum"] = c.MaxGridSize
	}

	if c.PageNumber != nil {
		m = c.PageNumber.AppendMap(m)
	}

	if c.Protection != nil {
		m = c.Protection.AppendMap(m)
	}

	if c.Compression {
		m["config_compression"] = c.Compression
	}

	if c.Metadata != nil {
		m = c.Metadata.AppendMap(m)
	}

	if c.BackgroundImage != nil {
		m = c.BackgroundImage.AppendMap(m)
	}

	if c.FirstPageBackgroundImage != nil {
		if len(c.FirstPageBackgroundImage.Bytes) != 0 {
			m["entity_first_page_background_image_bytes"] = len(c.FirstPageBackgroundImage.Bytes)
		}
		if c.FirstPageBackgroundImage.Extension != "" {
			m["entity_first_page_background_image_extension"] = c.FirstPageBackgroundImage.Extension
		}
	}

	if c.FirstPageForegroundImage != nil {
		if len(c.FirstPageForegroundImage.Bytes) != 0 {
			m["entity_first_page_foreground_image_bytes"] = len(c.FirstPageForegroundImage.Bytes)
		}
		if c.FirstPageForegroundImage.Extension != "" {
			m["entity_first_page_foreground_image_extension"] = c.FirstPageForegroundImage.Extension
		}
	}
	if len(c.FirstPageForegroundImages) != 0 {
		m["entity_first_page_foreground_images"] = len(c.FirstPageForegroundImages)
	}
	if len(c.FirstPageFinalForegroundImages) != 0 {
		m["entity_first_page_final_foreground_images"] = len(c.FirstPageFinalForegroundImages)
	}

	if c.DisableAutoPageBreak {
		m["config_disable_auto_page_break"] = c.DisableAutoPageBreak
	}

	if c.OutlineFromHeadings {
		m["config_outline_from_headings"] = c.OutlineFromHeadings
	}

	if c.Watermark != nil {
		m = c.Watermark.ToMap(m)
	}

	if c.HTMLLimits != (HTMLLimits{}) {
		m["config_html_max_image_pixels"] = c.HTMLLimits.MaxImagePixels
		m["config_html_max_image_bytes"] = c.HTMLLimits.MaxImageBytes
		m["config_html_max_dom_depth"] = c.HTMLLimits.MaxDOMDepth
		m["config_html_max_dom_nodes"] = c.HTMLLimits.MaxDOMNodes
		m["config_html_max_svg_pixels"] = c.HTMLLimits.MaxSVGPixels
		m["config_html_max_style_rules"] = c.HTMLLimits.MaxStyleRules
	}

	if c.AcroForm != nil {
		m["config_acroform_fields"] = len(c.AcroForm.Fields())
	}
	m = appendAnnotationsMap(c.Annotations, m)
	m = appendPageGeometriesMap(c.PageGeometries, m)
	if c.PdfA != nil {
		m["config_pdfa_level"] = c.PdfA.Level
		if c.PdfA.OutputCondition != "" {
			m["config_pdfa_output_condition"] = c.PdfA.OutputCondition
		}
	}
	if c.TaggedPDF {
		m["config_tagged_pdf"] = c.TaggedPDF
	}
	if c.Language != "" {
		m["config_language"] = c.Language
	}
	m = c.ViewerPreferences.AppendMap(m)
	m = appendPageLabelsMap(c.PageLabels, m)
	m = appendFileAttachmentsMap(c.Attachments, m)
	m = appendNamedDestinationsMap(c.NamedDestinations, m)
	if len(c.FileID) > 0 {
		m["config_file_id_bytes"] = len(c.FileID)
	}
	if c.Deterministic {
		m["config_deterministic"] = c.Deterministic
	}

	return m
}
