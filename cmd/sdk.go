package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/mgutz/ansi"

	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/str"

	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

type SDKCmd struct {
	All    bool
	GoOut  string
	PHPOut string
	VueOut string
}

type typeSpec struct {
	ReqName string
	ResName string
	method  string
	options map[string][]string
	result  map[string][]string
}

const (
	identType    = "ident"
	arrayType    = "array"
	mapType      = "map"
	arrayMapType = "arrayMap"
)

var internalRefs = map[string]bool{
	"typespec": true,
	"orm":      true,
	"time":     true,
}

type fieldMap map[string][]string

func NewSDKCmd(f factory.Factory) *cobra.Command {
	cmd := SDKCmd{}

	sdkCmd := &cobra.Command{
		Use:   "sdk",
		Short: "快速为项目生成SDK文件",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	sdkCmd.Flags().StringVar(&cmd.GoOut, "go_out", "", "指定要生成Go版本SDK路径.")
	sdkCmd.Flags().StringVar(&cmd.PHPOut, "php_out", "", "指定要生成PHP版本SDK路径.")
	sdkCmd.Flags().StringVar(&cmd.VueOut, "vue_out", "", "指定要生成Vue版本SDK路径.")
	sdkCmd.Flags().BoolVar(&cmd.All, "all", false, "是否指定要扫描所有的接口定义文件?")

	sdkCmd.MarkFlagRequired("go_out")

	return sdkCmd
}

var excludeFiles = map[string]bool{
	"rest":      true,
	"api.go":    true,
	"router.go": true,
}

func (cmd *SDKCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	root := fmt.Sprintf("%s/api/rest", wd)
	scanDirs := make([]string, 0)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if _, ok := excludeFiles[info.Name()]; ok {
			return nil
		}

		dir := strings.TrimPrefix(path, wd+"/")
		if !info.IsDir() && dir != "" {
			scanDirs = append(scanDirs, dir)
		}

		return nil
	})

	answer := strings.Join(scanDirs, ",")
	if !cmd.All {
		answer, err = f.GetLog().Question(&log.QuestionOptions{
			Question:      "Please select the scan file.",
			Options:       scanDirs,
			IsMultiSelect: true,
		})
		if err != nil {
			return err
		}
	}

	if cmd.GoOut == "" && cmd.PHPOut == "" {
		f.GetLog().Fatalf("Please use `%s`", ansi.Color("ginctl sdk --go_out xxx", "cyan+b"))
	}

	if cmd.GoOut != "" {
		found, err := file.PathExists(cmd.GoOut)
		if err != nil {
			return err
		}
		if !found {
			err = os.MkdirAll(cmd.GoOut, 0755)
			if err != nil {
				return err
			}
		}
	}
	if cmd.PHPOut != "" {
		found, err := file.PathExists(cmd.PHPOut)
		if err != nil {
			return err
		}
		if !found {
			err = os.MkdirAll(cmd.PHPOut, 0755)
			if err != nil {
				return err
			}
		}
	}
	//if cmd.VueOut == "" {
	//	cmd.VueOut = fmt.Sprintf("%s/pkg/sdk/vue", wd)
	//	found, err := file.PathExists(cmd.VueOut)
	//	if err != nil {
	//		return err
	//	}
	//	if !found {
	//		err = os.MkdirAll(cmd.VueOut, 0755)
	//		if err != nil {
	//			return err
	//		}
	//	}
	//}

	svcList := make(map[string]string)
	parts := strings.Split(answer, ",")
	f.GetLog().Info("Start crafting sdk for go project...")
	fmt.Println(fmt.Sprintf("%s Generate project sdk...", time.Now().Format("2006/01/02 15:04:05")))
	fmt.Println(fmt.Sprintf("%s Generate general SDK Info, search dir:./api", time.Now().Format("2006/01/02 15:04:05")))
	baseTypPath := fmt.Sprintf("%s/api/typespec/base.go", wd)
	found, err := file.PathExists(baseTypPath)
	if err != nil {
		return err
	}
	var gcache map[string]fieldMap
	if found {
		_, _, _, gcache, err = makeTypeSpecList(baseTypPath)
		if err != nil {
			return err
		}
	}
	for _, part := range parts {
		routeMap, err := makeRouteMap(fmt.Sprintf("%s/%s", wd, part))
		if err != nil {
			return err
		}

		filePath := strings.Replace(filepath.Dir(part)+"type/"+filepath.Base(part), "rest", "typespec", 1)
		fmt.Println(fmt.Sprintf("%s Parsing %s", time.Now().Format("2006/01/02 15:04:05"), strings.TrimPrefix(filePath, "api/")))
		mname, constant, tsList, cache, err := makeTypeSpecList(filePath)
		if err != nil {
			return err
		}

		keys := make([]string, len(tsList))
		i := 0
		for k := range tsList {
			keys[i] = k
			i++
		}

		filename := strings.Trim(filepath.Base(part), ".go")
		if strings.Contains(filename, "_") {
			mname = str.SnakeToCamel(filename)
		}
		svcList[mname] = mname + "Service"
		if cmd.PHPOut != "" {
			err = genPHPSDK(mname, cmd.PHPOut, constant, tsList, cache, gcache, routeMap)
			if err != nil {
				return err
			}
		}
		if cmd.GoOut != "" {
			err = genGoSDK(mname, cmd.GoOut, constant, tsList, cache, gcache, routeMap)
			if err != nil {
				return err
			}
		}
	}

	if cmd.PHPOut != "" {
		err = genPHPSDKFactory(cmd.PHPOut, svcList)
		f.GetLog().Donef("generate php sdk successful. !:)")
		if err != nil {
			return err
		}
	}
	if cmd.GoOut != "" {
		err = genGoSDKFactory(cmd.GoOut, svcList)
		f.GetLog().Donef("generate go sdk successful. !:)")
		if err != nil {
			return err
		}
	}

	return nil
}

