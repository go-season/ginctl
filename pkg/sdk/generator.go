package sdk

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-season/ginctl/pkg/util"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/go-season/ginctl/pkg/util/str"
	"github.com/mgutz/ansi"
)

const GinctlV100ReleaseTime = "g202105261512"
const TabIndentWith4Space = "    "
const TabIndentWith8Space = "        "

type Generator struct {
	*bytes.Buffer

	wd           string
	modName      string
	needQueryPkg bool
	isPublish    bool
	isOld        bool
	FileName     string
	APIName      string

	APIDecls map[string]*APIDecl

	RequestDecls  []*StructType
	ResponseDecls []*StructType
	GeneralDecls  []*StructType

	Log log.Logger

	Constants []*Constant
	ImportMap map[string]string
}

type APIDecl struct {
	method string
	path   string
}

type Field struct {
	Name         string
	Type         string
	Tag          string
	isStruct     bool
	StructFields []*Field
	Comment      string
}

type ConstantValue struct {
	isBinary bool
	value    string
	lv       string
	rv       string
	op       string
}

type Constant struct {
	Name  string
	Group []string
	Value ConstantValue
}

type StructType struct {
	Name         string
	Fields       []*Field
	isNeedUrlTag bool
	Pos          token.Pos
}

type Option func(g *Generator)

func NewGenerator(opts ...Option) *Generator {
	g := &Generator{
		Buffer:        new(bytes.Buffer),
		APIDecls:      make(map[string]*APIDecl),
		RequestDecls:  make([]*StructType, 0),
		ResponseDecls: make([]*StructType, 0),
		GeneralDecls:  make([]*StructType, 0),
		Constants:     make([]*Constant, 0),
		ImportMap:     make(map[string]string),
	}

	for _, o := range opts {
		o(g)
	}

	g.modName = "sdk"
	if g.wd != "" {
		g.modName = util.GetModeBaseName(g.wd)
	}

	return g
}

func WithLogger(log log.Logger) Option {
	return func(g *Generator) {
		g.Log = log
	}
}

func WithWorkDir(dir string) Option {
	return func(g *Generator) {
		g.wd = dir
	}
}

func WithOld(old bool) Option {
	return func(g *Generator) {
		g.isOld = old
	}
}

func WithPublish(publish bool) Option {
	return func(g *Generator) {
		g.isPublish = publish
	}
}

func (g *Generator) Parse(file string) error {
	if err := g.parseType(file); err != nil {
		return err
	}

	return g.parseAPI(file)
}

