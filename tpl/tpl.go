package tpl

import (
	"bytes"
)

func RestTemplate() []byte {
	return []byte(`package {{.SubPkgName | ToLower}}

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

{{ if .CreateFlag}}// 添加
// @Router /{{.RestName | ToLower}}/add [POST]
func Add{{.RestName | ToCamel}}(c *gin.Context) {
	var (
		req {{.RestName | ToLower}}type.Add{{.RestName | ToCamel}}Request
		resp {{.RestName | ToLower}}type.Add{{.RestName | ToCamel}}Response
		ginLog = log.GetFromGin(c)
	)

	err := apputil.BindReqAndValid(c, &req)
	if err != nil {
		httppkg.Error(c, http.StatusOK, err)
		return
	}
	
	ginLog.Debug("Add{{.RestName | ToCamel}}")

	app := &{{.RestName | ToLower}}app.{{.RestName | ToCamel}}{}
	ctx := server.NewContext(context.Background(), c)
	if err := app.Add{{.RestName | ToCamel}}(ctx, &req, &resp); err != nil {
		ginLog.Errorf("call app.Add{{.RestName | ToCamel}} trigger err: %v", err)
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	httppkg.Success(c, resp)
}{{end}}


{{if .ReadListFlag}}// 获取列表
// @Router /{{.RestName | ToLower}}/list [GET]
func Get{{.RestName | ToCamel}}List(c *gin.Context) {
	var (
		req {{.RestName | ToLower}}type.Get{{.RestName | ToCamel}}ListRequest
		resp {{.RestName | ToLower}}type.Get{{.RestName | ToCamel}}ListResponse
		ginLog = log.GetFromGin(c)
	)

	err := apputil.BindReqAndValid(c, &req)
	if err != nil {
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	ginLog.Debug("Get{{.RestName | ToCamel}}List")

	app := {{.RestName | ToLower}}app.{{.RestName | ToCamel}}{}
	ctx := server.NewContext(context.Background(), c)
    if err := app.Get{{.RestName | ToCamel}}List (ctx, &req, &resp); err != nil {
		ginLog.Errorf("call app.Get{{.RestName | ToCamel}}List trigger err: %v", err)
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	httppkg.Success(c, resp)
}{{end}}


{{if .ReadFlag}}// 获取详情
// @Router /{{.RestName | ToLower}}/info [GET]
func Get{{.RestName | ToCamel}}(c *gin.Context) {
	var (
		req {{.RestName | ToLower}}type.Get{{.RestName | ToCamel}}Request
		resp {{.RestName | ToLower}}type.Get{{.RestName | ToCamel}}Response
		ginLog = log.GetFromGin(c)
	)

	err := apputil.BindReqAndValid(c, &req)
	if err != nil {
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	ginLog.Debug("Get{{.RestName | ToCamel}}s")

	app := {{.RestName | ToLower}}app.{{.RestName | ToCamel}}{}
	ctx := server.NewContext(context.Background(), c)
	if err := app.Get{{.RestName | ToCamel}}(ctx, &req, &resp); err != nil {
		ginLog.Errorf("call app.Get{{.RestName | ToCamel}} trigger err: %v", err)
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	httppkg.Success(c, resp)
}{{end}}


{{if .UpdateFlag}}// 更新
// @Router /{{.RestName | ToLower}}/update [POST]
func Update{{.RestName | ToCamel}}(c *gin.Context) {
	var (
		req {{.RestName | ToLower}}type.Update{{.RestName | ToCamel}}Request
		resp {{.RestName | ToLower}}type.Update{{.RestName | ToCamel}}Response
		ginLog = log.GetFromGin(c)
	)

	err := apputil.BindReqAndValid(c, &req)
	if err != nil {
		httppkg.Error(c, http.StatusOK, err)
		return
	}
	
	ginLog.Debug("Update{{.RestName | ToCamel}}")
	
	app := {{.RestName | ToLower}}app.{{.RestName | ToCamel}}{}
	ctx := server.NewContext(context.Background(), c)
	if err := app.Update{{.RestName | ToCamel}}(ctx, &req, &resp); err != nil {
		ginLog.Errorf("call app.Update{{.RestName | ToCamel}} trigger err: %v", err)
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	httppkg.Success(c, resp)
}{{end}}


{{if .DeleteFlag}}// 删除
// @Router /{{.RestName | ToLower}}/delete [POST]
func Delete{{.RestName | ToCamel}}(c *gin.Context) {
	var (
		req {{.RestName | ToLower}}type.Delete{{.RestName | ToCamel}}Request
		resp {{.RestName | ToLower}}type.Delete{{.RestName | ToCamel}}Response
		ginLog = log.GetFromGin(c)
	)

	err := apputil.BindReqAndValid(c, &req)
	if err != nil {
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	ginLog.Debug("Delete{{.RestName | ToCamel}}")

	app := {{.RestName | ToLower}}app.{{.RestName | ToCamel}}{}
	ctx := server.NewContext(context.Background(), c)
	if err := app.Delete{{.RestName | ToCamel}}(ctx, &req, &resp); err != nil {
		ginLog.Errorf("call app.Delete{{.RestName | ToCamel}} trigger err: %v", err)
		httppkg.Error(c, http.StatusOK, err)
		return
	}

	httppkg.Success(c, resp)
}{{end}}
`)
}

