// nolint:errchkjson // not needed
package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/avdoseferovic/paper/v2/pkg/tree/node"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/avdoseferovic/paper/v2/pkg/core"
)

var (
	ErrCannotReadDir       = errors.New("cannot read directory")
	ErrCannotReadFile      = errors.New("cannot read file")
	ErrCannotUnmarshallYML = errors.New("cannot unmarshall yaml")
	ErrPaperYMLNotFound    = errors.New("found go.mod but not .paper.yml")
)

var (
	paperFile       = ".paper.yml"
	goModFile       = "go.mod"
	configSingleton *Config
	configOnce      sync.Once
)

type Node struct {
	Value   any            `json:"value,omitempty"`
	Type    string         `json:"type"`
	Details map[string]any `json:"details,omitempty"`
	Nodes   []*Node        `json:"nodes,omitempty"`
}

// PaperTest is the unit test instance.
type PaperTest struct {
	t    *testing.T
	node *node.Node[core.Structure]
}

// New creates the PaperTest instance to unit tests.
func New(t *testing.T) *PaperTest {
	t.Helper()
	var initErr error
	configOnce.Do(func() {
		path, err := getPaperConfigFilePath()
		if err != nil {
			initErr = err
			return
		}

		cfg, err := loadPaperConfigFile(path)
		if err != nil {
			initErr = err
			return
		}

		cfg.AbsolutePath = path
		configSingleton = cfg
	})
	if initErr != nil {
		assert.Fail(t, "could not load .paper.yml: "+initErr.Error())
	}

	return &PaperTest{
		t: t,
	}
}

// Assert validates if the structure is the same as defined by Equals method.
func (m *PaperTest) Assert(structure *node.Node[core.Structure]) *PaperTest {
	m.node = structure
	return m
}

// Equals defines which file will be loaded to do the comparison.
func (m *PaperTest) Equals(file string) *PaperTest {
	m.t.Helper()
	actual := m.buildNode(m.node)
	actualBytes, _ := json.Marshal(actual)
	actualString := string(actualBytes)

	indentedExpectBytes, err := os.ReadFile(configSingleton.getAbsoluteFilePath(file))
	if err != nil {
		assert.Fail(m.t, err.Error())
	}

	savedNode := &Node{}
	_ = json.Unmarshal(indentedExpectBytes, savedNode)
	expectedBytes, _ := json.Marshal(savedNode)

	assert.Equal(m.t, string(expectedBytes), actualString)
	return m
}

// Save is an auxiliary method to update the file to be asserted.
func (m *PaperTest) Save(file string) *PaperTest {
	actual := m.buildNode(m.node)
	actualBytes, _ := json.MarshalIndent(actual, "", "\t")

	err := os.WriteFile(configSingleton.getAbsoluteFilePath(file), actualBytes, os.ModePerm)
	if err != nil {
		assert.Fail(m.t, err.Error())
	}

	return m
}

func (m *PaperTest) buildNode(node *node.Node[core.Structure]) *Node {
	data := node.GetData()
	actual := &Node{
		Type:    data.Type,
		Value:   data.Value,
		Details: data.Details,
	}

	nexts := node.GetNexts()
	for _, next := range nexts {
		actual.Nodes = append(actual.Nodes, m.buildNode(next))
	}

	return actual
}

func getPaperConfigFilePath() (string, error) {
	path, _ := os.Getwd()
	path += "/"

	return getPaperConfigFilePathRecursive(path)
}

func loadPaperConfigFile(path string) (*Config, error) {
	bytes, err := os.ReadFile(path + "/" + paperFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotReadFile, err)
	}

	cfg := &Config{}
	err = yaml.Unmarshal(bytes, cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotUnmarshallYML, err)
	}

	return cfg, nil
}

func getPaperConfigFilePathRecursive(path string) (string, error) {
	hasPaper, err := hasFileInPath(paperFile, path)
	if err != nil {
		return "", err
	}

	if hasPaper {
		return path, nil
	}

	hasGoMod, err := hasFileInPath(goModFile, path)
	if err != nil {
		return "", err
	}

	if hasGoMod {
		return "", ErrPaperYMLNotFound
	}

	parentPath := getParentDir(path)
	return getPaperConfigFilePathRecursive(parentPath)
}

func hasFileInPath(file string, path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrCannotReadDir, err.Error())
	}

	for _, entry := range entries {
		if entry.Name() == file {
			return true, nil
		}
	}

	return false, nil
}

func getParentDir(path string) string {
	dirs := strings.Split(path, "/")
	dirs = dirs[:len(dirs)-2]

	var builder strings.Builder
	for _, dir := range dirs {
		builder.WriteString(dir + "/")
	}

	return builder.String()
}
