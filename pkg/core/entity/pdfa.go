package entity

// PdfALevel identifies the requested PDF/A part and conformance profile.
type PdfALevel int

const (
	PdfA2B PdfALevel = iota
	PdfA2U
	PdfA2A
	PdfA3B
	PdfA1B
	PdfA1A
	PdfA3A
	PdfA4
	PdfA4F
	PdfA4E
)

// PdfAConfig configures generated PDF/A identification metadata.
type PdfAConfig struct {
	Level           PdfALevel
	ICCProfile      []byte
	OutputCondition string
	XMPSchemas      []XMPSchema
	XMPProperties   []XMPPropertyBlock
}

// XMPSchema describes one PDF/A extension schema declaration.
type XMPSchema struct {
	Schema       string
	NamespaceURI string
	Prefix       string
	Properties   []XMPSchemaProperty
}

// XMPSchemaProperty declares one property within an XMP extension schema.
type XMPSchemaProperty struct {
	Name        string
	ValueType   string
	Category    string
	Description string
}

// XMPPropertyBlock carries actual XMP property values for a namespace.
type XMPPropertyBlock struct {
	Namespace  string
	Prefix     string
	Properties []XMPProperty
}

// XMPProperty is a single custom XMP property value.
type XMPProperty struct {
	Name  string
	Value string
}

// ClonePdfAConfig returns a deep copy of config.
func ClonePdfAConfig(config *PdfAConfig) *PdfAConfig {
	if config == nil {
		return nil
	}
	clone := *config
	clone.ICCProfile = append([]byte(nil), config.ICCProfile...)
	clone.XMPSchemas = cloneXMPSchemas(config.XMPSchemas)
	clone.XMPProperties = cloneXMPPropertyBlocks(config.XMPProperties)
	return &clone
}

func cloneXMPSchemas(schemas []XMPSchema) []XMPSchema {
	if len(schemas) == 0 {
		return nil
	}
	clones := make([]XMPSchema, len(schemas))
	for i, schema := range schemas {
		clones[i] = schema
		clones[i].Properties = append([]XMPSchemaProperty(nil), schema.Properties...)
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
		clones[i].Properties = append([]XMPProperty(nil), block.Properties...)
	}
	return clones
}
