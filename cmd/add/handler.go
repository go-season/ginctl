package add

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/go-season/ginctl/pkg/util/str"

	"github.com/go-season/ginctl/pkg/util"

	"github.com/go-season/ginctl/cmd/route"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type handlerCmd struct {
	method       string
	restPath     string
	typeSpecPath string
	appPath      string
	svcPath      string
	routePath    string
	appAlias     string
	appObjName   string
	specName     string
	description  string
	dirName      string
	fileName     string
	handleName   string
}

func newHandlerCmd(f factory.Factory) *cobra.Command {
	cmd := &handlerCmd{}

	addHandlerCmd := &cobra.Command{
		Use:   "handler [-d Dir] [-f File] HandleName",
		Short: "为`rest`层添加一个handler相关模板代码",
		Long: `
为rest层添加一个handler模板代码，同时生成req定义，application，service模板代码.

命令样例:
ginctl add handler -d user -f user UpdateUser
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	addHandlerCmd.Flags().StringVarP(&cmd.dirName, "dir", "d", "", "Specified a rest dir that want to add handler.")
	addHandlerCmd.Flags().StringVarP(&cmd.fileName, "file", "f", "", "Specified a rest file that want to add handler.")

	return addHandlerCmd
}

func (cmd *handlerCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if cmd.dirName != "" {
		if err := checkDir(cmd.dirName); err != nil {
			return err
		}
		dirRelPath := fmt.Sprintf("%s/api/rest/%s", wd, cmd.dirName)
		exists, err := file.PathExists(dirRelPath)
		if err != nil {
			return err
		}

		if !exists {
			err := os.Mkdir(dirRelPath, 0755)
			if err != nil {
				return err
			}
		}
		if cmd.fileName == "" {
			return errors.New("you must specified dir and file at same time")
		}
		fileName := strings.TrimSuffix(cmd.fileName, ".go")
		cmd.restPath = fmt.Sprintf("%s/%s.go", dirRelPath, fileName)
		exists, err = file.PathExists(cmd.restPath)
		if err != nil {
			return err
		}
		if !exists {
			if err = createRestImport(cmd.restPath); err != nil {
				return err
			}

			// for typespec
			dir := path.Dir(cmd.restPath)
			fi := path.Base(cmd.restPath)
			specPath := strings.Replace(dir, "rest", "typespec", 1) + "type/" + fi
			exists, err = file.PathExists(path.Dir(specPath))
			if err != nil {
				return err
			}
			if !exists {
				err = os.Mkdir(path.Dir(specPath), 0755)
				if err != nil {
					return err
				}
			}
			if err = createTypeSpecImport(specPath); err != nil {
				return err
			}

			// for app
			appPath := fmt.Sprintf("%s/application/%s/%s", wd, util.GetPkgBaseName(path.Dir(cmd.restPath)), fi)
			exists, err = file.PathExists(path.Dir(appPath))
			if err != nil {
				return err
			}
			if !exists {
				err = os.Mkdir(path.Dir(appPath), 0755)
			}
			if err = createApplicationImport(appPath); err != nil {
				return err
			}

			// for service
			servicePath := fmt.Sprintf("%s/service/%s", wd, fi)
			if err = createServiceImport(servicePath, util.GetPkgBaseName(path.Dir(cmd.restPath))); err != nil {
				return err
			}
		}
	}

	cmd.handleName = args[0]
	err = cmd.interactive(wd, f)
	if err != nil {
		return err
	}

	funcName := args[0]
	funcName = strings.ToUpper(funcName[0:1]) + funcName[1:]
	restTpl := getRestTpl()
	restHandler := fmt.Sprintf(restTpl,
		funcName,
		cmd.description,
		cmd.routePath,
		cmd.method,
		funcName,
		cmd.specName,
		funcName+"Request",
		cmd.specName,
		funcName+"Response",
		funcName,
		cmd.appAlias,
		cmd.appObjName,
		funcName,
		funcName,
		"%v",
	)

	f.GetLog().WriteString("\n")
	f.GetLog().Infof("crafting handler %s in %s", ansi.Color(funcName, "cyan+b"), ansi.Color(cmd.restPath, "cyan+b"))
	fs, err := os.OpenFile(cmd.restPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()

	_, err = fs.WriteString(restHandler)
	if err != nil {
		return err
	}
	f.GetLog().Done("create handler successful.")

	tpyTpl := getTypeSpecTpl()
	typ := fmt.Sprintf(tpyTpl,
		funcName,
		funcName,
	)
	f.GetLog().WriteString("\n")
	f.GetLog().Infof("crafting req: %s, resp %s in %s", ansi.Color(funcName+"Request", "cyan+b"), ansi.Color(funcName+"Response", "cyan+b"), ansi.Color(cmd.typeSpecPath, "cyan+b"))
	fs, err = os.OpenFile(cmd.typeSpecPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()

	_, err = fs.WriteString(typ)
	if err != nil {
		return err
	}
	f.GetLog().Done("create req/resp spec successful.")

	appTpl := getAppTpl()
	appHandler := fmt.Sprintf(appTpl,
		funcName,
		strings.ToLower(cmd.appObjName[0:1]),
		cmd.appObjName,
		funcName,
		cmd.specName,
		funcName+"Request",
		cmd.specName,
		funcName+"Response",
		strings.ToLower(cmd.appObjName[0:1])+cmd.appObjName[1:],
		cmd.appObjName,
		strings.ToLower(cmd.appObjName[0:1])+cmd.appObjName[1:],
		funcName,
	)
	f.GetLog().WriteString("\n")
	f.GetLog().Infof("crafting %s in %s", ansi.Color(funcName, "cyan+b"), ansi.Color(cmd.appPath, "cyan+b"))
	fs, err = os.OpenFile(cmd.appPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()

	_, err = fs.WriteString(appHandler)
	if err != nil {
		return err
	}
	f.GetLog().Done("create app successful.")

	svcTpl := getServiceTpl()
	svcHandler := fmt.Sprintf(svcTpl,
		funcName,
		strings.ToLower(cmd.appObjName[0:1]),
		cmd.appObjName,
		funcName,
	)
	f.GetLog().WriteString("\n")
	f.GetLog().Infof("crafting %s in %s", ansi.Color(funcName, "cyan+b"), ansi.Color(cmd.svcPath, "cyan+b"))
	fs, err = os.OpenFile(cmd.svcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fs.Close()

	_, err = fs.WriteString(svcHandler)
	if err != nil {
		return err
	}
	f.GetLog().Done("create service successful.")

	f.GetLog().WriteString("\n")
	f.GetLog().Infof("refresh route...")
	routeRefreshCmd := route.NewRefreshCmd(f)
	err = routeRefreshCmd.RunE(cobraCmd, []string{})
	if err != nil {
		return err
	}
	f.GetLog().Donef("refresh route completed, you can filling your logic now! :) :) :)")

	return nil
}

var whiteListMap = map[string]bool{
	"api.go":    true,
	"router.go": true,
}

func (cmd *handlerCmd) interactive(wd string, f factory.Factory) error {
	if cmd.restPath == "" {
		fsList := make([]string, 0)
		err := filepath.Walk(fmt.Sprintf("%s/api/rest", wd), func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				if ignore, ok := whiteListMap[info.Name()]; !ok || ok && !ignore {
					fsList = append(fsList, strings.TrimPrefix(path, wd+"/"))
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		choice, err := f.GetLog().Question(&log.QuestionOptions{
			Question:     "Which file do you want to append method?",
			DefaultValue: "",
			Options:      fsList,
		})
		if err != nil {
			return err
		}
		cmd.restPath = wd + "/" + choice
	}

	choice, err := f.GetLog().Question(&log.QuestionOptions{
		Question:     "Which http method do you want use?",
		DefaultValue: "GET",
		Options: []string{
			"GET",
			"POST",
		},
	})
	if err != nil {
		return err
	}
	cmd.method = choice

	var (
		appImpPath   string
		typeSpecPath string
	)

	fileTree, err := parser.ParseFile(token.NewFileSet(), cmd.restPath, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}
	cmd.appAlias = "app"
	for _, imp := range fileTree.Imports {
		if cmd.appAlias != "app" && cmd.specName != "" {
			break
		}

		if cmd.appAlias == "app" && strings.Contains(imp.Path.Value, "application") && imp.Name != nil {
			appImpPath = imp.Path.Value
			cmd.appAlias = imp.Name.String()
		}
		if cmd.specName == "" && strings.Contains(imp.Path.Value, "typespec") {
			pos := strings.LastIndex(imp.Path.Value, "/")
			typeSpecPath = imp.Path.Value
			cmd.specName = strings.TrimRight(imp.Path.Value[pos+1:], "\"")
		}
	}

	pkg := path.Base(path.Dir(cmd.restPath))
	routePath, err := f.GetLog().Question(&log.QuestionOptions{
		Question:     "Please given the action handler a route path: ",
		DefaultValue: guessPath(pkg, cmd.handleName),
	})
	if err != nil {
		return err
	}
	cmd.routePath = strings.ToLower(routePath)

	description, err := f.GetLog().Question(&log.QuestionOptions{
		Question:     "Please given the action handler a description: ",
		DefaultValue: "",
	})
	if err != nil {
		return err
	}
	cmd.description = description

	if appImpPath == "" {
		return errors.New(fmt.Sprintf("file %s missing application layer import or missing app import alias, ignore generate handler.", ansi.Color(cmd.restPath, "cyan+b")))
	}

	appImpPath = strings.TrimPrefix(strings.Trim(appImpPath, "\""), util.GetModuleName(wd))
	typeSpecPath = strings.TrimPrefix(strings.Trim(typeSpecPath, "\""), util.GetModuleName(wd))
	lpos := strings.LastIndex(cmd.restPath, "/")
	appFileName := cmd.restPath[lpos+1:]
	appPath := strings.Trim(appImpPath, "\"") + "/" + appFileName
	fpos := strings.Index(appPath, "/")
	appPath = appPath[fpos+1:]
	cmd.appPath = wd + "/" + appPath
	cmd.svcPath = wd + "/service/" + appFileName
	typeSpecPath = strings.Trim(typeSpecPath, "\"") + "/" + appFileName
	fpos = strings.Index(typeSpecPath, "/")
	typeSpecPath = typeSpecPath[fpos+1:]
	cmd.typeSpecPath = wd + "/" + typeSpecPath

	fileTree, err = parser.ParseFile(token.NewFileSet(), cmd.appPath, nil, parser.DeclarationErrors)
	if err != nil {
		return err
	}
loop:
	for _, decl := range fileTree.Decls {
		if generalDecl, ok := decl.(*ast.GenDecl); ok && generalDecl.Tok == token.TYPE {
			for _, astSpec := range generalDecl.Specs {
				if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
					cmd.appObjName = typeSpec.Name.String()
					break loop
				}
			}
		}
	}

	return nil
}

func getTypeSpecTpl() string {
	return `
type %sRequest struct {}

type %sResponse struct {}
`
}

func getServiceTpl() string {
	return `
// %s handles actual logic
func (%s *%s) %s(ctx context.Context) error {

	return nil
}
`
}

func getAppTpl() string {
	return `
// %s assemblies services logic
func (%s *%s) %s(ctx context.Context, req *%s.%s, resp *%s.%s) error {
	%sSvc := &service.%s{}
	if err := %sSvc.%s(ctx); err != nil {
		return err
	}
	
	return nil
}
`
}

func getRestTpl() string {
	return `
// %s
// %s 
// @Router %s [%s] 
func %s(c *gin.Context) {
	var (
		req %s.%s
		resp %s.%s
		ginLog = log.GetFromGin(c)
	)
	
	err := apputil.BindReqAndValid(c, &req)
	if err != nil {
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	ginLog.Debug("%s")

	app := %s.%s{}
	ctx := server.NewContext(context.Background(), c)
	if err := app.%s(ctx, &req, &resp); err != nil {
		ginLog.Errorf("call app.%s trigger err: %s", err)
		httppkg.Error(c, http.StatusOK, err)
		return
	}
	
	httppkg.Success(c, resp)
} 
`
}

func createRestImport(file string) error {
	tpl := []byte(`package {{.SubPkgName | ToLower}}

import (
	"net/http"
	"context"

	"github.com/gin-gonic/gin"

	"github.com/go-season/common/server"
	"github.com/go-season/common/log"
    apputil "github.com/go-season/common/util/app"

	httppkg "{{.RootPkgName}}/pkg/http"
	"{{.RootPkgName}}/api/typespec/{{.RestName | ToLower}}type"
	{{.RestName | ToLower}}app "{{.RootPkgName}}/application/{{.RestName | ToLower}}"
)

`)
	rootPkgName := util.GetModuleName(path.Dir(file))
	subPkgName := path.Base(path.Dir(file))
	restName := subPkgName

	restFile, err := os.Create(file)
	if err != nil {
		return err
	}
	defer restFile.Close()

	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}

	rest := struct {
		SubPkgName  string
		RestName    string
		RootPkgName string
	}{
		SubPkgName:  subPkgName,
		RestName:    restName,
		RootPkgName: rootPkgName,
	}

	restTemplate := template.Must(template.New("rest").Funcs(funcMap).Parse(string(tpl)))
	err = restTemplate.Execute(restFile, rest)
	if err != nil {
		return err
	}

	return NormalizeFile(restFile.Name())
}

func createTypeSpecImport(file string) error {
	tpl := []byte(`package {{.SubPkgName | ToLower}}

`)

	subPkgName := path.Base(path.Dir(file))

	tsFile, err := os.Create(file)
	if err != nil {
		return err
	}
	defer tsFile.Close()

	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}

	spec := struct {
		SubPkgName string
	}{
		SubPkgName: subPkgName,
	}

	restTemplate := template.Must(template.New("ts").Funcs(funcMap).Parse(string(tpl)))
	err = restTemplate.Execute(tsFile, spec)
	if err != nil {
		return err
	}

	return NormalizeFile(tsFile.Name())
}

func createApplicationImport(file string) error {
	tpl := []byte(`package {{.SubPkgName}}

import (
	"context"
	"{{.RootPkgName}}/service"
	"{{.RootPkgName}}/api/typespec/{{.AppName | ToLower}}type"
)

type {{.ObjName | ToCamel}} struct {}

`)
	rootPkgName := util.GetModuleName(path.Dir(file))
	appName := path.Base(path.Dir(file))
	objName := strings.TrimSuffix(path.Base(file), ".go")
	subPkgName := path.Base(path.Dir(file))

	appFile, err := os.Create(file)
	if err != nil {
		return err
	}
	defer appFile.Close()

	funcMap := template.FuncMap{
		"ToCamel": str.ToCamel,
		"ToLower": strings.ToLower,
	}

	spec := struct {
		RootPkgName string
		AppName     string
		ObjName     string
		SubPkgName  string
	}{
		RootPkgName: rootPkgName,
		AppName:     appName,
		ObjName:     objName,
		SubPkgName:  subPkgName,
	}

	restTemplate := template.Must(template.New("app").Funcs(funcMap).Parse(string(tpl)))
	err = restTemplate.Execute(appFile, spec)
	if err != nil {
		return err
	}

	return NormalizeFile(appFile.Name())
}

func createServiceImport(file string, typeSpec string) error {
	tpl := []byte(`package service

import (
	"context"
)

type {{.ServiceName | ToCamel}} struct {}

`)

	rootPkgName := util.GetModuleName(path.Dir(file))
	serviceName := strings.TrimSuffix(path.Base(file), ".go")

	svcFile, err := os.Create(file)
	if err != nil {
		return err
	}
	defer svcFile.Close()

	funcMap := template.FuncMap{
		"ToCamel": str.ToCamel,
		"ToLower": strings.ToLower,
	}

	spec := struct {
		RootPkgName string
		TypeSpec    string
		ServiceName string
	}{
		RootPkgName: rootPkgName,
		TypeSpec:    typeSpec,
		ServiceName: serviceName,
	}

	restTemplate := template.Must(template.New("service").Funcs(funcMap).Parse(string(tpl)))
	err = restTemplate.Execute(svcFile, spec)
	if err != nil {
		return err
	}

	return NormalizeFile(svcFile.Name())
}

func checkDir(dir string) error {
	if strings.Contains(dir, "/") {
		return fmt.Errorf("invalid dir %s, current only support single layer dir, please change it", dir)
	}

	return nil
}

func guessPath(pkg string, funcName string) string {
	matchFirstCap := regexp.MustCompile("([a-z])([A-Z]+)")

	s := matchFirstCap.ReplaceAllString(funcName, "${1}_${2}")

	parts := strings.Split(s, "_")

	if len(parts) == 1 {
		return fmt.Sprintf("/%s/%s", pkg, strings.ToLower(strings.Join(parts, "/")))
	}

	return fmt.Sprintf("/%s/%s", strings.ToLower(strings.Join(parts[1:], "/")), strings.ToLower(parts[0]))
}