func TypeSpecTemplate() []byte {
	return []byte(`package {{.SubPkgName | ToLower}}type

type {{.AppName | ToCamel}} struct {
	{{.Fields}}
}

{{if .CreateFlag}}type Add{{.AppName | ToCamel}}Request struct {
	{{.AppName | ToCamel}}
}

type Add{{.AppName | ToCamel}}Response struct {
	ID int64 {{.IdJsonTag}} // ID
}{{end}}


{{if .ReadListFlag}}type Get{{.AppName | ToCamel}}ListRequest struct {
	{{if eq .PageMode "page"}}Page     int {{.PageFormTag}} 	  // 分页页码
	PageSize int {{.PageSizeFormTag}} // 分页每页显示条数
	{{else}}Offset int {{.PageFormTag}}	 // 分页偏移
	Length int {{.PageSizeFormTag}} 	// 分页每页显示条数{{end}}
}

type Get{{.AppName | ToCamel}}ListResponse struct {
	{{if eq .PageMode "page"}}Page     int  {{.PageJsonTag}}   // 分页页码
	PageSize int  {{.PageSizeJsonTag}} // 分页每页显示条数
	{{else}}Offset int {{.PageJsonTag}}		 // 分页偏移
	Length int {{.PageSizeJsonTag}}  // 分页每页显示条数
	{{end}}Total    int64             		{{.PageTotalJsonTag}} // 总条数
	List []Get{{.AppName | ToCamel}}Response {{.ListJsonTag}}
}{{end}}


{{if .ReadFlag}}type Get{{.AppName | ToCamel}}Request struct {
	ID int64 {{.IdFormTagWithDefault}} // ID
}

type Get{{.AppName | ToCamel}}Response struct {
	ID int64	{{.IdJsonTag}} // ID
	CreatedAt string {{.CreatedAtJsonTag}} // 创建时间
	UpdatedAt string {{.UpdatedAtJsonTag}} // 更新时间
	{{.AppName | ToCamel}}
}{{end}}


{{if .UpdateFlag}}type Update{{.AppName | ToCamel}}Request struct {
	ID int64 {{.IdFormTag}} // ID
	{{.AppName | ToCamel}}
}

type Update{{.AppName | ToCamel}}Response struct {
	ID int64 {{.IdJsonTag}} // ID
}{{end}}


{{if .DeleteFlag}}type Delete{{.AppName | ToCamel}}Request struct {
	ID int64 {{.IdFormTag}} // ID
}

type Delete{{.AppName | ToCamel}}Response struct {
	ID int64 {{.IdJsonTag}} // ID
}{{end}}
`)
}

