package pkg

import (
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

func NewPkgCmd(f factory.Factory) *cobra.Command {
	pkgCmd := &cobra.Command{
		Use:   "pkg",
		Short: "快捷命令: PKG相关操作",
		Args:  cobra.NoArgs,
	}

	pkgCmd.AddCommand(NewRequireCmd(f))
	pkgCmd.AddCommand(NewUpdateCmd(f))
	pkgCmd.AddCommand(NewPublishCmd(f))

	return pkgCmd
}
