package doc

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/printer"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-season/ginctl/pkg/util"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/str"
	"github.com/mgutz/ansi"
	"github.com/prometheus/common/log"
	"golang.org/x/tools/go/loader"
)

const RouteEntryName = "router.go"
const APIDefinitionName = "api.go"

const (
	// CamelCase indicates using CamelCase strategy for struct field.
	CamelCase = "camelcase"

	// PascalCase indicates using PascalCase strategy for struct field.
	PascalCase = "pascalcase"

	// SnakeCase indicates using SnakeCase strategy for struct field.
	SnakeCase = "snakecase"
)

const MethodAny = "Any"

var builtinTypeMap = map[string]string{
	"orm.LocalTime": "string",
	"time.Time":     "string",
}

type Parser struct {
	Packages             *PackagesDefinitions
	excludes             map[string]bool
	ImportPaths          []string
	ImportPathsCache     map[string]bool
	IsHasMiddleware      bool
	TypePackagePathCache []string
	Debug                bool
	Apis                 []string
	ApiMap               map[string]string
	cwd                  string
}

func NewParser(options ...func(*Parser)) *Parser {
	parser := &Parser{
		excludes:             make(map[string]bool),
		Apis:                 make([]string, 0),
		ApiMap:               make(map[string]string),
		ImportPaths:          make([]string, 0),
		ImportPathsCache:     make(map[string]bool),
		TypePackagePathCache: make([]string, 0),
	}

	for _, option := range options {
		option(parser)
	}

	return parser
}

func WithPackagesDefinitions(pkgs *PackagesDefinitions) func(parser *Parser) {
	return func(p *Parser) {
		p.Packages = pkgs
	}
}

func WithDebug(debug bool) func(parser *Parser) {
	return func(p *Parser) {
		p.Debug = debug
	}
}

func WithWorkDir(cwd string) func(parser *Parser) {
	return func(p *Parser) {
		p.cwd = cwd
	}
}

func WithExcludedDirsAndFiles(excludes string) func(*Parser) {
	return func(p *Parser) {
		for _, f := range strings.Split(excludes, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				f = filepath.Clean(f)
				p.excludes[f] = true
			}
		}
	}
}

func (p *Parser) ParseTypeSpec(dryRun bool, propertyStrategy string, tagFlags []string, typeMap map[string]bool, info *AstFileInfo, astFile *ast.File) error {
	for _, astDescription := range astFile.Decls {
		if generalDeclaraction, ok := astDescription.(*ast.GenDecl); ok && generalDeclaraction.Tok == token.TYPE {
			for _, astSpec := range generalDeclaraction.Specs {
				if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
					handle := func() {
						lis := typeSpec.Type.(*ast.StructType).Fields.List
						for _, ls := range lis {
							names := ls.Names
							if len(names) == 0 {
								continue
							}
							fname := names[0].Name
							switch propertyStrategy {
							case CamelCase:
								fname = str.ToLowerCamelCase(fname)
							case PascalCase:
								fname = names[0].Name
							case SnakeCase:
								fname = str.ToSnakeCase(fname)
							default:
								fname = str.ToLowerCamelCase(fname)
							}
							var tagStr string
							var tag reflect.StructTag
							if ls.Tag != nil {
								tagStr = ls.Tag.Value
								tagStr = strings.Trim(tagStr, "`")
								tag = reflect.StructTag(tagStr)
							}
							for _, tagFlag := range tagFlags {
								if tag != "" {
									if tag.Get(tagFlag) == "" {
										tagStr += fmt.Sprintf(" %s:\"%s\"", tagFlag, fname)
									} else {
										propValue := tag.Get(tagFlag)
										tagStr = strings.Replace(tagStr, propValue, fname, 1)
									}
								} else {
									if tag == "" {
										tagStr += fmt.Sprintf(" %s:\"%s\"", tagFlag, fname)
									}
								}
							}
							ls.Tag = &ast.BasicLit{
								ValuePos: token.Pos(ls.Pos() + 15),
								Kind:     token.STRING,
								Value:    fmt.Sprintf("`%s`", strings.TrimSpace(tagStr)),
							}
						}
					}

					if len(typeMap) > 0 {
						if _, ok := typeMap[typeSpec.Name.String()]; !ok {
							continue
						}
					}

					handle()
				}
			}
		}
	}

	var buf bytes.Buffer
	(&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(&buf, info.FileSet, astFile)
	if dryRun {
		fmt.Println(buf.String())
	} else {
		fs, err := os.Create(fmt.Sprintf("%s/%s", p.cwd, info.Path))
		if err != nil {
			return err
		}
		defer fs.Close()
		fs.WriteString(buf.String())
	}

	return nil
}