func ApplicationTemplate() []byte {
	return []byte(`package application

import (
	"context"
	"{{.RootPkgName}}/service"
	"{{.RootPkgName}}/api/typespec/{{.AppName | ToLower}}type"
)

type {{.AppName | ToCamel}} struct {}

{{if .ReadListFlag}}// List{{.AppName | ToCamel}} returns {{.AppName}} list
func ({{.ShortName}} *{{.AppName | ToCamel}}) List{{.AppName | ToCamel}}(ctx context.Context, req *{{.AppName | ToLower}}type.Get{{.AppName | ToCamel}}ListRequest, resp *{{.AppName | ToLower}}type.Get{{.AppName | ToCamel}}ListResponse) error {
	{{.AppName}}Svc := &service.{{.AppName | ToCamel}}{}
	if err := {{.AppName}}Svc.GetMany(ctx, req, resp); err != nil {
		return err
	}

	return nil
}{{end}}

{{if .ReadFlag}}// Get{{.AppName | ToCamel}} returns data by criteria
func ({{.ShortName}} *{{.AppName | ToCamel}}) Get{{.AppName | ToCamel}}(ctx context.Context, req *{{.AppName | ToLower}}type.Get{{.AppName | ToCamel}}Request, resp *{{.AppName | ToLower}}type.Get{{.AppName | ToCamel}}Response) error {
	{{.AppName}}Svc := &service.{{.AppName | ToCamel}}{}
	if err := {{.AppName}}Svc.Get(ctx, req, resp); err != nil {
		return err
	}
	
	return nil
}{{end}}


{{if .CreateFlag}}// Add{{.AppName | ToCamel}} adds {{.AppName}} to database
func ({{.ShortName}} *{{.AppName | ToCamel}}) Add{{.AppName | ToCamel}}(ctx context.Context, req *{{.AppName | ToLower}}type.Add{{.AppName | ToCamel}}Request, resp *{{.AppName | ToLower}}type.Add{{.AppName | ToCamel}}Response) error {
	{{.AppName}}Svc := &service.{{.AppName | ToCamel}}{}
	if err := {{.AppName}}Svc.Add(ctx, req, resp); err != nil {
		return err
	}

	return nil
}{{end}}


{{if .UpdateFlag}}// Update{{.AppName | ToCamel}} updates {{.AppName}} by criteria
func ({{.ShortName}} *{{.AppName | ToCamel}}) Update{{.AppName | ToCamel}}(ctx context.Context, req *{{.AppName | ToLower}}type.Update{{.AppName | ToCamel}}Request, resp *{{.AppName | ToLower}}type.Update{{.AppName | ToCamel}}Response) error {
	{{.AppName}}Svc := &service.{{.AppName | ToCamel}}{}
	if err := {{.AppName}}Svc.Update(ctx, req, resp); err != nil {
		return err
	}

	return nil
}{{end}}


{{if .DeleteFlag}}// Delete{{.AppName | ToCamel}} deletes {{.AppName}} by criteria
func ({{.ShortName}} *{{.AppName | ToCamel}}) Delete{{.AppName | ToCamel}}(ctx context.Context, req *{{.AppName | ToLower}}type.Delete{{.AppName | ToCamel}}Request, resp *{{.AppName | ToLower}}type.Delete{{.AppName | ToCamel}}Response) error {
	{{.AppName}}Svc := &service.{{.AppName | ToCamel}}{}
	if err := {{.AppName}}Svc.Delete(ctx, req, resp); err != nil {
		return err
	}
	return nil
}{{end}}
`)
}

