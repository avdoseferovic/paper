package pdf

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func colorComp(v int) (int, float64) {
	if v < 0 {
		v = 0
	} else if v > 255 {
		v = 255
	}
	return v, float64(v) / 255.0
}

func rgbColorValue(r, g, b int, grayStr, fullStr string) colorType {
	var clr colorType
	clr.ir, clr.r = colorComp(r)
	clr.ig, clr.g = colorComp(g)
	clr.ib, clr.b = colorComp(b)
	clr.mode = colorModeRGB
	clr.gray = clr.ir == clr.ig && clr.r == clr.b
	if len(grayStr) > 0 {
		if clr.gray {
			clr.str = sprintf("%.3f %s", clr.r, grayStr)
		} else {
			clr.str = sprintf("%.3f %.3f %.3f %s", clr.r, clr.g, clr.b, fullStr)
		}
	} else {
		clr.str = sprintf("%.3f %.3f %.3f", clr.r, clr.g, clr.b)
	}
	return clr
}

// SetDrawColor defines the color used for all drawing operations (lines,
// rectangles and cell borders). It is expressed in RGB components (0 - 255).
// The method can be called before the first page is created. The value is
// retained from page to page.
func (f *PDF) SetDrawColor(r, g, b int) {
	f.setDrawColor(r, g, b)
}

func (f *PDF) setDrawColor(r, g, b int) {
	f.color.draw = rgbColorValue(r, g, b, "G", "RG")
	if f.page > 0 {
		f.out(f.color.draw.str)
	}
}

// GetDrawColor returns the most recently set draw color as RGB components (0 -
// 255). This will not be the current value if a draw color of some other type
// (for example, spot) has been more recently set.
func (f *PDF) GetDrawColor() (int, int, int) {
	return f.color.draw.ir, f.color.draw.ig, f.color.draw.ib
}

// SetFillColor defines the color used for all filling operations (filled
// rectangles and cell backgrounds). It is expressed in RGB components (0
// -255). The method can be called before the first page is created and the
// value is retained from page to page.
func (f *PDF) SetFillColor(r, g, b int) {
	f.setFillColor(r, g, b)
}

func (f *PDF) setFillColor(r, g, b int) {
	f.color.fill = rgbColorValue(r, g, b, "g", "rg")
	f.colorFlag = f.color.fill.str != f.color.text.str
	if f.page > 0 {
		f.out(f.color.fill.str)
	}
}

// GetFillColor returns the most recently set fill color as RGB components (0 -
// 255). This will not be the current value if a fill color of some other type
// (for example, spot) has been more recently set.
func (f *PDF) GetFillColor() (int, int, int) {
	return f.color.fill.ir, f.color.fill.ig, f.color.fill.ib
}

// SetTextColor defines the color used for text. It is expressed in RGB
// components (0 - 255). The method can be called before the first page is
// created. The value is retained from page to page.
func (f *PDF) SetTextColor(r, g, b int) {
	f.setTextColor(r, g, b)
}

func (f *PDF) setTextColor(r, g, b int) {
	f.color.text = rgbColorValue(r, g, b, "g", "rg")
	f.colorFlag = f.color.fill.str != f.color.text.str
}

// GetTextColor returns the most recently set text color as RGB components (0 -
// 255). This will not be the current value if a text color of some other type
// (for example, spot) has been more recently set.
func (f *PDF) GetTextColor() (int, int, int) {
	return f.color.text.ir, f.color.text.ig, f.color.text.ib
}

// GetAlpha returns the alpha blending channel, which consists of the
// alpha transparency value and the blend mode. See SetAlpha for more
// details.
func (f *PDF) GetAlpha() (float64, string) {
	return f.alpha, f.blendMode
}

// SetAlpha sets the alpha blending channel. The blending effect applies to
// text, drawings and images.
//
// alpha must be a value between 0.0 (fully transparent) to 1.0 (fully opaque).
// Values outside of this range result in an error.
//
// blendModeStr must be one of "Normal", "Multiply", "Screen", "Overlay",
// "Darken", "Lighten", "ColorDodge", "ColorBurn","HardLight", "SoftLight",
// "Difference", "Exclusion", "Hue", "Saturation", "Color", or "Luminosity". An
// empty string is replaced with "Normal".
//
// To reset normal rendering after applying a blending mode, call this method
// with alpha set to 1.0 and blendModeStr set to "Normal".
func (f *PDF) SetAlpha(alpha float64, blendModeStr string) {
	if f.err != nil {
		return
	}
	var bl blendModeType
	switch blendModeStr {
	case blendModeNormal, "Multiply", "Screen", "Overlay",
		"Darken", "Lighten", "ColorDodge", "ColorBurn", "HardLight", "SoftLight",
		"Difference", "Exclusion", "Hue", "Saturation", "Color", "Luminosity":
		bl.modeStr = blendModeStr
	case "":
		bl.modeStr = blendModeNormal
	default:
		f.err = fmt.Errorf("%w: %q", errUnrecognizedBlendMode, blendModeStr)
		return
	}
	if alpha < 0.0 || alpha > 1.0 {
		f.err = fmt.Errorf("%w: %.3f", errAlphaOutOfRange, alpha)
		return
	}
	f.alpha = alpha
	f.blendMode = blendModeStr
	alphaStr := sprintf("%.3f", alpha)
	keyStr := sprintf("%s %s", alphaStr, blendModeStr)
	pos, ok := f.blendMap[keyStr]
	if !ok {
		pos = len(f.blendList)
		f.blendList = append(f.blendList, blendModeType{alphaStr, alphaStr, blendModeStr, 0})
		f.blendMap[keyStr] = pos
	}
	f.outf("/GS%d gs", pos)
}

