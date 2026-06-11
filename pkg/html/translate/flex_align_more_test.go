package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
)

func TestNormalizeCrossAxisAlign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "start maps to flex-start", value: "start", want: flexAlignStart},
		{name: "self-start maps to flex-start", value: "self-start", want: flexAlignStart},
		{name: "end maps to flex-end", value: "end", want: flexAlignEnd},
		{name: "self-end maps to flex-end", value: "self-end", want: flexAlignEnd},
		{name: "center passes through lowercased", value: " Center ", want: "center"},
		{name: "flex-end passes through", value: "flex-end", want: flexAlignEnd},
		{name: "empty stays empty", value: "", want: ""},
		{name: "stretch passes through", value: "stretch", want: "stretch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, normalizeCrossAxisAlign(tt.value))
		})
	}
}

func TestEffectiveCrossAxisAlign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		container *css.ComputedStyle
		item      *css.ComputedStyle
		want      string
	}{
		{
			name:      "align-self wins over container",
			container: &css.ComputedStyle{AlignItems: "center"},
			item:      &css.ComputedStyle{AlignSelf: "end"},
			want:      flexAlignEnd,
		},
		{
			name:      "align-self auto falls back to container align-items",
			container: &css.ComputedStyle{AlignItems: "center"},
			item:      &css.ComputedStyle{AlignSelf: "auto"},
			want:      "center",
		},
		{
			name:      "empty item falls back to container",
			container: &css.ComputedStyle{AlignItems: "flex-end"},
			item:      &css.ComputedStyle{},
			want:      flexAlignEnd,
		},
		{name: "both nil yields empty", container: nil, item: nil, want: ""},
		{
			name:      "nil container keeps item value",
			container: nil,
			item:      &css.ComputedStyle{AlignSelf: "self-start"},
			want:      flexAlignStart,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, effectiveCrossAxisAlign(tt.container, tt.item))
		})
	}
}

func TestFlexItemCrossAxisBox(t *testing.T) {
	t.Parallel()

	t.Run("alignable value wraps child in crossAxisBox", func(t *testing.T) {
		t.Parallel()
		child := &fixedHeightComponent{height: 1}
		got := flexItemCrossAxisBox(child, &css.ComputedStyle{AlignItems: "center"}, nil)
		box, ok := got.(*crossAxisBox)
		assert.True(t, ok, "expected a crossAxisBox wrapper")
		assert.Equal(t, "center", box.align)
	})

	t.Run("stretch returns child unwrapped", func(t *testing.T) {
		t.Parallel()
		child := &fixedHeightComponent{height: 1}
		got := flexItemCrossAxisBox(child, &css.ComputedStyle{AlignItems: "stretch"}, nil)
		assert.Equal(t, child, got)
	})

	t.Run("nil styles return child unwrapped", func(t *testing.T) {
		t.Parallel()
		child := &fixedHeightComponent{height: 1}
		assert.Equal(t, child, flexItemCrossAxisBox(child, nil, nil))
	})
}

func TestCrossAxisBox_NilChild(t *testing.T) {
	t.Parallel()
	box := &crossAxisBox{align: flexAlignCenter}

	assert.Equal(t, 0.0, box.GetHeight(nil, &entity.Cell{}))
	box.Render(nil, &entity.Cell{Width: 10, Height: 10}) // must not panic
	box.SetConfig(&entity.Config{})                      // must not panic
}

func TestCrossAxisBox_GetHeightDelegatesToChild(t *testing.T) {
	t.Parallel()
	box := &crossAxisBox{child: &fixedHeightComponent{height: 6}, align: flexAlignCenter}
	assert.Equal(t, 6.0, box.GetHeight(nil, &entity.Cell{Width: 10, Height: 10}))
}

func TestCrossAxisBox_SetConfigWithChild(t *testing.T) {
	t.Parallel()
	box := &crossAxisBox{child: &fixedHeightComponent{}, align: flexAlignStart}
	box.SetConfig(&entity.Config{MaxGridSize: 12}) // fixedHeightComponent ignores it; line coverage only
}

func TestCrossAxisBox_GetStructure_IncludesAlign(t *testing.T) {
	t.Parallel()
	box := &crossAxisBox{child: &fixedHeightComponent{}, align: flexAlignEnd}
	str := box.GetStructure()
	assert.Equal(t, "cross_axis_box", str.GetData().Type)
	assert.Equal(t, flexAlignEnd, str.GetData().Details["align"])
	assert.Len(t, str.GetNexts(), 1)
}

func TestCrossAxisBox_Render_TallerChildKeepsCell(t *testing.T) {
	t.Parallel()
	// Child taller than the cell: center/end branches must not move the child.
	child := &fixedHeightComponent{height: 30}
	box := &crossAxisBox{child: child, align: flexAlignCenter}

	box.Render(nil, &entity.Cell{X: 1, Y: 2, Width: 10, Height: 10})

	assert.Equal(t, 2.0, child.renderedCell.Y)
	assert.Equal(t, 10.0, child.renderedCell.Height)
}
