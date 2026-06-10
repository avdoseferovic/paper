package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

const goldenFile = "paper_test.json"

func fixtureNode(rootType string) *node.Node[core.Structure] {
	rootNode := node.New[core.Structure](core.Structure{
		Type: rootType,
	})
	pageNode := node.New[core.Structure](core.Structure{
		Type: "page",
	})

	rootNode.AddNext(pageNode)
	return rootNode
}

func TestNew_WhenCalled_ShouldSetupSingleton(t *testing.T) {
	t.Parallel()

	sut := New(t)

	assert.NotNil(t, sut)
	assert.NotNil(t, configSingleton)
}

func TestPaperTest_Equals_WhenStructureMatchesGolden_ShouldPass(t *testing.T) {
	t.Parallel()

	n := fixtureNode("paper")
	innerT := &testing.T{}

	New(innerT).Assert(n).Equals(goldenFile)

	assert.False(t, innerT.Failed())
}

func TestPaperTest_Equals_WhenStructureDiffersFromGolden_ShouldFail(t *testing.T) {
	t.Parallel()

	n := fixtureNode("not_paper")
	innerT := &testing.T{}

	New(innerT).Assert(n).Equals(goldenFile)

	assert.True(t, innerT.Failed())
}

func TestPaperTest_Equals_WhenGoldenFileMissing_ShouldFail(t *testing.T) {
	t.Parallel()

	n := fixtureNode("paper")
	innerT := &testing.T{}

	New(innerT).Assert(n).Equals("does_not_exist.json")

	assert.True(t, innerT.Failed())
}

func TestPaperTest_Equals_WhenGoldenMatches_ShouldDecodeExpectedShape(t *testing.T) {
	t.Parallel()

	_ = New(t)
	bytes, err := os.ReadFile(configSingleton.getAbsoluteFilePath(goldenFile))
	assert.NoError(t, err)

	testNode := &Node{}
	err = json.Unmarshal(bytes, testNode)
	assert.NoError(t, err)
	assert.Equal(t, "paper", testNode.Type)
	assert.Equal(t, "page", testNode.Nodes[0].Type)
}

func TestGetPaperConfigFilePathRecursive_WhenModuleMatches_ShouldReturnThatDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "pkg", "test")
	err := os.MkdirAll(nested, os.ModePerm)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(root, goModFile), []byte("module "+paperModule+"\n"), os.ModePerm)
	assert.NoError(t, err)

	path, err := getPaperConfigFilePathRecursive(nested + string(os.PathSeparator))

	assert.NoError(t, err)
	assert.Equal(t, root+string(os.PathSeparator), path)
}

func TestGetPaperConfigFilePathRecursive_WhenIntermediateModuleDiffers_ShouldKeepWalkingUp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "sub")
	err := os.MkdirAll(nested, os.ModePerm)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(root, goModFile), []byte("module "+paperModule+"\n"), os.ModePerm)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(nested, goModFile), []byte("module example.test/other\n"), os.ModePerm)
	assert.NoError(t, err)

	path, err := getPaperConfigFilePathRecursive(nested)

	assert.NoError(t, err)
	assert.Equal(t, root+string(os.PathSeparator), path)
}

func TestGetPaperConfigFilePathRecursive_WhenGoModMissing_ShouldReturnError(t *testing.T) {
	t.Parallel()

	path, err := getPaperConfigFilePathRecursive(t.TempDir())

	assert.Empty(t, path)
	assert.ErrorIs(t, err, ErrGoModNotFound)
}
