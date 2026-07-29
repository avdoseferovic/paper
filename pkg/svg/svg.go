// Package svg parses SVG markup and adapts it to Paper components.
package svg

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	svgraster "github.com/avdoseferovic/paper/internal/svg"
	imagecomp "github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

var (
	// ErrSVGHasZeroDimensions is returned when SVG rasterization cannot infer dimensions.
	ErrSVGHasZeroDimensions = svgraster.ErrSVGHasZeroDimensions
	// ErrSVGTooLarge is returned when rasterization would exceed the pixel limit.
	ErrSVGTooLarge = svgraster.ErrSVGTooLarge
	// ErrNilDocument is returned when a method requiring an SVG document is called on nil.
	ErrNilDocument = errors.New("svg: nil document")
	// ErrNoSVGElement is returned when parsed XML does not contain an <svg> element.
	ErrNoSVGElement = errors.New("svg: no <svg> element found")
	// ErrEmptyDocument is returned when the SVG input has no XML element.
	ErrEmptyDocument = errors.New("svg: empty document")
)

// SVG is a parsed SVG document.
type SVG struct {
	data        []byte
	root        *Node
	width       float64
	height      float64
	viewBox     ViewBox
	aspectRatio PreserveAspectRatio
	defs        map[string]*Node
}

// Node represents a parsed SVG element.
type Node struct {
	Tag      string
	Attrs    map[string]string
	Children []*Node
	Text     string
}

// ViewBox defines the SVG coordinate system.
type ViewBox struct {
	MinX, MinY, Width, Height float64
	Valid                     bool
}

// AspectAlign controls how the viewBox is positioned within the viewport.
type AspectAlign int

const (
	// AlignXMidYMid centers the viewBox on both axes.
	AlignXMidYMid AspectAlign = iota
	// AlignXMinYMin aligns to the minimum x and minimum y edge.
	AlignXMinYMin
	// AlignXMidYMin centers x and aligns to the minimum y edge.
	AlignXMidYMin
	// AlignXMaxYMin aligns to the maximum x and minimum y edge.
	AlignXMaxYMin
	// AlignXMinYMid aligns to the minimum x edge and centers y.
	AlignXMinYMid
	// AlignXMaxYMid aligns to the maximum x edge and centers y.
	AlignXMaxYMid
	// AlignXMinYMax aligns to the minimum x and maximum y edge.
	AlignXMinYMax
	// AlignXMidYMax centers x and aligns to the maximum y edge.
	AlignXMidYMax
	// AlignXMaxYMax aligns to the maximum x and maximum y edge.
	AlignXMaxYMax
)

// AspectMeetOrSlice selects the uniform-scaling strategy.
type AspectMeetOrSlice int

const (
	// ScaleMeet scales uniformly so the entire viewBox fits.
	ScaleMeet AspectMeetOrSlice = iota
	// ScaleSlice scales uniformly so the viewport is covered.
	ScaleSlice
)

// PreserveAspectRatio captures the parsed preserveAspectRatio attribute.
type PreserveAspectRatio struct {
	Align       AspectAlign
	MeetOrSlice AspectMeetOrSlice
	None        bool
}

// DefaultPreserveAspectRatio returns the SVG default: xMidYMid meet.
func DefaultPreserveAspectRatio() PreserveAspectRatio {
	return PreserveAspectRatio{Align: AlignXMidYMid, MeetOrSlice: ScaleMeet}
}

// Parse parses SVG markup from a string.
func Parse(svgXML string) (*SVG, error) {
	return ParseBytes([]byte(svgXML))
}

// ParseBytes parses SVG markup from bytes.
func ParseBytes(data []byte) (*SVG, error) {
	return parse(data)
}

// ParseReader parses SVG markup from a reader.
func ParseReader(r io.Reader) (*SVG, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("svg: read: %w", err)
	}
	return parse(data)
}

// Rasterize converts SVG bytes to PNG bytes at the requested millimetre dimensions.
func Rasterize(data []byte, widthMM, heightMM float64) ([]byte, int, int, error) {
	return svgraster.Rasterize(data, widthMM, heightMM)
}

