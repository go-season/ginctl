package add

import (
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

func NewAddCmd(f factory.Factory) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "快捷命令: 为项目快速生成相关模板",
		Args:  cobra.NoArgs,
	}

	addCmd.AddCommand(newCronCmd(f))
	addCmd.AddCommand(newServiceCmd(f))
	addCmd.AddCommand(newModelCmd(f))
	addCmd.AddCommand(newRouteCmd(f))
	addCmd.AddCommand(newHandlerCmd(f))

	return addCmd
}
