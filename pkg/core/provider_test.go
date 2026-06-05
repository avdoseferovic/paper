package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderComposesFocusedServiceInterfaces(t *testing.T) {
	t.Parallel()

	provider := providerInterfaceDecl(t)
	require.NotNil(t, provider.Methods)
	require.Len(t, provider.Methods.List, 7)

	expectedEmbedded := []string{
		"GridProvider",
		"LineProvider",
		"TextProvider",
		"CodeProvider",
		"ImageProvider",
		"DocumentProvider",
		"DocumentConfigProvider",
	}
	for i, method := range provider.Methods.List {
		require.Empty(t, method.Names, "Provider should embed focused interfaces, not declare direct methods")
		ident, ok := method.Type.(*ast.Ident)
		require.True(t, ok)
		assert.Equal(t, expectedEmbedded[i], ident.Name)
	}
}

func providerInterfaceDecl(t *testing.T) *ast.InterfaceType {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join("provider.go"), nil, 0)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Provider" {
				continue
			}
			iface, ok := typeSpec.Type.(*ast.InterfaceType)
			require.True(t, ok)
			return iface
		}
	}
	require.FailNow(t, "Provider interface declaration not found")
	return nil
}
