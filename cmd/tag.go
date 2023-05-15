package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-season/ginctl/pkg/ginctl/doc"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type tagCmd struct {
	log log.Logger

	file             string
	typeName         string
	tag              string
	dryRun           bool
	save             bool
	propertyStrategy string
}

func NewTagCmd(f factory.Factory) *cobra.Command {
	cmd := &tagCmd{
		log: f.GetLog(),
	}

	tagCmd := &cobra.Command{
		Use:   "tag",
		Short: "快速生成项目指定结构体相关tag",
		Long: `为项目指定结构体生成相关tag

命令样例:
ginctl tag [file] [--tag -t form|json|gorm]
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	tagCmd.Flags().StringVarP(&cmd.tag, "tag", "t", "json", "Tag name that you want to add, comma separated, support form,json,gorm, default json")
	tagCmd.Flags().StringVar(&cmd.typeName, "tname", "", "The type name that you want to add tag, comma separated")
	tagCmd.Flags().BoolVar(&cmd.dryRun, "dry-run", true, "The default mode, just show change on stdout")
	tagCmd.Flags().BoolVarP(&cmd.save, "save", "s", false, "Rewrite stdout to file, is dry-run reverse operation")
	tagCmd.Flags().StringVarP(&cmd.propertyStrategy, "propertyStrategy", "p", "camelcase", "Property Naming Strategy like snakecase,camelcase,pascalcase")

	return tagCmd
}

func (cmd *tagCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	filepath := args[0]
	absolutePath := fmt.Sprintf("%s/%s", cwd, filepath)
	found, err := file.PathExists(absolutePath)
	if err != nil {
		return err
	}
	if !found {
		return errors.New(fmt.Sprintf("file %s not in path: %s, please specify correct path", filepath, cwd))
	}
	lastIndex := strings.LastIndex(filepath, "/")
	fileDir := filepath[:lastIndex]

	pkgs := doc.NewPackagesDefinitions()
	parser := doc.NewParser(doc.WithWorkDir(cwd), doc.WithPackagesDefinitions(pkgs))
	err = parser.ParseAPI(fileDir)
	if err != nil {
		return err
	}

	if cmd.save {
		cmd.dryRun = false
	}

	tagFlags := strings.Split(cmd.tag, ",")
	typeMap := make(map[string]bool)
	var typNames []string
	if cmd.typeName != "" {
		typNames = strings.Split(cmd.typeName, ",")
	}
	for _, typName := range typNames {
		typeMap[typName] = true
	}
	err = parser.Packages.RangeFileForInjectTag(cmd.dryRun, filepath, cmd.propertyStrategy, tagFlags, typeMap, parser.ParseTypeSpec)
	if err != nil {
		return err
	}

	if cmd.dryRun {
		cmd.log.WriteString("\n")
		cmd.log.Done("generate tag successful, you can specify --dry-run false save stdout to file")
	} else {
		cmd.log.Donef("generate tag successful, then you can open your file: %s code it", ansi.Color(absolutePath, "cyan+b"))
	}

	return nil
}
