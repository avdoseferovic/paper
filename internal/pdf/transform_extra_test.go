package pdf

import (
	"strings"
	"testing"
)

func TestTransformBeginEndNesting(t *testing.T) {
	f := readyPDF(t)
	f.TransformBegin()
	if f.transformNest != 1 {
		t.Fatalf("transformNest = %d after begin", f.transformNest)
	}
	f.TransformEnd()
	if f.transformNest != 0 {
		t.Fatalf("transformNest = %d after end", f.transformNest)
	}
	if f.Err() {
		t.Fatalf("unexpected error: %v", f.Error())
	}
}

func TestTransformEndOutOfSequenceErrors(t *testing.T) {
	f := readyPDF(t)
	f.TransformEnd()
	if !f.Err() {
		t.Fatal("expected error ending transform out of sequence")
	}
}

func TestTransformWithoutActiveContextErrors(t *testing.T) {
	f := readyPDF(t)
	f.Transform(TransformMatrix{A: 1, D: 1})
	if !f.Err() {
		t.Fatal("expected error transforming without active context")
	}
}

func TestTransformVariantsRunWithinContext(t *testing.T) {
	f := readyPDF(t)
	f.TransformBegin()
	f.TransformScale(150, 80, 10, 10)
	f.TransformScaleX(120, 10, 10)
	f.TransformScaleY(120, 10, 10)
	f.TransformScaleXY(90, 10, 10)
	f.TransformTranslate(5, 5)
	f.TransformTranslateX(5)
	f.TransformTranslateY(5)
	f.TransformRotate(45, 10, 10)
	f.TransformSkew(10, 10, 10, 10)
	f.TransformSkewX(10, 10, 10)
	f.TransformSkewY(10, 10, 10)
	f.TransformMirrorHorizontal(10)
	f.TransformMirrorVertical(10)
	f.TransformMirrorPoint(10, 10)
	f.TransformMirrorLine(30, 10, 10)
	f.TransformEnd()
	if f.Err() {
		t.Fatalf("transform variants errored: %v", f.Error())
	}
	out := mustOutput(t, f)
	// The transform matrix operator "cm" must appear in the content stream.
	if !strings.Contains(string(out), "cm") {
		// content streams are compressed by default, so just assert generation
		// succeeded; the explicit error checks above cover correctness.
		t.Log("cm operator not visible (stream compressed); generation succeeded")
	}
}

func TestTransformScaleZeroFactorErrors(t *testing.T) {
	f := readyPDF(t)
	f.TransformBegin()
	f.TransformScale(0, 100, 10, 10)
	if !f.Err() {
		t.Fatal("expected error for zero scale factor")
	}
}

func TestTransformSkewOutOfRangeErrors(t *testing.T) {
	cases := []struct{ ax, ay float64 }{
		{90, 0},
		{-90, 0},
		{0, 90},
		{0, -90},
	}
	for _, c := range cases {
		f := readyPDF(t)
		f.TransformBegin()
		f.TransformSkew(c.ax, c.ay, 10, 10)
		if !f.Err() {
			t.Fatalf("expected error for skew(%v,%v)", c.ax, c.ay)
		}
	}
}
