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

type serviceCmd struct {
	Conn           string
	TableName      string
	Database       string
	JsonTag        bool
	CreateFlag     bool
	UpdateFlag     bool
	ReadFlag       bool
	ReadListFlag   bool
	DeleteFlag     bool
	PageMode       string
	ORMV2          bool // --orm-v2
	WithSoftDelete bool // --with-soft
	NoORM          bool // --no-orm
}

func newServiceCmd(f factory.Factory) *cobra.Command {
	cmd := &serviceCmd{}

	addServiceCmd := &cobra.Command{
		Use:   "service [name]",
		Short: "生成`service`模板代码",
		Long: `
为项目生成service层模板

命令样例:
ginctl add service user
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAddService(f, cobraCmd, args)
		},
	}

	addServiceCmd.Flags().StringVarP(&cmd.TableName, "table", "t", "", "The table name of map to struct, default use model name")
	addServiceCmd.Flags().StringVar(&cmd.Database, "db", "", "Choice an database instance for current model")
	addServiceCmd.Flags().StringVar(&cmd.Conn, "conn", "", "Connection string used by the SQLDriver to connect to a database instance. eg: \"root:@tcp(127.0.0.1:3306)/dbname\"")
	addServiceCmd.Flags().BoolVar(&cmd.JsonTag, "json-tag", true, "Is generate json tag to struct")
	addServiceCmd.Flags().BoolVarP(&cmd.CreateFlag, "create", "c", false, "Is generate create service handler")
	addServiceCmd.Flags().BoolVarP(&cmd.UpdateFlag, "update", "u", false, "Is generate update service handler")
	addServiceCmd.Flags().BoolVarP(&cmd.ReadFlag, "read", "r", false, "Is generate read service handler")
	addServiceCmd.Flags().BoolVarP(&cmd.DeleteFlag, "delete", "d", false, "Is generate delete service handler")
	addServiceCmd.Flags().BoolVar(&cmd.WithSoftDelete, "with-soft", false, "Is expected model extends from orm.SoftDelete")
	addServiceCmd.Flags().BoolVar(&cmd.NoORM, "no-orm", false, "Is expected model not extends from orm.Base model")
	addServiceCmd.Flags().BoolVar(&cmd.ORMV2, "orm-v2", false, "Is expected use orm v2 extends.")

	return addServiceCmd
}

func (cmd *serviceCmd) RunAddService(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
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

	svcName := args[0]
	svc := &Service{
		IsEmpty:     cmd.Conn == "",
		ServiceName: str.ToLowerCamelCase(svcName),
		ShortName:   strings.ToLower(svcName[0:1]),
		RootPkgName: RootPkgName,
		PageMode:    cmd.PageMode,
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

	path := fmt.Sprintf("%s/service/%s.go", svc.AbsolutePath, strings.ToLower(svc.ServiceName))
	found, err := file.PathExists(path)
	if err != nil {
		return err
	}
	if found {
		result, err := f.GetLog().Question(&log.QuestionOptions{
			Question:     fmt.Sprintf("service file %s is exists, do you want overwrite it?", path),
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

	err = svc.Create()
	if err != nil {
		return err
	}

	if cmd.Conn != "" {
		modelCmd := &modelCmd{
			Conn:           cmd.Conn,
			TableName:      cmd.TableName,
			Database:       cmd.Database,
			JsonTag:        cmd.JsonTag,
			CreateFlag:     cmd.CreateFlag,
			UpdateFlag:     cmd.UpdateFlag,
			ReadFlag:       cmd.ReadFlag,
			ReadListFlag:   cmd.ReadListFlag,
			DeleteFlag:     cmd.DeleteFlag,
			PageModel:      cmd.PageMode,
			ORMV2:          cmd.ORMV2,
			WithSoftDelete: cmd.WithSoftDelete,
			NoORM:          cmd.NoORM,
		}
		err = modelCmd.RunAddModel(f, cobraCmd, []string{svcName})
		if err != nil {
			return err
		}
	}

	f.GetLog().Donef(fmt.Sprintf("%s.go created at %s/service\n", str.ToSnakeCase(svc.ServiceName), svc.AbsolutePath))

	return nil
}