func (p *Parser) ParseAPIInfo(info *AstFileInfo, astFile *ast.File) error {
	index := strings.LastIndex(info.PackagePath, "/")
	curPkg := info.PackagePath[index+1:]
	if _, ok := p.ImportPathsCache[info.PackagePath]; !ok {
		p.ImportPathsCache[info.PackagePath] = true
		p.ImportPaths = append(p.ImportPaths, fmt.Sprintf("\"%s\"", info.PackagePath))
	}
	for _, astDescription := range astFile.Decls {
		switch astDeclaraction := astDescription.(type) {
		case *ast.FuncDecl:
			funcName := astDeclaraction.Name.String()
			var httpMethod string
			var apiPath string
			var apiDecl string
			var beforeMiddlewares, afterMiddlewares []string
			if p.Debug {
				fmt.Println(fmt.Sprintf("parse funcName: %s in file %s", funcName, info.Path))
			}

			if astDeclaraction.Doc == nil {
				fmt.Println(ansi.Color(fmt.Sprintf("[Warn] func: %s missing route annotation, skipping...", funcName), "yellow+b"))
				continue
			}
			for _, comment := range astDeclaraction.Doc.List {
				commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "//"))
				fields := strings.Fields(commentLine)
				attribute := fields[0]
				lowerAttribute := strings.ToLower(attribute)
				switch lowerAttribute {
				case "@router":
					httpMethod = strings.TrimRight(strings.TrimLeft(fields[2], "["), "]")
					apiPath = fields[1]
				case "@beforemiddleware":
					beforeMiddlewares = strings.Split(fields[1], ",")
				case "@aftermiddleware":
					afterMiddlewares = strings.Split(fields[1], ",")
				}
			}
			if httpMethod == "" {
				continue
			}

			action := fmt.Sprintf("%s.%s", curPkg, funcName)
			methods := strings.Split(httpMethod, ",")
			for _, httpMethod := range methods {
				switch httpMethod {
				case http.MethodGet:
					apiDecl = fmt.Sprintf("route.GET(\"%s\"", apiPath)
				case http.MethodPost:
					apiDecl = fmt.Sprintf("route.POST(\"%s\"", apiPath)
				case http.MethodPatch:
					apiDecl = fmt.Sprintf("route.PATCH(\"%s\"", apiPath)
				case MethodAny:
					apiDecl = fmt.Sprintf("route.Any(\"%s\"", apiPath)
				default:
					return errors.New("unsupport method")
				}
				if beforeMiddlewares != nil {
					p.IsHasMiddleware = true
					for _, beforeMiddleware := range beforeMiddlewares {
						apiDecl = fmt.Sprintf("%s, %s", apiDecl, strings.TrimSpace(beforeMiddleware))
					}
				}
				apiDecl = fmt.Sprintf("%s, %s", apiDecl, action)
				if afterMiddlewares != nil {
					p.IsHasMiddleware = true
					for _, afterMiddleware := range afterMiddlewares {
						apiDecl = fmt.Sprintf("%s, %s", apiDecl, strings.TrimSpace(afterMiddleware))
					}
				}
				apiDecl = fmt.Sprintf("%s)", apiDecl)

				fmt.Printf("generate route: %s\n", ansi.Color(apiDecl, "cyan+b"))
				key := fmt.Sprintf("%s.%s", apiPath, httpMethod)
				p.Apis = append(p.Apis, key)
				p.ApiMap[key] = apiDecl
			}
		}
	}
	return nil
}

func (p *Parser) ParseImportsToMap(importPath string) map[string]string {
	pkgInfo := p.getPkgInfo(importPath)

	if pkgInfo == nil {
		return nil
	}

	importMap := make(map[string]string)
	for i := range pkgInfo.Files {
		astFile := pkgInfo.Files[i]
		for _, importSpec := range astFile.Imports {
			var key string
			name := importSpec.Name
			path := importSpec.Path.Value
			if name == nil {
				key = path[strings.LastIndex(path, "/")+1:]
			} else {
				key = name.Name
			}
			importMap[strings.TrimSuffix(key, "\"")] = strings.Trim(path, "\"")
		}
	}

	return importMap
}