func (g *Generator) parseType(file string) error {
	fset := token.NewFileSet()
	fileTree, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	g.FileName = strings.TrimSuffix(filepath.Base(file), ".go")
	g.APIName = strings.Title(str.SnakeToCamel(g.FileName))

	for _, importSpec := range fileTree.Imports {
		name := importSpec.Name.String()
		if importSpec.Name == nil {
			name = covertScopePkg(importSpec.Path.Value, g.wd, g.isPublish, g.isOld)
		}

		if pkgIsInternal(importSpec.Path.Value) {
			options := fset.Position(importSpec.Pos())
			return errors.New(fmt.Sprintf("无效的内部包%s导入，接口定义可导入范围仅支持(typespec、builtin、external): \nin File: %s:%s",
				ansi.Color(strings.Trim(importSpec.Path.Value, "\""), "red+b"),
				ansi.Color(strings.TrimPrefix(options.Filename, g.wd+"/"), "cyan+b"),
				ansi.Color(strconv.Itoa(options.Line), "cyan+b"),
			))
		}
		g.ImportMap[name] = covertScopePkg(importSpec.Path.Value, g.wd, g.isPublish, g.isOld)
	}

	for _, decl := range fileTree.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				cst := &Constant{
					Group: make([]string, 0),
				}
				switch spec.(type) {
				case *ast.ValueSpec:
					vs := spec.(*ast.ValueSpec)
					if vs.Values != nil {
						switch vs.Values[0].(type) {
						case *ast.Ident:
							v := ConstantValue{
								value: vs.Values[0].(*ast.Ident).String(),
							}
							cst.Value = v
							if vs.Values[0].(*ast.Ident).String() == "iota" {
								cst.Group = append(cst.Group, vs.Names[0].String())
							}
						case *ast.BasicLit:
							cst.Name = vs.Names[0].String()
							cst.Value = ConstantValue{
								value: vs.Values[0].(*ast.BasicLit).Value,
							}
						case *ast.BinaryExpr:
							expr := vs.Values[0].(*ast.BinaryExpr)
							lv := expr.X.(*ast.BasicLit).Value
							op := expr.Op.String()
							var rv string
							switch expr.Y.(type) {
							case *ast.Ident:
								rv = expr.Y.(*ast.Ident).String()
							case *ast.BasicLit:
								rv = expr.Y.(*ast.BasicLit).Value
							}
							if rv == "iota" || lv == "iota" {
								cst.Group = append(cst.Group, vs.Names[0].String())
							} else {
								cst.Name = vs.Names[0].String()
							}
							cst.Value = ConstantValue{
								isBinary: true,
								lv:       lv,
								op:       op,
								rv:       rv,
							}
						default:
							continue
						}
						g.Constants = append(g.Constants, cst)
					} else {
						prevConstant := g.Constants[len(g.Constants)-1]
						prevConstant.Group = append(prevConstant.Group, vs.Names[0].String())
					}
				case *ast.TypeSpec:
					ts := spec.(*ast.TypeSpec)
					switch ts.Type.(type) {
					case *ast.StructType:
						st := new(StructType)
						st.Name = ts.Name.String()
						st.Fields, err = g.parseFields(ts.Type.(*ast.StructType).Fields, fset)
						if err != nil {
							return err
						}
						st.Pos = ts.Pos()
						if typIsRequestDef(ts.Name.String()) {
							g.RequestDecls = append(g.RequestDecls, st)
						} else if typIsResponseDef(ts.Name.String()) {
							g.ResponseDecls = append(g.ResponseDecls, st)
						} else {
							if g.FileName == strings.ToLower(st.Name) {
								g.APIName = st.Name
							}
							g.GeneralDecls = append(g.GeneralDecls, st)
						}
					case *ast.ArrayType:
						position := fset.Position(ts.Pos())
						return errors.New(fmt.Sprintf("暂不支持解析数组类型in File: %s:%s",
							ansi.Color(strings.TrimPrefix(position.Filename, g.wd), "red+b"),
							ansi.Color(strconv.Itoa(position.Line), "red+b")))
					case *ast.MapType:
						position := fset.Position(ts.Pos())
						return errors.New(fmt.Sprintf("暂不支持解析Map类型in File: %s:%s",
							ansi.Color(strings.TrimPrefix(position.Filename, g.wd), "red+b"),
							ansi.Color(strconv.Itoa(position.Line), "red+b")))
					default:
						position := fset.Position(ts.Pos())
						return errors.New(fmt.Sprintf("not support data type %s of data:%s in File:%s:%s",
							ts.Type,
							ansi.Color(ts.Name.String(), "red+b"),
							ansi.Color(strings.TrimPrefix(position.Filename, g.wd), "red+b"),
							ansi.Color(strconv.Itoa(position.Line), "red+b")))
					}
				default:
				}
			}
		}
	}

	if !g.checkAPIDef(fset) {
		os.Exit(1)
	}

	return nil
}

func (g *Generator) parseAPI(file string) error {
	apiFile := specToAPIPath(file)
	fset := token.NewFileSet()
	fileTree, err := parser.ParseFile(fset, apiFile, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, decl := range fileTree.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			for _, doc := range funcDecl.Doc.List {
				commentLine := strings.TrimSpace(strings.TrimLeft(doc.Text, "//"))
				fields := strings.Fields(commentLine)
				attribute := fields[0]
				lowerAttribute := strings.ToLower(attribute)
				if lowerAttribute == "@router" {
					routePath := fields[1]
					method := strings.TrimRight(strings.TrimLeft(fields[2], "["), "]")
					if method == http.MethodGet && !g.needQueryPkg {
						g.needQueryPkg = true
					}

					g.APIDecls[funcDecl.Name.String()] = &APIDecl{
						method: method,
						path:   routePath,
					}
				}
			}
		}
	}

	return nil
}

