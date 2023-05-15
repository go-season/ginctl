package cc

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/spf13/cobra"
)

type initCmd struct {
	Env string
}

const (
	EnvTest = "test"
	EnvPre  = "pre"
	EnvProd = "prod"
	EnvAll  = "all"
)

func NewInitCmd(f factory.Factory) *cobra.Command {
	cmd := &initCmd{}

	icmd := &cobra.Command{
		Use:   "init",
		Short: "生成配置中心客户端配置文件",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	icmd.Flags().StringVarP(&cmd.Env, "env", "e", "all", "指定想要生成配置文件的相应环境，可支持的值:test,pre,prod,all，默认是all，即生成所有环境的")

	return icmd
}

func (cmd *initCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	prjPath, err := getProjectPath()
	if err != nil {
		return err
	}

	f.GetLog().Info("开始生成dotEnv相关配置文件...")

	// create test dotEnv
	if cmd.Env == EnvTest || cmd.Env == EnvAll {
		testConfFile, err := os.Create(fmt.Sprintf("%s/config/dotEnvGenerator_test.yaml", cwd))
		if err != nil {
			return err
		}
		defer testConfFile.Close()
		data1 := &struct {
			ZooKeeperHost string
			NodePath      string
		}{
			ZooKeeperHost: "  - pg-openstack-test-zk-00.idc.xiaozhu.com:2181",
			NodePath:      prjPath + "/test",
		}
		tpl1 := template.Must(template.New("testConf").Parse(string(getConfigTpl())))
		err = tpl1.Execute(testConfFile, data1)
		if err != nil {
			return nil
		}

		f.GetLog().Donef("生成测试环境dotEnv配置文件:%s/config/dotEnvGenerator_test.yaml", cwd)
	}

	// create pre dotEnv
	if cmd.Env == EnvPre || cmd.Env == EnvAll {
		preConfFile, err := os.Create(fmt.Sprintf("%s/config/dotEnvGenerator_pre.yaml", cwd))
		if err != nil {
			return err
		}
		defer preConfFile.Close()
		data2 := &struct {
			ZooKeeperHost string
			NodePath      string
		}{
			ZooKeeperHost: "- qconf-zk-node-n1.idc.xiaozhu.com:2181\n" + "  - qconf-zk-node-n2.idc.xiaozhu.com:2181\n" + "  - qconf-zk-node-n3.idc.xiaozhu.com:2181",
			NodePath:      prjPath + "/pre",
		}
		tpl2 := template.Must(template.New("preConf").Parse(string(getConfigTpl())))
		err = tpl2.Execute(preConfFile, data2)
		if err != nil {
			return nil
		}

		f.GetLog().Donef("生成预发环境dotEnv配置文件:%s/config/dotEnvGenerator_pre.yaml", cwd)
	}

	// create prod dotEnv
	if cmd.Env == EnvProd || cmd.Env == EnvAll {
		prodConfFile, err := os.Create(fmt.Sprintf("%s/config/dotEnvGenerator_prod.yaml", cwd))
		if err != nil {
			return err
		}
		defer prodConfFile.Close()
		data3 := &struct {
			ZooKeeperHost string
			NodePath      string
		}{
			ZooKeeperHost: "- qconf-zk-node-n1.idc.xiaozhu.com:2181\n" + "  - qconf-zk-node-n2.idc.xiaozhu.com:2181\n" + "  - qconf-zk-node-n3.idc.xiaozhu.com:2181",
			NodePath:      prjPath + "/prod",
		}
		tpl3 := template.Must(template.New("preConf").Parse(string(getConfigTpl())))
		err = tpl3.Execute(prodConfFile, data3)
		if err != nil {
			return nil
		}

		f.GetLog().Donef("生成生产环境dotEnv配置文件:%s/config/dotEnvGenerator_prod.yaml", cwd)
	}

	return nil
}

func getProjectPath() (string, error) {
	cmdStr := "git config --get remote.origin.url"
	output, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return "", err
	}
	remoteUrl := string(output)
	re := regexp.MustCompile(`github.com:?(.*?)\.git`)
	match := re.FindStringSubmatch(remoteUrl)
	if len(match) == 0 {
		return "", errors.New("parse error, can't get git remote addr.")
	}

	path := "/" + strings.TrimPrefix(match[1], "/")

	return path, nil
}

func getConfigTpl() []byte {
	return []byte(`hosts:
  {{ .ZooKeeperHost }}
timeInterval: 1
retryTime: 3
exitWhenError: true
backupNum: 3
envPath: .env
root: {{ .NodePath }}
hasShadow: false
shadowRoot: {{ .NodePath }}/shadow
logFile: out.log
debugMode: false
outputFile: config/app.yaml
alarmToken: 3d3cb349dd4c78acffaf29db990a15007923a88b0f283e4903eba8e195b2195a
alarmGateway: http://10.4.13.171:8091
`)
}