// SetLineWidth defines the line width. By default, the value equals 0.2 mm.
// The method can be called before the first page is created. The value is
// retained from page to page.
func (f *PDF) SetLineWidth(width float64) {
	f.setLineWidth(width)
}

func (f *PDF) setLineWidth(width float64) {
	f.lineWidth = width
	if f.page > 0 {
		f.outf("%.2f w", width*f.k)
	}
}

// GetLineWidth returns the current line thickness.
func (f *PDF) GetLineWidth() float64 {
	return f.lineWidth
}

// SetLineCapStyle defines the line cap style. styleStr should be "butt",
// "round" or "square". A square style projects from the end of the line. The
// method can be called before the first page is created. The value is
// retained from page to page.
func (f *PDF) SetLineCapStyle(styleStr string) {
	var capStyle int
	switch styleStr {
	case "round":
		capStyle = 1
	case "square":
		capStyle = 2
	default:
		capStyle = 0
	}
	f.capStyle = capStyle
	if f.page > 0 {
		f.outf("%d J", f.capStyle)
	}
}

// SetLineJoinStyle defines the line cap style. styleStr should be "miter",
// "round" or "bevel". The method can be called before the first page
// is created. The value is retained from page to page.
func (f *PDF) SetLineJoinStyle(styleStr string) {
	var joinStyle int
	switch styleStr {
	case "round":
		joinStyle = 1
	case "bevel":
		joinStyle = 2
	default:
		joinStyle = 0
	}
	f.joinStyle = joinStyle
	if f.page > 0 {
		f.outf("%d j", f.joinStyle)
	}
}

// SetDashPattern sets the dash pattern that is used to draw lines. The
// dashArray elements are numbers that specify the lengths, in units
// established in New(), of alternating dashes and gaps. The dash phase
// specifies the distance into the dash pattern at which to start the dash. The
// dash pattern is retained from page to page. Call this method with an empty
// array to restore solid line drawing.
//
// The Beziergon() example demonstrates this method.
func (f *PDF) SetDashPattern(dashArray []float64, dashPhase float64) {
	scaled := make([]float64, len(dashArray))
	for i, value := range dashArray {
		scaled[i] = value * f.k
	}
	dashPhase *= f.k

	f.dashArray = scaled
	f.dashPhase = dashPhase
	if f.page > 0 {
		f.outputDashPattern()
	}
}

func (f *PDF) outputDashPattern() {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, value := range f.dashArray {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(strconv.FormatFloat(value, 'f', 2, 64))
	}
	buf.WriteString("] ")
	buf.WriteString(strconv.FormatFloat(f.dashPhase, 'f', 2, 64))
	buf.WriteString(" d")
	f.outbuf(&buf)
}

// Line draws a line between points (x1, y1) and (x2, y2) using the current
// draw color, line width and cap style.
func (f *PDF) Line(x1, y1, x2, y2 float64) {
	f.outf("%.2f %.2f m %.2f %.2f l S", x1*f.k, (f.h-y1)*f.k, x2*f.k, (f.h-y2)*f.k)
}

// fillDrawOp corrects path painting operators
func fillDrawOp(styleStr string) string {
	switch strings.ToUpper(styleStr) {
	case "", "D":
		return "S"
	case "F":
		return "f"
	case "F*":
		return "f*"
	case "FD", "DF":
		return "B"
	case "FD*", "DF*":
		return "B*"
	default:
		return styleStr
	}
}

// Rect outputs a rectangle of width w and height h with the upper left corner
// positioned at point (x, y).
//
// It can be drawn (border only), filled (with no border) or both. styleStr can
// be "F" for filled, "D" for outlined only, or "DF" or "FD" for outlined and
// filled. An empty string will be replaced with "D". Drawing uses the current
// draw color and line width centered on the rectangle's perimeter. Filling
// uses the current fill color.
func (f *PDF) Rect(x, y, w, h float64, styleStr string) {
	f.outf("%.2f %.2f %.2f %.2f re %s", x*f.k, (f.h-y)*f.k, w*f.k, -h*f.k, fillDrawOp(styleStr))
}