func specToAPIPath(path string) string {
	prefix := strings.TrimSuffix(filepath.Dir(path), "type")

	return strings.Replace(prefix, "typespec", "rest", 1) + "/" + filepath.Base(path)
}

func (g *Generator) parseFields(fieldList *ast.FieldList, fset *token.FileSet) ([]*Field, error) {
	fields := make([]*Field, 0)
	for _, astField := range fieldList.List {
		field := new(Field)
		if astField.Tag != nil {
			field.Tag = astField.Tag.Value
		}
		if astField.Comment != nil {
			field.Comment = astField.Comment.Text()
		}
		if len(astField.Names) > 0 {
			field.Name = astField.Names[0].String()
		}
		switch astField.Type.(type) {
		case *ast.Ident:
			typ := astField.Type.(*ast.Ident)
			field.Type = typ.Name
		case *ast.SelectorExpr:
			pkgName := astField.Type.(*ast.SelectorExpr).X.(*ast.Ident).String()
			pkgName = covertPkgNameToModBase(pkgName, g.wd)
			typName := astField.Type.(*ast.SelectorExpr).Sel.String()
			field.Type = fmt.Sprintf("%s.%s", pkgName, typName)
		case *ast.ArrayType:
			elt := astField.Type.(*ast.ArrayType).Elt
			switch elt.(type) {
			case *ast.Ident:
				field.Type = fmt.Sprintf("[]%s", elt.(*ast.Ident).Name)
			case *ast.StructType:
				fields := elt.(*ast.StructType).Fields
				field.isStruct = true
				sfs, err := g.parseFields(fields, fset)
				if err != nil {
					return nil, err
				}
				field.Type = "[]"
				field.StructFields = sfs
			case *ast.SelectorExpr:
				pkgName := elt.(*ast.SelectorExpr).X.(*ast.Ident).String()
				pkgName = covertPkgNameToModBase(pkgName, g.wd)
				typName := elt.(*ast.SelectorExpr).Sel.String()
				field.Type = fmt.Sprintf("[]%s.%s", pkgName, typName)
			case *ast.MapType:
				typ := elt.(*ast.MapType)
				var key, value string
				switch typ.Key.(type) {
				case *ast.Ident:
					key = typ.Key.(*ast.Ident).Name
				default:
					position := fset.Position(astField.Pos())
					return nil, errors.New(fmt.Sprintf("暂不执行解析的字段类型，当前可支持mapKey: identity:\nin File: %s:%s",
						ansi.Color(strings.TrimPrefix(position.Filename, g.wd+"/"), "cyan+b"),
						ansi.Color(strconv.Itoa(position.Line), "cyan+b")))
				}
				switch typ.Value.(type) {
				case *ast.Ident:
					value = typ.Value.(*ast.Ident).Name
				default:
					position := fset.Position(astField.Pos())
					return nil, errors.New(fmt.Sprintf("暂不执行解析的字段类型，当前可支持mapValue: identity:\nin File: %s:%s",
						ansi.Color(strings.TrimPrefix(position.Filename, g.wd+"/"), "cyan+b"),
						ansi.Color(strconv.Itoa(position.Line), "cyan+b")))
				}
				field.Type = fmt.Sprintf("[]map[%s]%s", key, value)
			case *ast.StarExpr:
			default:
				position := fset.Position(astField.Pos())
				return nil, errors.New(fmt.Sprintf("暂不支持解析的字段类型，当前可支持arrayElt: identity、map、selector. \nin File: %s:%s",
					ansi.Color(strings.TrimPrefix(position.Filename, g.wd+"/"), "cyan+b"),
					ansi.Color(strconv.Itoa(position.Line), "cyan+b")))
			}
		case *ast.StructType:
			fields := astField.Type.(*ast.StructType).Fields
			field.isStruct = true
			sfs, err := g.parseFields(fields, fset)
			if err != nil {
				return nil, err
			}
			field.StructFields = sfs
		case *ast.InterfaceType:
			field.Type = "interface{}"
		case *ast.StarExpr:
			expr := astField.Type.(*ast.StarExpr).X
			switch expr.(type) {
			case *ast.Ident:
				field.Type = fmt.Sprintf("*%s", expr.(*ast.Ident).Name)
			case *ast.SelectorExpr:
				pkgName := expr.(*ast.SelectorExpr).X.(*ast.Ident).String()
				pkgName = covertPkgNameToModBase(pkgName, g.wd)
				typName := expr.(*ast.SelectorExpr).Sel.String()
				field.Type = fmt.Sprintf("*%s.%s", pkgName, typName)
			default:
				position := fset.Position(astField.Pos())
				return nil, errors.New(fmt.Sprintf("暂不支持解析非标识符指针类型！\nin File: %s:%s",
					ansi.Color(strings.TrimPrefix(position.Filename, g.wd+"/"), "cyan+b"),
					ansi.Color(strconv.Itoa(position.Line), "cyan+b")))
			}
		case *ast.MapType:
			typ := astField.Type.(*ast.MapType)
			var key, value string
			switch typ.Key.(type) {
			case *ast.Ident:
				key = typ.Key.(*ast.Ident).Name
			default:
				position := fset.Position(astField.Pos())
				return nil, errors.New(fmt.Sprintf("暂不执行解析的字段类型，当前可支持mapKey: identity:\nin File: %s:%s",
					ansi.Color(strings.TrimPrefix(position.Filename, g.wd+"/"), "cyan+b"),
					ansi.Color(strconv.Itoa(position.Line), "cyan+b")))
			}
			switch typ.Value.(type) {
			case *ast.Ident:
				value = typ.Value.(*ast.Ident).Name
			case *ast.ArrayType:
				elt := typ.Value.(*ast.ArrayType).Elt
				switch elt.(type) {
				case *ast.Ident:
					value = fmt.Sprintf("[]%s", elt.(*ast.Ident).Name)
				default:
					position := fset.Position(astField.Pos())
					return nil, errors.New(fmt.Sprintf("暂不支持解析的字段类型，当前可支持arrayElt: identity. \nin File: %s:%s",
						ansi.Color(strings.TrimPrefix(position.Filename, g.wd+"/"), "cyan+b"),
						ansi.Color(strconv.Itoa(position.Line), "cyan+b")))
				}
			case *ast.SelectorExpr:
			case *ast.MapType:
			case *ast.StructType:
			default:
				position := fset.Position(astField.Pos())
				return nil, errors.New(fmt.Sprintf("暂不执行解析的字段类型，当前可支持mapValue: identity, array:\nin File: %s:%s",
					ansi.Color(strings.TrimPrefix(position.Filename, g.wd+"/"), "cyan+b"),
					ansi.Color(strconv.Itoa(position.Line), "cyan+b")))
			}

			field.Type = fmt.Sprintf("map[%s]%s", key, value)
		default:
		}
		fields = append(fields, field)
	}

	return fields, nil
}

