// Package svg provides in-process SVG rasterisation helpers for PDF image
// embedding. The renderer is intentionally pure Go: it uses oksvg/rasterx for
// paths and a small text overlay pass for basic <text> elements.
package svg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"maps"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// DPIForRaster is the DPI used when rasterising SVG to PNG for PDF embedding.
const DPIForRaster = 150.0

// MaxRasterPixels caps the total pixel count of an SVG raster target. The
// dimensions feeding the rasteriser can be attacker-controlled in untrusted
// SVG input, so an unbounded image.NewRGBA allocation is a memory risk.
const MaxRasterPixels = 16 << 20

var (
	ErrSVGHasZeroDimensions = errors.New("svg has zero dimensions")
	ErrSVGTooLarge          = errors.New("svg raster dimensions exceed pixel budget")

	fontsOnce sync.Once
	errFonts  error
	regular   *opentype.Font
	bold      *opentype.Font
	mono      *opentype.Font
	monoBold  *opentype.Font
)

// Rasterize converts SVG bytes into PNG bytes at the requested mm dimensions
// using DPIForRaster. When widthMM/heightMM are zero it uses the SVG's viewBox.
func Rasterize(svgBytes []byte, widthMM, heightMM float64) ([]byte, int, int, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgBytes), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("svg parse: %w", err)
	}
	pxW, pxH := targetPixels(icon, widthMM, heightMM)
	if pxW <= 0 || pxH <= 0 {
		return nil, 0, 0, ErrSVGHasZeroDimensions
	}
	if int64(pxW)*int64(pxH) > MaxRasterPixels {
		return nil, 0, 0, fmt.Errorf("%w: %dx%d", ErrSVGTooLarge, pxW, pxH)
	}

	icon.SetTarget(0, 0, float64(pxW), float64(pxH))
	rgba := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	scanner := rasterx.NewScannerGV(pxW, pxH, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(pxW, pxH, scanner)
	icon.Draw(dasher, 1.0)
	drawText(svgBytes, icon, rgba)

	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.NoCompression}
	err = enc.Encode(&buf, rgba)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), pxW, pxH, nil
}

// MMFromPx converts pixels back to millimetres at DPIForRaster.
func MMFromPx(px int) float64 {
	return float64(px) / DPIForRaster * 25.4
}

func pxFromMM(mm float64) int {
	px := int(mm / 25.4 * DPIForRaster)
	if px < 1 {
		return 1
	}
	return px
}

func targetPixels(icon *oksvg.SvgIcon, widthMM, heightMM float64) (int, int) {
	vbW := icon.ViewBox.W
	vbH := icon.ViewBox.H
	switch {
	case widthMM > 0 && heightMM > 0:
		return pxFromMM(widthMM), pxFromMM(heightMM)
	case widthMM > 0:
		if vbW > 0 && vbH > 0 {
			pxW := pxFromMM(widthMM)
			return pxW, int(float64(pxW) * vbH / vbW)
		}
		return pxFromMM(widthMM), pxFromMM(widthMM)
	case heightMM > 0:
		if vbW > 0 && vbH > 0 {
			pxH := pxFromMM(heightMM)
			return int(float64(pxH) * vbW / vbH), pxH
		}
		return pxFromMM(heightMM), pxFromMM(heightMM)
	default:
		if vbW > 0 && vbH > 0 {
			return int(vbW), int(vbH)
		}
		return 32, 32
	}
}

type affine struct {
	a, b, c, d, e, f float64
}

func identity() affine {
	return affine{a: 1, d: 1}
}

func (m affine) multiply(n affine) affine {
	return affine{
		a: m.a*n.a + m.c*n.b,
		b: m.b*n.a + m.d*n.b,
		c: m.a*n.c + m.c*n.d,
		d: m.b*n.c + m.d*n.d,
		e: m.a*n.e + m.c*n.f + m.e,
		f: m.b*n.e + m.d*n.f + m.f,
	}
}

func (m affine) apply(x, y float64) (float64, float64) {
	return m.a*x + m.c*y + m.e, m.b*x + m.d*y + m.f
}

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