func ServiceTemplate() []byte {
	return []byte(`package service

import (
	"context"
	"{{.RootPkgName}}/model"
	"{{.RootPkgName}}/api/typespec/{{.ServiceName | ToLower}}type"
	"github.com/go-season/common/util"
)

type {{.ServiceName | ToCamel}} struct {}

{{if .ReadListFlag}}// GetMany returns {{.ServiceName}} list
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) GetMany(ctx context.Context, req *{{.ServiceName | ToLower}}type.Get{{.ServiceName | ToCamel}}ListRequest, resp *{{.ServiceName | ToLower}}type.Get{{.ServiceName | ToCamel}}ListResponse) error {
	var (
		{{.ServiceName}}List []model.{{.ServiceName | ToCamel}}
		{{.ServiceName}} model.{{.ServiceName | ToCamel}}
	)

	if err := util.Bind(&{{.ServiceName}}, req); err != nil {
		return err
	}

	{{if eq .PageMode "page"}}{{.ServiceName}}List, count, err := {{.ServiceName}}.Get{{.ServiceName | ToCamel}}List(ctx, req.Page, req.PageSize)
	{{else}}{{.ServiceName}}List, count, err := {{.ServiceName}}.Get{{.ServiceName | ToCamel}}List(ctx, req.Offset, req.Length)
	{{end}}if err != nil {
		return err
	}

	err = util.Bind(&resp.List, {{.ServiceName}}List)
	if err != nil {
		return err
	}

	{{if eq .PageMode "page"}}resp.Page = req.Page
	resp.PageSize = req.PageSize
	{{else}}resp.Offset = req.Offset
	resp.Length = req.Length
	{{end}}resp.Total = count

	return nil
}{{end}}

{{if .ReadFlag}}// Get returns a single {{.ServiceName}} data
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) Get(ctx context.Context, req *{{.ServiceName | ToLower}}type.Get{{.ServiceName | ToCamel}}Request, resp *{{.ServiceName | ToLower}}type.Get{{.ServiceName | ToCamel}}Response) error {
	var {{.ServiceName}} model.{{.ServiceName | ToCamel}}

	if err := util.Bind(&{{.ServiceName}}, req); err != nil {
		return err
	}

	{{.ServiceName}}, err := {{.ServiceName}}.Get{{.ServiceName | ToCamel}}(ctx)
	if err != nil {
		return err
	}

    return util.Bind(resp, {{.ServiceName}})
}{{end}}


{{if .CreateFlag}}// Add adds a record of {{.ServiceName}} 
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) Add(ctx context.Context, req *{{.ServiceName | ToLower}}type.Add{{.ServiceName | ToCamel}}Request, resp *{{.ServiceName | ToLower}}type.Add{{.ServiceName | ToCamel}}Response) error {
	var {{.ServiceName}} model.{{.ServiceName | ToCamel}}

	if err := util.Bind(&{{.ServiceName}}, req); err != nil {
		return err
	}

	return {{.ServiceName}}.Add{{.ServiceName | ToCamel}}(ctx)
}{{end}}


{{if .UpdateFlag}}// Update updates a record of {{.ServiceName}}
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) Update(ctx context.Context, req *{{.ServiceName | ToLower}}type.Update{{.ServiceName | ToCamel}}Request, resp *{{.ServiceName | ToLower}}type.Update{{.ServiceName | ToCamel}}Response) error {
	var {{.ServiceName}} model.{{.ServiceName | ToCamel}}

	if err := util.Bind(&{{.ServiceName}}, req); err != nil {
		return err
	}

	return {{.ServiceName}}.Update{{.ServiceName | ToCamel}}(ctx)
}{{end}}


{{if .DeleteFlag}}// Delete a record of {{.ServiceName}}
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) Delete(ctx context.Context, req *{{.ServiceName | ToLower}}type.Delete{{.ServiceName | ToCamel}}Request, resp *{{.ServiceName | ToLower}}type.Delete{{.ServiceName | ToCamel}}Response) error {
	var {{.ServiceName}} model.{{.ServiceName | ToCamel}}

	if err := util.Bind(&{{.ServiceName}}, req); err != nil {
		return err
	}

	return {{.ServiceName}}.Delete{{.ServiceName | ToCamel}}(ctx)
}{{end}}
`)
}

