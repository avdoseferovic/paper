package svg

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/vector"
)

// pathOpKind identifies a buffered drawing command in device pixel space.
type pathOpKind byte

const (
	opMove pathOpKind = iota
	opLine
	opQuad
	opCube
)

// pathOp is one buffered command. Commands are buffered rather than pushed
// straight into a rasterizer so the path's bounding box is known before a mask
// is allocated: the mask then only covers the shape instead of the whole canvas.
type pathOp struct {
	kind   pathOpKind
	coords [6]float32
}

type svgPath struct {
	renderer   *svgRenderer
	transform  affine
	ops        []pathOp
	bounds     deviceBounds
	lines      [][2]svgPoint
	current    svgPoint
	start      svgPoint
	hasCurrent bool
}

type svgPoint struct{ x, y float64 }

// deviceBounds accumulates the device-space extent of a path.
type deviceBounds struct {
	minX, minY, maxX, maxY float32
	valid                  bool
}

func (bounds *deviceBounds) add(x, y float32) {
	if !bounds.valid {
		bounds.minX, bounds.minY, bounds.maxX, bounds.maxY, bounds.valid = x, y, x, y, true
		return
	}
	bounds.minX = min(bounds.minX, x)
	bounds.minY = min(bounds.minY, y)
	bounds.maxX = max(bounds.maxX, x)
	bounds.maxY = max(bounds.maxY, y)
}

func (bounds deviceBounds) hasNaN() bool {
	return bounds.minX != bounds.minX || bounds.minY != bounds.minY ||
		bounds.maxX != bounds.maxX || bounds.maxY != bounds.maxY
}

// rect returns the pixel rectangle the path can touch, grown by pad and clipped
// to clip. A NaN bound (reachable through a degenerate transform) drops the
// shape: image.Rect would otherwise swap the inverted edges and hand back the
// whole canvas.
func (bounds deviceBounds) rect(clip image.Rectangle, pad float32) (image.Rectangle, bool) {
	if !bounds.valid || bounds.hasNaN() {
		return image.Rectangle{}, false
	}
	rect := image.Rect(
		clampInt(math.Floor(float64(bounds.minX-pad)), clip.Min.X, clip.Max.X),
		clampInt(math.Floor(float64(bounds.minY-pad)), clip.Min.Y, clip.Max.Y),
		// One extra pixel covers the partially painted edge pixel.
		clampInt(math.Ceil(float64(bounds.maxX+pad))+1, clip.Min.X, clip.Max.X),
		clampInt(math.Ceil(float64(bounds.maxY+pad))+1, clip.Min.Y, clip.Max.Y),
	).Intersect(clip)
	return rect, !rect.Empty()
}

// clampInt keeps an unbounded device coordinate inside the canvas; infinities
// saturate at the edges instead of overflowing the int conversion.
func clampInt(value float64, low, high int) int {
	if value <= float64(low) {
		return low
	}
	if value >= float64(high) {
		return high
	}
	return int(value)
}

func newSVGPath(renderer *svgRenderer, transform affine) *svgPath {
	return &svgPath{renderer: renderer, transform: transform}
}

func (path *svgPath) point(point svgPoint) (float32, float32) {
	x, y := path.transform.apply(point.x, point.y)
	return float32((x - path.renderer.viewBox.x) * path.renderer.scaleX), float32((y - path.renderer.viewBox.y) * path.renderer.scaleY)
}

func (path *svgPath) push(kind pathOpKind, coords ...float32) {
	op := pathOp{kind: kind}
	copy(op.coords[:], coords)
	path.ops = append(path.ops, op)
	for index := 0; index+1 < len(coords); index += 2 {
		path.bounds.add(coords[index], coords[index+1])
	}
}

func (path *svgPath) moveTo(x, y float64) {
	point := svgPoint{x, y}
	px, py := path.point(point)
	path.push(opMove, px, py)
	path.current, path.start, path.hasCurrent = point, point, true
}

func (path *svgPath) lineTo(x, y float64) {
	point := svgPoint{x, y}
	if !path.hasCurrent {
		path.moveTo(x, y)
		return
	}
	px, py := path.point(point)
	path.push(opLine, px, py)
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
	path.push(opCube, bx, by, cx, cy, dx, dy)
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
	path.push(opQuad, bx, by, cx, cy)
	path.lines = append(path.lines, [2]svgPoint{path.current, end})
	path.current = end
}

func (path *svgPath) close() {
	if !path.hasCurrent {
		return
	}
	px, py := path.point(path.start)
	path.push(opLine, px, py)
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
		path.fill(fill)
	}
	stroke := withOpacity(paint.stroke, paint.strokeOpacity*paint.opacity)
	if stroke.A == 0 || paint.strokeWidth <= 0 {
		return
	}
	width := paint.strokeWidth * math.Max(math.Abs(path.renderer.scaleX), math.Abs(path.renderer.scaleY))
	path.stroke(width, stroke)
}