func drawText(svgBytes []byte, icon *oksvg.SvgIcon, dst *image.RGBA) {
	runs := parseTextRuns(svgBytes)
	if len(runs) == 0 {
		return
	}

	scaleX, scaleY := svgScale(icon, dst.Bounds().Dx(), dst.Bounds().Dy())
	for _, run := range runs {
		if strings.TrimSpace(run.content) == "" || run.style.fontSize <= 0 || run.style.fill.A == 0 {
			continue
		}
		x, y := run.matrix.apply(run.x, run.y)
		pxX := (x - icon.ViewBox.X) * scaleX
		pxY := (y - icon.ViewBox.Y) * scaleY
		size := run.style.fontSize * math.Max(scaleX, scaleY)
		face, err := fontFace(run.style, size)
		if err != nil {
			continue
		}
		drawer := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(run.style.fill),
			Face: face,
		}
		if run.style.textAnchor != "" && run.style.textAnchor != "start" {
			advance := drawer.MeasureString(run.content)
			switch run.style.textAnchor {
			case "middle":
				pxX -= float64(advance) / 128
			case "end":
				pxX -= float64(advance) / 64
			}
		}
		drawer.Dot = fixed.P(int(math.Round(pxX)), int(math.Round(pxY)))
		drawer.DrawString(run.content)
		if closer, ok := face.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
}

func svgScale(icon *oksvg.SvgIcon, widthPx, heightPx int) (float64, float64) {
	vbW := icon.ViewBox.W
	vbH := icon.ViewBox.H
	if vbW <= 0 {
		vbW = float64(widthPx)
	}
	if vbH <= 0 {
		vbH = float64(heightPx)
	}
	return float64(widthPx) / vbW, float64(heightPx) / vbH
}

func parseTextRuns(svgBytes []byte) []textRun {
	decoder := xml.NewDecoder(bytes.NewReader(svgBytes))
	styles := map[string]map[string]string{}
	stack := []affine{identity()}
	runs := make([]textRun, 0)

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "style":
				styles = mergeStyleMaps(styles, parseStyleElement(decoder))
				continue
			case "text":
				current := stack[len(stack)-1].multiply(parseTransform(attrValue(t.Attr, "transform")))
				run := textRun{
					x:       parseNumber(attrValue(t.Attr, "x")),
					y:       parseNumber(attrValue(t.Attr, "y")),
					content: collectElementText(decoder),
					style:   resolveTextStyle(t.Attr, styles),
					matrix:  current,
				}
				runs = append(runs, run)
				continue
			}
			stack = append(stack, stack[len(stack)-1].multiply(parseTransform(attrValue(t.Attr, "transform"))))
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return runs
}

func parseStyleElement(decoder *xml.Decoder) map[string]map[string]string {
	content := collectElementText(decoder)
	ruleRe := regexp.MustCompile(`(?s)\.([A-Za-z0-9_-]+)\s*\{([^}]*)\}`)
	styles := map[string]map[string]string{}
	for _, match := range ruleRe.FindAllStringSubmatch(content, -1) {
		styles[match[1]] = parseDeclarations(match[2])
	}
	return styles
}

func collectElementText(decoder *xml.Decoder) string {
	depth := 1
	var parts []string
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			parts = append(parts, string(t))
		}
	}
	return strings.Trim(strings.Join(parts, ""), "\n\r\t")
}

func mergeStyleMaps(base, extra map[string]map[string]string) map[string]map[string]string {
	maps.Copy(base, extra)
	return base
}

func resolveTextStyle(attrs []xml.Attr, styles map[string]map[string]string) textStyle {
	declarations := map[string]string{
		"fill":        "#000000",
		"font-size":   "16",
		"font-weight": "400",
		"font-family": "sans-serif",
	}
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
		fill:       parseColor(declarations["fill"]),
		fontSize:   parseNumber(declarations["font-size"]),
		fontWeight: strings.ToLower(strings.TrimSpace(declarations["font-weight"])),
		fontFamily: strings.ToLower(strings.TrimSpace(declarations["font-family"])),
		textAnchor: strings.ToLower(strings.TrimSpace(declarations["text-anchor"])),
	}
}

