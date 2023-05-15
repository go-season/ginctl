package add

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/go-season/ginctl/pkg/util/str"
	"github.com/go-season/ginctl/tpl"
)

type Project struct {
	AbsolutePath string
}

type Route struct {
	CreateFlag   bool
	UpdateFlag   bool
	ReadFlag     bool
	ReadListFlag bool
	DeleteFlag   bool
}

type Rest struct {
	SubPkgName  string
	RestName    string
	RootPkgName string
	*Project
	*Route
}

type Cron struct {
	SubPkgName string
	Name       string
	*Project
}

type App struct {
	AppName     string
	ShortName   string
	RootPkgName string
	*Project
	*Route
}

type TypeSpec struct {
	AppName              string
	SubPkgName           string
	Fields               string
	ListJsonTag          string
	IdFormTag            string
	IdFormTagWithDefault string
	IdJsonTag            string
	PageMode             string
	CreatedAtJsonTag     string
	UpdatedAtJsonTag     string
	PageFormTag          string
	PageSizeFormTag      string
	PageJsonTag          string
	PageSizeJsonTag      string
	PageTotalJsonTag     string
	*Project
	*Route
}

type Service struct {
	IsEmpty     bool
	ServiceName string
	ShortName   string
	RootPkgName string
	PageMode    string
	*Project
	*Route
}

type Model struct {
	ModelStruct string
	ModelName   string
	ShortName   string
	PageMode    string
	UseDB       string
	*Project
	*Route
}

func (r *Rest) Create() error {
	pkgName := strings.ToLower(r.RestName)
	restFile, err := os.Create(fmt.Sprintf("%s/api/rest/%s/%s.go", r.AbsolutePath, pkgName, str.ToSnakeCase(r.RestName)))
	if err != nil {
		return err
	}
	defer restFile.Close()

	funcMap := template.FuncMap{
		"ToCamel":  str.ToCamel,
		"ToPlural": str.ToPlural,
		"ToLower":  strings.ToLower,
	}

	restTemplate := template.Must(template.New("rest").Funcs(funcMap).Parse(string(tpl.RestTemplate())))
	err = restTemplate.Execute(restFile, r)
	if err != nil {
		return err
	}

	if err := NormalizeFile(restFile.Name()); err != nil {
		return err
	}

	return nil
}

func (t *TypeSpec) Create() error {
	pkgName := strings.ToLower(t.AppName)
	tsFile, err := os.Create(fmt.Sprintf("%s/api/typespec/%stype/%s.go", t.AbsolutePath, pkgName, str.ToSnakeCase(t.AppName)))
	if err != nil {
		return err
	}
	defer tsFile.Close()

	funcMap := template.FuncMap{
		"ToCamel":  str.ToCamel,
		"ToPlural": str.ToPlural,
		"ToLower":  strings.ToLower,
	}

	tsTemplate := template.Must(template.New("ts").Funcs(funcMap).Parse(string(tpl.TypeSpecTemplate())))
	err = tsTemplate.Execute(tsFile, t)
	if err != nil {
		return err
	}
	if err := NormalizeFile(tsFile.Name()); err != nil {
		return err
	}

	return nil
}

func (a *App) Create() error {
	pkgName := strings.ToLower(a.AppName)
	appFile, err := os.Create(fmt.Sprintf("%s/application/%s/%s.go", a.AbsolutePath, pkgName, str.ToSnakeCase(a.AppName)))
	if err != nil {
		return err
	}
	defer appFile.Close()

	funcMap := template.FuncMap{
		"ToCamel":  str.ToCamel,
		"ToPlural": str.ToPlural,
		"ToLower":  strings.ToLower,
	}

	restTemplate := template.Must(template.New("app").Funcs(funcMap).Parse(string(tpl.ApplicationTemplate())))
	err = restTemplate.Execute(appFile, a)
	if err != nil {
		return err
	}
	if err := NormalizeFile(appFile.Name()); err != nil {
		return err
	}
	return nil
}

func (s *Service) Create() error {
	svcFile, err := os.Create(fmt.Sprintf("%s/service/%s.go", s.AbsolutePath, str.ToSnakeCase(s.ServiceName)))
	if err != nil {
		return err
	}
	defer svcFile.Close()

	funcMap := template.FuncMap{
		"ToCamel":  str.ToCamel,
		"ToPlural": str.ToPlural,
		"ToLower":  strings.ToLower,
	}

	svcTemplate := template.Must(template.New("svc").Funcs(funcMap).Parse(string(tpl.ServiceTemplate())))
	if s.IsEmpty {
		svcTemplate = template.Must(template.New("svc").Funcs(funcMap).Parse(string(tpl.EmptyServiceTemplate())))
	}

	err = svcTemplate.Execute(svcFile, s)
	if err != nil {
		return err
	}
	if err := NormalizeFile(svcFile.Name()); err != nil {
		return err
	}
	return nil
}

func (m *Model) Create() error {
	mFile, err := os.Create(fmt.Sprintf("%s/model/%s.go", m.AbsolutePath, str.ToSnakeCase(m.ModelName)))
	if err != nil {
		return err
	}
	defer mFile.Close()

	funcMap := template.FuncMap{
		"ToCamel":  str.ToCamel,
		"ToPlural": str.ToPlural,
	}

	mTemplate := template.Must(template.New("m").Funcs(funcMap).Parse(string(tpl.ModelTemplate())))
	err = mTemplate.Execute(mFile, m)
	if err != nil {
		return err
	}
	if err := NormalizeFile(mFile.Name()); err != nil {
		return err
	}
	return nil
}

func (r *Cron) Create() error {
	file, err := os.Create(fmt.Sprintf("%s/cmd/cron/cmd/%s.go", r.AbsolutePath, strings.ToLower(r.Name)))
	if err != nil {
		return err
	}
	defer file.Close()

	funcMap := template.FuncMap{
		"ToCamel": str.ToCamel,
	}

	cronTemplate := template.Must(template.New("cron").Funcs(funcMap).Parse(string(tpl.CronTemplate())))
	err = cronTemplate.Execute(file, r)
	if err != nil {
		return err
	}
	return nil
}

func NormalizeFile(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(data)
	fset := token.NewFileSet()
	original := buf.Bytes()
	fileAST, err := parser.ParseFile(fset, "", original, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.SortImports(fset, fileAST)
	buf.Reset()

	(&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(buf, fset, fileAST)

	ioutil.WriteFile(file, buf.Bytes(), 0644)

	return nil
}
