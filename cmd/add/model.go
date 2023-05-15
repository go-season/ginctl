package add

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-season/ginctl/pkg/db2struct"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/go-season/ginctl/pkg/util/str"
	"github.com/spf13/cobra"
)

type modelCmd struct {
	Conn           string
	TableName      string
	Database       string
	JsonTag        bool
	CreateFlag     bool
	UpdateFlag     bool
	ReadFlag       bool
	ReadListFlag   bool
	DeleteFlag     bool
	PageModel      string
	ORMV2          bool // --orm-v2
	WithSoftDelete bool // --with-soft
	NoORM          bool // --no-orm
}

func newModelCmd(f factory.Factory) *cobra.Command {
	cmd := &modelCmd{}

	addModelCmd := &cobra.Command{
		Use:   "model [name]",
		Short: "生成`model`层模板代码",
		Long: `
为项目生成一个model模板代码

命令样例:
ginctl add model user
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAddModel(f, cobraCmd, args)
		},
	}

	addModelCmd.Flags().StringVarP(&cmd.TableName, "table", "t", "", "The table name of map to struct, default use model name")
	addModelCmd.Flags().StringVar(&cmd.Conn, "conn", "", "Connection string used by the SQLDriver to connect to a database instance.eg:\"root:@tcp(127.0.0.1:3306)/dbname\"")
	addModelCmd.Flags().StringVar(&cmd.Database, "db", "", "Choice an database instance for current model")
	addModelCmd.Flags().BoolVar(&cmd.JsonTag, "json-tag", true, "Is generate json tag to struct")
	addModelCmd.Flags().BoolVarP(&cmd.CreateFlag, "create", "c", false, "Is generate create model handler")
	addModelCmd.Flags().BoolVarP(&cmd.UpdateFlag, "update", "u", false, "Is generate update model handler")
	addModelCmd.Flags().BoolVarP(&cmd.ReadFlag, "read", "r", false, "Is generate read model handler")
	addModelCmd.Flags().BoolVarP(&cmd.DeleteFlag, "delete", "d", false, "Is generate delete model handler")
	addModelCmd.Flags().BoolVar(&cmd.WithSoftDelete, "with-soft", false, "Is expected model extends from orm.SoftDelete")
	addModelCmd.Flags().BoolVar(&cmd.NoORM, "no-orm", false, "Is expected model not extends from orm.Base model")
	addModelCmd.Flags().BoolVar(&cmd.ORMV2, "orm-v2", false, "Is expected use orm v2 extends.")

	return addModelCmd
}

func (cmd *modelCmd) RunAddModel(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
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

	modelName := args[0]
	tableName := modelName
	if cmd.TableName != "" {
		tableName = cmd.TableName
	}

	if str.IsSnakeCase(modelName) {
		modelName = str.SnakeToCamel(modelName)
	}

	if str.IsBuiltinKeywords(modelName) {
		return errors.New(fmt.Sprintf("can't use builtin keywords [%s] as model name, please change other name!:(", modelName))
	}

	modelStruct := fmt.Sprintf(`
type %s struct {
	orm.Model
}
`, str.ToCamel(modelName))
	if cmd.Conn != "" {
		columnDataTypes, columnsSorted, err := db2struct.GetColumnsFromMysqlTable(cmd.Conn, tableName)
		if err != nil {
			return nil
		}

		expectedExtend := "default"
		if cmd.ORMV2 {
			expectedExtend = "ormV2"
		} else if cmd.NoORM {
			expectedExtend = "noORM"
		} else if cmd.WithSoftDelete {
			expectedExtend = "softDelete"
		}

		strct, err := db2struct.Generate(*columnDataTypes, columnsSorted, tableName, modelName, "model", cmd.JsonTag, true, expectedExtend)
		if err != nil {
			return err
		}
		modelStruct = string(strct)
	}

	model := &Model{
		ModelStruct: modelStruct,
		ModelName:   str.ToLowerCamelCase(modelName),
		ShortName:   strings.ToLower(modelName[0:1]),
		PageMode:    cmd.PageModel,
		UseDB:       cmd.Database,
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

	path := fmt.Sprintf("%s/model/%s.go", model.AbsolutePath, strings.ToLower(model.ModelName))
	found, err := file.PathExists(path)
	if err != nil {
		return err
	}
	if found {
		result, err := f.GetLog().Question(&log.QuestionOptions{
			Question:     fmt.Sprintf("model file %s is exists, do you want overwrite it?", path),
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

	err = model.Create()
	if err != nil {
		return err
	}

	f.GetLog().Donef(fmt.Sprintf("%s.go created at %s/model\n", str.ToSnakeCase(model.ModelName), model.AbsolutePath))

	return nil
}
