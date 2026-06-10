package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

var (
	ErrCannotReadDir = errors.New("cannot read directory")
	ErrGoModNotFound = errors.New("could not find go.mod")
)

var (
	goModFile       = "go.mod"
	paperModule     = "github.com/avdoseferovic/paper"
	defaultTestPath = "test/paper/"
	configSingleton = (*Config)(nil)
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

		cfg := &Config{AbsolutePath: path, TestPath: defaultTestPath}
		configSingleton = cfg
	})
	if initErr != nil {
		t.Errorf("could not configure paper tests: %s", initErr)
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
	actualBytes, err := json.Marshal(actual)
	if err != nil {
		m.t.Error(err.Error())
		return m
	}
	actualString := string(actualBytes)

	indentedExpectBytes, err := os.ReadFile(configSingleton.getAbsoluteFilePath(file))
	if err != nil {
		m.t.Error(err.Error())
		return m
	}

	savedNode := &Node{}
	err = json.Unmarshal(indentedExpectBytes, savedNode)
	if err != nil {
		m.t.Error(err.Error())
		return m
	}
	expectedBytes, err := json.Marshal(savedNode)
	if err != nil {
		m.t.Error(err.Error())
		return m
	}
	expectedString := string(expectedBytes)

	if expectedString != actualString {
		m.t.Errorf("structure mismatch:\nexpected: %s\nactual:   %s", expectedString, actualString)
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
	return getPaperConfigFilePathRecursive(path)
}

func getPaperConfigFilePathRecursive(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	modulePath, hasGoMod, err := modulePathInPath(cleanPath)
	if err != nil {
		return "", err
	}

	if hasGoMod && modulePath == paperModule {
		return cleanPath + string(os.PathSeparator), nil
	}

	parentPath := filepath.Dir(cleanPath)
	if parentPath == cleanPath {
		return "", ErrGoModNotFound
	}
	return getPaperConfigFilePathRecursive(parentPath)
}

func modulePathInPath(path string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(path, goModFile))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("%w: %s", ErrCannotReadDir, err.Error())
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], true, nil
		}
	}
	return "", true, nil
}
