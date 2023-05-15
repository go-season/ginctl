package polyfill

import (
	"fmt"
	"os"
	"text/template"

	tpl2 "github.com/go-season/ginctl/tpl"

	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

const (
	OverwriteTrue  = "true"
	OverwriteFalse = "false"
)

type responseCmd struct {
	WithCamelTimestamp bool
}

func newResponseCmd(f factory.Factory) *cobra.Command {
	cmd := &responseCmd{}

	responsePolyfillCmd := &cobra.Command{
		Use:   "response",
		Short: "兼容通用响应结构体，为老项目提供一种快速过渡的方式",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunPolyfillResponse(f, cobraCmd, args)
		},
	}

	responsePolyfillCmd.Flags().BoolVar(&cmd.WithCamelTimestamp, "camel-ts", false, "是否生成驼峰的Timestamp格式，默认是小写格式")

	return responsePolyfillCmd
}

func (cmd *responseCmd) RunPolyfillResponse(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	dirPath := fmt.Sprintf("%s/pkg/http", cwd)
	foundHTTP, err := file.PathExists(dirPath)
	if err != nil {
		return err
	}
	if foundHTTP {
		filePath := fmt.Sprintf("%s/response.go", dirPath)
		found, err := file.PathExists(filePath)
		if err != nil {
			return err
		}
		if found {
			overwrite, err := f.GetLog().Question(&log.QuestionOptions{
				Question:     fmt.Sprintf("%s已经存在，该操作会覆盖原始内容，请确认是否要覆盖？"),
				DefaultValue: "false",
				Options: []string{
					OverwriteTrue,
					OverwriteFalse,
				},
			})
			if err != nil {
				return err
			}
			if overwrite == OverwriteFalse {
				return nil
			}
		}
	}

	if !foundHTTP {
		err = os.Mkdir(dirPath, 0755)
		if err != nil {
			return err
		}
	}
	responseFile, err := os.Create(fmt.Sprintf("%s/response.go", dirPath))
	if err != nil {
		return err
	}
	defer responseFile.Close()
	tsJsonTag := "`json:\"timestamp\"`"
	if cmd.WithCamelTimestamp {
		tsJsonTag = "`json:\"timeStamp\"`"
	}
	responsePkgData := &struct {
		StatusJSONTag    string
		ContentJSONTag   string
		ErrorMsgJSONTag  string
		TimestampJSONTag string
	}{
		StatusJSONTag:    "`json:\"status\"`",
		ContentJSONTag:   "`json:\"content\"`",
		ErrorMsgJSONTag:  "`json:\"errorMsg\"`",
		TimestampJSONTag: tsJsonTag,
	}
	tpl := template.Must(template.New("httpPkg").Parse(string(tpl2.HTTPResponseTemplate())))
	err = tpl.Execute(responseFile, responsePkgData)
	if err != nil {
		return err
	}

	f.GetLog().Donef("生成响应结构体成功，在文件:%s", fmt.Sprintf("%s/response.go", dirPath))

	return nil
}