func (p *Parser) ParseCommentInfo(info *AstFileInfo, astFile *ast.File) error {
	filename := astFile.Name.Name
	typeSpecImportPath := strings.Replace(info.PackagePath, "rest", "typespec", 1)
	if p.Debug {
		fmt.Println(fmt.Sprintf("current parse rest: %s, typespec: %s", ansi.Color(info.PackagePath, "cyan+b"), ansi.Color(typeSpecImportPath+"type", "cyan+b")))
	}
	typeSpecImportPath += "type"

	typeSpecPath := strings.Replace(filepath.Dir(info.Path)+"type/"+filepath.Base(info.Path), "rest", "typespec", 1)
	//typeImportsMap := p.ParseImportsToMap(typeSpecPath)
	typeImportsMap := map[string]string{
		"typespec": fmt.Sprintf("%s/api/typespec/base.go", p.cwd),
	}
	p.TypePackagePathCache = append(p.TypePackagePathCache, "	_ "+fmt.Sprintf("\"%s\"", typeSpecImportPath))
	for _, astDescription := range astFile.Decls {
		switch astDeclaraction := astDescription.(type) {
		case *ast.FuncDecl:
			funcName := astDeclaraction.Name.String()
			if p.Debug {
				fmt.Println(fmt.Sprintf("current parse func: %s", ansi.Color(funcName, "cyan+b")))
			}
			var (
				httpMethod    string
				path          string
				acceptComment string
				funcDesc      string
			)
			if astDeclaraction.Doc == nil {
				log.Warnf("func: %s in %s not found doc, please confirm the func is deprecated.", ansi.Color(funcName, "cyan+b"), ansi.Color(info.PackagePath+".go", "cyan+b"))
				continue
			}
			for _, comment := range astDeclaraction.Doc.List {
				commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "//"))
				if !strings.Contains(commentLine, "@") {
					funcDesc = commentLine
					continue
				}
				fields := strings.Fields(commentLine)
				attribute := fields[0]
				lowerAttribute := strings.ToLower(attribute)
				if lowerAttribute == "@router" {
					path = fields[1]
					httpMethod = strings.TrimRight(strings.TrimLeft(fields[2], "["), "]")
				}
				if lowerAttribute == "@accept" {
					acceptComment = commentLine
				}
			}
			if httpMethod == "" {
				continue
			}

			if funcDesc == "" {
				funcDesc = str.ToCamel(filename)
			}
			methods := strings.Split(httpMethod, ",")
			for _, method := range methods {
				comment := fmt.Sprintf("// %s\n", funcDesc)
				comment += fmt.Sprintf("// @Tags %s\n", str.ToCamel(filename))
				comment += fmt.Sprintf("// @Summary %s\n", funcDesc)
				if acceptComment != "" {
					comment += fmt.Sprintf("// %s\n", acceptComment)
				}

				comment += "// @Produce json\n"
				// TODO optimize => module => file
				spec, _ := p.findTypeDef(typeSpecPath, fmt.Sprintf("%sRequest", funcName))
				typeSpecPkgName := typeSpecImportPath[strings.LastIndex(typeSpecImportPath, "/")+1:]
				if spec != nil {
					switch spec.Type.(type) {
					case *ast.StructType:
						lis := spec.Type.(*ast.StructType).Fields.List
						comment = p.parseTypeSpecComment(comment, httpMethod, typeSpecPath, 0, lis, typeImportsMap)
					}
					switch method {
					case http.MethodGet, http.MethodPatch:
						//comment += fmt.Sprintf("// @Param object query %s.%s false \"请求数据\"\n", typeSpecPkgName, funcName+"Request")
						if method != http.MethodPatch {
							comment += fmt.Sprintf("// @Success 200 object %s.%s \"请求成功\"\n", typeSpecPkgName, funcName+"Response")
							//if strings.HasSuffix(funcName, "List") {
							//	typename := fmt.Sprintf("%sResponse", strings.TrimSuffix(funcName, "List"))
							//	comment += fmt.Sprintf("// @Success 200 object []%s.%s \"请求成功\"\n", typeSpecPkgName, typename)
							//}
						}
					case http.MethodPost:
						if acceptComment != "" {
							comment += fmt.Sprintf("// @Param object formData %s.%s true \"请求数据\"\n", typeSpecPkgName, funcName+"Request")
						} else {
							comment += fmt.Sprintf("// @Param data body %s.%s true \"请求数据\"\n", typeSpecPkgName, funcName+"Request")
						}
						comment += fmt.Sprintf("// @Success 200 object %s.%s \"请求成功\"\n", typeSpecPkgName, funcName+"Response")
					}
				} else {
					comment += fmt.Sprintf("// @Success 200 object %s \"请求成功\"\n", funcName+"Response")
				}
				comment += "// @Failure 500 \"服务异常\"\n"
				comment += fmt.Sprintf("// @Router %s [%s]", path, method)
				astDeclaraction.Doc.List[0].Text = comment
				astDeclaraction.Doc.List = astDeclaraction.Doc.List[:1]
			}
		}
	}

	var buf bytes.Buffer
	printer.Fprint(&buf, info.FileSet, astFile)
	tplPathSuffix := strings.TrimPrefix(info.Path, fmt.Sprintf("%s/api/rest/", p.cwd))
	tplPath := fmt.Sprintf("%s/api/doc/%s", p.cwd, tplPathSuffix)

	err := p.buildDocTpl(buf, tplPath)
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) parseTypeSpecComment(comment, httpMethod, typeSpecPath string, depth int, lis []*ast.Field, typeImportsMap map[string]string) string {
	if depth >= 2 {
		panic("now just support parse dependency depth 1")
	}

	for _, ls := range lis {
		if p.Debug {
			fname := "anonymous"
			if ls.Names != nil {
				fname = ls.Names[0].String()
			}
			fmt.Println(fmt.Sprintf("current parse filed: %s", ansi.Color(fname, "cyan+b")))
		}
		var subTypePath, subTypeName, switchType string
		if httpMethod != http.MethodPost {
			switch ls.Type.(type) {
			case *ast.Ident:
				obj := ls.Type.(*ast.Ident).Obj
				if obj != nil && obj.Kind == ast.Typ {
					subTypePath = typeSpecPath
					subTypeName = obj.Name
				}
			case *ast.SelectorExpr:
				importName := ls.Type.(*ast.SelectorExpr).X.(*ast.Ident).Name
				subTypeName = ls.Type.(*ast.SelectorExpr).Sel.Name
				if typ, ok := builtinTypeMap[importName+"."+subTypeName]; ok {
					switchType = typ
				} else {
					if _, ok := typeImportsMap[importName]; !ok {
						panic("un import field path, can't parse un import type field")
					}
					subTypePath = typeImportsMap[importName]
				}
			case *ast.ArrayType:
				elt := ls.Type.(*ast.ArrayType).Elt
				switch elt.(type) {
				case *ast.Ident:
					switchType = "[]" + elt.(*ast.Ident).String()
				default:
					panic("not support field type")
				}
			case *ast.MapType:
				switchType = "object"
			default:
				panic("not support field type")
			}
		}
		if subTypePath != "" && subTypeName != "" {
			spec, err := p.findTypeDef(subTypePath, subTypeName)
			if err != nil {
				panic(err)
			}
			flis := spec.Type.(*ast.StructType).Fields.List
			comment = p.parseTypeSpecComment(comment, httpMethod, typeSpecPath, depth+1, flis, typeImportsMap)
			continue
		}

		var typ string
		if switchType != "" {
			typ = switchType
		} else {
			switch ls.Type.(type) {
			case *ast.Ident:
				typ = ls.Type.(*ast.Ident).Name
			default:
			}
		}
		if ls.Tag == nil {
			continue
		}
		tagV := strings.TrimRight(strings.TrimLeft(ls.Tag.Value, "`"), "`")
		tag := reflect.StructTag(tagV)
		name := tag.Get("form")
		if index := strings.Index(name, ","); index != -1 {
			name = name[:index]
		}
		desc := strings.TrimSpace(ls.Comment.Text())
		if desc == "" {
			desc = name
		}
		valid := tag.Get("valid")
		var required bool
		if valid != "" {
			parts := strings.Split(valid, ";")
			for _, part := range parts {
				if part == "Required" {
					required = true
				}
			}
		}
		if name != "" && httpMethod != http.MethodPost {
			comment += fmt.Sprintf("// @Param %s query %s %t \"%s\"\n", name, typ, required, desc)
		} else {
			header := tag.Get("header")
			swaggerIgnore := tag.Get("swagignore")
			if swaggerIgnore != "" {
				continue
			}
			if header != "" {
				if index := strings.Index(header, ","); index != -1 {
					header = header[:index]
				}
				comment += fmt.Sprintf("// @Param %s header %s %t \"%s\"\n", header, typ, required, header)
			}
		}
	}

	return comment
}

