package sdk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"

	"github.com/go-season/ginctl/pkg/ginctl/sdk"
	sdkpkg "github.com/go-season/ginctl/pkg/sdk"
	"github.com/go-season/ginctl/pkg/util"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

type GenerateCmd struct {
	All        bool
	GoOut      string
	PHPOut     string
	VueOut     string
	Publish    bool
	PublishOld bool
	Verbose    bool
}

var internalRefs = map[string]bool{
	"typespec": true,
	"orm":      true,
	"time":     true,
}

func NewGenerateCmd(f factory.Factory) *cobra.Command {
	cmd := GenerateCmd{}

	sdkCmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen", "g"},
		Short:   "快速生成项目SDK文件",
		Args:    cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	//sdkCmd.Flags().StringVar(&cmd.GoOut, "go_out", "", "指定要生成Go版本SDK路径.")
	//sdkCmd.Flags().StringVar(&cmd.PHPOut, "php_out", "", "指定要生成PHP版本SDK路径.")
	//sdkCmd.Flags().StringVar(&cmd.VueOut, "vue_out", "", "指定要生成Vue版本SDK路径.")
	sdkCmd.Flags().BoolVar(&cmd.All, "all", false, "是否指定要扫描所有的接口定义文件?")
	sdkCmd.Flags().BoolVar(&cmd.Publish, "publish", false, "是否发布SDK到远程仓库中?")
	sdkCmd.Flags().BoolVar(&cmd.PublishOld, "publish_old", false, "是否发布老版本SDK到远程仓库中?")
	sdkCmd.Flags().BoolVarP(&cmd.Verbose, "verbose", "v", false, "输出详细信息")

	//sdkCmd.MarkFlagRequired("go_out")

	return sdkCmd
}

var excludeFiles = map[string]bool{
	"base.go":   true,
	"readme.md": true,
}

func (cmd *GenerateCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := fmt.Sprintf("%s/api/typespec", wd)
	scanDirs := make([]string, 0)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if _, ok := excludeFiles[info.Name()]; ok {
			return nil
		}

		dir := strings.TrimPrefix(path, wd+"/")
		if !info.IsDir() && dir != "" {
			scanDirs = append(scanDirs, dir)
		}

		return nil
	})

	answer := strings.Join(scanDirs, ",")
	if !cmd.All {
		answer, err = f.GetLog().Question(&log.QuestionOptions{
			Question:      "Please select the scan file.",
			Options:       scanDirs,
			IsMultiSelect: true,
		})
		if err != nil {
			return err
		}
	}

	cmd.GoOut = fmt.Sprintf("%s/sdk/%s", wd, strings.Replace(util.GetModuleName(wd), "-", "", -1))
	found, err := file.PathExists(cmd.GoOut)
	if err != nil {
		return err
	}
	if !found {
		err = os.MkdirAll(cmd.GoOut, 0755)
		if err != nil {
			return err
		}
	}

	parts := strings.Split(answer, ",")
	f.GetLog().Info("Start crafting sdk for go project...")
	f.GetLog().WriteString(fmt.Sprintf("%s Generate project sdk...\n", time.Now().Format("2006/01/02 15:04:05")))
	f.GetLog().WriteString(fmt.Sprintf("%s Generate general SDK Info, search dir:./api\n", time.Now().Format("2006/01/02 15:04:05")))

	baseTypPath := fmt.Sprintf("%s/api/typespec/base.go", wd)
	found, err = file.PathExists(baseTypPath)
	if err != nil {
		return err
	}
	if found {
		if err := sdkpkg.GenerateBase(baseTypPath, cmd.GoOut, wd, f.GetLog()); err != nil {
			return err
		}
	}

	pkgList := make([]string, 0)
	serviceList := make([]string, 0)
	for _, part := range parts {
		filePath := strings.Replace(filepath.Dir(part)+"type/"+filepath.Base(part), "rest", "typespec", 1)
		f.GetLog().WriteString(fmt.Sprintf("%s Parsing %s\n", time.Now().Format("2006/01/02 15:04:05"), strings.TrimPrefix(filePath, "api/")))

		g := sdkpkg.NewGenerator(sdkpkg.WithLogger(f.GetLog()), sdkpkg.WithWorkDir(wd), sdkpkg.WithOld(cmd.PublishOld), sdkpkg.WithPublish(cmd.Publish))
		if err := g.Parse(part); err != nil {
			return err
		}

		g.GenGO()

		pkgList = append(pkgList, strings.Replace(g.FileName, "_", "", -1))
		serviceList = append(serviceList, g.APIName)
	}

	if cmd.Publish || cmd.PublishOld {
		home, _ := homedir.Dir()
		os.Remove(fmt.Sprintf("%s/.ginctl", home))
		found, err := file.PathExists(sdk.GetLocalPath())
		if err != nil {
			return err
		}
		if !found {
			if err := sdk.CloneRepoToLocal(sdk.GetLocalPath()); err != nil {
				return err
			}
		}

		f.GetLog().Info("start publish sdk...")

		sdkpkg.GenerateClientFactory(pkgList, serviceList, cmd.GoOut, true, cmd.PublishOld)
		if err := sdk.PublishRepo(cmd.GoOut, true, cmd.PublishOld, cmd.Verbose, f.GetLog()); err != nil {
			sdkpkg.GenerateClientFactory(pkgList, serviceList, cmd.GoOut, false, cmd.PublishOld)
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/sdk", wd))
		f.GetLog().Donef("publish sdk completed.")
	} else {
		sdkpkg.GenerateClientFactory(pkgList, serviceList, cmd.GoOut, false, cmd.PublishOld)
		f.GetLog().Donef("generate go sdk successful. !:)")
	}

	return nil
}
