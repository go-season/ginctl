package cc

import (
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

func NewCCCmd(f factory.Factory) *cobra.Command {
	ccCmd := &cobra.Command{
		Use:   "cc",
		Short: "快捷命令: 配置中心相关",
		Args:  cobra.NoArgs,
	}

	ccCmd.AddCommand(NewInitCmd(f))

	return ccCmd
}
