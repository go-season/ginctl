package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-season/ginctl/cmd/route"

	"github.com/go-season/ginctl/pkg/util/str"

	"github.com/go-season/ginctl/pkg/util/file"

	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

type CleanCmd struct {
	APIName string
}

func NewCleanCmd(f factory.Factory) *cobra.Command {
	cmd := &CleanCmd{}

	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "清除API关联的指定文件",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	cleanCmd.Flags().StringVar(&cmd.APIName, "api", "", "指定你要清除的API名称.")
	cleanCmd.MarkFlagRequired("api")

	return cleanCmd
}

func (cmd *CleanCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	if cmd.APIName == "" {
		return errors.New("you need specify an API name")
	}

	if str.IsSnakeCase(cmd.APIName) {
		cmd.APIName = str.SnakeToCamel(cmd.APIName)
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	restPath := fmt.Sprintf("%s/api/rest/%s", wd, strings.ToLower(cmd.APIName))
	rfound, err := file.PathExists(restPath)
	if err != nil {
		return err
	}
	if !rfound {
		f.GetLog().Warnf("dir %s not exists, ignore clean", restPath)
	}
	typespecPath := fmt.Sprintf("%s/api/typespec/%stype", wd, strings.ToLower(cmd.APIName))
	tfound, err := file.PathExists(typespecPath)
	if err != nil {
		return err
	}
	if !tfound {
		f.GetLog().Warnf("dir %s not exists, ignore clean", typespecPath)
	}
	appPath := fmt.Sprintf("%s/application/%s", wd, strings.ToLower(cmd.APIName))
	afound, err := file.PathExists(appPath)
	if err != nil {
		return err
	}
	if !afound {
		f.GetLog().Warnf("dir %s not exists, ignore clean", appPath)
	}
	servicePath := fmt.Sprintf("%s/service/%s.go", wd, str.ToSnakeCase(cmd.APIName))
	sfound, err := file.PathExists(servicePath)
	if err != nil {
		return err
	}
	if !sfound {
		f.GetLog().Warnf("file %s not exists, ignore clean", servicePath)
	}
	modelPath := fmt.Sprintf("%s/model/%s.go", wd, str.ToSnakeCase(cmd.APIName))
	mfound, err := file.PathExists(modelPath)
	if err != nil {
		return err
	}
	if !mfound {
		f.GetLog().Warnf("file %s not exists, ignore clean", modelPath)
	}
	f.GetLog().WriteString("\n")
	if rfound {
		f.GetLog().Info("cleaning dir %s...", restPath)
		err := os.RemoveAll(restPath)
		if err != nil {
			return err
		}
	}
	if tfound {
		f.GetLog().Info("cleaning dir %s...", typespecPath)
		err := os.RemoveAll(typespecPath)
		if err != nil {
			return err
		}
	}
	if afound {
		f.GetLog().Info("cleaning dir %s...", appPath)
		err := os.RemoveAll(appPath)
		if err != nil {
			return err
		}
	}
	if sfound {
		f.GetLog().Info("cleaning file %s...", servicePath)
		err := os.Remove(servicePath)
		if err != nil {
			return err
		}
	}
	if mfound {
		f.GetLog().Info("cleaning file %s...", modelPath)
		err := os.Remove(modelPath)
		if err != nil {
			return err
		}
	}

	(&route.RefreshCmd{}).Run(f, cobraCmd, nil)

	f.GetLog().WriteString("\n")
	f.GetLog().Donef("cleaned API successful.")

	return nil
}
