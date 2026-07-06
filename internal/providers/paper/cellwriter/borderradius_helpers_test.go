package cellwriter

import (
	"math"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/props"
)

type accentLineCall struct {
	x1 float64
	y1 float64
	x2 float64
	y2 float64
}

type accentPDFRecorder struct {
	lineWidth float64
	drawColor [3]int
	joinStyle string
	lines     []accentLineCall
	joins     []string
}

func (r *accentPDFRecorder) ClosePath()                                  {}
func (r *accentPDFRecorder) CurveBezierCubicTo(_, _, _, _, _, _ float64) {}
func (r *accentPDFRecorder) DrawPath(_ string)                           {}
func (r *accentPDFRecorder) Ellipse(_, _, _, _, _ float64, _ string)     {}
func (r *accentPDFRecorder) GetDrawColor() (int, int, int) {
	return r.drawColor[0], r.drawColor[1], r.drawColor[2]
}
func (r *accentPDFRecorder) GetFillColor() (int, int, int) { return 0, 0, 0 }
func (r *accentPDFRecorder) GetLineJoinStyle() string {
	if r.joinStyle == "" {
		return "miter"
	}
	return r.joinStyle
}
func (r *accentPDFRecorder) GetLineWidth() float64     { return r.lineWidth }
func (r *accentPDFRecorder) GetXY() (float64, float64) { return 0, 0 }
func (r *accentPDFRecorder) Line(x1, y1, x2, y2 float64) {
	r.lines = append(r.lines, accentLineCall{x1: x1, y1: y1, x2: x2, y2: y2})
}
func (r *accentPDFRecorder) LineTo(_, _ float64)          {}
func (r *accentPDFRecorder) MoveTo(_, _ float64)          {}
func (r *accentPDFRecorder) SetAlpha(_ float64, _ string) {}
func (r *accentPDFRecorder) SetDrawColor(red, green, blue int) {
	r.drawColor = [3]int{red, green, blue}
}
func (r *accentPDFRecorder) SetFillColor(_, _, _ int) {}
func (r *accentPDFRecorder) SetLineJoinStyle(style string) {
	r.joinStyle = style
	r.joins = append(r.joins, style)
}

func (r *accentPDFRecorder) SetLineWidth(width float64) {
	r.lineWidth = width
}

func TestPickStrokeColor(t *testing.T) {
	t.Parallel()

	t.Run("uniform border color wins", func(t *testing.T) {
		t.Parallel()
		uniform := &props.Color{Red: 1}
		prop := &props.Cell{
			BorderColor:    uniform,
			BorderTopColor: &props.Color{Red: 2},
		}
		assert.Equal(t, uniform, pickStrokeColor(prop))
	})

	t.Run("first non-nil per-side color is picked in top-right-bottom-left order", func(t *testing.T) {
		t.Parallel()
		top := &props.Color{Red: 1}
		right := &props.Color{Red: 2}
		bottom := &props.Color{Red: 3}
		left := &props.Color{Red: 4}

		assert.Equal(t, top, pickStrokeColor(&props.Cell{BorderTopColor: top, BorderLeftColor: left}))
		assert.Equal(t, right, pickStrokeColor(&props.Cell{BorderRightColor: right, BorderBottomColor: bottom}))
		assert.Equal(t, bottom, pickStrokeColor(&props.Cell{BorderBottomColor: bottom, BorderLeftColor: left}))
		assert.Equal(t, left, pickStrokeColor(&props.Cell{BorderLeftColor: left}))
	})

	t.Run("no colors returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, pickStrokeColor(&props.Cell{}))
	})
}

func TestDrawStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fill, stroke bool
		expected     string
	}{
		{"fill and stroke", true, true, "DF"},
		{"fill only", true, false, "F"},
		{"stroke only", false, true, "D"},
		{"neither", false, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, drawStyle(tt.fill, tt.stroke))
		})
	}
}

