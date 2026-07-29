package pdf

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// SetPdfA configures PDF/A identification metadata and output intents.
func (f *PDF) SetPdfA(config ConformanceConfig) {
	clone := config
	clone.ICCProfile = slices.Clone(config.ICCProfile)
	clone.XMPSchemas = cloneXMPSchemas(config.XMPSchemas)
	clone.XMPProperties = cloneXMPPropertyBlocks(config.XMPProperties)
	f.pdfA = &clone
	f.pdfVersion = pdfAVersion(config.Level)
	if isPdfALevelA(config.Level) {
		f.SetTaggedPDF(true)
	}
}

func pdfAVersion(level ConformanceLevel) string {
	switch level {
	case ConformanceA1B, ConformanceA1A:
		return pdfVersion14
	case ConformanceA2B, ConformanceA2U, ConformanceA2A, ConformanceA3B, ConformanceA3A:
		return cnPDFVersion
	case ConformanceA4, ConformanceA4F, ConformanceA4E:
		return "2.0"
	default:
		return cnPDFVersion
	}
}

func isPdfALevelA(level ConformanceLevel) bool {
	return level == ConformanceA1A || level == ConformanceA2A || level == ConformanceA3A
}

func allowsPdfAAttachments(level ConformanceLevel) bool {
	return level == ConformanceA3B || level == ConformanceA3A || level == ConformanceA4F || level == ConformanceA4E
}

func pdfAPart(level ConformanceLevel) int {
	switch level {
	case ConformanceA1B, ConformanceA1A:
		return 1
	case ConformanceA2B, ConformanceA2U, ConformanceA2A:
		return 2
	case ConformanceA3B, ConformanceA3A:
		return 3
	case ConformanceA4, ConformanceA4F, ConformanceA4E:
		return 4
	default:
		return 2
	}
}

func pdfAConformance(level ConformanceLevel) string {
	switch level {
	case ConformanceA1A, ConformanceA2A, ConformanceA3A:
		return "A"
	case ConformanceA1B, ConformanceA2B, ConformanceA3B:
		return "B"
	case ConformanceA2U:
		return "U"
	case ConformanceA4:
		return ""
	case ConformanceA4F:
		return "F"
	case ConformanceA4E:
		return "E"
	default:
		return ""
	}
}

func (f *PDF) preparePdfAMetadata() {
	if f.pdfA == nil {
		return
	}
	f.xmp = []byte(f.buildPdfAXMP())
}

