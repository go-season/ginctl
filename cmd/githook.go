package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
)

type gitHookCmd struct {
	Rule    string
	Preview bool
}

func NewGitHookCmd(f factory.Factory) *cobra.Command {
	cmd := &gitHookCmd{}

	gcmd := &cobra.Command{
		Use:   "githook",
		Short: "生成git相关钩子脚本",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	gcmd.Flags().StringVarP(&cmd.Rule, "rule", "r", "replace", "指定要生成git钩子相关脚本的规则，默认规则:replace")
	gcmd.Flags().BoolVar(&cmd.Preview, "preview", false, "指定是否仅输出钩子脚本内容")

	return gcmd
}

func (cmd *gitHookCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()

	shell := []byte(`#!/bin/bash
count=$(grep 'replace' go.mod |grep -c '/Users')
if [[ $count > 0 ]];then
	echo -e "\033[31m你的go.mod包含有本地替换指令，请剔除后再提交\033[0m";
	exit 1;
fi
`)
	if cmd.Preview {
		f.GetLog().WriteString(string(shell))
		return nil
	}
	target := fmt.Sprintf("%s/.git/hooks/pre-commit", cwd)
	found, err := file.PathExists(target)
	if err != nil {
		return err
	}
	if found {
		return errors.New("pre-commit钩子已经存在，请执行`ginctl githook --preview`获取脚本，并手动合并钩子触发内容")
	}

	fs, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	defer fs.Close()

	fs.Write(shell)

	f.GetLog().Donef("注入pre-commit钩子完成")

	return nil
}
