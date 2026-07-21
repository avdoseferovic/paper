// Package svg provides pure-Go SVG rasterisation helpers for PDF embedding.
package svg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"maps"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/avdoseferovic/paper/internal/htmllimits"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

const (
	DPIForRaster    = 150.0
	MaxRasterPixels = 50_000_000
	svgFill         = "fill"
)

var (
	ErrSVGHasZeroDimensions = errors.New("svg has zero dimensions")
	ErrSVGTooLarge          = htmllimits.ErrSVGTooLarge
	errMissingSVGRoot       = errors.New("svg parse: missing <svg> root")

	fontsOnce sync.Once
	errFonts  error
	regular   *opentype.Font
	bold      *opentype.Font
	mono      *opentype.Font
	monoBold  *opentype.Font
)

type svgViewBox struct {
	x, y float64
	w, h float64
}

// Rasterize converts SVG bytes into PNG bytes at the requested mm dimensions.
func Rasterize(svgBytes []byte, widthMM, heightMM float64) ([]byte, int, int, error) {
	return RasterizeWithLimit(svgBytes, widthMM, heightMM, MaxRasterPixels)
}

// RasterizeWithLimit converts SVG bytes into PNG bytes with a pixel cap.
func RasterizeWithLimit(svgBytes []byte, widthMM, heightMM float64, maxPixels int64) ([]byte, int, int, error) {
	viewBox, err := readSVGViewBox(svgBytes)
	if err != nil {
		return nil, 0, 0, err
	}
	pxW, pxH := targetPixels(viewBox, widthMM, heightMM)
	if pxW <= 0 || pxH <= 0 {
		return nil, 0, 0, ErrSVGHasZeroDimensions
	}
	if maxPixels > 0 && int64(pxW) > maxPixels/int64(pxH) {
		return nil, 0, 0, fmt.Errorf("%w: %dx%d exceeds limit %d", ErrSVGTooLarge, pxW, pxH, maxPixels)
	}

	dst := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	err = drawSVG(svgBytes, viewBox, dst)
	if err != nil {
		return nil, 0, 0, err
	}
	drawText(svgBytes, viewBox, dst)

	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	err = encoder.Encode(&encoded, dst)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("png encode: %w", err)
	}
	return encoded.Bytes(), pxW, pxH, nil
}

func MMFromPx(px int) float64 { return float64(px) / DPIForRaster * 25.4 }

func pxFromMM(mm float64) int {
	px := int(mm / 25.4 * DPIForRaster)
	if px < 1 {
		return 1
	}
	return px
}

func targetPixels(viewBox svgViewBox, widthMM, heightMM float64) (int, int) {
	switch {
	case widthMM > 0 && heightMM > 0:
		return pxFromMM(widthMM), pxFromMM(heightMM)
	case widthMM > 0:
		width := pxFromMM(widthMM)
		if viewBox.w > 0 && viewBox.h > 0 {
			return width, max(1, int(float64(width)*viewBox.h/viewBox.w))
		}
		return width, width
	case heightMM > 0:
		height := pxFromMM(heightMM)
		if viewBox.w > 0 && viewBox.h > 0 {
			return max(1, int(float64(height)*viewBox.w/viewBox.h)), height
		}
		return height, height
	default:
		if viewBox.w > 0 && viewBox.h > 0 {
			return max(1, int(viewBox.w)), max(1, int(viewBox.h))
		}
		return 32, 32
	}
}

func readSVGViewBox(data []byte) (svgViewBox, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return svgViewBox{}, errMissingSVGRoot
		}
		if err != nil {
			return svgViewBox{}, fmt.Errorf("svg parse: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || strings.ToLower(start.Name.Local) != "svg" {
			continue
		}
		viewBox := svgViewBox{w: parseNumber(attrValue(start.Attr, "width")), h: parseNumber(attrValue(start.Attr, "height"))}
		if parts := parseNumberList(attrValue(start.Attr, "viewBox")); len(parts) == 4 {
			viewBox = svgViewBox{x: parts[0], y: parts[1], w: parts[2], h: parts[3]}
		}
		if viewBox.w < 0 || viewBox.h < 0 {
			return svgViewBox{}, ErrSVGHasZeroDimensions
		}
		return viewBox, nil
	}
}

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

