# Attachments

Paper can embed files in generated PDFs through the document catalog. Attachments are stored as embedded file streams, referenced from `/Names /EmbeddedFiles`, and listed in the catalog `/AF` associated-files array.

## Builder API

```go
cfg := config.NewBuilder().
	WithFileAttachments(entity.FileAttachment{
		FileName:       "invoice.xml",
		MIMEType:       "application/xml",
		Description:    "Invoice XML",
		AFRelationship: "Data",
		Data:           []byte("<invoice/>"),
	}).
	Build()
```

## Runtime API

```go
m := paper.New(config.NewBuilder().Build())
m.AttachFile(entity.FileAttachment{
	FileName:       "report.json",
	MIMEType:       "application/json",
	AFRelationship: "Supplement",
	Data:           payload,
})
```

`AFRelationship` defaults to `Unspecified`, and `MIMEType` defaults to `application/octet-stream`.

For Factur-X/ZUGFeRD-style hybrid invoices, use PDF/A-3B or another
attachment-capable PDF/A profile and set `AFRelationship: "Alternative"` on
the XML attachment. Pair it with `entity.PdfAConfig.XMPSchemas` and
`XMPProperties` to declare the invoice metadata namespace in the PDF/A XMP
packet.

## Notes

- Attachments are document-level catalog entries, so Paper keeps generation on the whole-document path even if concurrent generation was requested.
- This feature embeds files in newly generated PDFs. It does not yet add PDF/A validation, attachment extraction, or reader/import support for existing PDFs.