// RasterizeWithLimit converts SVG bytes to PNG bytes with a custom pixel cap.
func RasterizeWithLimit(data []byte, widthMM, heightMM float64, maxPixels int64) ([]byte, int, int, error) {
	return svgraster.RasterizeWithLimit(data, widthMM, heightMM, maxPixels)
}

// Bytes returns a copy of the original SVG bytes.
func (s *SVG) Bytes() []byte {
	if s == nil {
		return nil
	}
	return slices.Clone(s.data)
}

// Width returns the explicit SVG width, or the viewBox width when width is absent.
func (s *SVG) Width() float64 {
	if s == nil {
		return 0
	}
	if s.width > 0 {
		return s.width
	}
	if s.viewBox.Valid {
		return s.viewBox.Width
	}
	return 0
}

// Height returns the explicit SVG height, or the viewBox height when height is absent.
func (s *SVG) Height() float64 {
	if s == nil {
		return 0
	}
	if s.height > 0 {
		return s.height
	}
	if s.viewBox.Valid {
		return s.viewBox.Height
	}
	return 0
}

// ViewBox returns the parsed viewBox.
func (s *SVG) ViewBox() ViewBox {
	if s == nil {
		return ViewBox{}
	}
	return s.viewBox
}

// AspectRatio returns width / height, or zero if dimensions are unavailable.
func (s *SVG) AspectRatio() float64 {
	if s == nil || s.Height() == 0 {
		return 0
	}
	return s.Width() / s.Height()
}

// Root returns the parsed root SVG node.
func (s *SVG) Root() *Node {
	if s == nil {
		return nil
	}
	return s.root
}

// Defs returns reusable elements indexed by id.
func (s *SVG) Defs() map[string]*Node {
	if s == nil {
		return nil
	}
	return s.defs
}

// PreserveAspectRatio returns the parsed preserveAspectRatio attribute.
func (s *SVG) PreserveAspectRatio() PreserveAspectRatio {
	if s == nil {
		return DefaultPreserveAspectRatio()
	}
	return s.aspectRatio
}

// Rasterize converts this SVG to PNG bytes at the requested millimetre dimensions.
func (s *SVG) Rasterize(widthMM, heightMM float64) ([]byte, int, int, error) {
	if s == nil {
		return nil, 0, 0, ErrNilDocument
	}
	return Rasterize(s.data, widthMM, heightMM)
}

// Component returns a Paper image component backed by the original SVG bytes.
func (s *SVG) Component(rectProps ...props.Rect) core.Component {
	if s == nil {
		return imagecomp.NewFromBytes(nil, extension.Svg, rectProps...)
	}
	return imagecomp.NewFromBytes(s.Bytes(), extension.Svg, rectProps...)
}

// Col returns a Paper column containing this SVG as an image component.
func (s *SVG) Col(size int, rectProps ...props.Rect) core.Col {
	if s == nil {
		return imagecomp.NewFromBytesCol(size, nil, extension.Svg, rectProps...)
	}
	return imagecomp.NewFromBytesCol(size, s.Bytes(), extension.Svg, rectProps...)
}

// Row returns a fixed-height Paper row containing this SVG as an image component.
func (s *SVG) Row(height float64, rectProps ...props.Rect) core.Row {
	if s == nil {
		return imagecomp.NewFromBytesRow(height, nil, extension.Svg, rectProps...)
	}
	return imagecomp.NewFromBytesRow(height, s.Bytes(), extension.Svg, rectProps...)
}

// AutoRow returns an automatic-height Paper row containing this SVG as an image component.
func (s *SVG) AutoRow(rectProps ...props.Rect) core.Row {
	if s == nil {
		return imagecomp.NewAutoFromBytesRow(nil, extension.Svg, rectProps...)
	}
	return imagecomp.NewAutoFromBytesRow(s.Bytes(), extension.Svg, rectProps...)
}

func parse(data []byte) (*SVG, error) {
	root, err := parseRootNode(data)
	if err != nil {
		return nil, err
	}
	svgRoot := findSVGRoot(root)
	if svgRoot == nil {
		return nil, ErrNoSVGElement
	}
	doc := &SVG{
		data:        slices.Clone(data),
		root:        svgRoot,
		aspectRatio: DefaultPreserveAspectRatio(),
	}
	doc.extractDimensions()
	doc.indexDefs()
	return doc, nil
}

