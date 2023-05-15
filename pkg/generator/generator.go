package generator

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

type Generator struct {
	*bytes.Buffer
}

func NewGenerator() *Generator {
	return &Generator{
		Buffer: new(bytes.Buffer),
	}
}

func (g *Generator) P(str ...string) {
	for _, v := range str {
		g.WriteString(v)
	}
	g.WriteByte('\n')
}

func (g *Generator) GenerateFile(file string) error {
	fset := token.NewFileSet()
	original := g.Bytes()
	fileAST, err := parser.ParseFile(fset, "", original, parser.ParseComments)
	if err != nil {
		return err
	}
	ast.SortImports(fset, fileAST)
	g.Reset()

	(&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(g, fset, fileAST)
	os.Remove(file)
	fs, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()

	fs.Write(g.Bytes())

	return nil
}
