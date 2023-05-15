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

type restCmd struct {
	Overwrite    bool
	CreateFlag   bool
	UpdateFlag   bool
	ReadFlag     bool
	ReadListFlag bool
	DeleteFlag   bool
}

func newRestCmd(f factory.Factory) *cobra.Command {
	cmd := &restCmd{}

	addRestCmd := &cobra.Command{
		Use:   "rest [name]",
		Short: "生成`rest`控制层模板代码",
		Long: `
为项目生成rest层控制代码

命令样例:
ginctl add rest user
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAddRest(f, cobraCmd, args)
		},
	}

	return addRestCmd
}

func (cmd *restCmd) RunAddRest(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if !cmd.CreateFlag && !cmd.UpdateFlag && !cmd.ReadFlag && !cmd.DeleteFlag && !cmd.ReadListFlag {
		cmd.CreateFlag = true
		cmd.UpdateFlag = true
		cmd.ReadFlag = true
		cmd.ReadListFlag = true
		cmd.DeleteFlag = true
	}

	CmdPkgName, err := util.GetPkgName(wd + "/cmd/apiserver")
	if err != nil {
		return err
	}
	RootPkgName := strings.TrimSuffix(CmdPkgName, "/cmd/apiserver")

	restName := args[0]
	rest := &Rest{
		SubPkgName:  restName,
		RestName:    restName,
		RootPkgName: RootPkgName,
		Project: &Project{
			AbsolutePath: wd,
		},
		Route: &Route{
			CreateFlag:   cmd.CreateFlag,
			UpdateFlag:   cmd.UpdateFlag,
			ReadFlag:     cmd.ReadFlag,
			ReadListFlag: cmd.ReadListFlag,
			DeleteFlag:   cmd.DeleteFlag,
		},
	}

	dir := fmt.Sprintf("%s/api/rest/%s", rest.AbsolutePath, strings.ToLower(rest.RestName))
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

	path := fmt.Sprintf("%s/%s.go", dir, strings.ToLower(restName))
	found, err = file.PathExists(path)
	if err != nil {
		return err
	}
	if found {
		result, err := f.GetLog().Question(&log.QuestionOptions{
			Question:     fmt.Sprintf("rest file %s is exists, do you want overwrite it?", path),
			DefaultValue: "No",
			Options:      []string{"Yes", "No"},
		})
		if err != nil {
			return err
		}
		if result != "Yes" {
			cmd.Overwrite = false
			return nil
		}
	}

	err = rest.Create()
	if err != nil {
		return err
	}

	f.GetLog().Donef(fmt.Sprintf("%s.go created at %s/api/rest/%s\n", str.ToSnakeCase(rest.RestName), rest.AbsolutePath, strings.ToLower(rest.RestName)))

	return nil
}