// RoundedRect outputs a rectangle of width w and height h with the upper left
// corner positioned at point (x, y). It can be drawn (border only), filled
// (with no border) or both. styleStr can be "F" for filled, "D" for outlined
// only, or "DF" or "FD" for outlined and filled. An empty string will be
// replaced with "D". Drawing uses the current draw color and line width
// centered on the rectangle's perimeter. Filling uses the current fill color.
// The rounded corners of the rectangle are specified by radius r. corners is a
// string that includes "1" to round the upper left corner, "2" to round the
// upper right corner, "3" to round the lower right corner, and "4" to round
// the lower left corner. The RoundedRect example demonstrates this method.
func (f *PDF) RoundedRect(x, y, w, h, r float64, corners string, stylestr string) {
	// This routine was adapted by Brigham Thompson from a script by Christophe Prugnaud
	var rTL, rTR, rBR, rBL float64 // zero means no rounded corner
	if strings.Contains(corners, "1") {
		rTL = r
	}
	if strings.Contains(corners, "2") {
		rTR = r
	}
	if strings.Contains(corners, "3") {
		rBR = r
	}
	if strings.Contains(corners, "4") {
		rBL = r
	}
	f.RoundedRectExt(x, y, w, h, rTL, rTR, rBR, rBL, stylestr)
}

// RoundedRectExt behaves the same as RoundedRect() but supports a different
// radius for each corner. A zero radius means squared corner. See
// RoundedRect() for more details. This method is demonstrated in the
// RoundedRect() example.
func (f *PDF) RoundedRectExt(x, y, w, h, rTL, rTR, rBR, rBL float64, stylestr string) {
	f.roundedRectPath(x, y, w, h, rTL, rTR, rBR, rBL)
	f.out(fillDrawOp(stylestr))
}

// Circle draws a circle centered on point (x, y) with radius r.
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D". Drawing uses
// the current draw color and line width centered on the circle's perimeter.
// Filling uses the current fill color.
func (f *PDF) Circle(x, y, r float64, styleStr string) {
	f.Ellipse(x, y, r, r, 0, styleStr)
}

// Ellipse draws an ellipse centered at point (x, y). rx and ry specify its
// horizontal and vertical radii.
//
// degRotate specifies the counter-clockwise angle in degrees that the ellipse
// will be rotated.
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D". Drawing uses
// the current draw color and line width centered on the ellipse's perimeter.
// Filling uses the current fill color.
//
// The Circle() example demonstrates this method.
func (f *PDF) Ellipse(x, y, rx, ry, degRotate float64, styleStr string) {
	f.arc(x, y, rx, ry, degRotate, 0, 360, styleStr, false)
}

// Polygon draws a closed figure defined by a series of vertices specified by
// points. The x and y fields of the points use the units established in New().
// The last point in the slice will be implicitly joined to the first to close
// the polygon.
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D". Drawing uses
// the current draw color and line width centered on the ellipse's perimeter.
// Filling uses the current fill color.
func (f *PDF) Polygon(points []PointType, styleStr string) {
	if len(points) > 2 {
		for j, pt := range points {
			if j == 0 {
				f.point(pt.X, pt.Y)
			} else {
				f.outf("%.5f %.5f l ", pt.X*f.k, (f.h-pt.Y)*f.k)
			}
		}
		f.outf("%.5f %.5f l ", points[0].X*f.k, (f.h-points[0].Y)*f.k)
		f.DrawPath(styleStr)
	}
}

// Beziergon draws a closed figure defined by a series of cubic Bézier curve
// segments. The first point in the slice defines the starting point of the
// figure. Each three following points p1, p2, p3 represent a curve segment to
// the point p3 using p1 and p2 as the Bézier control points.
//
// The x and y fields of the points use the units established in New().
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D". Drawing uses
// the current draw color and line width centered on the ellipse's perimeter.
// Filling uses the current fill color.
func (f *PDF) Beziergon(points []PointType, styleStr string) {
	if len(points) < 4 {
		return
	}
	f.point(points[0].XY())

	points = points[1:]
	for len(points) >= 3 {
		cx0, cy0 := points[0].XY()
		cx1, cy1 := points[1].XY()
		x1, y1 := points[2].XY()
		f.curve(cx0, cy0, cx1, cy1, x1, y1)
		points = points[3:]
	}

	f.DrawPath(styleStr)
}

// point outputs current point
func (f *PDF) point(x, y float64) {
	f.outf("%.2f %.2f m", x*f.k, (f.h-y)*f.k)
}