// stroke rasterises every segment of the path. Segments are drawn either into
// one mask covering them all or into one mask each, whichever touches fewer
// pixels: a dense polyline is cheapest in a single pass, while segments spread
// across the canvas are cheapest one at a time. Every quad is wound in the same
// direction (see quadFor), so batching unions overlaps instead of cancelling
// them.
func (path *svgPath) stroke(width float64, paint color.RGBA) {
	clip := path.renderer.dst.Bounds()
	quads := make([]strokeQuad, 0, len(path.lines))
	var union deviceBounds
	var separate int64
	for _, segment := range path.lines {
		quad, ok := path.quadFor(segment[0], segment[1], width)
		if !ok {
			continue
		}
		rect, ok := quad.bounds.rect(clip, 0)
		if !ok {
			continue
		}
		quads = append(quads, quad)
		separate += int64(rect.Dx()) * int64(rect.Dy())
		union.add(quad.bounds.minX, quad.bounds.minY)
		union.add(quad.bounds.maxX, quad.bounds.maxY)
	}
	if len(quads) == 0 {
		return
	}
	unionRect, ok := union.rect(clip, 0)
	if ok && int64(unionRect.Dx())*int64(unionRect.Dy()) <= separate {
		path.drawQuads(quads, unionRect, paint)
		return
	}
	for _, quad := range quads {
		rect, ok := quad.bounds.rect(clip, 0)
		if !ok {
			continue
		}
		path.drawQuads([]strokeQuad{quad}, rect, paint)
		if path.renderer.exhausted {
			return
		}
	}
}

// strokeQuad is one segment widened into a rectangle in device space.
type strokeQuad struct {
	x1, y1, x2, y2 float32
	offsetX        float32
	offsetY        float32
	bounds         deviceBounds
}

func (path *svgPath) quadFor(from, to svgPoint, width float64) (strokeQuad, bool) {
	x1, y1 := path.point(from)
	x2, y2 := path.point(to)
	dx, dy := float64(x2-x1), float64(y2-y1)
	length := math.Hypot(dx, dy)
	if length == 0 || math.IsNaN(length) || math.IsInf(length, 0) {
		return strokeQuad{}, false
	}
	quad := strokeQuad{
		x1: x1, y1: y1, x2: x2, y2: y2,
		offsetX: float32(-dy / length * width / 2),
		offsetY: float32(dx / length * width / 2),
	}
	quad.bounds.add(x1+quad.offsetX, y1+quad.offsetY)
	quad.bounds.add(x2+quad.offsetX, y2+quad.offsetY)
	quad.bounds.add(x2-quad.offsetX, y2-quad.offsetY)
	quad.bounds.add(x1-quad.offsetX, y1-quad.offsetY)
	return quad, true
}

func (path *svgPath) drawQuads(quads []strokeQuad, rect image.Rectangle, paint color.RGBA) {
	rasterizer, ok := path.renderer.maskFor(rect)
	if !ok {
		return
	}
	originX, originY := float32(rect.Min.X), float32(rect.Min.Y)
	for _, quad := range quads {
		rasterizer.MoveTo(quad.x1+quad.offsetX-originX, quad.y1+quad.offsetY-originY)
		rasterizer.LineTo(quad.x2+quad.offsetX-originX, quad.y2+quad.offsetY-originY)
		rasterizer.LineTo(quad.x2-quad.offsetX-originX, quad.y2-quad.offsetY-originY)
		rasterizer.LineTo(quad.x1-quad.offsetX-originX, quad.y1-quad.offsetY-originY)
		rasterizer.ClosePath()
	}
	path.draw(rasterizer, rect, paint)
}

func (path *svgPath) fill(paint color.RGBA) {
	rect, ok := path.bounds.rect(path.renderer.dst.Bounds(), 0)
	if !ok {
		return
	}
	rasterizer, ok := path.renderer.maskFor(rect)
	if !ok {
		return
	}
	offsetX, offsetY := float32(rect.Min.X), float32(rect.Min.Y)
	for _, op := range path.ops {
		coords := op.coords
		switch op.kind {
		case opMove:
			rasterizer.MoveTo(coords[0]-offsetX, coords[1]-offsetY)
		case opLine:
			rasterizer.LineTo(coords[0]-offsetX, coords[1]-offsetY)
		case opQuad:
			rasterizer.QuadTo(coords[0]-offsetX, coords[1]-offsetY, coords[2]-offsetX, coords[3]-offsetY)
		case opCube:
			rasterizer.CubeTo(
				coords[0]-offsetX, coords[1]-offsetY,
				coords[2]-offsetX, coords[3]-offsetY,
				coords[4]-offsetX, coords[5]-offsetY,
			)
		}
	}
	path.draw(rasterizer, rect, paint)
}

func (path *svgPath) draw(rasterizer *vector.Rasterizer, rect image.Rectangle, paint color.RGBA) {
	rasterizer.Draw(path.renderer.dst, rect, image.NewUniform(paint), image.Point{})
}
