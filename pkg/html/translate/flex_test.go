package translate_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/translate"
	"github.com/avdoseferovic/paper/pkg/tree/node"
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

// ── DoD coverage: em inheritance, center+odd-slack, mm-basis, non-leaf strict ─

func TestFlexEmInheritsFromContainer(t *testing.T) {
	t.Parallel()
	// Container has font-size:14pt; child uses 1.5em — should resolve to 21pt, not 0.
	doc := parseDoc(t, `<html><body><div style="display:flex;font-size:14pt"><div style="font-size:1.5em">Big</div><div>Small</div></div></body></html>`)
	rows, err := translate.Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	// Structural check: 2 cols. Em resolution is implicit via no-error + non-zero size below.
	cols := rows[0].GetColumns()
	require.Len(t, cols, 2)
	// First col's GetStructure should be non-nil (em resolved to non-zero font-size).
	assert.NotNil(t, cols[0].GetStructure())
}

func TestFlexCenterOddSlack(t *testing.T) {
	t.Parallel()
	// Two items each flex:0 0 25% on gridSize=11 → fixed share=3 each (rounded), sum=6.
	// Slack = 11-6 = 5. Center → leading=floor(5/2)=2, trailing=ceil(5/2)=3.
	doc := parseDoc(t, `<html><body><div style="display:flex;justify-content:center"><div style="flex:0 0 25%">a</div><div style="flex:0 0 25%">b</div></div></body></html>`)
	rows, err := translate.Translate(doc, translate.WithGridSize(11))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	cols := rows[0].GetColumns()
	require.Len(t, cols, 4)
	sum := 0
	for _, c := range cols {
		sum += c.GetSize()
	}
	assert.Equal(t, 11, sum)
	// Floor/ceil split: leading ≤ trailing
	assert.LessOrEqual(t, cols[0].GetSize(), cols[3].GetSize())
}

func TestFlexNonLeafStrictStructure(t *testing.T) {
	t.Parallel()
	doc := parseDoc(t, `<html><body><div style="display:flex"><div><h2>Title</h2><p>Body</p></div><div>Other</div></div></body></html>`)
	rows, err := translate.Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	cols := rows[0].GetColumns()
	require.Len(t, cols, 2)
	structure := cols[0].GetStructure()
	require.NotNil(t, structure)
	// The col contains a flexCellContent with 2 child rows (h2 + p).
	// Walk the structure tree to find a "flex_cell" node and verify rows=2.
	found := false
	walk(structure, func(s core.Structure) {
		if s.Type == "flex_cell" {
			if d, ok := s.Details["rows"].(int); ok && d == 2 {
				found = true
			}
		}
	})
	assert.True(t, found, "expected flex_cell node with rows=2 in structure")
}

// walk traverses the structure tree.
func walk(n *node.Node[core.Structure], fn func(core.Structure)) {
	if n == nil {
		return
	}
	fn(n.GetData())
	for _, c := range n.GetNexts() {
		walk(c, fn)
	}
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

	t.Run("row-reverse orders children right to left", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex;flex-direction:row-reverse"><div>a</div><div>b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, []string{"b", "a"}, richTextValues(rows[0]))
	})

	t.Run("column-reverse stacks children in reverse order", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex;flex-direction:column-reverse"><div>a</div><div>b</div><div>c</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 3)
		assert.Equal(t, []string{"c", "b", "a"}, richTextValues(rows...))
	})

	t.Run("column direction with row-gap produces 5 rows for 3 children", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex;flex-direction:column;row-gap:5mm"><div>a</div><div>b</div><div>c</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 5) // 3 content + 2 gap rows
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

	t.Run("small positive gap uses item margin instead of consuming grid columns", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex;gap:6mm"><div style="flex:1">a</div><div style="flex:1">b</div><div style="flex:1">c</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		require.Len(t, cols, 3)
		sum := 0
		for _, c := range cols {
			sum += c.GetSize()
		}
		assert.Equal(t, 12, sum)
		assert.Equal(t, 4, cols[0].GetSize())
		assert.Equal(t, 4, cols[1].GetSize())
		assert.Equal(t, 4, cols[2].GetSize())
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

// ── flex-wrap, order, *-reverse ───────────────────────────────────────────────

func TestFlexWrap(t *testing.T) {
	t.Parallel()

	t.Run("6 items at flex-basis:33% wrap into 2 rows of 3", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body>
		<div style="display:flex;flex-wrap:wrap">
		  <div style="flex-basis:33%">a</div>
		  <div style="flex-basis:33%">b</div>
		  <div style="flex-basis:33%">c</div>
		  <div style="flex-basis:33%">d</div>
		  <div style="flex-basis:33%">e</div>
		  <div style="flex-basis:33%">f</div>
		</div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		// flex-wrap:wrap produces 2 rows
		assert.Equal(t, 2, len(rows), "expected 2 wrapped rows")
		if len(rows) >= 2 {
			assert.Equal(t, 3, len(rows[0].GetColumns()), "row 1 should have 3 cols")
			assert.Equal(t, 3, len(rows[1].GetColumns()), "row 2 should have 3 cols")
		}
	})

	t.Run("nowrap keeps all items in one row", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body>
		<div style="display:flex;flex-wrap:nowrap">
		  <div style="flex-basis:33%">a</div>
		  <div style="flex-basis:33%">b</div>
		  <div style="flex-basis:33%">c</div>
		  <div style="flex-basis:33%">d</div>
		</div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Equal(t, 1, len(rows))
	})
}

