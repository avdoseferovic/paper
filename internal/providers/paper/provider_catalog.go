package paper

import (
	pdf "github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// catalogPDF is the subset of *pdf.PDF used to push document-catalog
// features (forms, PDF/A, tagged PDF, viewer hints, attachments, ...).
type catalogPDF interface {
	SetAcroFormFields(fields ...pdf.FormField)
	SetPageAnnotations(annotations ...pdf.PageAnnotation)
	SetPageGeometries(geometries ...pdf.PageGeometry)
	SetPdfA(config pdf.ConformanceConfig)
	SetTaggedPDF(enabled bool)
	SetLanguage(language string)
	SetViewerPreferences(prefs pdf.ViewerPreferences)
	SetPageLabels(labels ...pdf.PageLabelRange)
	SetAttachments(attachments ...pdf.FileAttachment)
	SetNamedDestinations(destinations ...pdf.NamedDestination)
	SetFileID(id []byte)
	SetDeterministic(deterministic bool)
}

// SetDocumentCatalog pushes the document-catalog features from the config
// into the underlying PDF writer.
func (g *provider) SetDocumentCatalog(cfg *entity.Config) {
	if cfg == nil {
		return
	}
	target, ok := any(g.fpdf).(catalogPDF)
	if !ok {
		return
	}
	if cfg.AcroForm != nil {
		target.SetAcroFormFields(pdfFormFields(cfg.AcroForm.Fields())...)
	}
	if len(cfg.Annotations) > 0 {
		target.SetPageAnnotations(pdfAnnotations(cfg.Annotations)...)
	}
	if len(cfg.PageGeometries) > 0 {
		target.SetPageGeometries(pdfPageGeometries(cfg.PageGeometries)...)
	}
	if cfg.PdfA != nil {
		target.SetPdfA(pdfPdfAConfig(cfg.PdfA))
	}
	if cfg.TaggedPDF {
		target.SetTaggedPDF(true)
	}
	if cfg.Language != "" {
		target.SetLanguage(cfg.Language)
	}
	if cfg.ViewerPreferences != nil {
		target.SetViewerPreferences(pdfViewerPreferences(cfg.ViewerPreferences))
	}
	if len(cfg.PageLabels) > 0 {
		target.SetPageLabels(pdfPageLabels(cfg.PageLabels)...)
	}
	if len(cfg.Attachments) > 0 {
		target.SetAttachments(pdfAttachments(cfg.Attachments)...)
	}
	if len(cfg.NamedDestinations) > 0 {
		target.SetNamedDestinations(pdfNamedDestinations(cfg.NamedDestinations)...)
	}
	if len(cfg.FileID) > 0 {
		target.SetFileID(cfg.FileID)
	}
	if cfg.Deterministic {
		target.SetDeterministic(true)
	}
}

func pdfFormFields(fields []*entity.Field) []pdf.FormField {
	out := make([]pdf.FormField, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		out = append(out, pdfFormField(field))
	}
	return out
}

func pdfFormField(field *entity.Field) pdf.FormField {
	converted := pdf.FormField{
		Name:        field.Name,
		Type:        pdf.FormFieldType(field.Type),
		Value:       field.Value,
		Default:     field.Default,
		Flags:       pdf.FormFieldFlags(field.Flags),
		Rect:        field.Rect,
		PageIndex:   field.PageIndex,
		FontSize:    field.FontSize,
		FontName:    field.FontName,
		TextColor:   field.TextColor,
		BGColor:     field.BGColor,
		BorderColor: field.BorderColor,
		BorderWidth: field.BorderWidth,
		Options:     append([]string(nil), field.Options...),
		ExportValue: field.ExportValue,
	}
	for _, child := range field.Children() {
		if child == nil {
			continue
		}
		converted.Children = append(converted.Children, pdfFormField(child))
	}
	return converted
}

func pdfAnnotations(annotations []entity.PageAnnotation) []pdf.PageAnnotation {
	out := make([]pdf.PageAnnotation, 0, len(annotations))
	for _, annotation := range annotations {
		out = append(out, pdf.PageAnnotation{
			PageIndex:  annotation.PageIndex,
			Subtype:    string(annotation.Type),
			Rect:       annotation.Rect,
			URI:        annotation.URI,
			DestName:   annotation.DestName,
			DestPage:   annotation.DestPage,
			Contents:   annotation.Contents,
			Name:       annotation.Icon,
			Open:       annotation.Open,
			Color:      annotation.Color,
			QuadPoints: annotation.QuadPoints,
		})
	}
	return out
}

func pdfPageGeometries(geometries []entity.PageGeometry) []pdf.PageGeometry {
	out := make([]pdf.PageGeometry, 0, len(geometries))
	for _, geometry := range geometries {
		out = append(out, pdf.PageGeometry{
			PageIndex: geometry.PageIndex,
			Rotate:    geometry.Rotate,
			CropBox:   geometry.CropBox,
			BleedBox:  geometry.BleedBox,
			TrimBox:   geometry.TrimBox,
			ArtBox:    geometry.ArtBox,
		})
	}
	return out
}

func pdfPdfAConfig(config *entity.PdfAConfig) pdf.ConformanceConfig {
	converted := pdf.ConformanceConfig{
		Level:           pdf.ConformanceLevel(config.Level),
		ICCProfile:      config.ICCProfile,
		OutputCondition: config.OutputCondition,
	}
	for _, schema := range config.XMPSchemas {
		convertedSchema := pdf.XMPSchema{
			Schema:       schema.Schema,
			NamespaceURI: schema.NamespaceURI,
			Prefix:       schema.Prefix,
		}
		for _, property := range schema.Properties {
			convertedSchema.Properties = append(convertedSchema.Properties, pdf.XMPSchemaProperty(property))
		}
		converted.XMPSchemas = append(converted.XMPSchemas, convertedSchema)
	}
	for _, block := range config.XMPProperties {
		convertedBlock := pdf.XMPPropertyBlock{
			Namespace: block.Namespace,
			Prefix:    block.Prefix,
		}
		for _, property := range block.Properties {
			convertedBlock.Properties = append(convertedBlock.Properties, pdf.XMPProperty(property))
		}
		converted.XMPProperties = append(converted.XMPProperties, convertedBlock)
	}
	return converted
}

func pdfViewerPreferences(prefs *entity.ViewerPreferences) pdf.ViewerPreferences {
	return pdf.ViewerPreferences{
		PageLayout:      string(prefs.PageLayout),
		PageMode:        string(prefs.PageMode),
		HideToolbar:     prefs.HideToolbar,
		HideMenubar:     prefs.HideMenubar,
		HideWindowUI:    prefs.HideWindowUI,
		FitWindow:       prefs.FitWindow,
		CenterWindow:    prefs.CenterWindow,
		DisplayDocTitle: prefs.DisplayDocTitle,
		OpenPage:        prefs.OpenPage,
		OpenZoom:        prefs.OpenZoom,
	}
}

func pdfPageLabels(labels []entity.PageLabelRange) []pdf.PageLabelRange {
	out := make([]pdf.PageLabelRange, 0, len(labels))
	for _, label := range labels {
		out = append(out, pdf.PageLabelRange{
			PageIndex: label.PageIndex,
			Style:     string(label.Style),
			Prefix:    label.Prefix,
			Start:     label.Start,
		})
	}
	return out
}

func pdfAttachments(attachments []entity.FileAttachment) []pdf.FileAttachment {
	out := make([]pdf.FileAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		out = append(out, pdf.FileAttachment{
			FileName:       attachment.FileName,
			MIMEType:       attachment.MIMEType,
			Description:    attachment.Description,
			AFRelationship: attachment.AFRelationship,
			Data:           attachment.Data,
			CreationDate:   attachment.CreationDate,
		})
	}
	return out
}

func pdfNamedDestinations(destinations []entity.NamedDestination) []pdf.NamedDestination {
	out := make([]pdf.NamedDestination, 0, len(destinations))
	for _, destination := range destinations {
		out = append(out, pdf.NamedDestination{
			Name:      destination.Name,
			PageIndex: destination.PageIndex,
			FitType:   string(destination.FitType),
			Top:       destination.Top,
			Left:      destination.Left,
			Zoom:      destination.Zoom,
		})
	}
	return out
}