func EmptyServiceTemplate() []byte {
	return []byte(`package service

import (
	"context"
	"{{.RootPkgName}}/api/typespec/{{.ServiceName | ToLower}}type"
)

type {{.ServiceName | ToCamel}} struct {}

{{if .ReadListFlag}}// GetMany returns {{.ServiceName}} list
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) GetMany(ctx context.Context, req *{{.ServiceName | ToLower}}type.Get{{.ServiceName | ToCamel}}ListRequest, resp *{{.ServiceName | ToLower}}type.Get{{.ServiceName | ToCamel}}ListResponse) error {

	return nil
}{{end}}

{{if .ReadFlag}}// Get returns a single {{.ServiceName}} data
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) Get(ctx context.Context, req *{{.ServiceName | ToLower}}type.Get{{.ServiceName | ToCamel}}Request, resp *{{.ServiceName | ToLower}}type.Get{{.ServiceName | ToCamel}}Response) error {

	return nil
}{{end}}


{{if .CreateFlag}}// Add adds a record of {{.ServiceName}} 
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) Add(ctx context.Context, req *{{.ServiceName | ToLower}}type.Add{{.ServiceName | ToCamel}}Request, resp *{{.ServiceName | ToLower}}type.Add{{.ServiceName | ToCamel}}Response) error {

	return nil
}{{end}}


{{if .UpdateFlag}}// Update updates a record of {{.ServiceName}}
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) Update(ctx context.Context, req *{{.ServiceName | ToLower}}type.Update{{.ServiceName | ToCamel}}Request, resp *{{.ServiceName | ToLower}}type.Update{{.ServiceName | ToCamel}}Response) error {
	return nil
}{{end}}


{{if .DeleteFlag}}// Delete a record of {{.ServiceName}}
func ({{.ShortName}} *{{.ServiceName | ToCamel}}) Delete(ctx context.Context, req *{{.ServiceName | ToLower}}type.Delete{{.ServiceName | ToCamel}}Request, resp *{{.ServiceName | ToLower}}type.Delete{{.ServiceName | ToCamel}}Response) error {

	return nil
}{{end}}
`)
}

