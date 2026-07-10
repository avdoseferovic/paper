package pdf

import "time"

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

type PageLabelRange struct {
	PageIndex int
	Style     string
	Prefix    string
	Start     int
}

type FileAttachment struct {
	FileName       string
	MIMEType       string
	Description    string
	AFRelationship string
	Data           []byte
	CreationDate   time.Time
}

type NamedDestination struct {
	Name      string
	PageIndex int
	FitType   string
	Top       float64
	Left      float64
	Zoom      float64
}

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

type FormFieldType int

const (
	FormFieldText FormFieldType = iota
	FormFieldCheckbox
	FormFieldRadio
	FormFieldDropdown
	FormFieldListBox
	FormFieldPushButton
	FormFieldSignature
)

type FormFieldFlags uint32

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

type PdfAConfig struct {
	Level           PdfALevel
	ICCProfile      []byte
	OutputCondition string
	XMPSchemas      []XMPSchema
	XMPProperties   []XMPPropertyBlock
}

type XMPSchema struct {
	Schema       string
	NamespaceURI string
	Prefix       string
	Properties   []XMPSchemaProperty
}

type XMPSchemaProperty struct {
	Name        string
	ValueType   string
	Category    string
	Description string
}

type XMPPropertyBlock struct {
	Namespace  string
	Prefix     string
	Properties []XMPProperty
}

type XMPProperty struct {
	Name  string
	Value string
}
