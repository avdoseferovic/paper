package svg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"maps"
	"math"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// maxFontSizeCanvasFactor caps a glyph face at this multiple of the canvas'
// larger side. A face is rasterised into a mask of its own em size before it is
// clipped to the canvas, so an unbounded font-size turns a few bytes of SVG
// into gigabytes of masks. Text this large is already far off-canvas.
const maxFontSizeCanvasFactor = 2

// builtinFonts holds the four Go faces the SVG text renderer can select from.
type builtinFonts struct {
	regular  *opentype.Font
	bold     *opentype.Font
	mono     *opentype.Font
	monoBold *opentype.Font
}

// loadBuiltinFonts parses the embedded Go fonts once per process, memoising the
// error alongside the value so every caller observes a parse failure.
var loadBuiltinFonts = sync.OnceValues(func() (builtinFonts, error) {
	var fonts builtinFonts
	for _, entry := range []struct {
		target **opentype.Font
		data   []byte
	}{
		{&fonts.regular, goregular.TTF},
		{&fonts.bold, gobold.TTF},
		{&fonts.mono, gomono.TTF},
		{&fonts.monoBold, gomonobold.TTF},
	} {
		parsed, err := opentype.Parse(entry.data)
		if err != nil {
			return builtinFonts{}, err
		}
		*entry.target = parsed
	}

	return fonts, nil
})

type textStyle struct {
	fill       color.RGBA
	fontSize   float64
	fontWeight string
	fontFamily string
	textAnchor string
}

type textRun struct {
	x, y    float64
	content string
	style   textStyle
	matrix  affine
}

func drawText(renderer *svgRenderer, svgBytes []byte) {
	dst := renderer.dst
	runs := parseTextRuns(svgBytes, renderer.classDeclarations)
	scaleX, scaleY := renderer.scaleX, renderer.scaleY
	sizeLimit := float64(maxFontSizeCanvasFactor * max(dst.Bounds().Dx(), dst.Bounds().Dy()))
	canvasArea := int64(dst.Bounds().Dx()) * int64(dst.Bounds().Dy())
	for _, run := range runs {
		if strings.TrimSpace(run.content) == "" || run.style.fontSize <= 0 || run.style.fill.A == 0 {
			continue
		}
		size := run.style.fontSize * max(scaleX, scaleY)
		if math.IsNaN(size) || size <= 0 {
			continue
		}
		size = min(size, sizeLimit)
		if !renderer.spend(textCost(size, run.content, canvasArea)) {
			return
		}
		x, y := run.matrix.apply(run.x, run.y)
		pxX, pxY := (x-renderer.viewBox.x)*scaleX, (y-renderer.viewBox.y)*scaleY
		face, err := fontFace(run.style, size)
		if err != nil {
			continue
		}
		drawer := &font.Drawer{Dst: dst, Src: image.NewUniform(run.style.fill), Face: face}
		switch run.style.textAnchor {
		case "middle":
			pxX -= float64(drawer.MeasureString(run.content)) / 128
		case "end":
			pxX -= float64(drawer.MeasureString(run.content)) / 64
		}
		drawer.Dot = fixed.P(clampDot(pxX), clampDot(pxY))
		drawer.DrawString(run.content)
		if closer, ok := face.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
}

// textCost charges a run roughly what its glyph masks cost to rasterise, capped
// at the canvas area because nothing larger can become visible.
func textCost(size float64, content string, canvasArea int64) int64 {
	glyphs := float64(len([]rune(content)))
	cost := size * size * glyphs
	if cost >= float64(canvasArea) {
		return canvasArea
	}
	return int64(cost) + 1
}

// clampDot keeps the pen inside the fixed-point range; fixed.P overflows for
// coordinates beyond ~2^25 and would wrap the baseline back onto the canvas.
func clampDot(value float64) int {
	const limit = 1 << 24
	if math.IsNaN(value) {
		return 0
	}

	return int(min(max(value, -limit), limit))
}

func parseTextRuns(svgBytes []byte, styles map[string]map[string]string) []textRun {
	decoder := xml.NewDecoder(bytes.NewReader(svgBytes))
	stack := []affine{identity()}
	var runs []textRun
	for {
		token, err := decoder.Token()
		if err != nil {
			return runs
		}
		switch current := token.(type) {
		case xml.StartElement:
			if strings.EqualFold(current.Name.Local, "text") {
				run := textRun{
					x:       parseNumber(attrValue(current.Attr, "x")),
					y:       parseNumber(attrValue(current.Attr, "y")),
					content: collectElementText(decoder),
					style:   resolveTextStyle(current.Attr, styles),
					matrix:  stack[len(stack)-1].multiply(parseTransform(attrValue(current.Attr, "transform"))),
				}
				runs = append(runs, run)
				continue
			}
			stack = append(stack, stack[len(stack)-1].multiply(parseTransform(attrValue(current.Attr, "transform"))))
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
}

func collectElementText(decoder *xml.Decoder) string {
	depth := 1
	var parts []string
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch current := token.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			parts = append(parts, string(current))
		}
	}
	return strings.Trim(strings.Join(parts, ""), "\n\r\t")
}

func resolveTextStyle(attrs []xml.Attr, styles map[string]map[string]string) textStyle {
	declarations := map[string]string{"fill": "#000000", "font-size": "16", "font-weight": "400", "font-family": "sans-serif"}
	for className := range strings.FieldsSeq(attrValue(attrs, "class")) {
		maps.Copy(declarations, styles[className])
	}
	maps.Copy(declarations, parseDeclarations(attrValue(attrs, "style")))
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "fill", "font-size", "font-weight", "font-family", "text-anchor":
			declarations[attr.Name.Local] = attr.Value
		}
	}
	return textStyle{
		fill:       parseColor(declarations[svgFill]),
		fontSize:   parseNumber(declarations["font-size"]),
		fontWeight: strings.ToLower(strings.TrimSpace(declarations["font-weight"])),
		fontFamily: strings.ToLower(strings.TrimSpace(declarations["font-family"])),
		textAnchor: strings.ToLower(strings.TrimSpace(declarations["text-anchor"])),
	}
}

func fontFace(style textStyle, size float64) (font.Face, error) {
	fonts, err := loadBuiltinFonts()
	if err != nil {
		return nil, fmt.Errorf("load svg fonts: %w", err)
	}
	selected := fonts.regular
	if strings.Contains(style.fontFamily, "mono") {
		selected = fonts.mono
	}
	if isBold(style.fontWeight) {
		selected = fonts.bold
		if strings.Contains(style.fontFamily, "mono") {
			selected = fonts.monoBold
		}
	}
	face, err := opentype.NewFace(selected, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, fmt.Errorf("new svg font face: %w", err)
	}
	return face, nil
}

func isBold(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "bold" || value == "bolder" {
		return true
	}
	weight, err := strconv.Atoi(value)
	return err == nil && weight >= 600
}
