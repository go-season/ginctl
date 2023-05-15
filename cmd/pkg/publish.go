package pkg

import (
	"errors"
	"os"
	"strings"

	"github.com/go-season/ginctl/pkg/ginctl/pkg"

	"github.com/go-season/ginctl/pkg/util"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

type PublishCmd struct{}

func NewPublishCmd(f factory.Factory) *cobra.Command {
	cmd := PublishCmd{}

	publishCmd := &cobra.Command{
		Use:     "publish",
		Aliases: []string{"pub", "p"},
		Short:   "发布包到内部包管理系统",
		Args:    cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	return publishCmd
}

func (cmd *PublishCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, _ := os.Getwd()

	f.GetLog().Info("开始推送包...")

	module := util.GetModuleName(wd)
	if !strings.HasPrefix(module, "github.com/") {
		return errors.New("无效的包项目，请将包模块名用: `github.com/xxx/xxx`格式")
	}

	if err := checkPkgIsExists(module, f); err != nil {
		return nil
	}

	if err := pkg.Publish(module); err != nil {
		return err
	}

	f.GetLog().Donef("发布包%s成功", module)

	return nil
}

func checkPkgIsExists(module string, f factory.Factory) error {
	list, err := pkg.List()
	if err != nil {
		return err
	}

	mmap := make(map[string]bool)
	for _, m := range list {
		mmap[m] = true
	}

	if _, ok := mmap[module]; ok {
		f.GetLog().Error("包已经被发布，请不要重复发布")
		os.Exit(1)
	}

	return nil
}
