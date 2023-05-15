package pkg

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-season/ginctl/pkg/util/log"

	"github.com/go-season/ginctl/pkg/ginctl/pkg"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

type UpdateCmd struct {
	ModuleName string
	ModuleList []string
	All        bool
}

func NewUpdateCmd(f factory.Factory) *cobra.Command {
	cmd := UpdateCmd{}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "更新内部指定包",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	updateCmd.Flags().BoolVar(&cmd.All, "all", false, "是否更新所有内部已经引入包")

	return updateCmd
}

func (cmd *UpdateCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	var err error

	cmd.ModuleList, err = pkg.FindRequiredModule()
	if err != nil {
		return err
	}

	alias := make(map[string][]string)
	for _, module := range cmd.ModuleList {
		pos := strings.LastIndex(module, "/")
		key := module[pos+1:]
		alias[key] = append(alias[key], module)
	}

	if cmd.All {
		return updateMulti(f, cmd.ModuleList)
	}

	if len(args) > 0 {
		if mod, ok := alias[args[0]]; ok {
			if len(mod) == 1 {
				if err := update(f, mod[0]); err != nil {
					return err
				}
				f.GetLog().Donef("更新完成")
				return nil
			}
			return QuestionUpdate(f, mod)
		}
		return errors.New("未匹配到包地址，请确定已经安装")
	}

	return QuestionUpdate(f, cmd.ModuleList)
}

func QuestionUpdate(f factory.Factory, modules []string) error {
	choiced, err := f.GetLog().Question(&log.QuestionOptions{
		Question:      "请选择你要更新的包",
		Options:       modules,
		IsMultiSelect: true,
	})
	if err != nil {
		return err
	}
	return updateMulti(f, strings.Split(choiced, ","))
}

func update(f factory.Factory, module string) error {
	cmdStr := fmt.Sprintf("go get %s", module)
	_, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return err
	}

	f.GetLog().Info(fmt.Sprintf("开始更新包%s...", module))

	return nil
}

func updateMulti(f factory.Factory, modules []string) error {
	for _, module := range modules {
		if err := update(f, module); err != nil {
			return err
		}
	}

	f.GetLog().Donef("更新完成")

	return nil
}
