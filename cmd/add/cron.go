package add

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

type cronCmd struct {
	Name    string
	Project *Project
}

func newCronCmd(f factory.Factory) *cobra.Command {

	cmd := &cronCmd{}

	cronCmd := &cobra.Command{
		Use:   "cron [name]",
		Short: "生成`cron`控制器模板代码",
		Long: `
为项目创建cron控制器模板代码

命令样例:
ginctl add cron user
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAddCron(f, cobraCmd, args)
		},
	}

	return cronCmd
}

func (cmd *cronCmd) RunAddCron(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	name := args[0]
	cron := &Cron{
		Name: name,
		Project: &Project{
			AbsolutePath: wd,
		},
	}

	//dir := fmt.Sprintf("%s/cmd/cron/cmd/%s", cron.AbsolutePath, cron.Name)
	//found, err := file.PathExists(dir)
	//if err != nil {
	//	f.GetLog().Donef(fmt.Sprintf("It seems that your project doesn't support cron."))
	//}
	//if !found {
	//	err = os.Mkdir(dir, 0755)
	//	if err != nil {
	//		return err
	//	}
	//}

	dir := fmt.Sprintf("%s/cmd/cron/cmd", cron.AbsolutePath)
	path := fmt.Sprintf("%s/%s.go", dir, strings.ToLower(name))
	found, err := file.PathExists(path)
	if err != nil {

		return err
	}
	if found {
		result, err := f.GetLog().Question(&log.QuestionOptions{
			Question:     fmt.Sprintf("cron file %s is exists, do you want overwrite it?", path),
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

	err = cron.Create()
	if err != nil {
		return err
	}

	f.GetLog().Donef(fmt.Sprintf("%s.go created at %s/cmd/cron/cmd/%s\n", strings.ToLower(cron.Name), cron.AbsolutePath, strings.ToLower(cron.Name)))

	return nil
}
