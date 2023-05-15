package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"syscall"

	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

var rebuild bool

type CronCmd struct {
	log log.Logger
}

func NewCronCmd(f factory.Factory) *cobra.Command {
	cmd := &CronCmd{
		log: f.GetLog(),
	}
	cronCmd := &cobra.Command{
		Use:   "cron",
		Short: "proxy cron cmd and rebuild when file changed.",
		Long:  "proxy cron cmd and rebuild when file changed.",
	}

	if len(os.Args) <= 1 || os.Args[1] != "cron" {
		return cronCmd
	}

	wd, _ := os.Getwd()

	var paths []string
	cmd.selectWatchPaths(fmt.Sprintf("%s/cmd/cron", wd), &paths)

	files := []string{"cmd/cron/main.go"}
	cmd.autoBuild(files)

	executablePath := fmt.Sprintf("%s/cron", wd)

	if err := syscall.Exec(executablePath, append([]string{executablePath}, os.Args[2:]...), os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cronCmd
}

func (cmd *CronCmd) autoBuild(files []string) {
	var (
		err    error
		stderr bytes.Buffer
	)

	cmdName := "go"
	args := []string{"build"}
	args = append(args, "-o", "cron")
	args = append(args, files...)

	ecmd1 := exec.Command(cmdName, args...)
	ecmd1.Env = append(os.Environ(), "GOGC=off")
	ecmd1.Stderr = &stderr

	state.Lock()
	if err = ecmd1.Run(); err != nil {
		state.Unlock()
		cmd.log.Errorf("Failed to build the cron script: %s with err: %v", stderr.String(), err)
		return
	}
	state.Unlock()
}

func (cmd *CronCmd) restart(appname, subCmd string) {
	cmd.log.WriteString("\n")
	cmd.log.Infof("Execute '%s'...", appname)
	cmd.log.WriteString("\n")
	if !strings.Contains(appname, "./") {
		appname = "./" + appname
	}
	var args []string
	if subCmd != "" {
		args = append(args, subCmd)
	}
	ecmd = exec.Command(appname, args...)
	ecmd.Env = []string{"APP_ENV=dev"}
	ecmd.Stdout = os.Stdout
	ecmd.Stderr = os.Stderr

	if err := ecmd.Run(); err != nil {
		cmd.log.Errorf("running %s failed...", appname)
	}

	answer, err := cmd.log.Question(&log.QuestionOptions{
		Question: "input you subcommand:",
	})
	if err != nil {
		return
	}

	cmd.restart(appname, answer)
}

func (cmd *CronCmd) shouldIgnoreFile(filename string) bool {
	for _, regex := range ignoredFilesRegExps {
		r, err := regexp.Compile(regex)
		if err != nil {
			cmd.log.Fatalf("Could not compile regular expression: %s", err)
		}
		if r.MatchString(filename) {
			return true
		}
		continue
	}
	return false
}

func (cmd *CronCmd) shouldWatchFileWithExtension(name string) bool {
	for _, s := range watchExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

func (cmd *CronCmd) selectWatchPaths(dir string, paths *[]string) {
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	useDir := false
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() && fileInfo.Name()[:1] != "." {
			cmd.selectWatchPaths(dir+"/"+fileInfo.Name(), paths)
			continue
		}
		if useDir {
			continue
		}

		if path.Ext(fileInfo.Name()) == ".go" {
			*paths = append(*paths, dir)
			useDir = true
		}
	}
}