type svgRenderer struct {
	dst               *image.RGBA
	viewBox           svgViewBox
	scaleX, scaleY    float64
	classDeclarations map[string]map[string]string
}

func drawSVG(data []byte, viewBox svgViewBox, dst *image.RGBA) error {
	scaleX, scaleY := svgScale(viewBox, dst.Bounds().Dx(), dst.Bounds().Dy())
	renderer := svgRenderer{
		dst:               dst,
		viewBox:           viewBox,
		scaleX:            scaleX,
		scaleY:            scaleY,
		classDeclarations: svgClassDeclarations(data),
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	states := []svgRenderState{{transform: identity(), paint: defaultPaint()}}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("svg parse: %w", err)
		}
		switch current := token.(type) {
		case xml.StartElement:
			state := applySVGState(states[len(states)-1], current.Attr, renderer.classDeclarations)
			states = append(states, state)
			renderer.drawElement(strings.ToLower(current.Name.Local), current.Attr, state)
		case xml.EndElement:
			if len(states) > 1 {
				states = states[:len(states)-1]
			}
		}
	}
	return nil
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
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func (renderer svgRenderer) drawElement(name string, attrs []xml.Attr, state svgRenderState) {
	if !state.paint.display {
		return
	}
	path := newSVGPath(renderer, state.transform)
	switch name {
	case "rect":
		x, y := parseNumber(attrValue(attrs, "x")), parseNumber(attrValue(attrs, "y"))
		width, height := parseNumber(attrValue(attrs, "width")), parseNumber(attrValue(attrs, "height"))
		if width <= 0 || height <= 0 {
			return
		}
		path.moveTo(x, y)
		path.lineTo(x+width, y)
		path.lineTo(x+width, y+height)
		path.lineTo(x, y+height)
		path.close()
	case "circle":
		cx, cy := parseNumber(attrValue(attrs, "cx")), parseNumber(attrValue(attrs, "cy"))
		radius := parseNumber(attrValue(attrs, "r"))
		path.ellipse(cx, cy, radius, radius)
	case "ellipse":
		cx, cy := parseNumber(attrValue(attrs, "cx")), parseNumber(attrValue(attrs, "cy"))
		path.ellipse(cx, cy, parseNumber(attrValue(attrs, "rx")), parseNumber(attrValue(attrs, "ry")))
	case "line":
		path.moveTo(parseNumber(attrValue(attrs, "x1")), parseNumber(attrValue(attrs, "y1")))
		path.lineTo(parseNumber(attrValue(attrs, "x2")), parseNumber(attrValue(attrs, "y2")))
	case "polyline", "polygon":
		points := parseNumberList(attrValue(attrs, "points"))
		if len(points) < 2 {
			return
		}
		path.moveTo(points[0], points[1])
		for index := 2; index+1 < len(points); index += 2 {
			path.lineTo(points[index], points[index+1])
		}
		if name == "polygon" {
			path.close()
		}
	case "path":
		if !path.parse(attrValue(attrs, "d")) {
			return
		}
	default:
		return
	}
	path.render(state.paint)
}

type svgPath struct {
	rasterizer *vector.Rasterizer
	renderer   svgRenderer
	transform  affine
	lines      [][2]svgPoint
	current    svgPoint
	start      svgPoint
	hasCurrent bool
}

type svgPoint struct{ x, y float64 }

func newSVGPath(renderer svgRenderer, transform affine) *svgPath {
	return &svgPath{
		rasterizer: vector.NewRasterizer(renderer.dst.Bounds().Dx(), renderer.dst.Bounds().Dy()),
		renderer:   renderer,
		transform:  transform,
	}
}

func (path *svgPath) point(point svgPoint) (float32, float32) {
	x, y := path.transform.apply(point.x, point.y)
	return float32((x - path.renderer.viewBox.x) * path.renderer.scaleX), float32((y - path.renderer.viewBox.y) * path.renderer.scaleY)
}

func (path *svgPath) moveTo(x, y float64) {
	point := svgPoint{x, y}
	px, py := path.point(point)
	path.rasterizer.MoveTo(px, py)
	path.current, path.start, path.hasCurrent = point, point, true
}

