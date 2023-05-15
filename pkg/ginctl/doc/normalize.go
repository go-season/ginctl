package doc

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-season/ginctl/pkg/util/log"

	"github.com/mgutz/ansi"
)

type Normalize struct {
	log         log.Logger
	debug       bool
	wdWithSlash string
}

func NewNormalize(log log.Logger, wdWithSlash string, debug bool) Normalize {
	return Normalize{
		log:         log,
		wdWithSlash: wdWithSlash,
		debug:       debug,
	}
}

func (n Normalize) Check() error {
	return filepath.Walk(fmt.Sprintf("%sapi/rest/", n.wdWithSlash), func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || info.Name() == "api.go" || info.Name() == "router.go" || strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		return n.check(path)
	})
}

func (n Normalize) check(path string) (err error) {
	fset := token.NewFileSet()
	fileTree, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, decl := range fileTree.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			funcName := funcDecl.Name.String()
			if n.debug {
				position := fset.Position(funcDecl.Pos())
				n.log.Debugf("check func %s in File: %s:%s",
					ansi.Color(funcName, "red+b"),
					ansi.Color(strings.TrimPrefix(position.Filename, n.wdWithSlash), "cyan+b"),
					ansi.Color(strconv.Itoa(position.Line), "cyan+b"))
			}
			for _, stmt := range funcDecl.Body.List {
				if declStmt, ok := stmt.(*ast.DeclStmt); ok {
					if genDecl, ok := declStmt.Decl.(*ast.GenDecl); ok {
						if genDecl.Tok == token.VAR {
							for _, spec := range genDecl.Specs {
								if vspec, ok := spec.(*ast.ValueSpec); ok {
									if vspec.Names[0].String() == "req" {
										position := fset.Position(vspec.Pos())
										switch vspec.Type.(type) {
										case *ast.SelectorExpr:
											expr := vspec.Type.(*ast.SelectorExpr)
											typName := expr.Sel.String()
											if strings.TrimSuffix(typName, "Request") != funcName {
												n.log.Errorf("方法%s未匹配的Request/Response类型%s in File: %s:%s",
													ansi.Color(funcName, "red+b"),
													ansi.Color(typName, "red+b"),
													ansi.Color(strings.TrimPrefix(position.Filename, n.wdWithSlash), "cyan+b"),
													ansi.Color(strconv.Itoa(position.Line), "cyan+b"),
												)
												err = errors.New("trigger error")
											}
										default:
											n.log.Errorf("错误的定义类型 in File: %s:%s", ansi.Color(strings.TrimPrefix(position.Filename, n.wdWithSlash), "cyan+b"),
												ansi.Color(strconv.Itoa(position.Line), "cyan+b"))
											err = errors.New("trigger error")
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// just irq remain handling
	if err != nil {
		os.Exit(-1)
	}

	return nil
}
