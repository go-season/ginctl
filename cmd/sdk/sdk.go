package sdk

import (
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

func NewSDKCmd(f factory.Factory) *cobra.Command {
	sdkCmd := &cobra.Command{
		Use:   "sdk",
		Short: "快捷命令: SDK相关操作",
		Args:  cobra.NoArgs,
	}

	sdkCmd.AddCommand(NewGenerateCmd(f))
	sdkCmd.AddCommand(NewLocallyCmd(f))
	sdkCmd.AddCommand(NewUpdateCmd(f))

	return sdkCmd
}