func makeTypeSpecList(file string) (string, map[string][]string, map[string]typeSpec, map[string]fieldMap, error) {
	fileTree, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
	if err != nil {
		return "", nil, nil, nil, err
	}

	tsList := make(map[string]typeSpec)
	cache := make(map[string]fieldMap)
	constant := make(map[string][]string)
	mname := str.ToCamel(strings.Replace(fileTree.Name.String(), "type", "", 1))
	for _, decl := range fileTree.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			var preKey string
			for _, spec := range genDecl.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					if vs.Values != nil {
						switch vs.Values[0].(type) {
						case *ast.Ident:
							if vs.Values[0].(*ast.Ident).String() == "iota" {
								constant["iota"] = []string{vs.Names[0].String()}
								preKey = "iota"
							} else {
								constant[vs.Names[0].String()] = []string{vs.Values[0].(*ast.Ident).String()}
							}
						case *ast.BasicLit:
							constant[vs.Names[0].String()] = []string{vs.Values[0].(*ast.BasicLit).Value}
						case *ast.BinaryExpr:
							lv := vs.Values[0].(*ast.BinaryExpr).X.(*ast.BasicLit).Value
							op := vs.Values[0].(*ast.BinaryExpr).Op.String()
							rexpr := vs.Values[0].(*ast.BinaryExpr).Y
							var rv string
							switch rexpr.(type) {
							case *ast.Ident:
								rv = rexpr.(*ast.Ident).String()
							case *ast.BasicLit:
								rv = rexpr.(*ast.BasicLit).Value
							}
							constant[lv+":"+op+":"+rv] = []string{vs.Names[0].String()}
							preKey = lv + ":" + op + ":" + rv
						}
					} else {
						constant[preKey] = append(constant[preKey], vs.Names[0].String())
					}
				}
				options := make(map[string][]string)
				if ts, ok := spec.(*ast.TypeSpec); ok {
					tsName := ts.Name.String()
					if strings.ToLower(tsName) == strings.ToLower(mname) {
						mname = tsName
					}
					switch ts.Type.(type) {
					case *ast.ArrayType:
						objName := ts.Type.(*ast.ArrayType).Elt.(*ast.Ident).Name
						options[tsName+":"+objName] = []string{arrayType, ""}
					case *ast.StructType:
						fields := ts.Type.(*ast.StructType).Fields.List
						fillFields(fields, options, cache, tsList)
					}
					stdName := strings.TrimSuffix(strings.TrimSuffix(tsName, "Request"), "Response")
					if strings.Contains(tsName, "Request") || strings.Contains(tsName, "Response") {
						if ts, ok := tsList[stdName]; ok {
							if strings.Contains(tsName, "Request") {
								ts.ReqName = tsName
								ts.options = options
							} else if strings.Contains(tsName, "Response") {
								ts.ResName = tsName
								ts.result = options
							}
							tsList[stdName] = ts
						} else {
							ts := typeSpec{
								method: strings.Replace(stdName, mname, "", 1),
							}
							if strings.Contains(tsName, "Request") {
								ts.ReqName = tsName
								ts.options = options
							} else if strings.Contains(tsName, "Response") {
								ts.ResName = tsName
								ts.result = options
							}
							tsList[stdName] = ts
						}
					} else {
						cache[stdName] = options
					}
				}
			}
		}
	}

	for stdName, ts := range tsList {
		for name, option := range ts.options {
			if option == nil {
				if opt, ok := cache[name]; ok {
					ts.options = opt
					for fd, params := range opt {
						ts.options[fd] = params
					}
					delete(ts.options, name)
				}
			}
		}
		for name, result := range ts.result {
			if result == nil {
				if opt, ok := cache[name]; ok {
					for fd, params := range opt {
						ts.result[fd] = params
					}
					delete(ts.result, name)
				}
			}
		}
		tsList[stdName] = ts
	}

	for stdName, ts := range tsList {
		if _, ok := cache[ts.ReqName]; ok {
			cache[stdName] = ts.options
			delete(cache, ts.ReqName)
		}
		if _, ok := cache[ts.ResName]; ok {
			cache[stdName] = ts.result
			delete(cache, ts.ResName)
		}
	}

	return mname, constant, tsList, cache, nil
}

