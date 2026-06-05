package paper

import (
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/tree/node"
)

// GetStructure is responsible for return the component tree, this is useful
// on unit tests cases.
func (m *Paper) GetStructure() *node.Node[core.Structure] {
	return m.pageBuilder.getStructure()
}