// curve outputs a single cubic Bézier curve segment from current point
func (f *PDF) curve(cx0, cy0, cx1, cy1, x, y float64) {
	f.outf("%.5f %.5f %.5f %.5f %.5f %.5f c", cx0*f.k, (f.h-cy0)*f.k, cx1*f.k,
		(f.h-cy1)*f.k, x*f.k, (f.h-y)*f.k)
}

// Curve draws a single-segment quadratic Bézier curve. The curve starts at
// the point (x0, y0) and ends at the point (x1, y1). The control point (cx,
// cy) specifies the curvature. At the start point, the curve is tangent to the
// straight line between the start point and the control point. At the end
// point, the curve is tangent to the straight line between the end point and
// the control point.
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D". Drawing uses
// the current draw color, line width, and cap style centered on the curve's
// path. Filling uses the current fill color.
//
// The Circle() example demonstrates this method.
func (f *PDF) Curve(x0, y0, cx, cy, x1, y1 float64, styleStr string) {
	f.point(x0, y0)
	f.outf("%.5f %.5f %.5f %.5f v %s", cx*f.k, (f.h-cy)*f.k, x1*f.k, (f.h-y1)*f.k,
		fillDrawOp(styleStr))
}

// CurveCubic draws a single-segment cubic Bézier curve. This routine performs
// the same function as CurveBezierCubic() but has a nonstandard argument order.
// It is retained to preserve backward compatibility.
func (f *PDF) CurveCubic(x0, y0, cx0, cy0, x1, y1, cx1, cy1 float64, styleStr string) {
	f.CurveBezierCubic(x0, y0, cx0, cy0, cx1, cy1, x1, y1, styleStr)
}

// CurveBezierCubic draws a single-segment cubic Bézier curve. The curve starts at
// the point (x0, y0) and ends at the point (x1, y1). The control points (cx0,
// cy0) and (cx1, cy1) specify the curvature. At the start point, the curve is
// tangent to the straight line between the start point and the control point
// (cx0, cy0). At the end point, the curve is tangent to the straight line
// between the end point and the control point (cx1, cy1).
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D". Drawing uses
// the current draw color, line width, and cap style centered on the curve's
// path. Filling uses the current fill color.
//
// This routine performs the same function as CurveCubic() but uses standard
// argument order.
//
// The Circle() example demonstrates this method.
func (f *PDF) CurveBezierCubic(x0, y0, cx0, cy0, cx1, cy1, x1, y1 float64, styleStr string) {
	f.point(x0, y0)
	f.outf("%.5f %.5f %.5f %.5f %.5f %.5f c %s", cx0*f.k, (f.h-cy0)*f.k,
		cx1*f.k, (f.h-cy1)*f.k, x1*f.k, (f.h-y1)*f.k, fillDrawOp(styleStr))
}

// Arc draws an elliptical arc centered at point (x, y). rx and ry specify its
// horizontal and vertical radii.
//
// degRotate specifies the angle that the arc will be rotated. degStart and
// degEnd specify the starting and ending angle of the arc. All angles are
// specified in degrees and measured counter-clockwise from the 3 o'clock
// position.
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D". Drawing uses
// the current draw color, line width, and cap style centered on the arc's
// path. Filling uses the current fill color.
//
// The Circle() example demonstrates this method.
func (f *PDF) Arc(x, y, rx, ry, degRotate, degStart, degEnd float64, styleStr string) {
	f.arc(x, y, rx, ry, degRotate, degStart, degEnd, styleStr, false)
}

func (f *PDF) gradientClipStart(x, y, w, h float64) {
	f.outf("q %.2f %.2f %.2f %.2f re W n", x*f.k, (f.h-y)*f.k, w*f.k, -h*f.k)

	f.outf("%.5f 0 0 %.5f %.5f %.5f cm", w*f.k, h*f.k, x*f.k, (f.h-(y+h))*f.k)
}

func (f *PDF) gradientClipEnd() {
	f.out("Q")
}

func (f *PDF) gradient(tp, r1, g1, b1, r2, g2, b2 int, x1, y1, x2, y2, r float64) {
	pos := len(f.gradientList)
	clr1 := rgbColorValue(r1, g1, b1, "", "")
	clr2 := rgbColorValue(r2, g2, b2, "", "")
	f.gradientList = append(f.gradientList, gradientType{
		tp, clr1.str, clr2.str,
		x1, y1, x2, y2, r, 0,
	})
	f.outf("/Sh%d sh", pos)
}