func (path *svgPath) lineTo(x, y float64) {
	point := svgPoint{x, y}
	if !path.hasCurrent {
		path.moveTo(x, y)
		return
	}
	px, py := path.point(point)
	path.rasterizer.LineTo(px, py)
	path.lines = append(path.lines, [2]svgPoint{path.current, point})
	path.current = point
}

func (path *svgPath) cubeTo(control1, control2, end svgPoint) {
	if !path.hasCurrent {
		path.moveTo(end.x, end.y)
		return
	}
	bx, by := path.point(control1)
	cx, cy := path.point(control2)
	dx, dy := path.point(end)
	path.rasterizer.CubeTo(bx, by, cx, cy, dx, dy)
	path.lines = append(path.lines, [2]svgPoint{path.current, end})
	path.current = end
}

func (path *svgPath) quadTo(control, end svgPoint) {
	if !path.hasCurrent {
		path.moveTo(end.x, end.y)
		return
	}
	bx, by := path.point(control)
	cx, cy := path.point(end)
	path.rasterizer.QuadTo(bx, by, cx, cy)
	path.lines = append(path.lines, [2]svgPoint{path.current, end})
	path.current = end
}

func (path *svgPath) close() {
	if !path.hasCurrent {
		return
	}
	path.rasterizer.ClosePath()
	path.lines = append(path.lines, [2]svgPoint{path.current, path.start})
	path.current = path.start
}

func (path *svgPath) ellipse(cx, cy, rx, ry float64) {
	if rx <= 0 || ry <= 0 {
		return
	}
	const kappa = 0.5522847498307936
	path.moveTo(cx+rx, cy)
	path.cubeTo(svgPoint{cx + rx, cy + kappa*ry}, svgPoint{cx + kappa*rx, cy + ry}, svgPoint{cx, cy + ry})
	path.cubeTo(svgPoint{cx - kappa*rx, cy + ry}, svgPoint{cx - rx, cy + kappa*ry}, svgPoint{cx - rx, cy})
	path.cubeTo(svgPoint{cx - rx, cy - kappa*ry}, svgPoint{cx - kappa*rx, cy - ry}, svgPoint{cx, cy - ry})
	path.cubeTo(svgPoint{cx + kappa*rx, cy - ry}, svgPoint{cx + rx, cy - kappa*ry}, svgPoint{cx + rx, cy})
	path.close()
}

func (path *svgPath) render(paint svgPaint) {
	if fill := withOpacity(paint.fill, paint.fillOpacity*paint.opacity); fill.A > 0 {
		path.rasterizer.Draw(path.renderer.dst, path.renderer.dst.Bounds(), image.NewUniform(fill), image.Point{})
	}
	if stroke := withOpacity(paint.stroke, paint.strokeOpacity*paint.opacity); stroke.A > 0 && paint.strokeWidth > 0 {
		width := paint.strokeWidth * math.Max(math.Abs(path.renderer.scaleX), math.Abs(path.renderer.scaleY))
		for _, segment := range path.lines {
			path.strokeSegment(segment[0], segment[1], width, stroke)
		}
	}
}

func (path *svgPath) strokeSegment(from, to svgPoint, width float64, paint color.RGBA) {
	x1, y1 := path.point(from)
	x2, y2 := path.point(to)
	dx, dy := float64(x2-x1), float64(y2-y1)
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	px, py := -dy/length*width/2, dx/length*width/2
	stroke := vector.NewRasterizer(path.renderer.dst.Bounds().Dx(), path.renderer.dst.Bounds().Dy())
	stroke.MoveTo(float32(float64(x1)+px), float32(float64(y1)+py))
	stroke.LineTo(float32(float64(x2)+px), float32(float64(y2)+py))
	stroke.LineTo(float32(float64(x2)-px), float32(float64(y2)-py))
	stroke.LineTo(float32(float64(x1)-px), float32(float64(y1)-py))
	stroke.ClosePath()
	stroke.Draw(path.renderer.dst, path.renderer.dst.Bounds(), image.NewUniform(paint), image.Point{})
}