func (f *PDF) buildPdfAXMP() string {
	level := f.pdfA.Level
	part := pdfAPart(level)
	conf := pdfAConformance(level)
	created := timeOrNow(f.creationDate).Format("2006-01-02T15:04:05Z07:00")
	modified := timeOrNow(f.modDate).Format("2006-01-02T15:04:05Z07:00")
	creator := cmp.Or(f.creator, "Paper")
	producer := cmp.Or(f.producer, "Paper")

	var b strings.Builder
	b.WriteString(`<?xpacket begin="` + "\xef\xbb\xbf" + `" id="W5M0MpCehiHzreSzNTczkc9d"?>` + "\n")
	b.WriteString(`<x:xmpmeta xmlns:x="adobe:ns:meta/">` + "\n")
	b.WriteString(`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` + "\n")
	b.WriteString(`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` + "\n")
	if f.title != "" {
		b.WriteString(`<dc:title><rdf:Alt><rdf:li xml:lang="x-default">` + xmlEscape(f.title) + `</rdf:li></rdf:Alt></dc:title>` + "\n")
	}
	if f.author != "" {
		b.WriteString(`<dc:creator><rdf:Seq><rdf:li>` + xmlEscape(f.author) + `</rdf:li></rdf:Seq></dc:creator>` + "\n")
	}
	if f.language != "" {
		b.WriteString(`<dc:language><rdf:Bag><rdf:li>` + xmlEscape(f.language) + `</rdf:li></rdf:Bag></dc:language>` + "\n")
	}
	b.WriteString(`</rdf:Description>` + "\n")
	b.WriteString(`<rdf:Description rdf:about="" xmlns:xmp="http://ns.adobe.com/xap/1.0/">` + "\n")
	b.WriteString(`<xmp:CreatorTool>` + xmlEscape(creator) + `</xmp:CreatorTool>` + "\n")
	b.WriteString(`<xmp:CreateDate>` + created + `</xmp:CreateDate>` + "\n")
	b.WriteString(`<xmp:ModifyDate>` + modified + `</xmp:ModifyDate>` + "\n")
	b.WriteString(`</rdf:Description>` + "\n")
	b.WriteString(`<rdf:Description rdf:about="" xmlns:pdf="http://ns.adobe.com/pdf/1.3/">` + "\n")
	b.WriteString(`<pdf:Producer>` + xmlEscape(producer) + `</pdf:Producer>` + "\n")
	b.WriteString(`</rdf:Description>` + "\n")
	b.WriteString(`<rdf:Description rdf:about="" xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/">` + "\n")
	_, _ = fmt.Fprintf(&b, `<pdfaid:part>%d</pdfaid:part>`, part)
	b.WriteByte('\n')
	if part == 4 {
		b.WriteString(`<pdfaid:rev>2020</pdfaid:rev>` + "\n")
	}
	if conf != "" {
		b.WriteString(`<pdfaid:conformance>` + conf + `</pdfaid:conformance>` + "\n")
	}
	b.WriteString(`</rdf:Description>` + "\n")
	f.writePdfAXMPExtensionSchemas(&b, level)
	f.writePdfAXMPPropertyBlocks(&b)
	b.WriteString(`</rdf:RDF>` + "\n")
	b.WriteString(`</x:xmpmeta>` + "\n")
	b.WriteString(`<?xpacket end="w"?>`)
	return b.String()
}

func (f *PDF) writePdfAXMPExtensionSchemas(b *strings.Builder, level ConformanceLevel) {
	emitAFSchema := allowsPdfAAttachments(level)
	if !emitAFSchema && len(f.pdfA.XMPSchemas) == 0 {
		return
	}

	b.WriteString(`<rdf:Description rdf:about=""`)
	b.WriteString(` xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/"`)
	b.WriteString(` xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#"`)
	b.WriteString(` xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#">` + "\n")
	b.WriteString(`<pdfaExtension:schemas><rdf:Bag>` + "\n")
	if emitAFSchema {
		writePdfAAssociatedFileSchema(b)
	}
	for _, schema := range f.pdfA.XMPSchemas {
		writePdfAXMPSchema(b, schema)
	}
	b.WriteString(`</rdf:Bag></pdfaExtension:schemas>` + "\n")
	b.WriteString(`</rdf:Description>` + "\n")
}

func writePdfAAssociatedFileSchema(b *strings.Builder) {
	b.WriteString(`<rdf:li rdf:parseType="Resource">`)
	b.WriteString(`<pdfaSchema:schema>PDF/A Associated File Attachment</pdfaSchema:schema>`)
	b.WriteString(`<pdfaSchema:namespaceURI>http://www.aiim.org/pdfa/ns/f#</pdfaSchema:namespaceURI>`)
	b.WriteString(`<pdfaSchema:prefix>pdfaf</pdfaSchema:prefix>`)
	b.WriteString(`<pdfaSchema:property><rdf:Seq><rdf:li rdf:parseType="Resource">`)
	b.WriteString(`<pdfaProperty:name>file</pdfaProperty:name>`)
	b.WriteString(`<pdfaProperty:valueType>URI</pdfaProperty:valueType>`)
	b.WriteString(`<pdfaProperty:category>external</pdfaProperty:category>`)
	b.WriteString(`<pdfaProperty:description>Associated file</pdfaProperty:description>`)
	b.WriteString(`</rdf:li></rdf:Seq></pdfaSchema:property>`)
	b.WriteString(`</rdf:li>` + "\n")
}