func fillFields(fields []*ast.Field, options fieldMap, cache map[string]fieldMap, tsList map[string]typeSpec) {
	for _, field := range fields {
		switch field.Type.(type) {
		case *ast.SelectorExpr:
			packageName := field.Type.(*ast.SelectorExpr).X.(*ast.Ident).String()
			if _, ok := internalRefs[packageName]; !ok {
				panic("api define just can references typespec package!")
			}
			if packageName == "orm" || packageName == "time" {
				options[field.Names[0].String()] = []string{"string", field.Tag.Value}
				continue
			}
			typName := field.Type.(*ast.SelectorExpr).Sel.String()
			if field.Names != nil {
				options[field.Names[0].String()+":"+typName] = []string{identType, field.Tag.Value}
			} else {
				options[typName] = nil
			}
		case *ast.Ident:
			if field.Type.(*ast.Ident).Obj == nil {
				options[field.Names[0].String()] = []string{field.Type.(*ast.Ident).String(), field.Tag.Value}
			} else {
				typName := field.Type.(*ast.Ident).String()
				if field.Names != nil {
					options[field.Names[0].String()+":"+typName] = []string{identType, field.Tag.Value}
				} else {
					options[typName] = nil
				}
			}
		case *ast.MapType:
			key := field.Type.(*ast.MapType).Key.(*ast.Ident).String()
			value := field.Type.(*ast.MapType).Value
			switch value.(type) {
			case *ast.Ident:
				vname := value.(*ast.Ident).String()
				if isGenType(vname) {
					options[field.Names[0].String()] = []string{fmt.Sprintf("map[%s]%s", key, value), field.Tag.Value}
				} else {
					options[field.Names[0].String()+":"+key+"|"+vname] = []string{mapType, field.Tag.Value}
					stdName := strings.TrimSuffix(strings.TrimSuffix(vname, "Request"), "Response")
					if ts, ok := tsList[stdName]; ok {
						if strings.Contains(vname, "Request") {
							cache[vname] = ts.options
						} else if strings.Contains(vname, "Response") {
							cache[vname] = ts.result
						}
					} else {
						if _, ok := cache[vname]; !ok {
							cache[vname] = nil
						}
					}
				}
			case *ast.ArrayType:
				elt := value.(*ast.ArrayType).Elt
				switch elt.(type) {
				case *ast.Ident:
					vname := elt.(*ast.Ident).String()
					if isGenType(vname) {
						options[field.Names[0].String()] = []string{fmt.Sprintf("map[%s][]%s", key, elt.(*ast.Ident).String()), field.Tag.Value}
					} else {
						options[field.Names[0].String()+":"+key+"|"+vname] = []string{arrayMapType, field.Tag.Value}
						stdName := strings.TrimSuffix(strings.TrimSuffix(vname, "Request"), "Response")
						if ts, ok := tsList[stdName]; ok {
							if strings.Contains(vname, "Request") {
								cache[vname] = ts.options
							} else if strings.Contains(vname, "Response") {
								cache[vname] = ts.result
							}
						} else {
							if _, ok := cache[vname]; !ok {
								cache[vname] = nil
							}
						}
					}
				default:
					panic("unsupported deep type!")
				}
			}
		case *ast.StructType:
			fieldList := field.Type.(*ast.StructType).Fields.List
			innerOptions := make(fieldMap)
			innerTsList := make(map[string]typeSpec)
			fillFields(fieldList, innerOptions, cache, innerTsList)
			options[field.Names[0].String()+":"+field.Names[0].String()+"0"] = []string{identType, field.Tag.Value}
			cache[field.Names[0].String()+"0"] = innerOptions
		case *ast.ArrayType:
			elt := field.Type.(*ast.ArrayType).Elt
			switch elt.(type) {
			case *ast.MapType:
				panic("unsupported type!")
			case *ast.StructType:
				fieldList := elt.(*ast.StructType).Fields.List
				innerOptions := make(fieldMap)
				innerTsList := make(map[string]typeSpec)
				fillFields(fieldList, innerOptions, cache, innerTsList)
				options[field.Names[0].String()+":"+field.Names[0].String()+"0"] = []string{arrayType, field.Tag.Value}
				cache[field.Names[0].String()+"0"] = innerOptions
			case *ast.Ident:
				typName := elt.(*ast.Ident).String()
				if !isGenType(typName) {
					stdName := strings.Replace(strings.Replace(typName, "Request", "", 1), "Response", "", 1)
					if ts, ok := tsList[stdName]; ok {
						if strings.Contains(typName, "Request") {
							cache[typName] = ts.options
						} else if strings.Contains(typName, "Response") {
							cache[typName] = ts.result
						}
					} else {
						if _, ok := cache[typName]; !ok {
							cache[typName] = nil
						}
					}
					options[field.Names[0].String()+":"+stdName] = []string{arrayType, field.Tag.Value}
				} else {
					options[field.Names[0].String()] = []string{"[]" + typName, field.Tag.Value}
				}
			}
		}
	}
}

