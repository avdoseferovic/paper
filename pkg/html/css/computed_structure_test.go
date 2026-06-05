package css

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputedStyleApplyCtxDelegatesToFocusedHandlers(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "computed.go", nil, 0)
	require.NoError(t, err)

	applyCtx := findFuncDecl(file, "ApplyCtx")
	require.NotNil(t, applyCtx)

	ast.Inspect(applyCtx.Body, func(node ast.Node) bool {
		_, ok := node.(*ast.SwitchStmt)
		assert.False(t, ok, "ApplyCtx should delegate to focused handlers instead of owning the property switch")
		return !ok
	})

	handlers := map[string]bool{
		"applyEffectsProperty":    false,
		"applyFontProperty":       false,
		"applyBoxProperty":        false,
		"applyBorderProperty":     false,
		"applyFlexProperty":       false,
		"applyTypographyProperty": false,
	}
	ast.Inspect(applyCtx.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if _, found := handlers[selector.Sel.Name]; found {
			handlers[selector.Sel.Name] = true
		}
		return true
	})

	for handler, found := range handlers {
		assert.True(t, found, "%s should be called from ApplyCtx", handler)
	}
}

func TestComputedStyleRemainsFlatStruct(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "computed.go", nil, 0)
	require.NoError(t, err)

	style := findStructType(file, "ComputedStyle")
	require.NotNil(t, style)

	fields := map[string]bool{}
	for _, field := range style.Fields.List {
		require.NotEmpty(t, field.Names, "ComputedStyle should not use embedded grouped structs")
		for _, name := range field.Names {
			fields[name.Name] = true
		}
	}

	for _, field := range []string{"FontSize", "PaddingTop", "BorderTopColor", "FlexGrow", "WhiteSpace"} {
		assert.True(t, fields[field], "%s should remain a direct ComputedStyle field", field)
	}
}

func findFuncDecl(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

func findStructType(file *ast.File, name string) *ast.StructType {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != name {
				continue
			}
			structType, _ := typeSpec.Type.(*ast.StructType)
			return structType
		}
	}
	return nil
}