func (g *Generator) checkAPIDef(fset *token.FileSet) bool {
	valid := true
	wd, _ := os.Getwd()
	reqMap := make(map[string]*StructType)
	respMap := make(map[string]*StructType)

	for _, request := range g.RequestDecls {
		reqMap[strings.TrimSuffix(request.Name, "Request")] = request
	}

	for _, response := range g.ResponseDecls {
		respMap[strings.TrimSuffix(response.Name, "Response")] = response
	}

	for name, st := range reqMap {
		if _, ok := respMap[name]; !ok {
			position := fset.Position(st.Pos)
			g.Log.Errorf("未找到与请求结构%s匹配的响应结构:\nin File: %s:%s",
				ansi.Color(name+"Request", "red+b"),
				ansi.Color(strings.TrimPrefix(position.Filename, wd+"/"), "cyan+b"),
				ansi.Color(strconv.Itoa(position.Line), "cyan+b"))
			valid = false
		}
	}

	for name, st := range respMap {
		if _, ok := reqMap[name]; !ok {
			position := fset.Position(st.Pos)
			g.Log.Errorf("未找到与响应结构%s匹配的请求结构:\nin File: %s:%s",
				ansi.Color(name+"Response", "red+b"),
				ansi.Color(strings.TrimPrefix(position.Filename, wd+"/"), "cyan+b"),
				ansi.Color(strconv.Itoa(position.Line), "cyan+b"))
			valid = false
		}
	}

	return valid
}