// LinearGradient draws a rectangular area with a blending of one color to
// another. The rectangle is of width w and height h. Its upper left corner is
// positioned at point (x, y).
//
// Each color is specified with three component values, one each for red, green
// and blue. The values range from 0 to 255. The first color is specified by
// (r1, g1, b1) and the second color by (r2, g2, b2).
//
// The blending is controlled with a gradient vector that uses normalized
// coordinates in which the lower left corner is position (0, 0) and the upper
// right corner is (1, 1). The vector's origin and destination are specified by
// the points (x1, y1) and (x2, y2). In a linear gradient, blending occurs
// perpendicularly to the vector. The vector does not necessarily need to be
// anchored on the rectangle edge. Color 1 is used up to the origin of the
// vector and color 2 is used beyond the vector's end point. Between the points
// the colors are gradually blended.
func (f *PDF) LinearGradient(x, y, w, h float64, r1, g1, b1, r2, g2, b2 int, x1, y1, x2, y2 float64) {
	f.gradientClipStart(x, y, w, h)
	f.gradient(2, r1, g1, b1, r2, g2, b2, x1, y1, x2, y2, 0)
	f.gradientClipEnd()
}

// RadialGradient draws a rectangular area with a blending of one color to
// another. The rectangle is of width w and height h. Its upper left corner is
// positioned at point (x, y).
//
// Each color is specified with three component values, one each for red, green
// and blue. The values range from 0 to 255. The first color is specified by
// (r1, g1, b1) and the second color by (r2, g2, b2).
//
// The blending is controlled with a point and a circle, both specified with
// normalized coordinates in which the lower left corner of the rendered
// rectangle is position (0, 0) and the upper right corner is (1, 1). Color 1
// begins at the origin point specified by (x1, y1). Color 2 begins at the
// circle specified by the center point (x2, y2) and radius r. Colors are
// gradually blended from the origin to the circle. The origin and the circle's
// center do not necessarily have to coincide, but the origin must be within
// the circle to avoid rendering problems.
//
// The LinearGradient() example demonstrates this method.
func (f *PDF) RadialGradient(x, y, w, h float64, r1, g1, b1, r2, g2, b2 int, x1, y1, x2, y2, r float64) {
	f.gradientClipStart(x, y, w, h)
	f.gradient(3, r1, g1, b1, r2, g2, b2, x1, y1, x2, y2, r)
	f.gradientClipEnd()
}

func (f *PDF) putBlendModes() {
	count := len(f.blendList)
	for j := 1; j < count; j++ {
		bl := f.blendList[j]
		f.newobj()
		f.blendList[j].objNum = f.n
		f.outf("<</Type /ExtGState /ca %s /CA %s /BM /%s>>",
			bl.fillStr, bl.strokeStr, bl.modeStr)
		f.out("endobj")
	}
}

func (f *PDF) putGradients() {
	count := len(f.gradientList)
	for j := 1; j < count; j++ {
		var f1 int
		gr := f.gradientList[j]
		switch gr.tp {
		case 2, 3:
			f.newobj()
			f.outf("<</FunctionType 2 /Domain [0.0 1.0] /C0 [%s] /C1 [%s] /N 1>>", gr.clr1Str, gr.clr2Str)
			f.out("endobj")
			f1 = f.n
		}
		f.newobj()
		f.outf("<</ShadingType %d /ColorSpace /DeviceRGB", gr.tp)
		switch gr.tp {
		case 2:
			f.outf("/Coords [%.5f %.5f %.5f %.5f] /Function %d 0 R /Extend [true true]>>",
				gr.x1, gr.y1, gr.x2, gr.y2, f1)
		case 3:
			f.outf("/Coords [%.5f %.5f 0 %.5f %.5f %.5f] /Function %d 0 R /Extend [true true]>>",
				gr.x1, gr.y1, gr.x2, gr.y2, gr.r, f1)
		}
		f.out("endobj")
		f.gradientList[j].objNum = f.n
	}
}

// ClipRect begins a rectangular clipping operation. The rectangle is of width
// w and height h. Its upper left corner is positioned at point (x, y). outline
// is true to draw a border with the current draw color and line width centered
// on the rectangle's perimeter. Only the outer half of the border will be
// shown. After calling this method, all rendering operations (for example,
// Image(), LinearGradient(), etc) will be clipped by the specified rectangle.
// Call ClipEnd() to restore unclipped operations.
//
// This ClipText() example demonstrates this method.
func (f *PDF) ClipRect(x, y, w, h float64, outline bool) {
	f.clipNest++
	f.outf("q %.2f %.2f %.2f %.2f re W %s", x*f.k, (f.h-y)*f.k, w*f.k, -h*f.k, strIf(outline, "S", "n"))
}

