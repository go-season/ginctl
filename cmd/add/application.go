package add

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-season/ginctl/pkg/util/str"

	"github.com/go-season/ginctl/pkg/util"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

type appCmd struct {
	log log.Logger

	Fields       string
	CreateFlag   bool
	UpdateFlag   bool
	ReadFlag     bool
	DeleteFlag   bool
	ReadListFlag bool
	PageModel    string
}

func newAppCmd(f factory.Factory) *cobra.Command {
	cmd := &appCmd{
		log: f.GetLog(),
	}

	addAppCmd := &cobra.Command{
		Use:   "app [name]",
		Short: "生成`application`层模板代码",
		Long: `
为项目生成application层模板逻辑

命令样例:
ginctl add app user
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAddApp(f, cobraCmd, args)
		},
	}

	return addAppCmd
}

func (cmd *appCmd) RunAddApp(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if !cmd.CreateFlag && !cmd.UpdateFlag && !cmd.ReadFlag && !cmd.DeleteFlag && !cmd.ReadListFlag {
		cmd.CreateFlag = true
		cmd.UpdateFlag = true
		cmd.ReadFlag = true
		cmd.DeleteFlag = true
		cmd.ReadListFlag = true
	}

	CmdPkgName, err := util.GetPkgName(wd + "/cmd/apiserver")
	if err != nil {
		return err
	}
	RootPkgName := strings.TrimSuffix(CmdPkgName, "/cmd/apiserver")

	appName := args[0]
	app := &App{
		AppName:     str.ToLowerCamelCase(appName),
		ShortName:   strings.ToLower(appName[0:1]),
		RootPkgName: RootPkgName,
		Project: &Project{
			AbsolutePath: wd,
		},
		Route: &Route{
			CreateFlag:   cmd.CreateFlag,
			UpdateFlag:   cmd.UpdateFlag,
			ReadFlag:     cmd.ReadFlag,
			DeleteFlag:   cmd.DeleteFlag,
			ReadListFlag: cmd.ReadListFlag,
		},
	}

	dir := fmt.Sprintf("%s/application/%s", wd, strings.ToLower(appName))
	found, err := file.PathExists(dir)
	if err != nil {
		return err
	}
	if !found {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			return err
		}
	}

	path := fmt.Sprintf("%s/%s.go", dir, strings.ToLower(appName))
	found, err = file.PathExists(path)
	if err != nil {
		return err
	}
	if found {
		result, err := f.GetLog().Question(&log.QuestionOptions{
			Question:     fmt.Sprintf("application file %s is exists, do you want overwrite it?", path),
			DefaultValue: "No",
			Options:      []string{"Yes", "No"},
		})
		if err != nil {
			return err
		}
		if result != "Yes" {
			return nil
		}
	}

	err = app.Create()
	if err != nil {
		return err
	}
	f.GetLog().Donef(fmt.Sprintf("%s.go created at %s\n", str.ToSnakeCase(appName), dir))

	// generate typeSpec layer
	dirName := fmt.Sprintf("%s/api/typespec/%stype", wd, strings.ToLower(appName))
	found, err = file.PathExists(dirName)
	if err != nil {
		return err
	}
	if !found {
		err = os.Mkdir(dirName, 0755)
		if err != nil {
			return err
		}
	}
	pageFormTag := fmt.Sprintf("`form:\"page,default=1\"`")
	pageSizeFormTag := fmt.Sprintf("`form:\"pageSize,default=10\"`")
	pageJsonTag := fmt.Sprintf("`json:\"page\"`")
	pageSizeJsonTag := fmt.Sprintf("`json:\"pageSize\"`")
	if cmd.PageModel == "offset" {
		pageFormTag = fmt.Sprintf("`form:\"offset,default=0\"`")
		pageSizeFormTag = fmt.Sprintf("`form:\"length,default=10\"`")
		pageJsonTag = fmt.Sprintf("`json:\"offset\"`")
		pageSizeJsonTag = fmt.Sprintf("`json:\"length\"`")
	}

	ts := &TypeSpec{
		AppName:              appName,
		SubPkgName:           appName,
		Fields:               cmd.Fields,
		PageMode:             cmd.PageModel,
		ListJsonTag:          fmt.Sprintf("`json:\"list\"`"),
		IdFormTag:            fmt.Sprintf("`form:\"id\"`"),
		IdFormTagWithDefault: fmt.Sprintf("`form:\"id,default=1\"`"),
		IdJsonTag:            fmt.Sprintf("`json:\"id\"`"),
		CreatedAtJsonTag:     fmt.Sprintf("`json:\"createdAt\"`"),
		UpdatedAtJsonTag:     fmt.Sprintf("`json:\"updatedAt\"`"),
		PageFormTag:          pageFormTag,
		PageSizeFormTag:      pageSizeFormTag,
		PageJsonTag:          pageJsonTag,
		PageSizeJsonTag:      pageSizeJsonTag,
		PageTotalJsonTag:     fmt.Sprintf("`json:\"total\"`"),
		Project: &Project{
			AbsolutePath: wd,
		},
		Route: &Route{
			CreateFlag:   cmd.CreateFlag,
			UpdateFlag:   cmd.UpdateFlag,
			ReadFlag:     cmd.ReadFlag,
			DeleteFlag:   cmd.DeleteFlag,
			ReadListFlag: cmd.ReadListFlag,
		},
	}

	err = ts.Create()
	if err != nil {
		return err
	}

	f.GetLog().Donef(fmt.Sprintf("%s.go created at %s/typespec/%stype\n", str.ToSnakeCase(appName), wd, strings.ToLower(appName)))

	return nil
}