func makeRouteMap(file string) (fieldMap, error) {
	fileTree, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	routeMap := make(fieldMap)

	for _, decl := range fileTree.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Doc != nil {
				comments := funcDecl.Doc.List
				var httpMethod string
				var route string
				for _, comment := range comments {
					commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "//"))
					if !strings.Contains(commentLine, "@") {
						continue
					}
					fields := strings.Fields(commentLine)
					attribute := fields[0]
					lowerAttribute := strings.ToLower(attribute)
					if lowerAttribute == "@router" {
						route = fields[1]
						httpMethod = strings.TrimRight(strings.TrimLeft(fields[2], "["), "]")
						routeMap[funcDecl.Name.String()] = []string{route, httpMethod}
					}
				}
			}
		}
	}

	return routeMap, nil
}

func genGoSDKFactory(output string, svcList map[string]string) error {
	factoryPath := fmt.Sprintf("%s/factory.go", output)
	os.Remove(factoryPath)
	tpl := getGOSDKFactoryTpl(svcList)
	fs, err := os.OpenFile(factoryPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()
	fs.WriteString(tpl)

	return nil
}

func genGoSDK(mname, output string, constant map[string][]string, tsList map[string]typeSpec, cache map[string]fieldMap, gcache map[string]fieldMap, routeMap fieldMap) error {
	filePath := fmt.Sprintf("%s/%s.go", output, strings.ToLower(mname))
	os.Remove(filePath)
	tpl := getGOSDKFileTpl(mname, constant, tsList, cache, gcache, routeMap)
	fs, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()
	fmt.Println(fmt.Sprintf("%s Generating %s", time.Now().Format("2006/01/02 15:04:05"), filePath))
	fs.WriteString(tpl)

	return nil
}

func genPHPSDKFactory(output string, svcList map[string]string) error {
	factoryPath := fmt.Sprintf("%s/ClientFactory.php", output)
	os.Remove(factoryPath)
	tpl := getPHPSDKFactoryTpl(svcList)
	fs, err := os.OpenFile(factoryPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()
	fs.WriteString(tpl)

	return nil
}

func getPHPSDKFactoryTpl(svcList map[string]string) string {
	tpl := "<?php\n\n"
	tpl += "namespace App\\SDK;\n\n"
	tpl += "use BundleLib\\GuzzleBundle\\Client;\nuse Psr\\Http\\Message\\ResponseInterface;\nuse Symfony\\Component\\HttpFoundation\\Response;\n\n"
	tpl += "class ClientFactory extends Client\n{\n"
	for _, svcName := range svcList {
		tpl += fmt.Sprintf("\tprivate $%s;\n", str.ToLowerCamelCase(svcName))
	}
	tpl += "\n\tpublic function __construct(array $config = [])\n\t{\n"
	for _, svcName := range svcList {
		tpl += fmt.Sprintf("\t\t$this->%s = new %s($this);\n", str.ToLowerCamelCase(svcName), svcName)
	}
	tpl += "\t\tparent::__construct($config);\n\t}\n\n"
	for _, svcName := range svcList {
		tpl += fmt.Sprintf("\tpublic function %s()\n\t{\n", str.ToLowerCamelCase(svcName))
		tpl += fmt.Sprintf("\t\treturn $this->%s;\n\t}\n", str.ToLowerCamelCase(svcName))
	}
	tpl += `
    public function extractBody(ResponseInterface $response)
    {
        if (Response::HTTP_OK == $response->getStatusCode()) {
            $result = $response->getBody()->getContents();
            if ($result) {
                return json_decode($result, true);
            }
        }

        return [];
    }

    public function getContent(array $result)
    {
        if (!empty($result) && isset($result['status'])) {
            if (Response::HTTP_OK != $result['status']) {
                throw new \Exception($result['errorMsg'], $result['status']);
            }
            return $result['content'];
        }

        throw new \Exception('client error', 4000);
    }
`
	tpl += "\n}"

	return tpl
}

func genPHPSDK(mname, output string, constant map[string][]string, tsList map[string]typeSpec, cache map[string]fieldMap, gcache map[string]fieldMap, routeMap fieldMap) error {
	filePath := fmt.Sprintf("%s/%sService.php", output, mname)
	os.Remove(filePath)
	tpl := getPHPSDKFileTpl(mname, constant, tsList, cache, gcache, routeMap)
	fs, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()
	fmt.Println(fmt.Sprintf("%s Generating %s", time.Now().Format("2006/01/02 15:04:05"), filePath))
	fs.WriteString(tpl)
	return nil
}

func getPHPSDKFileTpl(mname string, constant map[string][]string, tsList map[string]typeSpec, cache map[string]fieldMap, gcache map[string]fieldMap, routeMap fieldMap) string {
	tpl := "<?php\n\n"
	tpl += "namespace App\\SDK;\n\n"
	tpl += fmt.Sprintf("class %sService\n{\n", mname)
	if constant != nil {
		for name, ks := range constant {
			if name == "iota" {
				for i, v := range ks {
					tpl += fmt.Sprintf("\tconst %s = %d;\n", v, i)
				}
			} else if strings.Contains(name, ":") {
				parts := strings.Split(name, ":")
				for i, v := range ks {
					tpl += fmt.Sprintf("\tconst %s = %s %s %d;\n", v, parts[0], parts[1], i)
				}
			} else {
				tpl += fmt.Sprintf("\tconst %s = %s;\n", name, ks[0])
			}
		}
	}
	tpl += "\tprivate $client;\n\n"
	tpl += "\tpublic function __construct(ClientFactory $client)\n\t{\n"
	tpl += "\t\t$this->client = $client;\n\t}\n\n"
	keys := make([]string, len(tsList))
	i := 0
	for k := range tsList {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for i, stdName := range keys {
		ts := tsList[stdName]
		route := routeMap[stdName]
		if route == nil {
			continue
		}
		path := route[0]
		method := str.ToCamel(strings.ToLower(route[1]))
		params := make([]string, 0, len(ts.options))
		query := make(map[string]string)
		if ts.options != nil {
			for k, v := range ts.options {
				if v == nil {
					continue
				}
				var typ string
				switch v[0] {
				case "int", "int32", "int64":
					typ = "int"
				case "float", "float32", "float64":
					typ = "float"
				case "[]int", "[]int32", "[]int64":
					typ = "array|intval"
				case "[]float", "[]float32", "[]float64":
					typ = "array|floatval"
				case "[]string":
					typ = "array|strval"
				default:
					typ = v[0]
				}
				camelCaseK := str.ToLowerCamelCase(k)
				params = append(params, fmt.Sprintf("%s $%s", typ, camelCaseK))
				parts := strings.Split(typ, "|")
				if len(parts) == 2 {
					query[camelCaseK] = fmt.Sprintf("array_map('%s', (%s) $options['%s']),", parts[1], parts[0], camelCaseK)
				} else {
					query[camelCaseK] = fmt.Sprintf("(%s) $options['%s'],", typ, camelCaseK)
				}
			}
			if method == "Get" {
				tpl += fmt.Sprintf("\tpublic function %s(%s)\n\t{\n", str.ToLowerCamelCase(ts.method), strings.Join(params, ", "))
			} else {
				tpl += fmt.Sprintf("\tpublic function %s(array $options)\n\t{\n", str.ToLowerCamelCase(ts.method))
			}
		} else {
			tpl += fmt.Sprintf("\tpublic function %s()\n\t{\n", str.ToLowerCamelCase(ts.method))
		}
		tpl += fmt.Sprintf("\t\t$response = $this->client->%s('%s'", strings.ToLower(route[1]), path)
		if len(query) > 0 {
			if method == "Get" {
				tpl += fmt.Sprintf(", [\n\t\t\t'query' => [")
			} else {
				tpl += fmt.Sprintf(", [\n\t\t\t'json' => [")
			}
			for k, v := range query {
				if strings.Contains(k, ":") {
					parts := strings.Split(k, ":")
					if !isGenType(parts[1]) {
						tpl += fmt.Sprintf("\n\t\t\t\t'%s' => [\n", parts[0])
						tpl = recursiveStructFillPHPTpl(tpl, 5, cache[parts[1]], cache, gcache)
						tpl += fmt.Sprintf("\n\t\t\t\t]")
					}
				} else {
					tpl += fmt.Sprintf("\n\t\t\t\t'%s' => %s", k, v)
				}
			}
			tpl += "\n\t\t\t]\n\t\t]);\n\n"
		} else {
			tpl += ");\n\n"
		}
		tpl += "\t\treturn $this->client->extractBody($response);\n\t}\n"
		if i < len(tsList)-1 {
			tpl += "\n"
		}
	}
	tpl += "}"

	return tpl
}

func getGOSDKFactoryTpl(svcList map[string]string) string {
	tpl := "package sdk\n\n"
	tpl += "import (\n\t\"context\"\n\n\t\"gitlab.idc.xiaozhu.com/xz/lib/component/xzapi\"\n)\n\n"
	tpl += "type Client struct {\n"
	tpl += "\txzapi.Client\n\n"
	for short, service := range svcList {
		tpl += fmt.Sprintf("\t%s *%s\n", short, service)
	}
	tpl += "}\n\n"
	tpl += "type Option func(client *Client)\n\n"
	tpl += "func NewClient(ctx context.Context, opt ...Option) *Client {\n"
	tpl += "\treturn newClient(ctx, opt...)\n}\n\n"
	tpl += "func newClient(ctx context.Context, opt ...Option) *Client {\n"
	tpl += "\tc := &Client{}\n\tc.Ctx = ctx\n\n"
	tpl += "\tfor _, o := range opt {\n"
	tpl += "\t\to(c)\n\t}\n\n"
	for short, service := range svcList {
		tpl += fmt.Sprintf("\tc.%s =&%s{Client:c}\n", short, service)
	}
	tpl += "\treturn c\n}\n\n"
	tpl += "func WithName(name string) Option {\n"
	tpl += "\treturn func(client *Client) {\n"
	tpl += "\t\tclient.Name = name\n\t}\n}"

	return tpl
}

func getGOSDKFileTpl(mname string, constant map[string][]string, tsList map[string]typeSpec, cache map[string]fieldMap, gcache map[string]fieldMap, routeMap fieldMap) string {
	var needImportQuery bool
	for _, param := range routeMap {
		if strings.ToLower(param[1]) == "get" {
			needImportQuery = true
			break
		}
	}
	tpl := "package sdk\n\n"
	tpl += "import ("
	tpl += "\n\t\"github.com/go-season/common/client\"\n"
	if needImportQuery {
		tpl += "\t\"github.com/google/go-querystring/query\"\n"
	}
	tpl += ")\n\n"
	if constant != nil {
		for name, ks := range constant {
			if name == "iota" {
				tpl += fmt.Sprintf("const (\n\t%s = iota", ks[0])
				for i, k := range ks {
					if i == 0 {
						continue
					}
					tpl += fmt.Sprintf("\n\t%s", k)
				}
				tpl += "\n)\n\n"
			} else if strings.Contains(name, ":") {
				parts := strings.Split(name, ":")
				tpl += fmt.Sprintf("const (\n\t%s = %s %s %s", ks[0], parts[0], parts[1], parts[2])
				for i, k := range ks {
					if i == 0 {
						continue
					}
					tpl += fmt.Sprintf("\n\t%s", k)
				}
				tpl += "\n)\n\n"
			} else {
				tpl += fmt.Sprintf("const %s = %s\n\n", name, ks[0])
			}
		}
	}
	tpl += fmt.Sprintf("type %sService struct {\n\t*Client \n}\n\n", mname)
	keys := make([]string, len(tsList))
	i := 0
	for k := range tsList {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, stdName := range keys {
		ts := tsList[stdName]
		needClose := true
		if cacheName, typ, ok := isNonStructType(ts.options); ok {
			if !isGenType(cacheName) {
				if typ == arrayType {
					tpl += fmt.Sprintf("type %sOptions []struct {\n", stdName)
				} else {
					tpl += fmt.Sprintf("type %sOptions struct {\n", stdName)
				}
				tpl = recursiveStructFillGoTpl(tpl, 1, cache[cacheName], cache, gcache)
			} else {
				if typ == arrayType {
					tpl += fmt.Sprintf("type %sOptions []%s\n\n", stdName, cacheName)
				} else {
					tpl += fmt.Sprintf("type %sOptions %s\n\n", stdName, cacheName)
				}
				needClose = false
			}
		} else {
			tpl += fmt.Sprintf("type %sOptions struct {\n", stdName)
			tpl = recursiveStructFillGoTpl(tpl, 1, ts.options, cache, gcache)
		}
		if needClose {
			tpl += fmt.Sprintf("}\n")
		}
		needClose = true
		if cacheName, typ, ok := isNonStructType(ts.result); ok {
			if !isGenType(cacheName) {
				if typ == arrayType {
					tpl += fmt.Sprintf("type %sResult []struct {\n", stdName)
				} else {
					tpl += fmt.Sprintf("type %sResult struct {\n", stdName)
				}
				parts := strings.Split(cacheName, "|")
				if len(parts) == 2 {
					tpl = recursiveStructFillGoTpl(tpl, 1, ts.result, cache, gcache)
				} else {
					tpl = recursiveStructFillGoTpl(tpl, 1, cache[cacheName], cache, gcache)
				}
			} else {
				if typ == arrayType {
					tpl += fmt.Sprintf("type %sResult []%s\n\n", stdName, cacheName)
				} else {
					tpl += fmt.Sprintf("type %sResult %s\n\n", stdName, cacheName)
				}
				needClose = false
			}
		} else {
			tpl += fmt.Sprintf("type %sResult struct {\n", stdName)
			tpl = recursiveStructFillGoTpl(tpl, 1, ts.result, cache, gcache)
		}
		if needClose {
			tpl += fmt.Sprintf("}\n\n")
		}
		tpl += fmt.Sprintf("func (%s *%sService) %s(opt *%sOptions) (%sResult, error) {\n", strings.ToLower(mname), mname, ts.method, stdName, stdName)
		tpl += fmt.Sprintf("\tvar (\n\t\tresult %sResult\n\t\terr error\n\t)\n", stdName)
		if route, ok := routeMap[stdName]; ok {
			path := route[0]
			method := str.ToCamel(strings.ToLower(route[1]))
			optK := "JSON"
			optV := "opt"
			if method == "Get" || strings.Contains(stdName, "Delete") || strings.Contains(stdName, "Remove") {
				tpl += fmt.Sprintf("\tv, _ := query.Values(opt)\n")
				optK = "Query"
				optV = "v.Encode()"
			}
			tpl += fmt.Sprintf("\t_, err = %s.ClientWithParseContent(&result).%s(\"%s\", client.Options{\n\t\t%s: %s,\n\t})\n\n",
				strings.ToLower(mname),
				method,
				path,
				optK,
				optV)

		}
		tpl += "\treturn result, err\n}\n\n"
	}

	return tpl
}

func isNonStructType(options map[string][]string) (string, string, bool) {
	for name, params := range options {
		if strings.Contains(name, "Request") || strings.Contains(name, "Response") {
			if strings.Contains(name, ":") {
				parts := strings.Split(name, ":")
				return parts[1], params[0], true
			}
		}
	}

	return "", "", false
}

func recursiveStructFillPHPTpl(tpl string, depth int, fmap map[string][]string, cache map[string]fieldMap, gcache map[string]fieldMap) string {
	tabSize := strings.Repeat("\t", depth)
	for field, params := range fmap {
		if strings.Contains(field, ":") {
			parts := strings.Split(field, ":")
			if ffmap, ok := cache[parts[1]]; ok {
				if params[0] == arrayType {
					// wait implementation
				} else {
					tpl += fmt.Sprintf("%s'%s' => [\n", tabSize, str.ToLowerCamelCase(parts[0]))
				}
				tpl = recursiveStructFillPHPTpl(tpl, depth+1, ffmap, cache, gcache)
			} else if gcache != nil {
				if ffmap, ok := gcache[parts[1]]; ok {
					if params[0] == arrayType {
						// wait implementation
					} else {
						tpl += fmt.Sprintf("%s'%s' => [\n", tabSize, str.ToLowerCamelCase(parts[0]))
					}
					tpl = recursiveStructFillPHPTpl(tpl, depth+1, ffmap, cache, gcache)
				}
			}
		} else {
			if params == nil {
				if fields, ok := cache[field]; ok {
					tpl = recursiveStructFillPHPTpl(tpl, depth, fields, cache, gcache)
				} else if gcache != nil {
					if fields, ok := gcache[field]; ok {
						tpl = recursiveStructFillPHPTpl(tpl, depth, fields, cache, gcache)
					}
				}
			} else {
				tpl += fmt.Sprintf("%s'%s' => (%s) $options['%s'],\n", tabSize, str.ToLowerCamelCase(field), params[0], str.ToLowerCamelCase(field))
			}
		}

		if params != nil && strings.Contains(field, ":") {
			tpl += fmt.Sprintf("%s]\n", tabSize)
		}
	}

	return tpl
}

func recursiveStructFillGoTpl(tpl string, depth int, fmap map[string][]string, cache map[string]fieldMap, gcache map[string]fieldMap) string {
	tabSize := strings.Repeat("\t", depth)
	keys := make([]string, len(fmap))
	i := 0
	for k := range fmap {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, field := range keys {
		params := fmap[field]
		if strings.Contains(field, ":") {
			parts := strings.Split(field, ":")
			parts2 := strings.Split(parts[1], "|")
			if len(parts2) == 2 {
				cacheName := strings.TrimSuffix(strings.TrimSuffix(parts2[1], "Request"), "Response")
				if ffmap, ok := cache[cacheName]; ok {
					if params[0] == mapType {
						tpl += fmt.Sprintf("%s%s map[%s]struct{\n", tabSize, parts[0], parts2[0])
					} else if params[0] == arrayMapType {
						tpl += fmt.Sprintf("%s%s map[%s][]struct{\n", tabSize, parts[0], parts2[0])
					}
					tpl = recursiveStructFillGoTpl(tpl, depth+1, ffmap, cache, gcache)
				}
			} else {
				if ffmap, ok := cache[parts[1]]; ok {
					if params[0] == arrayType {
						tpl += fmt.Sprintf("%s%s []struct{\n", tabSize, parts[0])
					} else {
						tpl += fmt.Sprintf("%s%s struct{\n", tabSize, parts[0])
					}
					tpl = recursiveStructFillGoTpl(tpl, depth+1, ffmap, cache, gcache)
				} else if gcache != nil {
					if ffmap, ok := gcache[parts[1]]; ok {
						if params[0] == arrayType {
							tpl += fmt.Sprintf("%s%s []struct{\n", tabSize, parts[0])
						} else {
							tpl += fmt.Sprintf("%s%s struct{\n", tabSize, parts[0])
						}
						tpl = recursiveStructFillGoTpl(tpl, depth+1, ffmap, cache, gcache)
					}
				}
			}
		} else {
			if params == nil {
				if fields, ok := cache[field]; ok {
					tpl = recursiveStructFillGoTpl(tpl, depth, fields, cache, gcache)
				} else if gcache != nil {
					if fields, ok := gcache[field]; ok {
						tpl = recursiveStructFillGoTpl(tpl, depth, fields, cache, gcache)
					}
				}
			} else {
				tagStr := params[1]
				tag := reflect.StructTag(strings.Trim(tagStr, "`"))
				if tag.Get("form") != "" {
					name := tag.Get("form")
					tagStr = fmt.Sprintf("`%s %s`", strings.Trim(tagStr, "`"), fmt.Sprintf("url:\"%s\"", name))
				}
				tpl += fmt.Sprintf("%s%s %s %s\n", tabSize, field, params[0], tagStr)
			}
		}
		if params != nil && strings.Contains(field, ":") {
			tpl += fmt.Sprintf("%s} %s\n", tabSize, params[1])
		}
	}

	return tpl
}

func isGenType(name string) bool {
	switch name {
	case "int", "int32", "int64", "string", "float", "float32", "float64", "bool":
		return true
	}

	return false
}