func parseDeclarations(value string) map[string]string {
	declarations := map[string]string{}
	for part := range strings.SplitSeq(value, ";") {
		key, val, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		declarations[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(val)
	}
	return declarations
}

func parseTransform(value string) affine {
	value = strings.TrimSpace(value)
	if value == "" {
		return identity()
	}
	result := identity()
	re := regexp.MustCompile(`([a-zA-Z]+)\(([^)]*)\)`)
	for _, match := range re.FindAllStringSubmatch(value, -1) {
		args := parseNumberList(match[2])
		switch strings.ToLower(match[1]) {
		case "translate":
			tx, ty := numberAt(args, 0, 0), numberAt(args, 1, 0)
			result = result.multiply(affine{a: 1, d: 1, e: tx, f: ty})
		case "scale":
			sx := numberAt(args, 0, 1)
			sy := numberAt(args, 1, sx)
			result = result.multiply(affine{a: sx, d: sy})
		case "matrix":
			if len(args) >= 6 {
				result = result.multiply(affine{a: args[0], b: args[1], c: args[2], d: args[3], e: args[4], f: args[5]})
			}
		}
	}
	return result
}

func parseNumberList(value string) []float64 {
	value = strings.ReplaceAll(value, ",", " ")
	tokens := strings.Fields(value)
	numbers := make([]float64, 0, len(tokens))
	for _, token := range tokens {
		numbers = append(numbers, parseNumber(token))
	}
	return numbers
}

func numberAt(numbers []float64, index int, fallback float64) float64 {
	if index < len(numbers) {
		return numbers[index]
	}
	return fallback
}

func parseNumber(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	value = strings.TrimSuffix(value, "px")
	value = strings.TrimSuffix(value, "pt")
	value = strings.TrimSuffix(value, "mm")
	value = strings.TrimSuffix(value, "cm")
	n, _ := strconv.ParseFloat(value, 64)
	return n
}

func parseColor(value string) color.RGBA {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "none" || strings.HasPrefix(value, "transparent") {
		return color.RGBA{}
	}
	switch value {
	case "black":
		return color.RGBA{A: 255}
	case "white":
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if hex, ok := strings.CutPrefix(value, "#"); ok {
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) == 6 {
			n, err := strconv.ParseUint(hex, 16, 32)
			if err == nil {
				return color.RGBA{R: byte((n >> 16) & 0xFF), G: byte((n >> 8) & 0xFF), B: byte(n & 0xFF), A: 255}
			}
		}
	}
	if strings.HasPrefix(value, "rgb(") && strings.HasSuffix(value, ")") {
		parts := parseNumberList(strings.TrimSuffix(strings.TrimPrefix(value, "rgb("), ")"))
		return color.RGBA{R: byte(numberAt(parts, 0, 0)), G: byte(numberAt(parts, 1, 0)), B: byte(numberAt(parts, 2, 0)), A: 255}
	}
	return color.RGBA{A: 255}
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func fontFace(style textStyle, size float64) (font.Face, error) {
	fontsOnce.Do(func() {
		regular, errFonts = opentype.Parse(goregular.TTF)
		if errFonts != nil {
			return
		}
		bold, errFonts = opentype.Parse(gobold.TTF)
		if errFonts != nil {
			return
		}
		mono, errFonts = opentype.Parse(gomono.TTF)
		if errFonts != nil {
			return
		}
		monoBold, errFonts = opentype.Parse(gomonobold.TTF)
	})
	if errFonts != nil {
		return nil, fmt.Errorf("load svg fonts: %w", errFonts)
	}

	selected := regular
	if strings.Contains(style.fontFamily, "mono") {
		selected = mono
	}
	if isBold(style.fontWeight) {
		selected = bold
		if strings.Contains(style.fontFamily, "mono") {
			selected = monoBold
		}
	}

	face, err := opentype.NewFace(selected, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
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