func withOpacity(value color.RGBA, opacity float64) color.RGBA {
	value.A = uint8(math.Round(float64(value.A) * clampUnit(opacity)))
	return value
}

//nolint:gocognit,gocyclo // SVG paths are command-driven; keeping their state transitions together prevents divergence.
func (path *svgPath) parse(data string) bool {
	tokens := scanPathTokens(data)
	if len(tokens) == 0 {
		return false
	}
	index := 0
	command := byte(0)
	previousCommand := byte(0)
	lastCubic, lastQuadratic := svgPoint{}, svgPoint{}
	for index < len(tokens) {
		if tokens[index].command != 0 {
			command = tokens[index].command
			index++
		} else if command == 0 {
			return false
		}
		relative := command >= 'a' && command <= 'z'
		kind := command
		if kind >= 'a' && kind <= 'z' {
			kind -= 'a' - 'A'
		}
		read := func(count int) ([]float64, bool) {
			if index+count > len(tokens) {
				return nil, false
			}
			values := make([]float64, count)
			for offset := range values {
				if tokens[index+offset].command != 0 {
					return nil, false
				}
				values[offset] = tokens[index+offset].number
			}
			index += count
			return values, true
		}
		point := func(x, y float64) svgPoint {
			if relative && path.hasCurrent {
				return svgPoint{path.current.x + x, path.current.y + y}
			}
			return svgPoint{x, y}
		}
		switch kind {
		case 'M':
			values, ok := read(2)
			if !ok {
				return false
			}
			path.moveTo(point(values[0], values[1]).x, point(values[0], values[1]).y)
			if relative {
				command = 'l'
			} else {
				command = 'L'
			}
		case 'L':
			values, ok := read(2)
			if !ok {
				return false
			}
			end := point(values[0], values[1])
			path.lineTo(end.x, end.y)
		case 'H':
			values, ok := read(1)
			if !ok || !path.hasCurrent {
				return false
			}
			x := values[0]
			if relative {
				x += path.current.x
			}
			path.lineTo(x, path.current.y)
		case 'V':
			values, ok := read(1)
			if !ok || !path.hasCurrent {
				return false
			}
			y := values[0]
			if relative {
				y += path.current.y
			}
			path.lineTo(path.current.x, y)
		case 'C':
			values, ok := read(6)
			if !ok {
				return false
			}
			control1, control2, end := point(values[0], values[1]), point(values[2], values[3]), point(values[4], values[5])
			path.cubeTo(control1, control2, end)
			lastCubic = control2
		case 'S':
			values, ok := read(4)
			if !ok || !path.hasCurrent {
				return false
			}
			control1 := path.current
			if previousCommand == 'C' || previousCommand == 'c' || previousCommand == 'S' || previousCommand == 's' {
				control1 = svgPoint{2*path.current.x - lastCubic.x, 2*path.current.y - lastCubic.y}
			}
			control2, end := point(values[0], values[1]), point(values[2], values[3])
			path.cubeTo(control1, control2, end)
			lastCubic = control2
		case 'Q':
			values, ok := read(4)
			if !ok {
				return false
			}
			control, end := point(values[0], values[1]), point(values[2], values[3])
			path.quadTo(control, end)
			lastQuadratic = control
		case 'T':
			values, ok := read(2)
			if !ok || !path.hasCurrent {
				return false
			}
			control := path.current
			if previousCommand == 'Q' || previousCommand == 'q' || previousCommand == 'T' || previousCommand == 't' {
				control = svgPoint{2*path.current.x - lastQuadratic.x, 2*path.current.y - lastQuadratic.y}
			}
			end := point(values[0], values[1])
			path.quadTo(control, end)
			lastQuadratic = control
		case 'A':
			values, ok := read(7)
			if !ok {
				return false
			}
			// Arc endpoint semantics are preserved; this compact renderer draws
			// the chord until elliptical-arc flattening is required by Paper.
			end := point(values[5], values[6])
			path.lineTo(end.x, end.y)
		case 'Z':
			path.close()
		default:
			return false
		}
		previousCommand = command
	}
	return path.hasCurrent
}

type pathToken struct {
	command byte
	number  float64
}

