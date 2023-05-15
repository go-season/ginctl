package apitest

import (
	"os"
	"time"

	"github.com/go-season/ginctl/pkg/util"

	"github.com/go-season/ginctl/pkg/apitest"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

type generateCmd struct {
}

func NewGenerateCmd(f factory.Factory) *cobra.Command {
	cmd := &generateCmd{}

	genCmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"g", "gen"},
		Short:   "生成APITest相关代码",
		Args:    cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	return genCmd
}

func (g *generateCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, _ := os.Getwd()

	gen := apitest.NewGenerator(apitest.WithLog(f.GetLog()), apitest.WithWorkDir(wd))

	f.GetLog().StartWait("crafting api test...")
	time.Sleep(1 * time.Second)

	if err := gen.Parse(); err != nil {
		return err
	}

	if err := util.ReloadModule(); err != nil {
		return err
	}

	f.GetLog().StopWait()
	f.GetLog().Done("generated api test successful.")

	return nil
}