func ModelTemplate() []byte {
	return []byte(`package model

import (
	"context"
	"gorm.io/gorm"

	"github.com/go-season/common/orm"
)

{{.ModelStruct}}

{{if .ReadListFlag}}// List{{.ModelName|ToCamel}} retrieves a list of {{.ModelName}} from database
func ({{.ShortName}} *{{.ModelName | ToCamel}}) List{{.ModelName | ToCamel}}(ctx context.Context, page, pageSize int) ([]{{.ModelName | ToCamel}}, int64, error) {
	var (
		{{.ModelName}}List []{{.ModelName | ToCamel}}
		count int64
		db = orm.FromContext(ctx{{if .UseDB}}, "{{.UseDB}}"{{end}})
	)

	{{if eq .PageMode "page"}}result := db.Scopes(orm.Paginate(page, pageSize, false)).Find(&{{.ModelName}}List)
	{{else}}result := db.Scopes(orm.Paginate(page, pageSize, true)).Find(&{{.ModelName}}List)
	{{end}}if err := result.Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, 0, err
	}

	orm.DB.Model(&{{.ModelName | ToCamel}}{}).Count(&count)

	return {{.ModelName}}List, count, nil
}{{end}}

{{if .ReadFlag}}// Get{{.ModelName | ToCamel}} retrieves a single record of {{.ModelName}} from database
func ({{.ShortName}} *{{.ModelName | ToCamel}}) Get{{.ModelName | ToCamel}}(ctx context.Context) ({{.ModelName | ToCamel}}, error) {
	var (
			{{.ModelName}} {{.ModelName | ToCamel}}
			err error
			db = orm.FromContext(ctx{{if .UseDB}}, "{{.UseDB}}"{{end}})
	)

	curErr := db.Where("id = ?", {{.ShortName}}.ID).First(&{{.ModelName}}).Error
	if curErr != nil && curErr != gorm.ErrRecordNotFound {
		err = curErr
	}
	
	return  {{.ModelName}}, err
}{{end}}

{{if .CreateFlag}}// Add{{.ModelName | ToCamel}} persists {{.ModelName}} to database
func ({{.ShortName}} *{{.ModelName | ToCamel}}) Add{{.ModelName | ToCamel}}(ctx context.Context) error {
	var db = orm.FromContext(ctx{{if .UseDB}}, "{{.UseDB}}"{{end}})

	if err := db.Create({{.ShortName}}).Error; err != nil {
		return err
	}
	
	return nil
}{{end}}

{{if .UpdateFlag}}// Update{{.ModelName | ToCamel}} changes {{.ModelName}} by id
func ({{.ShortName}} *{{.ModelName | ToCamel}}) Update{{.ModelName | ToCamel}}(ctx context.Context) error {
	var db = orm.FromContext(ctx{{if .UseDB}}, "{{.UseDB}}"{{end}})

	if err := db.Model(&{{.ModelName | ToCamel}}{}).Where("id = ?", {{.ShortName}}.ID).Updates({{.ShortName}}).Error; err != nil {
		return err
	}
	
	return nil
}{{end}}

{{if .DeleteFlag}}// Delete{{.ModelName | ToCamel}} {{.ModelName}} by id
func ({{.ShortName}} *{{.ModelName | ToCamel}}) Delete{{.ModelName | ToCamel}}(ctx context.Context) error {
	var db = orm.FromContext(ctx{{if .UseDB}}, "{{.UseDB}}"{{end}})

	if err := db.Model(&{{.ModelName | ToCamel}}{}).Where("id = ?", {{.ShortName}}.ID).Delete({{.ShortName}}).Error; err != nil {
		return err
	}

	return nil
}{{end}}
`)
}

func RouterTemplate() []byte {
	return []byte(`package rest
import (
	"os"

	"{{.RootPkgName}}/api/rest/hello"
	"{{.RootPkgName}}/pkg/constant"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/go-season/common/log"
	"github.com/go-season/common/plugins/middleware/trace/zipkin"
)

var route gin.IRouter

func InitRouter() *gin.Engine {
    r := gin.New()
	r.Use(log.GinHandler(), gin.Recovery(), zipkin.Trace())
	r.GET("/", hello.Greeter)
	if os.Getenv(constant.EnvFlag) != constant.EnvProd {
		pprof.Register(r)
		r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	route = r.Group("/")
	registerRoute()
	return r
}
`)
}

func ServiceEntryTemplate() []byte {
	return []byte(`package main

import (
	"fmt"
	"os"

	"{{.RootPkgName}}/api/rest"
	_ "{{.RootPkgName}}/docs"
	_ "go.uber.org/automaxprocs"

	"github.com/go-season/common/client"
	"github.com/go-season/common/config"
	"github.com/go-season/common/orm"
	"github.com/go-season/common/plugins/wrapper/k8straffic"
	"github.com/go-season/common/plugins/wrapper/trace/zipkin"
	"github.com/go-season/common/server"
	"github.com/go-season/common/util/trace"
)

func main() {
	// init common component.
	// there default comment orm and redis component,
	// if your project dependency these component,
	// you should open the comment.
	if _, err := config.NewConfig(os.Getenv("APP_ENV")); err != nil {
		panic(fmt.Sprintf("init config failed, err: %v", err))
	}
	orm.Setup()
	//redis.Setup()

	// register route and init server
	router := rest.InitRouter()
	srv := server.NewServer(router)

	// Setting trace's service name
	trace.SetServiceName("{{.RootPkgName}}")

	// Add client wrappers
	// Out of the box inject xve and trace wrapper
	client.AddDefaultWrappers(k8straffic.WrapperChain)
	client.AddDefaultWrappers(zipkin.WrapperChain)

	srv.Run()
}
`)
}