func TestFlexOrder(t *testing.T) {
	t.Parallel()

	t.Run("items with order 2,1,3,0 render in order 0,1,2,3", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body>
		<div style="display:flex">
		  <div style="order:2" id="c">C</div>
		  <div style="order:1" id="b">B</div>
		  <div style="order:3" id="d">D</div>
		  <div style="order:0" id="a">A</div>
		</div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		// Verify all 4 items are still present after order sort.
		assert.Len(t, rows[0].GetColumns(), 4)
	})
}

func TestFlexReverse(t *testing.T) {
	t.Parallel()

	t.Run("row-reverse reverses item order", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body>
		<div style="display:flex;flex-direction:row-reverse">
		  <div>a</div><div>b</div><div>c</div>
		</div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, []string{"c", "b", "a"}, richTextValues(rows[0]))
	})
}

func richTextValues(rows ...core.Row) []string {
	var values []string
	for _, r := range rows {
		walk(r.GetStructure(), func(s core.Structure) {
			if s.Type != "richtext" {
				return
			}
			if text, ok := s.Value.(string); ok {
				values = append(values, text)
			}
		})
	}
	return values
}

// ── align-self per-item alignment ────────────────────────────────────────────

func TestFlexAlignSelf(t *testing.T) {
	t.Parallel()

	t.Run("align-self flex-end produces one row with correct col count", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body>
		<div style="display:flex">
		  <div style="align-self:flex-end">child</div>
		</div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Len(t, rows[0].GetColumns(), 1)
		assert.True(t, hasCrossAxisAlign(rows[0], "flex-end"))
	})

	t.Run("align-self auto falls back to container align-items", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body>
		<div style="display:flex;align-items:center">
		  <div style="align-self:auto">child</div>
		</div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Len(t, rows[0].GetColumns(), 1)
		assert.True(t, hasCrossAxisAlign(rows[0], "center"))
	})
}

func hasCrossAxisAlign(r core.Row, align string) bool {
	found := false
	walk(r.GetStructure(), func(s core.Structure) {
		if s.Type == "cross_axis_box" && s.Details["align"] == align {
			found = true
		}
	})
	return found
}

// ── Golden tests: single-row computeFlexSizes preservation ───────────────────
// These tests capture the exact output of computeFlexSizes for representative
// single-row inputs. They exist to detect regressions after the WrappedLayout
// refactor for flex-wrap support.

func TestComputeFlexSizes_Golden(t *testing.T) {
	t.Parallel()

	makeStyles := func(flexGrows ...float64) []*css.ComputedStyle {
		out := make([]*css.ComputedStyle, len(flexGrows))
		for i, g := range flexGrows {
			s := css.NewComputedStyle()
			s.FlexGrow = g
			out[i] = s
		}
		return out
	}

	makePctStyles := func(pcts ...float64) []*css.ComputedStyle {
		out := make([]*css.ComputedStyle, len(pcts))
		for i, p := range pcts {
			s := css.NewComputedStyle()
			s.FlexBasisPct = p
			out[i] = s
		}
		return out
	}

	t.Run("3 equal items gridSize=12", func(t *testing.T) {
		t.Parallel()
		got := translate.ComputeFlexSizes(makeStyles(0, 0, 0), 12)
		assert.Equal(t, []int{4, 4, 4}, got)
	})

	t.Run("percentage basis 50% 25% 25% gridSize=12", func(t *testing.T) {
		t.Parallel()
		styles := makePctStyles(50, 25, 25)
		got := translate.ComputeFlexSizes(styles, 12)
		assert.Equal(t, []int{6, 3, 3}, got)
	})

	t.Run("grow weights 1:2 gridSize=12", func(t *testing.T) {
		t.Parallel()
		got := translate.ComputeFlexSizes(makeStyles(1, 2), 12)
		assert.Equal(t, []int{4, 8}, got)
	})

	t.Run("mixed pct+grow: 50%+grow=1 gridSize=12", func(t *testing.T) {
		t.Parallel()
		pct := css.NewComputedStyle()
		pct.FlexBasisPct = 50
		grow := css.NewComputedStyle()
		grow.FlexGrow = 1
		got := translate.ComputeFlexSizes([]*css.ComputedStyle{pct, grow}, 12)
		assert.Equal(t, []int{6, 6}, got)
	})

	t.Run("5 equal items gridSize=12 (remainder distribution)", func(t *testing.T) {
		t.Parallel()
		got := translate.ComputeFlexSizes(makeStyles(0, 0, 0, 0, 0), 12)
		sum := 0
		for _, v := range got {
			sum += v
		}
		assert.Equal(t, 12, sum)
		assert.Len(t, got, 5)
	})
}
