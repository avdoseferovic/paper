package svg

import (
	"encoding/xml"
	"image/color"
	"maps"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Compiled once: both expressions run for every element of every SVG, so
// compiling them per call dominated parse time on shape-heavy documents.
var (
	classRuleRe = regexp.MustCompile(`(?s)\.([A-Za-z0-9_-]+)\s*\{([^}]*)\}`)
	transformRe = regexp.MustCompile(`([a-zA-Z]+)\(([^)]*)\)`)
)

type affine struct{ a, b, c, d, e, f float64 }

func identity() affine { return affine{a: 1, d: 1} }

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

type svgPaint struct {
	fill, stroke               color.RGBA
	fillOpacity, strokeOpacity float64
	opacity                    float64
	strokeWidth                float64
	display                    bool
}

func defaultPaint() svgPaint {
	return svgPaint{fill: color.RGBA{A: 255}, fillOpacity: 1, strokeOpacity: 1, opacity: 1, strokeWidth: 1, display: true}
}

type svgRenderState struct {
	transform affine
	paint     svgPaint
}

func applySVGState(parent svgRenderState, attrs []xml.Attr, classes map[string]map[string]string) svgRenderState {
	state := parent
	state.transform = parent.transform.multiply(parseTransform(attrValue(attrs, "transform")))
	declarations := map[string]string{}
	for className := range strings.FieldsSeq(attrValue(attrs, "class")) {
		maps.Copy(declarations, classes[className])
	}
	maps.Copy(declarations, parseDeclarations(attrValue(attrs, "style")))
	for _, attr := range attrs {
		switch strings.ToLower(attr.Name.Local) {
		case svgFill, "stroke", "stroke-width", "fill-opacity", "stroke-opacity", "opacity", "display", "visibility":
			declarations[strings.ToLower(attr.Name.Local)] = attr.Value
		}
	}
	for key, value := range declarations {
		switch key {
		case svgFill:
			state.paint.fill = parseColor(value)
		case "stroke":
			state.paint.stroke = parseColor(value)
		case "stroke-width":
			if width := parseNumber(value); width >= 0 {
				state.paint.strokeWidth = width
			}
		case "fill-opacity":
			state.paint.fillOpacity = clampUnit(parseNumber(value))
		case "stroke-opacity":
			state.paint.strokeOpacity = clampUnit(parseNumber(value))
		case "opacity":
			state.paint.opacity = parent.paint.opacity * clampUnit(parseNumber(value))
		case "display":
			state.paint.display = parent.paint.display && strings.ToLower(strings.TrimSpace(value)) != "none"
		case "visibility":
			state.paint.display = parent.paint.display && strings.ToLower(strings.TrimSpace(value)) != "hidden"
		}
	}
	return state
}

func clampUnit(value float64) float64 {
	if math.IsNaN(value) || value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func withOpacity(value color.RGBA, opacity float64) color.RGBA {
	value.A = uint8(math.Round(float64(value.A) * clampUnit(opacity)))
	return value
}

func svgClassDeclarations(data []byte) map[string]map[string]string {
	styles := map[string]map[string]string{}
	for _, match := range classRuleRe.FindAllSubmatch(data, -1) {
		styles[string(match[1])] = parseDeclarations(string(match[2]))
	}
	return styles
}

func parseDeclarations(value string) map[string]string {
	result := map[string]string{}
	for part := range strings.SplitSeq(value, ";") {
		key, value, ok := strings.Cut(part, ":")
		if ok {
			result[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
		}
	}
	return result
}

func parseTransform(value string) affine {
	result := identity()
	if strings.TrimSpace(value) == "" {
		return result
	}
	for _, match := range transformRe.FindAllStringSubmatch(value, -1) {
		args := parseNumberList(match[2])
		switch strings.ToLower(match[1]) {
		case "translate":
			result = result.multiply(affine{a: 1, d: 1, e: numberAt(args, 0, 0), f: numberAt(args, 1, 0)})
		case "scale":
			sx := numberAt(args, 0, 1)
			result = result.multiply(affine{a: sx, d: numberAt(args, 1, sx)})
		case "rotate":
			angle := numberAt(args, 0, 0) * math.Pi / 180
			rotation := affine{a: math.Cos(angle), b: math.Sin(angle), c: -math.Sin(angle), d: math.Cos(angle)}
			if len(args) >= 3 {
				translate := affine{a: 1, d: 1, e: args[1], f: args[2]}
				inverse := affine{a: 1, d: 1, e: -args[1], f: -args[2]}
				result = result.multiply(translate).multiply(rotation).multiply(inverse)
			} else {
				result = result.multiply(rotation)
			}
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
	numbers := make([]float64, 0, strings.Count(value, " ")+1)
	for token := range strings.FieldsSeq(value) {
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
	for _, unit := range []string{"px", "pt", "mm", "cm"} {
		value = strings.TrimSuffix(value, unit)
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0
	}
	return number
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
	case "red":
		return color.RGBA{R: 255, A: 255}
	case "green":
		return color.RGBA{G: 128, A: 255}
	case "blue":
		return color.RGBA{B: 255, A: 255}
	}
	if hex, ok := strings.CutPrefix(value, "#"); ok {
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) == 6 {
			red, redErr := strconv.ParseUint(hex[:2], 16, 8)
			green, greenErr := strconv.ParseUint(hex[2:4], 16, 8)
			blue, blueErr := strconv.ParseUint(hex[4:], 16, 8)
			if redErr == nil && greenErr == nil && blueErr == nil {
				return color.RGBA{R: uint8(red), G: uint8(green), B: uint8(blue), A: 255}
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
		if strings.EqualFold(attr.Name.Local, name) {
			return attr.Value
		}
	}
	return ""
}
