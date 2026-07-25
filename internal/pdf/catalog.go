package pdf

import "time"

// ViewerPreferences holds the catalog hints that tell a PDF reader how to
// present the document when it opens.
type ViewerPreferences struct {
	PageLayout      string
	PageMode        string
	HideToolbar     bool
	HideMenubar     bool
	HideWindowUI    bool
	FitWindow       bool
	CenterWindow    bool
	DisplayDocTitle bool
	OpenPage        int
	OpenZoom        string
}

// PageLabelRange numbers a run of pages, starting at PageIndex. Style selects
// the numbering (decimal, roman, letters) and Prefix is prepended to it.
type PageLabelRange struct {
	PageIndex int
	Style     string
	Prefix    string
	Start     int
}

// FileAttachment is a file embedded in the document.
type FileAttachment struct {
	FileName       string
	MIMEType       string
	Description    string
	AFRelationship string
	Data           []byte
	CreationDate   time.Time
}

// NamedDestination is a named jump target: a page plus how the reader should
// fit it in the window.
type NamedDestination struct {
	Name      string
	PageIndex int
	FitType   string
	Top       float64
	Left      float64
	Zoom      float64
}

// PageAnnotation is an annotation placed on one page, such as a link, a text
// note, or a highlight.
type PageAnnotation struct {
	PageIndex  int
	Subtype    string
	Rect       [4]float64
	URI        string
	DestName   string
	DestPage   *int
	Contents   string
	Name       string
	Open       bool
	Color      *[3]float64
	QuadPoints [][8]float64
}

// PageGeometry overrides a single page's rotation and box sizes.
type PageGeometry struct {
	PageIndex int
	Rotate    int
	CropBox   *[4]float64
	BleedBox  *[4]float64
	TrimBox   *[4]float64
	ArtBox    *[4]float64
}

type attachmentFileSpecRef struct {
	name string
	ref  int
}

// FormFieldType is the kind of interactive form field to draw.
type FormFieldType int

// The interactive form field kinds.
const (
	FormFieldText FormFieldType = iota
	FormFieldCheckbox
	FormFieldRadio
	FormFieldDropdown
	FormFieldListBox
	FormFieldPushButton
	FormFieldSignature
)

// FormFieldFlags is the bit set of PDF field flags, such as read-only or
// required.
type FormFieldFlags uint32

// FormField describes one interactive form field: what it is, where it sits on
// the page, and how it looks.
type FormField struct {
	Name      string
	Type      FormFieldType
	Value     string
	Default   string
	Flags     FormFieldFlags
	Rect      [4]float64
	PageIndex int

	FontSize    float64
	FontName    string
	TextColor   [3]float64
	BGColor     *[3]float64
	BorderColor *[3]float64
	BorderWidth float64

	Options     []string
	ExportValue string
	Children    []FormField
}

// ConformanceLevel is the PDF/A conformance level to write.
type ConformanceLevel int

// The supported PDF/A conformance levels.
const (
	ConformanceA2B ConformanceLevel = iota
	ConformanceA2U
	ConformanceA2A
	ConformanceA3B
	ConformanceA1B
	ConformanceA1A
	ConformanceA3A
	ConformanceA4
	ConformanceA4F
	ConformanceA4E
)

// ConformanceConfig configures PDF/A output: the conformance level, the ICC profile to
// embed, and any extra XMP metadata.
type ConformanceConfig struct {
	Level           ConformanceLevel
	ICCProfile      []byte
	OutputCondition string
	XMPSchemas      []XMPSchema
	XMPProperties   []XMPPropertyBlock
}

// XMPSchema declares a custom XMP schema in the PDF/A extension metadata, so a
// validator can resolve properties outside the standard namespaces.
type XMPSchema struct {
	Schema       string
	NamespaceURI string
	Prefix       string
	Properties   []XMPSchemaProperty
}

// XMPSchemaProperty describes one property of an XMPSchema.
type XMPSchemaProperty struct {
	Name        string
	ValueType   string
	Category    string
	Description string
}

// XMPPropertyBlock is a group of XMP properties written under one namespace.
type XMPPropertyBlock struct {
	Namespace  string
	Prefix     string
	Properties []XMPProperty
}

// XMPProperty is a single name/value pair of XMP metadata.
type XMPProperty struct {
	Name  string
	Value string
}