// ClipText begins a clipping operation in which rendering is confined to the
// character string specified by txtStr. The origin (x, y) is on the left of
// the first character at the baseline. The current font is used. outline is
// true to draw a border with the current draw color and line width centered on
// the perimeters of the text characters. Only the outer half of the border
// will be shown. After calling this method, all rendering operations (for
// example, Image(), LinearGradient(), etc) will be clipped. Call ClipEnd() to
// restore unclipped operations.
func (f *PDF) ClipText(x, y float64, txtStr string, outline bool) {
	f.clipNest++
	f.outf("q BT %.5f %.5f Td %d Tr (%s) Tj ET", x*f.k, (f.h-y)*f.k, intIf(outline, 5, 7), f.escape(txtStr))
}

func (f *PDF) clipArc(x1, y1, x2, y2, x3, y3 float64) {
	h := f.h
	f.outf("%.5f %.5f %.5f %.5f %.5f %.5f c ", x1*f.k, (h-y1)*f.k,
		x2*f.k, (h-y2)*f.k, x3*f.k, (h-y3)*f.k)
}

// ClipRoundedRect begins a rectangular clipping operation. The rectangle is of
// width w and height h. Its upper left corner is positioned at point (x, y).
// The rounded corners of the rectangle are specified by radius r. outline is
// true to draw a border with the current draw color and line width centered on
// the rectangle's perimeter. Only the outer half of the border will be shown.
// After calling this method, all rendering operations (for example, Image(),
// LinearGradient(), etc) will be clipped by the specified rectangle. Call
// ClipEnd() to restore unclipped operations.
//
// This ClipText() example demonstrates this method.
func (f *PDF) ClipRoundedRect(x, y, w, h, r float64, outline bool) {
	f.ClipRoundedRectExt(x, y, w, h, r, r, r, r, outline)
}

// ClipRoundedRectExt behaves the same as ClipRoundedRect() but supports a
// different radius for each corner, given by rTL (top-left), rTR (top-right)
// rBR (bottom-right), rBL (bottom-left). See ClipRoundedRect() for more
// details. This method is demonstrated in the ClipText() example.
func (f *PDF) ClipRoundedRectExt(x, y, w, h, rTL, rTR, rBR, rBL float64, outline bool) {
	f.clipNest++
	f.roundedRectPath(x, y, w, h, rTL, rTR, rBR, rBL)
	f.outf(" W %s", strIf(outline, "S", "n"))
}

// add a rectangle path with rounded corners.
// routine shared by RoundedRect() and ClipRoundedRect(), which add the
// drawing operation
func (f *PDF) roundedRectPath(x, y, w, h, rTL, rTR, rBR, rBL float64) {
	k := f.k
	hp := f.h
	myArc := (4.0 / 3.0) * (math.Sqrt2 - 1.0)
	f.outf("q %.5f %.5f m", (x+rTL)*k, (hp-y)*k)
	xc := x + w - rTR
	yc := y + rTR
	f.outf("%.5f %.5f l", xc*k, (hp-y)*k)
	if rTR != 0 {
		f.clipArc(xc+rTR*myArc, yc-rTR, xc+rTR, yc-rTR*myArc, xc+rTR, yc)
	}
	xc = x + w - rBR
	yc = y + h - rBR
	f.outf("%.5f %.5f l", (x+w)*k, (hp-yc)*k)
	if rBR != 0 {
		f.clipArc(xc+rBR, yc+rBR*myArc, xc+rBR*myArc, yc+rBR, xc, yc+rBR)
	}
	xc = x + rBL
	yc = y + h - rBL
	f.outf("%.5f %.5f l", xc*k, (hp-(y+h))*k)
	if rBL != 0 {
		f.clipArc(xc-rBL*myArc, yc+rBL, xc-rBL, yc+rBL*myArc, xc-rBL, yc)
	}
	xc = x + rTL
	yc = y + rTL
	f.outf("%.5f %.5f l", x*k, (hp-yc)*k)
	if rTL != 0 {
		f.clipArc(xc-rTL, yc-rTL*myArc, xc-rTL*myArc, yc-rTL, xc, yc-rTL)
	}
}

