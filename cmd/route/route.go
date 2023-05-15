package route

import (
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

func NewRouteCmd(f factory.Factory) *cobra.Command {
	routeCmd := &cobra.Command{
		Use:   "route",
		Short: "快捷命令: 路由相关操作",
		Long:  `项目中的路由相关操作`,
		Args:  cobra.NoArgs,
	}

	routeCmd.AddCommand(NewRefreshCmd(f))

	return routeCmd
}
