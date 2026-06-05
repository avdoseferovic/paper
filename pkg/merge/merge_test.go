package merge_test

import (
	"bytes"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johnfercher/paper/v2"
	"github.com/johnfercher/paper/v2/pkg/components/image"
	"github.com/johnfercher/paper/v2/pkg/components/text"
	"github.com/johnfercher/paper/v2/pkg/config"
	"github.com/johnfercher/paper/v2/pkg/consts/align"
	"github.com/johnfercher/paper/v2/pkg/consts/extension"
	"github.com/johnfercher/paper/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/paper/v2/pkg/core/entity"
	"github.com/johnfercher/paper/v2/pkg/fontrepository"
	"github.com/johnfercher/paper/v2/pkg/merge"
	"github.com/johnfercher/paper/v2/pkg/props"
)

func TestBytes(t *testing.T) {
	t.Parallel()
	t.Run("when valid PDFs are provided, should merge and return bytes", func(t *testing.T) {
		t.Parallel()
		// Arrange
		m1 := paper.New()
		m1.AddRows(text.NewRow(10, "text1"))
		doc1, _ := m1.Generate()
		doc1Bytes := doc1.GetBytes()

		m2 := paper.New()
		m2.AddRows(text.NewRow(10, "text2"))
		doc2, _ := m2.Generate()
		doc2Bytes := doc2.GetBytes()

		// Act
		result, err := merge.Bytes(doc1Bytes, doc2Bytes)

		// Assert
		require.NoError(t, err)
		assertPDFPageGraph(t, result, 2)
	})
	t.Run("when invalid PDF bytes are provided, should return wrapped error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		invalidPDF := []byte("not a valid pdf")

		// Act
		result, err := merge.Bytes(invalidPDF, invalidPDF)

		// Assert
		assert.Nil(t, result)
		assert.ErrorIs(t, err, merge.ErrCannotMergePDFs)
	})
	t.Run("when generated PDFs use resources, should keep page references resolvable", func(t *testing.T) {
		t.Parallel()

		pdfs := [][]byte{
			generateTextPDF(t, "plain text"),
			generateImagePDF(t),
			generateGradientPDF(t),
			generateLinkedPDF(t),
			generateCustomFontPDF(t),
			generateTextPDF(t, "compressed text", config.NewBuilder().WithCompression(true).Build()),
		}

		result, err := merge.Bytes(pdfs...)

		require.NoError(t, err)
		assertPDFPageGraph(t, result, len(pdfs))
	})
}

func generateTextPDF(t *testing.T, value string, cfg ...*entity.Config) []byte {
	t.Helper()

	m := paper.New(cfg...)
	m.AddRows(text.NewRow(10, value))
	doc, err := m.Generate()
	require.NoError(t, err)

	return doc.GetBytes()
}

func generateImagePDF(t *testing.T) []byte {
	t.Helper()

	img, err := os.ReadFile("../../docs/assets/images/logo.png")
	require.NoError(t, err)

	m := paper.New()
	m.AddRow(30, image.NewFromBytesCol(12, img, extension.Png, props.Rect{Center: true, Percent: 80}))
	doc, err := m.Generate()
	require.NoError(t, err)

	return doc.GetBytes()
}

func generateGradientPDF(t *testing.T) []byte {
	t.Helper()

	gradient := &props.Gradient{
		Kind:     props.GradientLinear,
		AngleDeg: 0,
		Stops: []props.GradientStop{
			{Color: props.RedColor, Position: 0},
			{Color: props.BlueColor, Position: 1},
		},
	}
	m := paper.New()
	m.AddRow(20, text.NewCol(12, "gradient")).WithStyle(&props.Cell{BackgroundGradient: gradient})
	doc, err := m.Generate()
	require.NoError(t, err)

	return doc.GetBytes()
}

func generateLinkedPDF(t *testing.T) []byte {
	t.Helper()

	link := "https://example.com"
	m := paper.New()
	m.AddRows(text.NewRow(10, "linked text", props.Text{Hyperlink: &link, Align: align.Center}))
	doc, err := m.Generate()
	require.NoError(t, err)

	return doc.GetBytes()
}

func generateCustomFontPDF(t *testing.T) []byte {
	t.Helper()

	const customFont = "arial-unicode-ms"
	fonts, err := fontrepository.New().
		AddUTF8Font(customFont, fontstyle.Normal, "../../docs/assets/fonts/arial-unicode-ms.ttf").
		Load()
	require.NoError(t, err)

	cfg := config.NewBuilder().
		WithCustomFonts(fonts).
		WithDefaultFont(&props.Font{Family: customFont}).
		Build()
	m := paper.New(cfg)
	m.AddRows(text.NewRow(10, "custom font"))
	doc, err := m.Generate()
	require.NoError(t, err)

	return doc.GetBytes()
}