func TestClampRadius(t *testing.T) {
	t.Parallel()

	t.Run("negative radius clamps to zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, clampRadius(-1, 10, 10))
	})

	t.Run("radius above half the smaller dimension clamps", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2.0, clampRadius(8, 10, 4))
		assert.Equal(t, 3.0, clampRadius(8, 6, 20))
	})

	t.Run("radius within bounds passes through", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1.5, clampRadius(1.5, 10, 10))
	})
}

func TestIsCompactFullRoundCell(t *testing.T) {
	t.Parallel()

	assert.True(t, isCompactFullRoundCell(10, 10, 5, 5, 5, 5))
	assert.True(t, isCompactFullRoundCell(11, 10, 5, 5, 5, 5))
	assert.False(t, isCompactFullRoundCell(20, 10, 5, 5, 5, 5))
	assert.False(t, isCompactFullRoundCell(11, 10, 4, 5, 5, 5))
}

func TestBaseBorderThickness(t *testing.T) {
	t.Parallel()

	t.Run("uniform thickness wins", func(t *testing.T) {
		t.Parallel()
		prop := &props.Cell{BorderThickness: 2, BorderTopThickness: 10}
		assert.Equal(t, 2.0, baseBorderThickness(prop))
	})

	t.Run("uses the thinnest non-zero per-side thickness (base for the rounded stroke)", func(t *testing.T) {
		t.Parallel()
		// e.g. thin all-side border (0.5) + a thick left accent (5) → base is 0.5;
		// the 5pt left side is overlaid separately as an accent.
		prop := &props.Cell{
			BorderTopThickness: 0.5, BorderRightThickness: 0.5,
			BorderBottomThickness: 0.5, BorderLeftThickness: 5,
		}
		assert.Equal(t, 0.5, baseBorderThickness(prop))
	})

	t.Run("no thickness set returns zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, baseBorderThickness(&props.Cell{}))
	})
}

func TestDrawAccentSides_UsesPartialCornerInsetForVerticalAccents(t *testing.T) {
	t.Parallel()

	rec := &accentPDFRecorder{lineWidth: 0.2, drawColor: [3]int{1, 2, 3}}
	styler := &borderRadiusStyler{}
	prop := &props.Cell{
		BorderLeftThickness: 3,
		BorderLeftColor:     &props.Color{Red: 9, Green: 8, Blue: 7},
	}

	styler.drawAccentSides(rec, prop, 10, 20, 100, 50, 8, 8, 8, 6, 0.5)

	assert.Len(t, rec.lines, 1)
	line := rec.lines[0]
	assertClose(t, 11.5, line.x1)
	assertClose(t, 20+8*accentCornerInsetFactor, line.y1)
	assertClose(t, 11.5, line.x2)
	assertClose(t, 20+50-6*accentCornerInsetFactor, line.y2)
	assert.Equal(t, 0.2, rec.lineWidth)
	assert.Equal(t, [3]int{1, 2, 3}, rec.drawColor)
}

func TestUseRoundJoinForStrokeRestoresPreviousJoin(t *testing.T) {
	t.Parallel()

	rec := &accentPDFRecorder{joinStyle: "bevel"}
	styler := &borderRadiusStyler{stylerTemplate: stylerTemplate{fpdf: rec}}

	restore := styler.useRoundJoinForStroke(true)
	assert.Equal(t, "round", rec.GetLineJoinStyle())

	restore()
	assert.Equal(t, "bevel", rec.GetLineJoinStyle())
	assert.Equal(t, []string{"round", "bevel"}, rec.joins)
}

func TestUseRoundJoinForStrokeNoopsWithoutStroke(t *testing.T) {
	t.Parallel()

	rec := &accentPDFRecorder{joinStyle: "bevel"}
	styler := &borderRadiusStyler{stylerTemplate: stylerTemplate{fpdf: rec}}

	restore := styler.useRoundJoinForStroke(false)
	restore()

	assert.Equal(t, "bevel", rec.GetLineJoinStyle())
	assert.Len(t, rec.joins, 0)
}

func assertClose(t *testing.T, want, got float64) {
	t.Helper()
	if math.Abs(want-got) > 0.000001 {
		t.Fatalf("got %f, want %f", got, want)
	}
}
