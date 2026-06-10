package merge_test

import (
	"bytes"
	"regexp"
	"strconv"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/merge"
	"github.com/avdoseferovic/paper/pkg/props"
)

func outlinePDF(t *testing.T, titlesAndLevels map[string]int, order []string) []byte {
	t.Helper()
	cfg := config.NewBuilder().WithCompression(false).Build()
	m := paper.New(cfg)
	for _, title := range order {
		m.AddAutoRow(col.New(12).Add(text.New(title, props.Text{
			Outline: &props.Outline{Level: titlesAndLevels[title]},
		})))
	}
	doc, err := m.Generate()
	require.NoError(t, err)
	return doc.GetBytes()
}

func plainPDF(t *testing.T, content string) []byte {
	t.Helper()
	cfg := config.NewBuilder().WithCompression(false).Build()
	m := paper.New(cfg)
	m.AddAutoRow(col.New(12).Add(text.New(content, props.Text{})))
	doc, err := m.Generate()
	require.NoError(t, err)
	return doc.GetBytes()
}

// mergedObjects splits a merged PDF into objectID -> object body.
func mergedObjects(t *testing.T, pdf []byte) map[int][]byte {
	t.Helper()
	objects := make(map[int][]byte)
	re := regexp.MustCompile(`(?s)(\d+) 0 obj\n(.*?)\nendobj`)
	for _, m := range re.FindAllSubmatch(pdf, -1) {
		id, err := strconv.Atoi(string(m[1]))
		require.NoError(t, err)
		objects[id] = m[2]
	}
	return objects
}

func refAfter(t *testing.T, content []byte, key string) (int, bool) {
	t.Helper()
	re := regexp.MustCompile(regexp.QuoteMeta(key) + `\s+(\d+)\s+\d+\s+R`)
	m := re.FindSubmatch(content)
	if m == nil {
		return 0, false
	}
	id, err := strconv.Atoi(string(m[1]))
	require.NoError(t, err)
	return id, true
}

func TestBytes_WhenSourcesHaveOutlines_ShouldMergeOutlineTrees(t *testing.T) {
	t.Parallel()

	doc1 := outlinePDF(t, map[string]int{"One": 0, "One-Sub": 1}, []string{"One", "One-Sub"})
	doc2 := outlinePDF(t, map[string]int{"Two": 0}, []string{"Two"})

	merged, err := merge.Bytes(doc1, doc2)
	require.NoError(t, err)

	objects := mergedObjects(t, merged)

	// catalog (object 1) references the merged outline root + page mode
	catalog := objects[1]
	rootID, ok := refAfter(t, catalog, "/Outlines")
	require.True(t, ok, "merged catalog must reference /Outlines")
	assert.True(t, bytes.Contains(catalog, []byte("/PageMode /UseOutlines")))

	// root mirrors the engine dialect: /First, /Last, no /Count
	root := objects[rootID]
	assert.False(t, bytes.Contains(root, []byte("/Count")), "merged root must not carry /Count")
	firstID, ok := refAfter(t, root, "/First")
	require.True(t, ok, "merged root needs /First")
	lastID, ok := refAfter(t, root, "/Last")
	require.True(t, ok, "merged root needs /Last")

	// walk the top-level chain: One -> Two
	first := objects[firstID]
	assert.True(t, bytes.Contains(first, []byte("One")), "first top-level entry is doc1's")
	parentID, ok := refAfter(t, first, "/Parent")
	require.True(t, ok)
	assert.Equal(t, rootID, parentID, "top-level /Parent must point at merged root")

	nextID, ok := refAfter(t, first, "/Next")
	require.True(t, ok, "cross-document /Next chain missing")
	second := objects[nextID]
	assert.True(t, bytes.Contains(second, []byte("Two")), "second top-level entry is doc2's")
	assert.Equal(t, lastID, nextID, "doc2's top entry is the root's /Last")
	prevID, ok := refAfter(t, second, "/Prev")
	require.True(t, ok, "cross-document /Prev chain missing")
	assert.Equal(t, firstID, prevID)
	secondParent, ok := refAfter(t, second, "/Parent")
	require.True(t, ok)
	assert.Equal(t, rootID, secondParent)

	// nested item keeps its parent pointing at doc1's top item, not the root
	childID, ok := refAfter(t, first, "/First")
	require.True(t, ok, "doc1's nested child must remain linked")
	child := objects[childID]
	assert.True(t, bytes.Contains(child, []byte("One-Sub")))
	childParent, ok := refAfter(t, child, "/Parent")
	require.True(t, ok)
	assert.Equal(t, firstID, childParent, "nested /Parent must point at its level-0 item")

	// /Dest of each entry must reference a real /Type /Page object
	destRe := regexp.MustCompile(`/Dest\s*\[\s*(\d+)\s+\d+\s+R`)
	for _, id := range []int{firstID, nextID, childID} {
		m := destRe.FindSubmatch(objects[id])
		require.NotNil(t, m, "outline item missing /Dest")
		destID, err := strconv.Atoi(string(m[1]))
		require.NoError(t, err)
		dest := objects[destID]
		assert.True(t, bytes.Contains(dest, []byte("/Type /Page")), "outline /Dest must dereference to a /Type /Page object")
	}
}

func TestBytes_WhenOnlyOneSourceHasOutlines_ShouldKeepThatOutline(t *testing.T) {
	t.Parallel()

	doc1 := plainPDF(t, "no bookmarks here")
	doc2 := outlinePDF(t, map[string]int{"Solo": 0}, []string{"Solo"})

	merged, err := merge.Bytes(doc1, doc2)
	require.NoError(t, err)

	objects := mergedObjects(t, merged)
	rootID, ok := refAfter(t, objects[1], "/Outlines")
	require.True(t, ok)
	firstID, ok := refAfter(t, objects[rootID], "/First")
	require.True(t, ok)
	assert.True(t, bytes.Contains(objects[firstID], []byte("Solo")))
}

func TestBytes_WhenNoSourceHasOutlines_ShouldNotEmitOutlines(t *testing.T) {
	t.Parallel()

	merged, err := merge.Bytes(plainPDF(t, "a"), plainPDF(t, "b"))
	require.NoError(t, err)

	objects := mergedObjects(t, merged)
	_, ok := refAfter(t, objects[1], "/Outlines")
	assert.False(t, ok, "outline-free sources must produce an outline-free merge")
}

func TestBytes_WhenSourceOutlineRootIsMalformed_ShouldDropThatOutlineGracefully(t *testing.T) {
	t.Parallel()

	doc1 := outlinePDF(t, map[string]int{"Broken": 0}, []string{"Broken"})
	// Corrupt the outline root's /First reference so the chain walk fails.
	corrupted := bytes.Replace(doc1, []byte("/Type /Outlines /First"), []byte("/Type /Outlines /Frst "), 1)
	require.False(t, bytes.Equal(doc1, corrupted), "fixture must actually corrupt the outline root")

	merged, err := merge.Bytes(corrupted, plainPDF(t, "ok"))

	require.NoError(t, err, "malformed outline must not fail the merge")
	objects := mergedObjects(t, merged)
	_, ok := refAfter(t, objects[1], "/Outlines")
	assert.False(t, ok, "corrupted outline source contributes no outline")
}