func parseRootNode(data []byte) (*Node, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil, ErrEmptyDocument
		}
		if err != nil {
			return nil, fmt.Errorf("svg: parse: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		node := newNode(start)
		err = parseChildren(decoder, node)
		if err != nil {
			return nil, err
		}
		return node, nil
	}
}

func parseChildren(decoder *xml.Decoder, parent *Node) error {
	var text strings.Builder
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			parent.Text = strings.TrimSpace(text.String())
			return nil
		}
		if err != nil {
			return fmt.Errorf("svg: parse children: %w", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			child := newNode(value)
			err = parseChildren(decoder, child)
			if err != nil {
				return err
			}
			parent.Children = append(parent.Children, child)
		case xml.CharData:
			text.Write(value)
		case xml.EndElement:
			parent.Text = strings.TrimSpace(text.String())
			return nil
		}
	}
}

func newNode(start xml.StartElement) *Node {
	node := &Node{
		Tag:   start.Name.Local,
		Attrs: make(map[string]string, len(start.Attr)),
	}
	for _, attr := range start.Attr {
		node.Attrs[attr.Name.Local] = attr.Value
	}
	return node
}

func findSVGRoot(node *Node) *Node {
	if node == nil {
		return nil
	}
	if node.Tag == "svg" {
		return node
	}
	for _, child := range node.Children {
		if found := findSVGRoot(child); found != nil {
			return found
		}
	}
	return nil
}

func (s *SVG) extractDimensions() {
	if s.root == nil {
		return
	}
	if width, ok := s.root.Attrs["width"]; ok {
		s.width = parseDimension(width)
	}
	if height, ok := s.root.Attrs["height"]; ok {
		s.height = parseDimension(height)
	}
	if viewBox, ok := s.root.Attrs["viewBox"]; ok {
		s.viewBox = parseViewBox(viewBox)
	}
	if par, ok := s.root.Attrs["preserveAspectRatio"]; ok {
		s.aspectRatio = parsePreserveAspectRatio(par)
	}
}

func parseDimension(value string) float64 {
	value = strings.TrimSpace(value)
	for _, suffix := range []string{"px", "pt", "em", "rem", "cm", "mm", "in", "%"} {
		value = strings.TrimSuffix(value, suffix)
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseViewBox(value string) ViewBox {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", " ")
	parts := strings.Fields(value)
	if len(parts) < 4 {
		return ViewBox{}
	}
	minX, err1 := strconv.ParseFloat(parts[0], 64)
	minY, err2 := strconv.ParseFloat(parts[1], 64)
	width, err3 := strconv.ParseFloat(parts[2], 64)
	height, err4 := strconv.ParseFloat(parts[3], 64)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return ViewBox{}
	}
	return ViewBox{
		MinX:   minX,
		MinY:   minY,
		Width:  width,
		Height: height,
		Valid:  width > 0 && height > 0,
	}
}

func parsePreserveAspectRatio(value string) PreserveAspectRatio {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return DefaultPreserveAspectRatio()
	}
	if strings.EqualFold(fields[0], "none") {
		return PreserveAspectRatio{None: true}
	}

	result := DefaultPreserveAspectRatio()
	switch fields[0] {
	case "xMinYMin":
		result.Align = AlignXMinYMin
	case "xMidYMin":
		result.Align = AlignXMidYMin
	case "xMaxYMin":
		result.Align = AlignXMaxYMin
	case "xMinYMid":
		result.Align = AlignXMinYMid
	case "xMaxYMid":
		result.Align = AlignXMaxYMid
	case "xMinYMax":
		result.Align = AlignXMinYMax
	case "xMidYMax":
		result.Align = AlignXMidYMax
	case "xMaxYMax":
		result.Align = AlignXMaxYMax
	}
	if len(fields) > 1 && strings.EqualFold(fields[1], "slice") {
		result.MeetOrSlice = ScaleSlice
	}
	return result
}

func (s *SVG) indexDefs() {
	s.defs = make(map[string]*Node)
	indexDefs(s.root, s.defs)
}

func indexDefs(node *Node, defs map[string]*Node) {
	if node == nil {
		return
	}
	if id, ok := node.Attrs["id"]; ok && id != "" {
		defs[id] = node
	}
	for _, child := range node.Children {
		indexDefs(child, defs)
	}
}