func (p *Parser) buildDocTpl(buf bytes.Buffer, tplPath string) error {
	pos := strings.LastIndex(tplPath, "/")
	docDir := tplPath[:pos]
	found, err := file.PathExists(docDir)
	if err != nil {
		return err
	}
	if !found {
		os.MkdirAll(docDir, 0755)
	}

	f, err := os.Create(tplPath)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(buf.String())
	return nil
}

func (p *Parser) getPkgInfo(importPath string) *loader.PackageInfo {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	conf := loader.Config{
		ParserMode: goparser.ParseComments,
		Cwd:        cwd,
	}

	conf.Import(importPath)

	lprog, err := conf.Load()
	if err != nil {
		return nil
	}

	for k := range lprog.AllPackages {
		realPkgPath := k.Path()

		if strings.Contains(realPkgPath, "vendor/"+importPath) {
			importPath = realPkgPath
		}
	}

	return lprog.Package(importPath)
}

func (p *Parser) findTypeDef(file, typeName string) (*ast.TypeSpec, error) {
	fset := token.NewFileSet()
	fileTree, err := goparser.ParseFile(fset, file, nil, goparser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range fileTree.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.String() == typeName {
						return typeSpec, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("type spec not found")
}

func (p *Parser) findTypeDefV2(importPath, typeName string) (*ast.TypeSpec, error) {
	pkgInfo := p.getPkgInfo(importPath)

	if pkgInfo == nil {
		return nil, fmt.Errorf("package was nil")
	}

	for i := range pkgInfo.Files {
		for _, astDeclaraction := range pkgInfo.Files[i].Decls {
			if generalDeclaraction, ok := astDeclaraction.(*ast.GenDecl); ok && generalDeclaraction.Tok == token.TYPE {
				for _, astSpec := range generalDeclaraction.Specs {
					if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
						if typeSpec.Name.String() == typeName {
							return typeSpec, nil
						}
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("type spec not found")
}

func (p *Parser) ParseAPI(searchDir string) error {
	packageDir, err := util.GetPkgName(searchDir)
	if err != nil {
		fmt.Printf("warning: failed to get package name in dir: %s, error: %s", searchDir, err.Error())
	}

	if err = p.getAllGoFileInfo(packageDir, searchDir); err != nil {
		return err
	}

	return nil
}

func (p *Parser) getAllGoFileInfo(packageDir, searchDir string) error {
	return filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if err := p.Skip(path, f); err != nil {
			return err
		} else if f.IsDir() {
			return nil
		} else if f.Name() == RouteEntryName || f.Name() == APIDefinitionName {
			return nil
		}
		if strings.Contains(f.Name(), "_test.go") {
			return nil
		}

		relPath, err := filepath.Rel(searchDir, path)
		if err != nil {
			return err
		}
		return p.parseFile(filepath.ToSlash(filepath.Dir(filepath.Clean(filepath.Join(packageDir, relPath)))), path, nil)
	})
}

func (p *Parser) parseFile(packageDir, path string, src interface{}) error {
	if strings.HasSuffix(strings.ToLower(path), "_test.go") || filepath.Ext(path) != ".go" {
		return nil
	}

	fset := token.NewFileSet()
	astFile, err := goparser.ParseFile(fset, path, src, goparser.ParseComments)
	if err != nil {
		return fmt.Errorf("ParseFile error:%+v", err)
	}
	p.Packages.CollectAstFile(packageDir, path, astFile, fset)
	return nil
}

func (p *Parser) Skip(path string, f os.FileInfo) error {
	if f.IsDir() {
		if f.Name() == "docs" || len(f.Name()) > 1 && f.Name()[:1] == "." {
			return filepath.SkipDir
		}
		if p.excludes != nil {
			if _, ok := p.excludes[path]; ok {
				return filepath.SkipDir
			}
		}
	}

	return nil
}
