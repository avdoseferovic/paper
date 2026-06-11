package pdf

import "testing"

func TestColorSettersAndGetters(t *testing.T) {
	f := readyPDF(t)
	f.SetDrawColor(10, 20, 30)
	if r, g, b := f.GetDrawColor(); r != 10 || g != 20 || b != 30 {
		t.Errorf("GetDrawColor = %d,%d,%d", r, g, b)
	}
	f.SetFillColor(40, 50, 60)
	if r, g, b := f.GetFillColor(); r != 40 || g != 50 || b != 60 {
		t.Errorf("GetFillColor = %d,%d,%d", r, g, b)
	}
	f.SetTextColor(70, 80, 90)
	if r, g, b := f.GetTextColor(); r != 70 || g != 80 || b != 90 {
		t.Errorf("GetTextColor = %d,%d,%d", r, g, b)
	}
}

func TestAlphaSetAndGet(t *testing.T) {
	f := readyPDF(t)
	f.SetAlpha(0.5, "Multiply")
	if f.Err() {
		t.Fatalf("SetAlpha errored: %v", f.Error())
	}
	a, mode := f.GetAlpha()
	if a != 0.5 || mode != "Multiply" {
		t.Fatalf("GetAlpha = %v, %q", a, mode)
	}
}

func TestSetAlphaOutOfRangeErrors(t *testing.T) {
	f := readyPDF(t)
	f.SetAlpha(1.5, "Normal")
	if !f.Err() {
		t.Fatal("expected error for alpha > 1.0")
	}
}

func TestLineWidthSetAndGet(t *testing.T) {
	f := readyPDF(t)
	f.SetLineWidth(2.5)
	if got := f.GetLineWidth(); got != 2.5 {
		t.Fatalf("GetLineWidth = %v", got)
	}
}

func TestLineCapAndJoinStyles(t *testing.T) {
	f := readyPDF(t)
	for _, s := range []string{"butt", "round", "square", "unknown"} {
		f.SetLineCapStyle(s)
	}
	for _, s := range []string{"miter", "round", "bevel", "unknown"} {
		f.SetLineJoinStyle(s)
	}
	if f.Err() {
		t.Fatalf("style setters errored: %v", f.Error())
	}
}

func TestDashPattern(t *testing.T) {
	f := readyPDF(t)
	f.SetDashPattern([]float64{2, 1}, 0)
	f.Line(10, 10, 50, 50)
	// Reset to solid.
	f.SetDashPattern([]float64{}, 0)
	if f.Err() {
		t.Fatalf("dash pattern errored: %v", f.Error())
	}
}

func TestPrimitiveShapes(t *testing.T) {
	f := readyPDF(t)
	f.Line(10, 10, 100, 10)
	f.Rect(10, 20, 50, 30, "D")
	f.Rect(10, 60, 50, 30, "F")
	f.Rect(10, 100, 50, 30, "FD")
	f.Circle(40, 160, 20, "D")
	f.Ellipse(120, 160, 30, 15, 0, "FD")
	f.Curve(10, 200, 30, 180, 60, 200, "D")
	f.CurveCubic(10, 230, 30, 210, 60, 230, 45, 250, "D")
	f.Arc(120, 230, 30, 20, 0, 0, 180, "D")
	if f.Err() {
		t.Fatalf("shapes errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestPolygonAndBeziergon(t *testing.T) {
	f := readyPDF(t)
	pts := []PointType{{X: 10, Y: 10}, {X: 60, Y: 10}, {X: 35, Y: 50}}
	f.Polygon(pts, "DF")
	bez := []PointType{
		{X: 10, Y: 100}, {X: 30, Y: 80}, {X: 60, Y: 120}, {X: 90, Y: 100},
	}
	f.Beziergon(bez, "D")
	if f.Err() {
		t.Fatalf("polygon/beziergon errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestRoundedRectVariants(t *testing.T) {
	f := readyPDF(t)
	f.RoundedRect(10, 10, 60, 40, 5, "1234", "D")
	f.RoundedRectExt(10, 60, 60, 40, 3, 6, 9, 12, "FD")
	if f.Err() {
		t.Fatalf("rounded rect errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestPathDrawingAPI(t *testing.T) {
	f := readyPDF(t)
	f.MoveTo(10, 10)
	f.LineTo(60, 10)
	f.CurveTo(80, 30, 60, 50)
	f.CurveBezierCubicTo(40, 60, 20, 60, 10, 50)
	f.ArcTo(40, 80, 20, 10, 0, 0, 180)
	f.ClosePath()
	f.DrawPath("D")
	if f.Err() {
		t.Fatalf("path API errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestGradients(t *testing.T) {
	f := readyPDF(t)
	f.LinearGradient(10, 10, 80, 40, 255, 0, 0, 0, 0, 255, 0, 0, 1, 0)
	f.RadialGradient(10, 60, 80, 40, 255, 255, 0, 0, 0, 255, 0.5, 0.5, 0.5, 0.5, 1)
	if f.Err() {
		t.Fatalf("gradients errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestClippingRegions(t *testing.T) {
	f := readyPDF(t)

	f.ClipRect(10, 10, 50, 50, true)
	f.Rect(10, 10, 50, 50, "F")
	f.ClipEnd()

	f.ClipCircle(120, 40, 25, false)
	f.Circle(120, 40, 25, "F")
	f.ClipEnd()

	f.ClipEllipse(60, 120, 40, 20, true)
	f.Rect(20, 100, 80, 40, "F")
	f.ClipEnd()

	f.ClipRoundedRect(10, 160, 80, 40, 6, true)
	f.Rect(10, 160, 80, 40, "F")
	f.ClipEnd()

	f.ClipPolygon([]PointType{{X: 10, Y: 220}, {X: 90, Y: 220}, {X: 50, Y: 260}}, true)
	f.Rect(10, 220, 80, 40, "F")
	f.ClipEnd()

	f.ClipText(10, 290, "clipped", true)
	f.Rect(10, 280, 80, 20, "F")
	f.ClipEnd()

	if f.Err() {
		t.Fatalf("clipping errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestClipEndOutOfSequenceErrors(t *testing.T) {
	f := readyPDF(t)
	f.ClipEnd()
	if !f.Err() {
		t.Fatal("expected error ending clip out of sequence")
	}
}
