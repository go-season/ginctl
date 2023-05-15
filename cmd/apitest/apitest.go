package apitest

import (
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

func NewAPITestCmd(f factory.Factory) *cobra.Command {
	atestCmd := &cobra.Command{
		Use:   "api:test",
		Short: "快捷命令: APITest相关操作",
		Args:  cobra.NoArgs,
	}

	atestCmd.AddCommand(NewGenerateCmd(f))
	atestCmd.AddCommand(NewMockCmd(f))

	return atestCmd
}
