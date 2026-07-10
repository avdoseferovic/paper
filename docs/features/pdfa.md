# PDF/A

Paper can emit PDF/A identification foundations for generated PDFs:

```go
cfg := config.NewBuilder().
	WithTitle("Archive Copy", false).
	WithPdfA(entity.PdfAConfig{Level: entity.PdfA2B}).
	Build()

doc := paper.New(cfg)
```

Runtime configuration is also available:

```go
doc.SetPdfA(entity.PdfAConfig{Level: entity.PdfA3B})
```

Supported levels:

- `entity.PdfA1B`, `PdfA1A`
- `entity.PdfA2B`, `PdfA2U`, `PdfA2A`
- `entity.PdfA3B`, `PdfA3A`
- `entity.PdfA4`, `PdfA4F`, `PdfA4E`

Paper writes PDF/A XMP identification metadata, references that metadata from
the catalog, emits an sRGB ICC output intent by default, selects the PDF
version implied by the level, and enables tagged PDF automatically for Level A
profiles.

## Custom XMP Extensions

`entity.PdfAConfig` can declare PDF/A extension schemas and write namespaced
property values into the XMP packet. This is useful for hybrid document
profiles such as Factur-X/ZUGFeRD, where a PDF/A-3B document embeds an XML
invoice attachment and declares a profile-specific XMP namespace:

```go
cfg := config.NewBuilder().
	WithPdfA(entity.PdfAConfig{
		Level: entity.PdfA3B,
		XMPSchemas: []entity.XMPSchema{{
			Schema:       "Factur-X PDFA Extension Schema",
			NamespaceURI: "urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#",
			Prefix:       "fx",
			Properties: []entity.XMPSchemaProperty{{
				Name:        "DocumentFileName",
				ValueType:   "Text",
				Category:    "external",
				Description: "Name of the embedded XML invoice file",
			}},
		}},
		XMPProperties: []entity.XMPPropertyBlock{{
			Namespace: "urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#",
			Prefix:    "fx",
			Properties: []entity.XMPProperty{
				{Name: "DocumentFileName", Value: "factur-x.xml"},
				{Name: "DocumentType", Value: "INVOICE"},
				{Name: "Version", Value: "1.0"},
				{Name: "ConformanceLevel", Value: "BASIC"},
			},
		}},
	}).
	WithFileAttachments(entity.FileAttachment{
		FileName:       "factur-x.xml",
		MIMEType:       "application/xml",
		AFRelationship: "Alternative",
		Data:           invoiceXML,
	}).
	Build()
```

For PDF/A-3 and PDF/A-4 profiles that allow associated files, Paper also
declares the built-in PDF/A associated-file (`pdfaf`) extension schema.

Current scope: strict PDF/A validation, guaranteed validator conformance, and
embedded-font enforcement are not implemented yet.
