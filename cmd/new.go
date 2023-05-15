package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	"github.com/go-season/ginctl/pkg/util"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	tpl2 "github.com/go-season/ginctl/tpl"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
)

const SkeletonGitLabRepo = "https://github.com/go-season/gin-skeleton.git"

type newCmd struct {
	log log.Logger

	PkgName    string
	CamelTSTag bool
}

func NewNewCmd(f factory.Factory) *cobra.Command {
	cmd := &newCmd{
		log: f.GetLog(),
	}

	newCmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "创建一个项目模板",
		Args:  cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	newCmd.Flags().StringVar(&cmd.PkgName, "pkg-name", "", "指定模块名称.")
	newCmd.Flags().BoolVar(&cmd.CamelTSTag, "camel-ts", false, "指定框架生成的响应结构中timestamp是否开启驼峰格式，默认关闭.")

	return newCmd
}

func (cmd *newCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	err := checkGoVersion()
	if err != nil {
		return err
	}

	projectName := args[0]

	wd, err := os.Getwd()
	if err != nil {
		cmd.log.Error()
		return err
	}

	projectPath := wd + "/" + projectName
	exists, err := file.PathExists(projectPath)
	if err != nil {
		cmd.log.Error()
		return err
	}
	if exists == true {
		err = errors.New("the project is exists, please use other name to create")
		return err
	}

	cmd.log.Info("Crafting gin skeleton...")
	_, err = git.PlainClone(projectPath, false, &git.CloneOptions{
		URL: SkeletonGitLabRepo,
	})
	if err != nil {
		return err
	}

	rootPkgName := projectName
	if cmd.PkgName != "" {
		rootPkgName = cmd.PkgName
	}

	data := &struct {
		RootPkgName string
	}{
		RootPkgName: rootPkgName,
	}

	// generate %project%/pkg/http/response.go
	err = os.Mkdir(fmt.Sprintf("%s/pkg/http", projectPath), 0755)
	if err != nil {
		return err
	}
	responseFile, err := os.Create(fmt.Sprintf("%s/pkg/http/response.go", projectPath))
	if err != nil {
		return err
	}
	defer responseFile.Close()
	tsJsonTag := "`json:\"timestamp\"`"
	if cmd.CamelTSTag {
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

	// regenerate %project%/api/rest/router.go
	routerFile, err := os.Create(fmt.Sprintf("%s/api/rest/router.go", projectPath))
	if err != nil {
		return err
	}
	defer routerFile.Close()

	tpl = template.Must(template.New("router").Parse(string(tpl2.RouterTemplate())))
	err = tpl.Execute(routerFile, data)
	if err != nil {
		return err
	}

	// regenerate %project%/cmd/apiserver/main.go
	mainFile, err := os.Create(fmt.Sprintf("%s/cmd/apiserver/main.go", projectPath))
	if err != nil {
		return err
	}
	defer mainFile.Close()

	tpl = template.Must(template.New("entry").Parse(string(tpl2.ServiceEntryTemplate())))
	err = tpl.Execute(mainFile, data)
	if err != nil {
		return err
	}

	// regenerate %project%/go.mod
	goModFile, err := os.Create(fmt.Sprintf("%s/go.mod", projectPath))
	if err != nil {
		return err
	}
	defer goModFile.Close()

	tpl = template.Must(template.New("gomod").Parse(string(tpl2.PkgModuleTemplate())))
	err = tpl.Execute(goModFile, data)
	if err != nil {
		return err
	}

	fl, err := os.OpenFile(fmt.Sprintf("%s/.gitignore", projectPath), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fl.Close()
	if _, err := fl.WriteString("\n" + rootPkgName); err != nil {
		return err
	}

	// generate %project%/app_dev.yaml for local development
	appDefConfFile, err := os.Create(fmt.Sprintf("%s/config/app_dev.yaml", projectPath))
	if err != nil {
		return err
	}
	defer appDefConfFile.Close()

	tpl = template.Must(template.New("devconf").Parse(string(tpl2.DevAppConfigTemplate())))
	err = tpl.Execute(appDefConfFile, nil)
	if err != nil {
		return err
	}

	// generate %project%/cmd/cron/main.go
	importPath, err := util.GetPkgName(fmt.Sprintf("%s/cmd/cron/cmd", projectPath))
	if err != nil {
		return err
	}
	cronData := struct {
		SubCmdPath string
	}{
		SubCmdPath: importPath,
	}

	cronEntryFile, err := os.Create(fmt.Sprintf("%s/cmd/cron/main.go", projectPath))
	if err != nil {
		return err
	}
	defer cronEntryFile.Close()
	tpl = template.Must(template.New("cron").Parse(string(tpl2.CronEntryTemplate())))
	err = tpl.Execute(cronEntryFile, cronData)
	if err != nil {
		return err
	}

	err = os.RemoveAll(projectPath + "/.git")
	if err != nil {
		return err
	}

	f.GetLog().Infof("download go mod dependency")
	ecmd := exec.Command("go", "mod", "tidy")
	ecmd.Dir = projectPath
	output, err := ecmd.Output()
	if err != nil {
		cmd.log.Debug("Initialized project failed")
		return err
	}

	cmd.log.Write(output)
	cmd.log.WriteString("\n")

	cmd.log.Done("Project successfully initialized")

	cmd.log.WriteString("\n")
	cmd.log.Infof("Now you can:\n\n	cd %s\n	vim %s\n	run %s", ansi.Color(projectName, "cyan+b"), ansi.Color("config/app_dev.yaml", "cyan+b"), ansi.Color("ginctl run", "cyan+b"))

	return nil
}

func checkGoVersion() error {
	version := runtime.Version()
	semver := strings.TrimPrefix(version, "go")
	parts := strings.Split(semver, ".")
	if parts[0] == "1" && parts[1] == "15" {
		return nil
	}

	return errors.New(fmt.Sprintf("incorrect go version: %s, please upgrade to v1.15", semver))
}