// ClipEllipse begins an elliptical clipping operation. The ellipse is centered
// at (x, y). Its horizontal and vertical radii are specified by rx and ry.
// outline is true to draw a border with the current draw color and line width
// centered on the ellipse's perimeter. Only the outer half of the border will
// be shown. After calling this method, all rendering operations (for example,
// Image(), LinearGradient(), etc) will be clipped by the specified ellipse.
// Call ClipEnd() to restore unclipped operations.
//
// This ClipText() example demonstrates this method.
func (f *PDF) ClipEllipse(x, y, rx, ry float64, outline bool) {
	f.clipNest++
	lx := (4.0 / 3.0) * rx * (math.Sqrt2 - 1)
	ly := (4.0 / 3.0) * ry * (math.Sqrt2 - 1)
	k := f.k
	h := f.h
	f.outf("q %.5f %.5f m %.5f %.5f %.5f %.5f %.5f %.5f c",
		(x+rx)*k, (h-y)*k,
		(x+rx)*k, (h-(y-ly))*k,
		(x+lx)*k, (h-(y-ry))*k,
		x*k, (h-(y-ry))*k)
	f.outf("%.5f %.5f %.5f %.5f %.5f %.5f c",
		(x-lx)*k, (h-(y-ry))*k,
		(x-rx)*k, (h-(y-ly))*k,
		(x-rx)*k, (h-y)*k)
	f.outf("%.5f %.5f %.5f %.5f %.5f %.5f c",
		(x-rx)*k, (h-(y+ly))*k,
		(x-lx)*k, (h-(y+ry))*k,
		x*k, (h-(y+ry))*k)
	f.outf("%.5f %.5f %.5f %.5f %.5f %.5f c W %s",
		(x+lx)*k, (h-(y+ry))*k,
		(x+rx)*k, (h-(y+ly))*k,
		(x+rx)*k, (h-y)*k,
		strIf(outline, "S", "n"))
}

// ClipCircle begins a circular clipping operation. The circle is centered at
// (x, y) and has radius r. outline is true to draw a border with the current
// draw color and line width centered on the circle's perimeter. Only the outer
// half of the border will be shown. After calling this method, all rendering
// operations (for example, Image(), LinearGradient(), etc) will be clipped by
// the specified circle. Call ClipEnd() to restore unclipped operations.
//
// The ClipText() example demonstrates this method.
func (f *PDF) ClipCircle(x, y, r float64, outline bool) {
	f.ClipEllipse(x, y, r, r, outline)
}

// ClipPolygon begins a clipping operation within a polygon. The figure is
// defined by a series of vertices specified by points. The x and y fields of
// the points use the units established in New(). The last point in the slice
// will be implicitly joined to the first to close the polygon. outline is true
// to draw a border with the current draw color and line width centered on the
// polygon's perimeter. Only the outer half of the border will be shown. After
// calling this method, all rendering operations (for example, Image(),
// LinearGradient(), etc) will be clipped by the specified polygon. Call
// ClipEnd() to restore unclipped operations.
//
// The ClipText() example demonstrates this method.
func (f *PDF) ClipPolygon(points []PointType, outline bool) {
	f.clipNest++
	var s fmtBuffer
	h := f.h
	k := f.k
	s.printf("q ")
	for j, pt := range points {
		s.printf("%.5f %.5f %s ", pt.X*k, (h-pt.Y)*k, strIf(j == 0, "m", "l"))
	}
	s.printf("h W %s", strIf(outline, "S", "n"))
	f.out(s.String())
}

// ClipEnd ends a clipping operation that was started with a call to
// ClipRect(), ClipRoundedRect(), ClipText(), ClipEllipse(), ClipCircle() or
// ClipPolygon(). Clipping operations can be nested. The document cannot be
// successfully output while a clipping operation is active.
//
// The ClipText() example demonstrates this method.
func (f *PDF) ClipEnd() {
	if f.err == nil {
		if f.clipNest > 0 {
			f.clipNest--
			f.out("Q")
		} else {
			f.err = errClipEndSequence
		}
	}
}

// MoveTo moves the stylus to (x, y) without drawing the path from the
// previous point. Paths must start with a MoveTo to set the original
// stylus location or the result is undefined.
//
// Create a "path" by moving a virtual stylus around the page (with
// MoveTo, LineTo, CurveTo, CurveBezierCubicTo, ArcTo & ClosePath)
// then draw it or  fill it in (with DrawPath). The main advantage of
// using the path drawing routines rather than multiple PDF.Line is
// that PDF creates nice line joins at the angles, rather than just
// overlaying the lines.
func (f *PDF) MoveTo(x, y float64) {
	f.point(x, y)
	f.x, f.y = x, y
}

// LineTo creates a line from the current stylus location to (x, y), which
// becomes the new stylus location. Note that this only creates the line in
// the path; it does not actually draw the line on the page.
//
// The MoveTo() example demonstrates this method.
func (f *PDF) LineTo(x, y float64) {
	f.outf("%.2f %.2f l", x*f.k, (f.h-y)*f.k)
	f.x, f.y = x, y
}

// CurveTo creates a single-segment quadratic Bézier curve. The curve starts at
// the current stylus location and ends at the point (x, y). The control point
// (cx, cy) specifies the curvature. At the start point, the curve is tangent
// to the straight line between the current stylus location and the control
// point. At the end point, the curve is tangent to the straight line between
// the end point and the control point.
//
// The MoveTo() example demonstrates this method.
func (f *PDF) CurveTo(cx, cy, x, y float64) {
	f.outf("%.5f %.5f %.5f %.5f v", cx*f.k, (f.h-cy)*f.k, x*f.k, (f.h-y)*f.k)
	f.x, f.y = x, y
}

