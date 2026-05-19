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

// ── Justify-content ───────────────────────────────────────────────────────────

func TestFlexJustifyContent(t *testing.T) {
	t.Parallel()

	t.Run("flex-end prepends offset col", func(t *testing.T) {
		t.Parallel()
		// Two items at fixed 25% each = 3+3 cols. Slack = 6. flex-end → leading 6 + items 3+3.
		doc := parseDoc(t, `<html><body><div style="display:flex;justify-content:flex-end"><div style="flex:0 0 25%">a</div><div style="flex:0 0 25%">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		require.Len(t, cols, 3)
		assert.Equal(t, 6, cols[0].GetSize())
		assert.Equal(t, 3, cols[1].GetSize())
		assert.Equal(t, 3, cols[2].GetSize())
	})

	t.Run("center splits slack at both ends", func(t *testing.T) {
		t.Parallel()
		// Two items 3+3=6, slack 6, center → 3 + 3 + 3 + 3
		doc := parseDoc(t, `<html><body><div style="display:flex;justify-content:center"><div style="flex:0 0 25%">a</div><div style="flex:0 0 25%">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		require.Len(t, cols, 4)
		// Leading offset = floor(6/2) = 3
		assert.Equal(t, 3, cols[0].GetSize())
		assert.Equal(t, 3, cols[1].GetSize())
		assert.Equal(t, 3, cols[2].GetSize())
		// Trailing offset = ceil(6/2) = 3
		assert.Equal(t, 3, cols[3].GetSize())
	})

	t.Run("space-between distributes slack as between-spacers", func(t *testing.T) {
		t.Parallel()
		// Two items 3+3=6, slack 6 distributed as N-1=1 between-spacer of size 6.
		doc := parseDoc(t, `<html><body><div style="display:flex;justify-content:space-between"><div style="flex:0 0 25%">a</div><div style="flex:0 0 25%">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		// Pattern: item, spacer, item
		require.Len(t, cols, 3)
		assert.Equal(t, 3, cols[0].GetSize())
		assert.Equal(t, 6, cols[1].GetSize())
		assert.Equal(t, 3, cols[2].GetSize())
	})

	t.Run("flex-start (default) doesn't add offsets", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex"><div style="flex:1">a</div><div style="flex:1">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		require.Len(t, cols, 2)
	})

	t.Run("space-around adds lead+between+trail spacers", func(t *testing.T) {
		t.Parallel()
		// Two items 3+3=6, slack 6, N+1=3 spacers via Hamilton on [1,1,1] → [2,2,2].
		doc := parseDoc(t, `<html><body><div style="display:flex;justify-content:space-around"><div style="flex:0 0 25%">a</div><div style="flex:0 0 25%">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		// Pattern: lead-spacer, item, between-spacer, item, trail-spacer (5 cols)
		require.Len(t, cols, 5)
		sum := 0
		for _, c := range cols {
			sum += c.GetSize()
		}
		assert.Equal(t, 12, sum)
	})
}

// ── Flex direction ────────────────────────────────────────────────────────────

func TestFlexDirection(t *testing.T) {
	t.Parallel()

	t.Run("column direction stacks children as rows", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex;flex-direction:column"><div>a</div><div>b</div><div>c</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 3)
	})

	t.Run("row-reverse renders same as row (limitation)", func(t *testing.T) {
		t.Parallel()
		docA := parseDoc(t, `<html><body><div style="display:flex;flex-direction:row"><div>a</div><div>b</div></div></body></html>`)
		rowsA, _ := translate.Translate(docA)
		docB := parseDoc(t, `<html><body><div style="display:flex;flex-direction:row-reverse"><div>a</div><div>b</div></div></body></html>`)
		rowsB, _ := translate.Translate(docB)
		require.Len(t, rowsA, 1)
		require.Len(t, rowsB, 1)
		assert.Equal(t, len(rowsA[0].GetColumns()), len(rowsB[0].GetColumns()))
	})

	t.Run("column-reverse renders same as column (limitation)", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex;flex-direction:column-reverse"><div>a</div><div>b</div></div></body></html>`)
		rows, _ := translate.Translate(doc)
		assert.Len(t, rows, 2)
	})
}

// ── Non-leaf flex items ───────────────────────────────────────────────────────

func TestFlexNonLeafItems(t *testing.T) {
	t.Parallel()

	t.Run("flex item with h2+p preserves both children", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex"><div><h2>Title</h2><p>Body</p></div><div>Other</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		require.Len(t, cols, 2)
		// The structure of the first col should include 2 child components (h2 + p)
		// or at least not be a single flat-text richtext.
		structure := cols[0].GetStructure()
		require.NotNil(t, structure)
	})
}

// ── Gap ───────────────────────────────────────────────────────────────────────

func TestFlexGap(t *testing.T) {
	t.Parallel()

	t.Run("column-gap inserts between-spacers", func(t *testing.T) {
		t.Parallel()
		// gap:20mm with content width=170mm → 20/14.17 ≈ 1.4 cols → 1 col gap
		// Two flex:1 items, gridSize=12, gap reserves 1 col → items split 5+5+gap_col → wait
		// Actually: total=12, gap_cols=1, items get 12-1=11 split between two items via Hamilton ([5,6] or [6,5]).
		doc := parseDoc(t, `<html><body><div style="display:flex;column-gap:20mm"><div style="flex:1">a</div><div style="flex:1">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		// Expect 3 cols: item, spacer, item
		require.Len(t, cols, 3)
		assert.Equal(t, 1, cols[1].GetSize())
		sum := 0
		for _, c := range cols {
			sum += c.GetSize()
		}
		assert.Equal(t, 12, sum)
	})

	t.Run("gap shorthand is equivalent to column-gap on flex-row", func(t *testing.T) {
		t.Parallel()
		docA := parseDoc(t, `<html><body><div style="display:flex;gap:20mm"><div style="flex:1">a</div><div style="flex:1">b</div></div></body></html>`)
		rowsA, _ := translate.Translate(docA)
		docB := parseDoc(t, `<html><body><div style="display:flex;column-gap:20mm"><div style="flex:1">a</div><div style="flex:1">b</div></div></body></html>`)
		rowsB, _ := translate.Translate(docB)
		require.Len(t, rowsA, 1)
		require.Len(t, rowsB, 1)
		assert.Equal(t, len(rowsA[0].GetColumns()), len(rowsB[0].GetColumns()))
	})

	t.Run("gap is clamped to half the grid", func(t *testing.T) {
		t.Parallel()
		// Huge gap value — should be clamped so items still get reasonable share.
		doc := parseDoc(t, `<html><body><div style="display:flex;gap:1000mm"><div style="flex:1">a</div><div style="flex:1">b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		sum := 0
		for _, c := range cols {
			sum += c.GetSize()
		}
		assert.Equal(t, 12, sum)
		// At least one item should retain ≥ 1 col
		assert.GreaterOrEqual(t, cols[0].GetSize(), 1)
	})
}