func assertPDFPageGraph(t *testing.T, pdf []byte, expectedPages int) {
	t.Helper()

	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF-")))
	objects := parseTestObjects(t, pdf)
	rootID := findTestObject(t, objects, regexp.MustCompile(`/Type\s*/Catalog\b`))
	pagesID := firstReferenceForKey(t, objects[rootID], "Pages")

	visited := map[int]bool{}
	pageIDs := collectTestPages(t, objects, pagesID, visited)
	require.Len(t, pageIDs, expectedPages)

	for _, pageID := range pageIDs {
		page := objects[pageID]
		contentRefs := referencesForKey(page, "Contents")
		require.NotEmpty(t, contentRefs, "page %d should reference content streams", pageID)
		for _, ref := range contentRefs {
			require.Contains(t, objects, ref, "page %d content reference %d should resolve", pageID, ref)
		}
		for _, ref := range referencesForKey(page, "Resources") {
			require.Contains(t, objects, ref, "page %d resource reference %d should resolve", pageID, ref)
		}
	}
}

func parseTestObjects(t *testing.T, pdf []byte) map[int][]byte {
	t.Helper()

	objectRe := regexp.MustCompile(`(?s)(\d+)\s+\d+\s+obj\s*(.*?)\s*endobj`)
	matches := objectRe.FindAllSubmatch(pdf, -1)
	require.NotEmpty(t, matches)

	objects := make(map[int][]byte, len(matches))
	for _, match := range matches {
		id, err := strconv.Atoi(string(match[1]))
		require.NoError(t, err)
		objects[id] = match[2]
	}
	return objects
}

func findTestObject(t *testing.T, objects map[int][]byte, pattern *regexp.Regexp) int {
	t.Helper()

	ids := make([]int, 0, len(objects))
	for id := range objects {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if pattern.Match(objects[id]) {
			return id
		}
	}
	t.Fatalf("object matching %s not found", pattern.String())
	return 0
}

func collectTestPages(t *testing.T, objects map[int][]byte, objectID int, visited map[int]bool) []int {
	t.Helper()

	require.False(t, visited[objectID], "page tree contains a cycle at object %d", objectID)
	visited[objectID] = true

	object, ok := objects[objectID]
	require.True(t, ok, "page tree object %d should resolve", objectID)
	if regexp.MustCompile(`/Type\s*/Page\b`).Match(object) {
		return []int{objectID}
	}

	require.Regexp(t, regexp.MustCompile(`/Type\s*/Pages\b`), string(object))
	kids := referencesForKey(object, "Kids")
	require.NotEmpty(t, kids, "pages object %d should have kids", objectID)

	var pages []int
	for _, kid := range kids {
		pages = append(pages, collectTestPages(t, objects, kid, visited)...)
	}
	return pages
}

func firstReferenceForKey(t *testing.T, object []byte, key string) int {
	t.Helper()

	refs := referencesForKey(object, key)
	require.NotEmpty(t, refs, "object should contain /%s reference", key)
	return refs[0]
}

func referencesForKey(object []byte, key string) []int {
	directRe := regexp.MustCompile(`/` + key + `\s+(\d+)\s+\d+\s+R`)
	arrayRe := regexp.MustCompile(`/` + key + `\s*\[(.*?)\]`)
	refRe := regexp.MustCompile(`(\d+)\s+\d+\s+R`)

	var refs []int
	for _, match := range directRe.FindAllSubmatch(object, -1) {
		if id, err := strconv.Atoi(string(match[1])); err == nil {
			refs = append(refs, id)
		}
	}
	for _, match := range arrayRe.FindAllSubmatch(object, -1) {
		for _, refMatch := range refRe.FindAllSubmatch(match[1], -1) {
			if id, err := strconv.Atoi(string(refMatch[1])); err == nil {
				refs = append(refs, id)
			}
		}
	}
	return refs
}

func TestReferencesForKeyDoesNotTreatSimilarKeysAsMatches(t *testing.T) {
	t.Parallel()

	object := []byte(`/Contents 10 0 R /OtherContents 20 0 R /Kids [30 0 R 40 0 R]`)

	assert.Equal(t, []int{10}, referencesForKey(object, "Contents"))
	assert.Equal(t, []int{30, 40}, referencesForKey(object, "Kids"))
	assert.True(t, strings.Contains(string(object), "OtherContents"))
}
