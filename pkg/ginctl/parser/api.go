package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

type ApiParser struct {
	ApiDecls map[string]map[string]*ApiDecl
}

type ApiDecl struct {
	Action         string
	Path           string
	Method         string
	RequestOption  interface{}
	ResponseOption interface{}
}

//

func NewApiParser() *ApiParser {
	return &ApiParser{
		ApiDecls: make(map[string]map[string]*ApiDecl),
	}
}

func (p *ApiParser) Parse(file string) error {
	err := p.parseAPI(file)
	if err != nil {
		return err
	}

	//err = p.parseTypeSpec(ConvToTypeDeclPath(file))

	return nil
}

func (p *ApiParser) parseAPI(file string) error {
	fileTree, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	apiDecls := make(map[string]*ApiDecl)
	for _, decl := range fileTree.Decls {
		apiDecl := new(ApiDecl)
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Doc != nil {
				comments := funcDecl.Doc.List
				for _, comment := range comments {
					commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "//"))
					if !strings.Contains(commentLine, "@") {
						continue
					}
					fields := strings.Fields(commentLine)
					attribute := fields[0]
					lowerAttribute := strings.ToLower(attribute)
					if lowerAttribute == "@router" {
						apiDecl.Path = fields[1]
						apiDecl.Method = strings.TrimRight(strings.TrimLeft(fields[2], "["), "]")
						apiDecl.Action = funcDecl.Name.String()
						apiDecls[apiDecl.Action] = apiDecl
					}
				}
			}
		}
	}

	filename := extractFilename(file)
	p.ApiDecls[filename] = apiDecls

	return nil
}

func (p *ApiParser) parseTypeSpecs(file string) error {
	//filename := extractFilename(file)
	//if apiDecls, ok := p.ApiDecls[filename]; ok {
	//	fileTree, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
	//	if err != nil {
	//		return err
	//	}
	//	for _, decl := range fileTree.Decls {
	//		if genDecl, ok := decl.(*ast.GenDecl); ok {
	//			for _, spec := range genDecl.Specs {
	//				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
	//					if apiDecl, ok := apiDecls[typeSpec.Name.String()]; ok {
	//						typeSpec
	//					}
	//				}
	//			}
	//		}
	//	}
	//}
	//
	return nil
}

func (p *ApiParser) parseTypeSpec(typeSpec *ast.TypeSpec) {

}

func ConvToTypeDeclPath(file string) string {
	dir := filepath.Dir(file)
	dir = strings.Replace(file, "rest", "typespec", 1)

	return fmt.Sprintf("%stype/%s", dir, filepath.Base(file))
}

func extractFilename(file string) string {
	return strings.TrimSuffix(filepath.Base(file), ".go")
}