// CurveBezierCubicTo creates a single-segment cubic Bézier curve. The curve
// starts at the current stylus location and ends at the point (x, y). The
// control points (cx0, cy0) and (cx1, cy1) specify the curvature. At the
// current stylus, the curve is tangent to the straight line between the
// current stylus location and the control point (cx0, cy0). At the end point,
// the curve is tangent to the straight line between the end point and the
// control point (cx1, cy1).
//
// The MoveTo() example demonstrates this method.
func (f *PDF) CurveBezierCubicTo(cx0, cy0, cx1, cy1, x, y float64) {
	f.curve(cx0, cy0, cx1, cy1, x, y)
	f.x, f.y = x, y
}

// ClosePath creates a line from the current location to the last MoveTo point
// (if not the same) and mark the path as closed so the first and last lines
// join nicely.
//
// The MoveTo() example demonstrates this method.
func (f *PDF) ClosePath() {
	f.outf("h")
}

// DrawPath actually draws the path on the page.
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D".
// Path-painting operators as defined in the PDF specification are also
// allowed: "S" (Stroke the path), "s" (Close and stroke the path),
// "f" (fill the path, using the nonzero winding number), "f*"
// (Fill the path, using the even-odd rule), "B" (Fill and then stroke
// the path, using the nonzero winding number rule), "B*" (Fill and
// then stroke the path, using the even-odd rule), "b" (Close, fill,
// and then stroke the path, using the nonzero winding number rule) and
// "b*" (Close, fill, and then stroke the path, using the even-odd
// rule).
// Drawing uses the current draw color, line width, and cap style
// centered on the
// path. Filling uses the current fill color.
//
// The MoveTo() example demonstrates this method.
func (f *PDF) DrawPath(styleStr string) {
	f.out(fillDrawOp(styleStr))
}

// ArcTo draws an elliptical arc centered at point (x, y). rx and ry specify its
// horizontal and vertical radii. If the start of the arc is not at
// the current position, a connecting line will be drawn.
//
// degRotate specifies the angle that the arc will be rotated. degStart and
// degEnd specify the starting and ending angle of the arc. All angles are
// specified in degrees and measured counter-clockwise from the 3 o'clock
// position.
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled. An empty string will be replaced with "D". Drawing uses
// the current draw color, line width, and cap style centered on the arc's
// path. Filling uses the current fill color.
//
// The MoveTo() example demonstrates this method.
func (f *PDF) ArcTo(x, y, rx, ry, degRotate, degStart, degEnd float64) {
	f.arc(x, y, rx, ry, degRotate, degStart, degEnd, "", true)
}

func (f *PDF) arc(x, y, rx, ry, degRotate, degStart, degEnd float64,
	styleStr string, path bool,
) {
	x *= f.k
	y = (f.h - y) * f.k
	rx *= f.k
	ry *= f.k
	segments := max(int(degEnd-degStart)/60, 2)
	angleStart := degStart * math.Pi / 180
	angleEnd := degEnd * math.Pi / 180
	angleTotal := angleEnd - angleStart
	dt := angleTotal / float64(segments)
	dtm := dt / 3
	if degRotate != 0 {
		a := -degRotate * math.Pi / 180
		f.outf("q %.5f %.5f %.5f %.5f %.5f %.5f cm",
			math.Cos(a), -1*math.Sin(a),
			math.Sin(a), math.Cos(a), x, y)
		x = 0
		y = 0
	}
	t := angleStart
	a0 := x + rx*math.Cos(t)
	b0 := y + ry*math.Sin(t)
	c0 := -rx * math.Sin(t)
	d0 := ry * math.Cos(t)
	sx := a0 / f.k
	sy := f.h - (b0 / f.k)
	if path {
		if f.x != sx || f.y != sy {
			f.LineTo(sx, sy)
		}
	} else {
		f.point(sx, sy)
	}
	for j := 1; j <= segments; j++ {
		t = (float64(j) * dt) + angleStart
		a1 := x + rx*math.Cos(t)
		b1 := y + ry*math.Sin(t)
		c1 := -rx * math.Sin(t)
		d1 := ry * math.Cos(t)
		f.curve((a0+(c0*dtm))/f.k,
			f.h-((b0+(d0*dtm))/f.k),
			(a1-(c1*dtm))/f.k,
			f.h-((b1-(d1*dtm))/f.k),
			a1/f.k,
			f.h-(b1/f.k))
		a0 = a1
		b0 = b1
		c0 = c1
		d0 = d1
		if path {
			f.x = a1 / f.k
			f.y = f.h - (b1 / f.k)
		}
	}
	if !path {
		f.out(fillDrawOp(styleStr))
	}
	if degRotate != 0 {
		f.out("Q")
	}
}
