package polyfill

import (
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

func NewPolyfillCmd(f factory.Factory) *cobra.Command {
	polyfillCmd := &cobra.Command{
		Use:   "polyfill",
		Short: "快捷命令: 为老项目生成新项目兼容相关的代码，以便使用新特性",
		Args:  cobra.NoArgs,
	}

	polyfillCmd.AddCommand(newResponseCmd(f))

	return polyfillCmd
}