func (g *Generator) GenGO() {
	g.generateSpec()
	dir := fmt.Sprintf("%s/sdk/%s/%s", g.wd, strings.Replace(util.GetModeBaseName(g.wd), "-", "", -1), strings.Replace(g.FileName, "_", "", -1))
	if ok, _ := file.PathExists(dir); !ok {
		os.MkdirAll(dir, 0755)
	}

	err := g.generateFile(fmt.Sprintf("%s/%s_spec.go", dir, g.FileName))
	if err != nil {
		panic(err)
	}

	g.generateClientMethod()
	err = g.generateFile(fmt.Sprintf("%s/%s.go", dir, g.FileName))
	if err != nil {
		panic(err)
	}
}

func (g *Generator) GenPHP() {
	fmt.Println(util.GetProjectPath())
	os.Exit(0)
	g.generateRequestClass()
}

func (g *Generator) generateClientServiceClass() {
	g.Reset()
	g.Buffer = new(bytes.Buffer)
}

func (g *Generator) generateRequestClass() {
	for _, decl := range g.RequestDecls {
		g.P("<?php")
		g.P()
		g.P("namespace SDK\\UserCenter\\Request;")
		g.P()
		g.P("class ", decl.Name)
		g.P("{")
		mslice := make([]string, 0)
		for _, field := range decl.Fields {
			prop := g.extractPropertyName(field.Name, field.Tag)
			if !g.typIsBasicType(field.Type) {
				if !field.isStruct {
					prop = g.extractPropertyName(field.Type, field.Tag)
				}

				if field.Tag == "" {
					mslice = g.generateProperty(g.getGeneralDecl(field.Type), mslice)
				} else {
					mslice = append(mslice, prop)
					g.P(TabIndentWith4Space, "private $", prop, " = array();")
					g.P()
				}
			} else {
				mslice = append(mslice, prop)
				g.P(TabIndentWith4Space, "private $", prop, ";")
				g.P()
			}
		}

		for _, m := range mslice {
			method := strings.ToUpper(m[:1]) + m[1:]
			g.P(TabIndentWith4Space, "public function set", method, "($", m, ")")
			g.P(TabIndentWith4Space, "{")
			g.P(TabIndentWith8Space, "$this->", m, " = $", m, ";")
			g.P(TabIndentWith8Space, "return $this;")
			g.P(TabIndentWith4Space, "}")
			g.P()
			g.P(TabIndentWith4Space, "public function get", method, "()")
			g.P(TabIndentWith4Space, "{")
			g.P(TabIndentWith8Space, "return $this->", m, ";")
			g.P(TabIndentWith4Space, "}")
			g.P()
		}
		g.P("}")
	}
}

func (g *Generator) extractPropertyName(name, tag string) string {
	if tag != "" {
		refTag := reflect.StructTag(strings.Trim(tag, "`"))
		prop := refTag.Get("form")
		if prop == "" {
			prop = refTag.Get("json")
		}
		if prop != "" {
			parts := strings.Split(prop, ",")
			return parts[0]
		}
	}

	if name == "" {
		return ""
	}

	return strings.ToLower(name[:1]) + name[1:]
}

