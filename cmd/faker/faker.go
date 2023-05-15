package faker

import (
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

func NewFakerCmd(f factory.Factory) *cobra.Command {
	fakerCmd := &cobra.Command{
		Use:   "faker",
		Short: "快捷命令: Faker相关操作",
		Args:  cobra.NoArgs,
	}

	fakerCmd.AddCommand(NewGenerateCmd(f))

	return fakerCmd
}
