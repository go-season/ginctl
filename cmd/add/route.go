package add

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-season/ginctl/pkg/util/str"

	route2 "github.com/go-season/ginctl/cmd/route"
	"github.com/go-season/ginctl/pkg/db2struct"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

type routeCmd struct {
	log                log.Logger
	Conn               string
	TableName          string
	Database           string
	JsonTag            bool
	CreateFlag         bool
	UpdateFlag         bool
	ReadFlag           bool
	ReadListFlag       bool
	DeleteFlag         bool
	PaginationStrategy string
	ORMV2              bool // --orm-v2
	WithSoftDelete     bool // --with-soft
	NoORM              bool // --no-orm
	Verbose            bool
}

func newRouteCmd(f factory.Factory) *cobra.Command {
	cmd := &routeCmd{
		log: f.GetLog(),
	}

	addRouteCmd := &cobra.Command{
		Use:   "route [name]",
		Short: "生成接口相关的curd模板逻辑",
		Long: `
为项目生成请求接口相关模板代码，默认会生成rest控制器，service，
model模板.

命令样例:
ginctl add route user
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	addRouteCmd.Flags().StringVarP(&cmd.TableName, "table", "t", "", "The table name of map to struct, default use model name")
	addRouteCmd.Flags().StringVar(&cmd.Database, "db", "", "Choice an database instance for current model")
	addRouteCmd.Flags().StringVar(&cmd.Conn, "conn", "", "Connection string used by the SQLDriver to connect to a database instance. eg: \"root:@tcp(127.0.0.1:3306)/dbname\"")
	addRouteCmd.Flags().BoolVar(&cmd.JsonTag, "json-tag", true, "Is generate json tag to struct")
	addRouteCmd.Flags().BoolVarP(&cmd.CreateFlag, "create", "c", false, "Is generate create request handler")
	addRouteCmd.Flags().BoolVarP(&cmd.UpdateFlag, "update", "u", false, "Is generate update request handler")
	addRouteCmd.Flags().BoolVarP(&cmd.ReadFlag, "read", "r", false, "Is generate read request handler")
	addRouteCmd.Flags().BoolVarP(&cmd.ReadListFlag, "read-list", "l", false, "Is generate read list request handler")
	addRouteCmd.Flags().BoolVarP(&cmd.DeleteFlag, "delete", "d", false, "Is generate delete request handler")
	addRouteCmd.Flags().StringVar(&cmd.PaginationStrategy, "page-mode", "offset", "The pagination field show strategy, default: offset, support: offset, page")
	addRouteCmd.Flags().BoolVar(&cmd.WithSoftDelete, "with-soft", false, "Is expected model extends from orm.SoftDelete")
	addRouteCmd.Flags().BoolVar(&cmd.NoORM, "no-orm", false, "Is expected model not extends from orm.Base model")
	addRouteCmd.Flags().BoolVar(&cmd.ORMV2, "orm-v2", false, "Is expected use orm v2 extends.")
	addRouteCmd.Flags().BoolVarP(&cmd.Verbose, "verbose", "v", false, "Is output verbose info.")

	return addRouteCmd
}

func (cmd *routeCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	routeName := args[0]
	if str.IsSnakeCase(routeName) {
		routeName = str.SnakeToCamel(routeName)
	}

	if str.IsBuiltinKeywords(routeName) {
		return errors.New(fmt.Sprintf("can't use builtin keywords [%s] as route name, please change other name!:(", routeName))
	}

	// default any curd flag not passed, generate all logic
	if !cmd.CreateFlag && !cmd.UpdateFlag && !cmd.ReadFlag && !cmd.DeleteFlag && !cmd.ReadListFlag {
		cmd.CreateFlag = true
		cmd.UpdateFlag = true
		cmd.ReadFlag = true
		cmd.ReadListFlag = true
		cmd.DeleteFlag = true
	}

	// generate rest layer
	restCmd := &restCmd{
		Overwrite:    true,
		CreateFlag:   cmd.CreateFlag,
		UpdateFlag:   cmd.UpdateFlag,
		ReadFlag:     cmd.ReadFlag,
		ReadListFlag: cmd.ReadListFlag,
		DeleteFlag:   cmd.DeleteFlag,
	}
	err := restCmd.RunAddRest(f, cobraCmd, []string{routeName})
	if err != nil {
		return err
	}

	// generate application layer
	tableName := routeName
	if cmd.TableName != "" {
		tableName = cmd.TableName
	}
	var fileds string
	if cmd.Conn != "" {
		columnTyp, columnName, err := db2struct.GetColumnsFromMysqlTable(cmd.Conn, tableName)
		if err != nil {
			return err
		}
		fileds = db2struct.GenerateReqAndRespTypes(*columnTyp, columnName, true, true)
	}
	appCmd := &appCmd{
		Fields:       strings.TrimSpace(fileds),
		CreateFlag:   cmd.CreateFlag,
		UpdateFlag:   cmd.UpdateFlag,
		ReadFlag:     cmd.ReadFlag,
		DeleteFlag:   cmd.DeleteFlag,
		ReadListFlag: cmd.ReadListFlag,
		PageModel:    cmd.PaginationStrategy,
	}
	err = appCmd.RunAddApp(f, cobraCmd, []string{routeName})
	if err != nil {
		return err
	}

	// generate service layer
	svcCmd := &serviceCmd{
		Conn:           cmd.Conn,
		TableName:      cmd.TableName,
		Database:       cmd.Database,
		JsonTag:        cmd.JsonTag,
		CreateFlag:     cmd.CreateFlag,
		UpdateFlag:     cmd.UpdateFlag,
		ReadFlag:       cmd.ReadFlag,
		ReadListFlag:   cmd.ReadListFlag,
		DeleteFlag:     cmd.DeleteFlag,
		PageMode:       cmd.PaginationStrategy,
		ORMV2:          cmd.ORMV2,
		WithSoftDelete: cmd.WithSoftDelete,
		NoORM:          cmd.NoORM,
	}
	err = svcCmd.RunAddService(f, cobraCmd, []string{routeName})
	if err != nil {
		return err
	}

	if restCmd.Overwrite {
		refreshCmd := &route2.RefreshCmd{
			Verbose: cmd.Verbose,
		}
		if err := refreshCmd.Run(f, cobraCmd, []string{}); err != nil {
			return err
		}
	}

	cmd.log.WriteString("\n")
	cmd.log.Donef("create route successfully, you can filling your logic now.")

	return nil
}