func (g *Generator) generateProperty(st *StructType, mslice []string) []string {
	if st != nil {
		for _, field := range st.Fields {
			prop := g.extractPropertyName(field.Name, field.Tag)
			if !g.typIsBasicType(field.Type) {
				if !field.isStruct {
					prop = g.extractPropertyName(field.Type, field.Tag)
				}
				mslice = append(mslice, prop)
				g.P(TabIndentWith4Space, "private $", prop, " = array();")
			} else {
				mslice = append(mslice, prop)
				g.P(TabIndentWith4Space, "private $", prop, ";")
			}
			g.P()
		}
	}

	return mslice
}

func (g *Generator) typIsBasicType(typ string) bool {
	switch typ {
	case "int", "int32", "int64", "float", "float32", "float64", "string":
		return true
	default:
		return false
	}
}

func (g *Generator) getGeneralDecl(name string) *StructType {
	for _, decl := range g.GeneralDecls {
		if decl.Name == name {
			return decl
		}
	}

	return nil
}

func (g *Generator) generateFile(file string) error {
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

func (g *Generator) generateClientMethod() {
	g.Reset()
	g.Buffer = new(bytes.Buffer)

	g.P("// Code generated by ginctl. DO NOT EDIT.")
	g.P()
	g.P("package ", strings.Replace(g.FileName, "_", "", -1))
	g.P("import (")
	if g.needQueryPkg {
		g.P(strconv.Quote("github.com/google/go-querystring/query"))
	}
	g.P(strconv.Quote("github.com/go-season/common/client"))
	g.P(strconv.Quote("gitlab.idc.xiaozhu.com/xz/lib/component/xzapi"))
	g.P(")")
	g.P()

	g.P("type Service interface {")
	for i, decl := range g.RequestDecls {
		funcName := strings.TrimSuffix(decl.Name, "Request")
		apidecl := g.APIDecls[funcName]
		if apidecl == nil {
			continue
		}
		g.P(fmt.Sprintf("%s(req *%s) (%s, error)", funcName, decl.Name, g.ResponseDecls[i].Name))
	}
	g.P("}")
	g.P()

	g.P(fmt.Sprintf("type %sService struct {", str.ToLowerCamelCase(g.APIName)))
	g.P("xzapi.Client")
	g.P("}")
	g.P()

	g.P(fmt.Sprintf("func New%sService(client xzapi.Client) Service {", g.APIName))
	g.P(fmt.Sprintf("return &%sService{", str.ToLowerCamelCase(g.APIName)))
	g.P("Client: client,")
	g.P("}")
	g.P("}")
	g.P()

	for i, decl := range g.RequestDecls {
		short := strings.ToLower(g.APIName[0:1])
		funcName := strings.TrimSuffix(decl.Name, "Request")
		apidecl := g.APIDecls[funcName]
		if apidecl == nil {
			continue
		}
		g.P(fmt.Sprintf("func (%s *%sService) %s(req *%s) (%s, error) {",
			short, str.ToLowerCamelCase(g.APIName), funcName, decl.Name, g.ResponseDecls[i].Name))
		g.P("var (")
		g.P("resp ", g.ResponseDecls[i].Name)
		g.P("err error")
		g.P(")")
		g.P()
		if apidecl.method == http.MethodGet {
			g.P("val, _ := query.Values(req)")
		}
		g.P(fmt.Sprintf("_, err = %s.ClientWithParseContent(&resp).%s(%s, client.Options{",
			short, strings.Title(strings.ToLower(apidecl.method)), strconv.Quote(apidecl.path)))
		if apidecl.method == http.MethodGet {
			g.P("Query: val.Encode(),")
		} else {
			g.P("JSON: req,")
		}
		g.P("})")
		g.P()
		g.P("return resp, err")
		g.P("}")
		g.P()
	}
}

func (g *Generator) generateSpec() {
	g.P("// Code generated by ginctl. DO NOT EDIT.")
	g.P()
	g.P("package ", strings.Replace(g.FileName, "_", "", -1))
	g.P()

	g.generateImports()
	g.generateConstant()

	for _, decl := range g.GeneralDecls {
		g.P("type ", decl.Name, " struct {")
		g.generateStruct(decl.Fields)
		g.P("}")
		g.P()
	}

	g.P()

	for i, decl := range g.RequestDecls {
		funcName := strings.TrimSuffix(decl.Name, "Request")
		apidecl := g.APIDecls[funcName]
		if apidecl != nil {
			decl.isNeedUrlTag = true
		}
		g.P("type ", decl.Name, " struct {")
		g.generateStruct(decl.Fields)
		g.P("}")
		g.P()
		g.P("type ", g.ResponseDecls[i].Name, " struct {")
		g.generateStruct(g.ResponseDecls[i].Fields)
		g.P("}")
		g.P()
	}
}

func (g *Generator) generateImports() {
	if len(g.ImportMap) == 1 {
		for name, path := range g.ImportMap {
			if name != path {
				g.P("import ", name, " ", path)
			} else {
				g.P("import ", " ", path)
			}
		}
	} else {
		if len(g.ImportMap) > 1 {
			g.P("import (")
			for name, path := range g.ImportMap {
				if name != path {
					g.P(name, " ", path)
				} else {
					g.P(path)
				}
			}
			g.P(")")
		}
	}
}

func (g *Generator) generateConstant() {
	g.P()
	for _, cnst := range g.Constants {
		if len(cnst.Group) > 0 {
			g.P()
			g.P("const (")
			if cnst.Value.isBinary {
				g.P(cnst.Group[0], " = ", cnst.Value.lv, " ", cnst.Value.op, " ", cnst.Value.rv)
			} else {
				g.P(cnst.Group[0], " = ", cnst.Value.value)
			}
			for _, name := range cnst.Group[1:] {
				g.P(name)
			}
			g.P(")")
			g.P()
		} else {
			if cnst.Value.isBinary {
				g.P("const ", cnst.Name, " = ", cnst.Value.lv, " ", cnst.Value.op, " ", cnst.Value.rv)
			} else {
				g.P("const ", cnst.Name, " = ", cnst.Value.value)
			}
		}
	}
	g.P()
}

func (g *Generator) generateStruct(fields []*Field) {
	for _, field := range fields {
		tag := field.Tag
		structTag := reflect.StructTag(strings.Trim(tag, "`"))
		if ft := structTag.Get("form"); ft != "" {
			tag = fmt.Sprintf("`%s %s`", strings.Trim(tag, "`"), fmt.Sprintf("url:%s", strconv.Quote(ft)))
		}
		if field.Name == "" {
			g.P(field.Type)
		} else if field.isStruct {
			g.P(field.Name, " ", field.Type, "struct {")
			g.generateStruct(field.StructFields)
			g.P("}", " ", tag)
		} else {
			g.P(field.Name, " ", field.Type, " ", tag)
		}
	}
}

func (g *Generator) P(str ...string) {
	for _, v := range str {
		g.WriteString(v)
	}
	g.WriteByte('\n')
}

func GenerateBase(path, output, dir string, log log.Logger) error {
	g := NewGenerator(WithLogger(log), WithWorkDir(dir))
	if err := g.parseType(path); err != nil {
		return err
	}

	g.FileName = GinctlV100ReleaseTime
	g.generateSpec()

	baseDir := fmt.Sprintf("%s/%s", output, GinctlV100ReleaseTime)
	found, err := file.PathExists(baseDir)
	if err != nil {
		return err
	}
	if !found {
		os.Mkdir(baseDir, 0755)
	}

	return g.generateFile(fmt.Sprintf("%s/base.go", baseDir))
}

func GenerateClientFactory(pkgList, serviceList []string, output string, isPublish bool, isOld bool) {
	wd, _ := os.Getwd()

	g := NewGenerator()

	prefix := getServicePackagePrefix(isPublish, isOld)

	g.P("// Code generated by ginctl. DO NOT EDIT.")
	g.P()
	g.P("package ", strings.Replace(util.GetModeBaseName(wd), "-", "", -1))
	g.P()
	g.P("import (")
	g.P(strconv.Quote("context"))
	g.P(strconv.Quote("gitlab.idc.xiaozhu.com/xz/lib/component/xzapi"))
	for _, pkg := range pkgList {
		g.P(strconv.Quote(fmt.Sprintf("%s/%s", prefix, pkg)))
	}
	g.P(")")
	g.P()

	g.P("type Client interface {")
	for i, service := range serviceList {
		g.P(fmt.Sprintf("%sService() %s.Service", service, pkgList[i]))
	}
	g.P("}")
	g.P()

	g.P("type client struct {")
	g.P("xzapi.Client")
	g.P("}")
	g.P()

	g.P("type Options struct {")
	g.P("Name string")
	g.P("}")
	g.P()
	g.P("type Option func(opt *Options)")
	g.P()
	g.P("func WithName(name string) Option {")
	g.P("return func(opt *Options) {")
	g.P("opt.Name = name")
	g.P("}")
	g.P("}")
	g.P()

	g.P("func NewClient(ctx context.Context, opt ...Option) Client {")
	g.P("c := &client{}")
	g.P("c.Ctx = ctx")
	g.P()
	g.P("opts := new(Options)")
	g.P("for _, o := range opt {")
	g.P("o(opts)")
	g.P("}")
	g.P("c.Name = opts.Name")
	g.P()
	g.P("return c")
	g.P("}")
	g.P()

	for i, s := range serviceList {
		g.P(fmt.Sprintf("func (c *client) %sService() %s.Service {", s, pkgList[i]))
		g.P(fmt.Sprintf("return %s.New%sService(c.Client)", pkgList[i], s))
		g.P("}")
		g.P()
	}

	clientPath := fmt.Sprintf("%s/client.go", output)

	g.generateFile(clientPath)
}

func getServicePackagePrefix(isPublish bool, isOld bool) string {
	wd, _ := os.Getwd()
	basename := util.GetModeBaseName(wd)
	if isPublish || isOld {
		if isOld {
			return "gitlab.idc.xiaozhu.com/xz/lib/sdk/" + strings.Replace(basename, "-", "", -1)
		}
		prjPath, _ := util.GetProjectPath()
		prjPath = strings.TrimPrefix(prjPath, "/")
		return "gitlab.idc.xiaozhu.com/xz/lib/sdk/" + strings.Replace(prjPath, "-", "", -1)
	}
	return util.GetModuleName(wd) + "/sdk/" + strings.Replace(basename, "-", "", -1)
}

func typIsRequestDef(name string) bool {
	return strings.HasSuffix(name, "Request")
}

func typIsResponseDef(name string) bool {
	return strings.HasSuffix(name, "Response")
}

func pkgIsInternal(name string) bool {
	wd, _ := os.Getwd()
	module := util.GetModuleName(wd)
	tsPkg := fmt.Sprintf("%s/api/typespec", module)
	name = strings.Trim(name, "\"")
	return strings.HasPrefix(name, module) && !strings.HasPrefix(name, tsPkg)
}

func covertScopePkg(name, dir string, isPublish bool, isOld bool) string {
	if name == strconv.Quote(fmt.Sprintf("%s/api/typespec", util.GetModuleName(dir))) {
		if isPublish || isOld {
			modepath := util.GetModeBaseName(dir)
			if !isOld {
				modepath, _ = util.GetProjectPath()
				modepath = strings.TrimPrefix(modepath, "/")
			}
			return strconv.Quote(fmt.Sprintf("%s/%s/%s", "gitlab.idc.xiaozhu.com/xz/lib/sdk", modepath, GinctlV100ReleaseTime))
		}
		return strconv.Quote(fmt.Sprintf("%s/sdk/%s/%s", util.GetModuleName(dir), util.GetModeBaseName(dir), GinctlV100ReleaseTime))
	}

	return name
}

func covertPkgNameToModBase(name, dir string) string {
	if name == "typespec" {
		return GinctlV100ReleaseTime
	}
	return name
}
