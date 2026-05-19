package translate_test

import (
	"testing"

	"github.com/johnfercher/maroto/v2/pkg/html/translate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Hamilton LRM quantizer ────────────────────────────────────────────────────

func TestHamilton(t *testing.T) {
	t.Parallel()

	t.Run("equal weights 3 items", func(t *testing.T) {
		t.Parallel()
		got := translate.Hamilton([]float64{1, 1, 1}, 12)
		assert.Equal(t, []int{4, 4, 4}, got)
	})

	t.Run("proportional weights 1:2:1", func(t *testing.T) {
		t.Parallel()
		got := translate.Hamilton([]float64{1, 2, 1}, 12)
		assert.Equal(t, []int{3, 6, 3}, got)
	})

	t.Run("5 equal items distribute remainder correctly", func(t *testing.T) {
		t.Parallel()
		got := translate.Hamilton([]float64{1, 1, 1, 1, 1}, 12)
		sum := 0
		for _, v := range got {
			sum += v
		}
		assert.Equal(t, 12, sum)
		// Each item gets 2 or 3; first 2 get 3, last 3 get 2
		assert.GreaterOrEqual(t, got[0], 2)
		assert.LessOrEqual(t, got[len(got)-1], 3)
	})

	t.Run("1:3 split", func(t *testing.T) {
		t.Parallel()
		got := translate.Hamilton([]float64{1, 3}, 12)
		assert.Equal(t, []int{3, 9}, got)
	})

	t.Run("single item gets all columns", func(t *testing.T) {
		t.Parallel()
		got := translate.Hamilton([]float64{1}, 12)
		assert.Equal(t, []int{12}, got)
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		t.Parallel()
		got := translate.Hamilton([]float64{}, 12)
		assert.Equal(t, []int{}, got)
	})

	t.Run("all-zero weights get equal split", func(t *testing.T) {
		t.Parallel()
		got := translate.Hamilton([]float64{0, 0, 0}, 12)
		sum := 0
		for _, v := range got {
			sum += v
		}
		assert.Equal(t, 12, sum)
		for _, v := range got {
			assert.Greater(t, v, 0)
		}
	})

	t.Run("custom grid size 8", func(t *testing.T) {
		t.Parallel()
		got := translate.Hamilton([]float64{1, 1}, 8)
		assert.Equal(t, []int{4, 4}, got)
	})
}

// ── Flex row translator ───────────────────────────────────────────────────────

func TestFlexRow_ColCount(t *testing.T) {
	t.Parallel()

	t.Run("two equal flex items produce 2 cols", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex"><div>a</div><div>b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Len(t, rows[0].GetColumns(), 2)
	})

	t.Run("whitespace between flex children is ignored", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, "<html><body><div style=\"display:flex\">\n  <div>a</div>\n  <div>b</div>\n</div></body></html>")
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Len(t, rows[0].GetColumns(), 2)
	})

	t.Run("flex:1 and flex:2 produce sizes 4 and 8", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex"><div style="flex:1">a</div><div style="flex:2">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		require.Len(t, cols, 2)
		assert.Equal(t, 4, cols[0].GetSize())
		assert.Equal(t, 8, cols[1].GetSize())
	})

	t.Run("flex-basis 25% weight resolves correctly", func(t *testing.T) {
		t.Parallel()
		// flex:0 0 25% → weight 3 (25% of 12); flex:1 → weight 9 (remaining)
		doc := parseDoc(t, `<html><body><div style="display:flex"><div style="flex:0 0 25%">a</div><div style="flex:1">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		require.Len(t, cols, 2)
		assert.Equal(t, 3, cols[0].GetSize())
		assert.Equal(t, 9, cols[1].GetSize())
	})

	t.Run("empty flex container produces 0 rows", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex"></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 0)
	})

	t.Run("display:inline-flex also produces a flex row", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:inline-flex"><div>a</div><div>b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Len(t, rows[0].GetColumns(), 2)
	})
}
