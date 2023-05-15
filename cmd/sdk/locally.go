package sdk

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-season/ginctl/pkg/ginctl/sdk"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/spf13/cobra"
)

type LocallyCmd struct {
	Branch string
}

func NewLocallyCmd(f factory.Factory) *cobra.Command {
	cmd := LocallyCmd{}

	locallyCmd := &cobra.Command{
		Use:     "locally",
		Aliases: []string{"loc", "l"},
		Short:   "本地化sdk，方便快速调试",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	locallyCmd.Flags().StringVarP(&cmd.Branch, "branch", "b", "master", "指定要本地调试的SDK分支")

	return locallyCmd
}

func (cmd *LocallyCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	f.GetLog().StartWait("start binding sdk to locally...")

	found, err := file.PathExists(sdk.GetLocallyPath())
	if err != nil {
		return err
	}
	if !found {
		if err := sdk.CloneRepoToLocal(sdk.GetLocallyPath()); err != nil {
			return err
		}
	}

	// 获取当前本地分支
	cmdStr := fmt.Sprintf("git rev-parse --abbrev-ref HEAD")
	ecmd := exec.Command("bash", "-c", cmdStr)
	ecmd.Dir = sdk.GetLocallyPath()
	branch, err := ecmd.Output()
	if err != nil {
		return err
	}

	if cmd.Branch != strings.TrimSpace(string(branch)) {
		cmdStr = fmt.Sprintf("git fetch && git checkout %s", cmd.Branch)
		ecmd = exec.Command("bash", "-c", cmdStr)
		ecmd.Dir = sdk.GetLocallyPath()
		if err := ecmd.Run(); err != nil {
			return err
		}
	}

	cmdStr = fmt.Sprintf("git pull origin %s", cmd.Branch)
	ecmd = exec.Command("bash", "-c", cmdStr)
	ecmd.Dir = sdk.GetLocallyPath()
	if err := ecmd.Run(); err != nil {
		return err
	}

	// check sdk package is exists project
	cmdStr = "go get gitlab.idc.xiaozhu.com/xz/lib/sdk"
	_, err = exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return err
	}

	cmdStr = fmt.Sprintf("go mod edit --replace=gitlab.idc.xiaozhu.com/xz/lib/sdk=%s", sdk.GetLocallyPath())
	_, err = exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return err
	}

	f.GetLog().Donef("bind locally sdk branch:%s to project", cmd.Branch)

	return nil
}
