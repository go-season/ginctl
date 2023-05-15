package faker

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-season/ginctl/pkg/util/file"

	pkgfaker "github.com/go-season/ginctl/pkg/faker"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

type generateCmd struct {
	log log.Logger

	config  string
	preview bool
}

func NewGenerateCmd(f factory.Factory) *cobra.Command {
	cmd := &generateCmd{
		log: f.GetLog(),
	}

	gcmd := &cobra.Command{
		Use:     "generate -c File",
		Aliases: []string{"g", "gen"},
		Short:   "生成faker相关provide代码",
		Args:    cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	gcmd.Flags().StringVarP(&cmd.config, "config", "c", "", "指定要生成的faker配置文件")
	gcmd.Flags().BoolVarP(&cmd.preview, "preview", "p", false, "是否预览模式，而不写入文件")

	_ = gcmd.MarkFlagRequired("config")

	return gcmd
}

func (g *generateCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, _ := os.Getwd()

	path := g.config
	if idx := strings.Index(g.config, "./"); idx != -1 {
		path = wd + "/" + g.config[idx+2:]
	} else if strings.Index(g.config, "/") != 0 {
		path = wd + "/" + g.config
	}

	outDir := wd + "/pkg/faker/"
	exists, err := file.PathExists(outDir)
	if err != nil {
		return err
	}
	if !exists {
		os.Mkdir(outDir, 0755)
	}
	os.Remove(fmt.Sprintf("%s/faker.go", outDir))

	f.GetLog().WriteString("\n")
	f.GetLog().Infof("parse config file: %s", path)

	gen := pkgfaker.NewGenerator(pkgfaker.WithPreview(g.preview), pkgfaker.WithWorkDir(wd))
	if err := gen.Parse(path); err != nil {
		return err
	}

	f.GetLog().StartWait("crafting faker provide...")
	time.Sleep(1 * time.Second)

	gen.GenFaker()

	f.GetLog().Done("generated successful. now you can preview faker/faker.go")

	return nil
}
