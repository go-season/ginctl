package pkg

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mgutz/ansi"

	"github.com/go-season/ginctl/pkg/ginctl/pkg"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

type RequireCmd struct {
	ModuleName string
	ModuleList []string
}

func NewRequireCmd(f factory.Factory) *cobra.Command {
	cmd := RequireCmd{}

	requireCmd := &cobra.Command{
		Use:     "require [name]",
		Aliases: []string{"req", "r"},
		Short:   "安装指定的内部包",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	return requireCmd
}

func (cmd *RequireCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	list, err := pkg.List()
	if err != nil {
		return err
	}

	if len(list) == 0 {
		return errors.New("还没有包注册，请联系liulei@xiaozhu.com")
	}

	mmap := make(map[string]bool)
	alias := make(map[string][]string)
	for _, module := range list {
		mmap[module] = true
		pos := strings.LastIndex(module, "/")
		key := module[pos+1:]
		alias[key] = append(alias[key], module)
	}

	cmd.ModuleList, err = pkg.FindDiffModule(mmap)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if err := cmd.interactive(f); err != nil {
			return err
		}
	} else {
		matched := alias[args[0]]
		if len(matched) > 1 {
			choiced, err := f.GetLog().Question(&log.QuestionOptions{
				Question: "请选择你要引入的包:",
				Options:  matched,
			})
			if err != nil {
				return err
			}
			cmd.ModuleName = choiced
		} else {
			cmd.ModuleName = matched[0]
		}
	}

	f.GetLog().StartWait("开始下载包...")

	cmdStr := fmt.Sprintf("go get %s", cmd.ModuleName)
	_, err = exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return err
	}

	f.GetLog().Donef("安装包%s完成", ansi.Color(cmd.ModuleName, "cyan+b"))

	return nil
}

func (cmd *RequireCmd) interactive(f factory.Factory) error {
	choiced, err := f.GetLog().Question(&log.QuestionOptions{
		Question: "请选择你要引入的包:",
		Options:  cmd.ModuleList,
	})

	if err != nil {
		return err
	}

	cmd.ModuleName = choiced

	return nil
}
