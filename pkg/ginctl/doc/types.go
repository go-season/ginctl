package doc

import (
	"go/ast"
	"go/token"
)

type AstFileInfo struct {
	File        *ast.File
	FileSet     *token.FileSet
	Path        string
	PackagePath string
}

type PackageDefinitions struct {
	Name  string
	Files map[string]*ast.File
}
