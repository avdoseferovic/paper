package paper

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderGoOnlyContainsProviderConstruction(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join("provider.go"), nil, 0)
	require.NoError(t, err)

	allowed := map[string]bool{
		"New":             true,
		"MeasureString":   true,
		"AddTextAt":       true,
		"AddRichText":     true,
		"MeasureRichText": true,
	}
	var unexpected []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || allowed[fn.Name.Name] {
			continue
		}
		unexpected = append(unexpected, fn.Name.Name)
	}

	assert.Empty(t, unexpected)
}
