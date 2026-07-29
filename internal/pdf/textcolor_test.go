package pdf

import "testing"

// TestSetTextColor_CachesOnlyIdenticalColors guards the allocation shortcut in
// setTextColor: reusing the current value must never skip a real change, and
// the very first assignment must still produce an operator string even when the
// requested color is black (the zero value of the cached components).
func TestSetTextColor_CachesOnlyIdenticalColors(t *testing.T) {
	t.Parallel()

	t.Run("black on a fresh document still emits an operator", func(t *testing.T) {
		t.Parallel()
		f := &PDF{}
		f.setTextColor(0, 0, 0)

		if f.color.text.str == "" {
			t.Fatal("text color operator is empty after the first assignment")
		}
		if got := f.color.text.str; got != "0.000 g" {
			t.Errorf("operator = %q, want %q", got, "0.000 g")
		}
	})

	t.Run("changing color updates the operator", func(t *testing.T) {
		t.Parallel()
		f := &PDF{}
		f.setTextColor(0, 0, 0)
		first := f.color.text.str

		f.setTextColor(255, 0, 0)
		if f.color.text.str == first {
			t.Fatal("operator did not change after setting a different color")
		}
		if r, g, b := f.GetTextColor(); r != 255 || g != 0 || b != 0 {
			t.Errorf("components = (%d,%d,%d), want (255,0,0)", r, g, b)
		}
	})

	t.Run("repeating a color keeps the same operator", func(t *testing.T) {
		t.Parallel()
		f := &PDF{}
		f.setTextColor(12, 34, 56)
		want := f.color.text.str

		f.setTextColor(12, 34, 56)
		if got := f.color.text.str; got != want {
			t.Errorf("operator = %q, want %q", got, want)
		}
	})

	t.Run("colorFlag tracks a fill change even when text is unchanged", func(t *testing.T) {
		t.Parallel()
		f := &PDF{}
		f.setTextColor(10, 20, 30)
		f.setFillColor(10, 20, 30)
		f.setTextColor(10, 20, 30)
		if f.colorFlag {
			t.Error("colorFlag should be false when fill and text match")
		}

		f.setFillColor(200, 100, 50)
		f.setTextColor(10, 20, 30)
		if !f.colorFlag {
			t.Error("colorFlag should be true after the fill color diverged")
		}
	})
}