func PkgModuleTemplate() []byte {
	return []byte(`module {{.RootPkgName}}

go 1.15

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/gin-gonic/gin v1.6.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/swaggo/files v0.0.0-20190704085106-630677cd5c14
	github.com/swaggo/gin-swagger v1.3.0
	github.com/swaggo/swag v1.7.0
	go.uber.org/automaxprocs v1.4.0
	github.com/go-season/common latest
)
`)
}

func DevAppConfigTemplate() []byte {
	return []byte(`app:
  http_port: 8080
  read_timeout: 10
  write_timeout: 20
mysql:
  - name: dbname
    master: user:password@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local
    slaves:
      - user:password@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local
`)
}

func APIDefinitionTemplate() []byte {
	return []byte(`package rest
{{if gt .Count 0}}
import (
	{{.ImportPath}}
)
{{end}}
func registerRoute() {
	{{.APIDefinition}}
}
`)
}

func CronEntryTemplate() []byte {
	return []byte(`package main

import (
	"fmt"
	"github.com/go-season/common/config"
	"os"
	_ "go.uber.org/automaxprocs"
	"{{.SubCmdPath}}"
)

func main() {
	if _, err := config.NewConfig(os.Getenv("APP_ENV")); err != nil {
		panic(fmt.Sprintf("init config failed, err: %v", err))
	}
	cmd.Execute()
}
`)
}

func CronTemplate() []byte {
	return bytes.NewBufferString(`package cmd
import (
	"fmt"
	"github.com/spf13/cobra"
)
// 接受参数列表
//var name string
// {{.Name}}Cmd represents the {{.Name}} command
var {{.Name}}Cmd = &cobra.Command{
	Use:   "{{.Name}}",
	Short: "一句话总结的 cron 任务信息",
	Long: ` + "`描述 cron 任务的具体信息`," + `
	Run: func(cmd *cobra.Command, args []string) {
		//do what you want
		fmt.Println("do what you want")
	},
}

func init() {
	rootCmd.AddCommand({{.Name}}Cmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// helloCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	//{{.Name}}Cmd.Flags().StringVarP(&name, "name", "n", "", "-n/--name name")
}

`).Bytes()
}

func HTTPResponseTemplate() []byte {
	return []byte(`package http
import (
	"github.com/gin-gonic/gin"
	"github.com/go-season/common/errors"
	"net/http"
	"time"
)

const (
	defaultErrCode = 4000
	defaultErrMsg  = "server internal error"
)

type response struct {
	ctx       *gin.Context
	code      int
	Status    int         {{.StatusJSONTag}}
	Content   interface{} {{.ContentJSONTag}}
	ErrorMsg  string      {{.ErrorMsgJSONTag}}
	Timestamp int64       {{.TimestampJSONTag}}
}

func Success(c *gin.Context, data interface{}) {
	newResponse(c, http.StatusOK, http.StatusOK, data).render()
}

func Error(c *gin.Context, httpCode int, err error) {
	if ce, ok := err.(errors.CustomError); ok {
		errCode := ce.Code()
		errMsg := ce.Error()
		newResponse(c, httpCode, errCode, errMsg).render()
		return
	}

	newResponse(c, httpCode, defaultErrCode, defaultErrMsg).render()
}

func newResponse(ctx *gin.Context, httpCode, status int, data interface{}) response {
	var errMsg string
	if status != http.StatusOK {
		errMsg = data.(string)
		data = nil
	}
	return response{
		ctx:       ctx,
		code:      httpCode,
		Status:    status,
		ErrorMsg:  errMsg,
		Content: data,
		Timestamp: time.Now().Unix(),
	}
}

func (r response) render() {
	r.ctx.JSON(r.code, r)
}
`)
}