//nolint:gocognit // The scanner deliberately accepts compact SVG number syntax in one bounded pass.
func scanPathTokens(value string) []pathToken {
	var tokens []pathToken
	for index := 0; index < len(value); {
		if strings.ContainsRune(" \t\r\n,", rune(value[index])) {
			index++
			continue
		}
		if value[index] >= 'A' && value[index] <= 'Z' || value[index] >= 'a' && value[index] <= 'z' {
			tokens = append(tokens, pathToken{command: value[index]})
			index++
			continue
		}
		start := index
		if value[index] == '+' || value[index] == '-' {
			index++
		}
		digits := false
		for index < len(value) && value[index] >= '0' && value[index] <= '9' {
			index++
			digits = true
		}
		if index < len(value) && value[index] == '.' {
			index++
			for index < len(value) && value[index] >= '0' && value[index] <= '9' {
				index++
				digits = true
			}
		}
		if index < len(value) && (value[index] == 'e' || value[index] == 'E') {
			exponent := index + 1
			if exponent < len(value) && (value[exponent] == '+' || value[exponent] == '-') {
				exponent++
			}
			end := exponent
			for end < len(value) && value[end] >= '0' && value[end] <= '9' {
				end++
			}
			if end > exponent {
				index = end
			}
		}
		if !digits {
			index++
			continue
		}
		number, err := strconv.ParseFloat(value[start:index], 64)
		if err == nil {
			tokens = append(tokens, pathToken{number: number})
		}
	}
	return tokens
}

func svgScale(viewBox svgViewBox, widthPx, heightPx int) (float64, float64) {
	if viewBox.w <= 0 {
		viewBox.w = float64(widthPx)
	}
	if viewBox.h <= 0 {
		viewBox.h = float64(heightPx)
	}
	return float64(widthPx) / viewBox.w, float64(heightPx) / viewBox.h
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

func drawText(svgBytes []byte, viewBox svgViewBox, dst *image.RGBA) {
	runs := parseTextRuns(svgBytes)
	scaleX, scaleY := svgScale(viewBox, dst.Bounds().Dx(), dst.Bounds().Dy())
	for _, run := range runs {
		if strings.TrimSpace(run.content) == "" || run.style.fontSize <= 0 || run.style.fill.A == 0 {
			continue
		}
		x, y := run.matrix.apply(run.x, run.y)
		pxX, pxY := (x-viewBox.x)*scaleX, (y-viewBox.y)*scaleY
		face, err := fontFace(run.style, run.style.fontSize*math.Max(scaleX, scaleY))
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
		drawer.Dot = fixed.P(int(math.Round(pxX)), int(math.Round(pxY)))
		drawer.DrawString(run.content)
		if closer, ok := face.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
}

func parseTextRuns(svgBytes []byte) []textRun {
	decoder := xml.NewDecoder(bytes.NewReader(svgBytes))
	styles := svgClassDeclarations(svgBytes)
	stack := []affine{identity()}
	var runs []textRun
	for {
		token, err := decoder.Token()
		if err != nil {
			return runs
		}
		switch current := token.(type) {
		case xml.StartElement:
			if strings.ToLower(current.Name.Local) == "text" {
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

func svgClassDeclarations(data []byte) map[string]map[string]string {
	styles := map[string]map[string]string{}
	rule := regexp.MustCompile(`(?s)\.([A-Za-z0-9_-]+)\s*\{([^}]*)\}`)
	for _, match := range rule.FindAllStringSubmatch(string(data), -1) {
		styles[match[1]] = parseDeclarations(match[2])
	}
	return styles
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
	re := regexp.MustCompile(`([a-zA-Z]+)\(([^)]*)\)`)
	for _, match := range re.FindAllStringSubmatch(value, -1) {
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
	number, _ := strconv.ParseFloat(value, 64)
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

func fontFace(style textStyle, size float64) (font.Face, error) {
	fontsOnce.Do(func() {
		regular, errFonts = opentype.Parse(goregular.TTF)
		if errFonts == nil {
			bold, errFonts = opentype.Parse(gobold.TTF)
		}
		if errFonts == nil {
			mono, errFonts = opentype.Parse(gomono.TTF)
		}
		if errFonts == nil {
			monoBold, errFonts = opentype.Parse(gomonobold.TTF)
		}
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
