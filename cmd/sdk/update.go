package sdk

import (
	"fmt"
	"os/exec"

	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

type updateCmd struct{}

func NewUpdateCmd(f factory.Factory) *cobra.Command {
	cmd := updateCmd{}

	ucmd := &cobra.Command{
		Use:   "update",
		Short: "快速更新SDK包",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	return ucmd
}

func (cmd *updateCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	f.GetLog().StartWait("开始更新SDK包...")

	cmdStr := fmt.Sprintf("go get gitlab.idc.xiaozhu.com/xz/lib/sdk")
	_, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return err
	}

	f.GetLog().Donef("更新SDK包完成.")

	return nil
}
