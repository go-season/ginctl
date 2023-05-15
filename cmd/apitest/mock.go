package apitest

import (
	"os"
	"time"

	"github.com/go-season/ginctl/pkg/mock"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

type MockCmd struct {
	config string
}

func NewMockCmd(f factory.Factory) *cobra.Command {
	cmd := &MockCmd{}

	mockCmd := &cobra.Command{
		Use:     "mock",
		Aliases: []string{"m"},
		Short:   "生成mock客户端代码或配置",
		Args:    cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	mockCmd.Flags().StringVarP(&cmd.config, "config", "c", "", "请指定要生成Mock客户端的配置文件.")

	_ = mockCmd.MarkFlagRequired("config")

	return mockCmd
}

func (m *MockCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, _ := os.Getwd()

	g := mock.NewGenerator(mock.WithWorkDir(wd))

	f.GetLog().Infof("parsing config %s", m.config)

	if err := g.Parse(m.config); err != nil {
		return err
	}

	f.GetLog().StartWait("crafting mock client...")

	time.Sleep(1 * time.Second)

	if err := g.GenMockClient(); err != nil {
		return err
	}

	f.GetLog().StopWait()
	f.GetLog().Done("generate mock client in pkg/mock/mock.go successful.")

	return nil
}