func writePdfAXMPSchema(b *strings.Builder, schema XMPSchema) {
	b.WriteString(`<rdf:li rdf:parseType="Resource">` + "\n")
	b.WriteString(`<pdfaSchema:schema>` + xmlEscape(schema.Schema) + `</pdfaSchema:schema>` + "\n")
	b.WriteString(`<pdfaSchema:namespaceURI>` + xmlEscape(schema.NamespaceURI) + `</pdfaSchema:namespaceURI>` + "\n")
	b.WriteString(`<pdfaSchema:prefix>` + xmlEscape(schema.Prefix) + `</pdfaSchema:prefix>` + "\n")
	if len(schema.Properties) > 0 {
		b.WriteString(`<pdfaSchema:property><rdf:Seq>` + "\n")
		for _, property := range schema.Properties {
			writePdfAXMPSchemaProperty(b, property)
		}
		b.WriteString(`</rdf:Seq></pdfaSchema:property>` + "\n")
	}
	b.WriteString(`</rdf:li>` + "\n")
}

func writePdfAXMPSchemaProperty(b *strings.Builder, property XMPSchemaProperty) {
	b.WriteString(`<rdf:li rdf:parseType="Resource">` + "\n")
	b.WriteString(`<pdfaProperty:name>` + xmlEscape(property.Name) + `</pdfaProperty:name>` + "\n")
	b.WriteString(`<pdfaProperty:valueType>` + xmlEscape(property.ValueType) + `</pdfaProperty:valueType>` + "\n")
	b.WriteString(`<pdfaProperty:category>` + xmlEscape(property.Category) + `</pdfaProperty:category>` + "\n")
	b.WriteString(`<pdfaProperty:description>` + xmlEscape(property.Description) + `</pdfaProperty:description>` + "\n")
	b.WriteString(`</rdf:li>` + "\n")
}

func (f *PDF) writePdfAXMPPropertyBlocks(b *strings.Builder) {
	for _, block := range f.pdfA.XMPProperties {
		b.WriteString(`<rdf:Description rdf:about=""`)
		b.WriteString(` xmlns:` + block.Prefix + `="` + xmlEscape(block.Namespace) + `">` + "\n")
		for _, property := range block.Properties {
			b.WriteString(`<` + block.Prefix + `:` + property.Name + `>`)
			b.WriteString(xmlEscape(property.Value))
			b.WriteString(`</` + block.Prefix + `:` + property.Name + `>` + "\n")
		}
		b.WriteString(`</rdf:Description>` + "\n")
	}
}

func cloneXMPSchemas(schemas []XMPSchema) []XMPSchema {
	if len(schemas) == 0 {
		return nil
	}
	clones := make([]XMPSchema, len(schemas))
	for i, schema := range schemas {
		clones[i] = schema
		clones[i].Properties = slices.Clone(schema.Properties)
	}
	return clones
}

func cloneXMPPropertyBlocks(blocks []XMPPropertyBlock) []XMPPropertyBlock {
	if len(blocks) == 0 {
		return nil
	}
	clones := make([]XMPPropertyBlock, len(blocks))
	for i, block := range blocks {
		clones[i] = block
		clones[i].Properties = slices.Clone(block.Properties)
	}
	return clones
}

func (f *PDF) putPdfAOutputIntent() {
	f.outputIntentObj = 0
	if f.pdfA == nil {
		return
	}
	profile := f.pdfA.ICCProfile
	if len(profile) == 0 {
		profile = srgbICCProfile()
	}
	condition := cmp.Or(f.pdfA.OutputCondition, "sRGB IEC61966-2.1")

	f.newobj()
	profileRef := f.n
	stream := f.encryptedStream(profile)
	if f.err != nil {
		return
	}
	f.out("<<")
	f.out("/N 3")
	f.outf("/Length %d", len(stream))
	f.out(">>")
	f.putstream(stream)
	f.out("endobj")

	f.newobj()
	f.outputIntentObj = f.n
	f.out("<<")
	f.out("/Type /OutputIntent")
	f.out("/S /GTS_PDFA1")
	f.outf("/OutputConditionIdentifier %s", f.textstring(condition))
	f.out("/RegistryName (http://www.color.org)")
	f.outf("/Info %s", f.textstring(condition))
	f.outf("/DestOutputProfile %d 0 R", profileRef)
	f.out(">>")
	f.out("endobj")
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
