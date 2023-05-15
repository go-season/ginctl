package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/file"
	"github.com/go-season/ginctl/pkg/util/log"
	"github.com/spf13/cobra"
)

type runCmd struct {
	log log.Logger

	Env      string
	MainFile string
}

var (
	ecmd                *exec.Cmd
	currpath            string
	appname             string
	exit                chan bool
	state               sync.Mutex
	eventTime           = make(map[string]int64)
	scheduleTime        time.Time
	defaultMainFile     = "cmd/apiserver/main.go"
	watchExts           = []string{".go"}
	ignoredFilesRegExps = []string{
		`.#(\w+).go$`,
		`.(\w+).go.swp$`,
		`.(\w+).go~$`,
		`.(\w+).tmp$`,
	}
)

var started = make(chan bool)

func NewRunCmd(f factory.Factory) *cobra.Command {
	cmd := &runCmd{
		log: f.GetLog(),
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "运行本地开发服务",
		Args:  cobra.NoArgs,
		Run: func(cobraCmd *cobra.Command, args []string) {
			cmd.Run(f, cobraCmd, args)
		},
	}

	runCmd.Flags().StringVarP(&cmd.Env, "env", "e", "dev", "指定当前运行环境，默认是: dev")
	runCmd.Flags().StringVar(&cmd.MainFile, "main", "", "指定入口路径, 默认是: cmd/apiserver/main.go")

	return runCmd
}

func (cmd *runCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) {
	log.PrintLogo()

	appPath, _ := os.Getwd()

	addBuildFileToIgnoreIfNotIn(appPath)

	var paths []string
	cmd.readAppDir(appPath, &paths)
	appname = path.Base(appPath)

	mainFile := defaultMainFile
	if cmd.MainFile != "" {
		mainFile = cmd.MainFile
	}
	found, err := file.PathExists(mainFile)
	if err != nil {
		panic(err)
	}
	if !found {
		mainFile = "cmd/main.go"
	}

	files := []string{mainFile}

	cmd.newWatcher(paths, files)
	cmd.autoBuild(files)

	for {
		<-exit
		runtime.Goexit()
	}
}

func (cmd *runCmd) readAppDir(dir string, paths *[]string) {
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	useDir := false
	for _, fileInfo := range fileInfos {
		if strings.HasSuffix(fileInfo.Name(), "docs") {
			continue
		}
		if strings.HasSuffix(fileInfo.Name(), "cron") {
			continue
		}
		if fileInfo.IsDir() && fileInfo.Name()[0:1] != "." {
			cmd.readAppDir(dir+"/"+fileInfo.Name(), paths)
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

func (cmd *runCmd) newWatcher(paths []string, files []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cmd.log.Failf("Failed to create watcher: %s", err)
	}

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				isBuild := true

				if cmd.shouldIgnoreFile(e.Name) {
					continue
				}

				if !cmd.shouldWatchFileWithExtension(e.Name) {
					continue
				}

				mt := file.GetFileModTime(e.Name)
				if t := eventTime[e.Name]; mt == t {
					cmd.log.Infof("Skipping: %s", e.String())
					isBuild = false
				}

				eventTime[e.Name] = mt

				if isBuild {
					cmd.log.Infof("Event fired: %s", e)
					go func() {
						scheduleTime = time.Now().Add(1 * time.Second)
						time.Sleep(time.Until(scheduleTime))
						cmd.autoBuild(files)
					}()
				}
			case err := <-watcher.Errors:
				cmd.log.Warnf("Watcher error: %s", err.Error())
			}
		}
	}()
	cmd.log.Infof("Initializing watcher...")
	for _, p := range paths {
		cmd.log.Infof("Watching: %s", p)
		err = watcher.Add(p)
		if err != nil {
			cmd.log.Fatalf("Failed to watch directory: %s", err)
		}
	}
}

func (cmd *runCmd) autoBuild(files []string) {
	var (
		err    error
		stderr bytes.Buffer
	)

	cmdName := "go"
	appname = "app"
	args := []string{"build"}
	args = append(args, "-o", appname)
	args = append(args, files...)

	bcmd := exec.Command(cmdName, args...)
	bcmd.Env = append(os.Environ(), "GOGC=off")
	bcmd.Stderr = &stderr
	state.Lock()
	err = bcmd.Run()
	if err != nil {
		state.Unlock()
		cmd.log.Errorf("Failed to build the application: %s with err: %v", stderr.String(), err)
		return
	}
	state.Unlock()

	cmd.log.Donef("Built Successfully!")
	cmd.restart(appname)
}

func (cmd *runCmd) restart(appname string) {
	cmd.log.Debugf("Kill running process", file.FILE(), file.LINE())
	cmd.kill()
	cmd.start(appname)
}

func (cmd *runCmd) kill() {
	defer func() {
		if e := recover(); e != nil {
			cmd.log.Info("Kill recover: %s", e)
		}
	}()
	if ecmd != nil && ecmd.Process != nil {
		if runtime.GOOS == "windows" {
			ecmd.Process.Signal(os.Kill)
		} else {
			ecmd.Process.Signal(os.Interrupt)
		}

		ch := make(chan struct{}, 1)
		go func() {
			ecmd.Wait()
			ch <- struct{}{}
		}()

		select {
		case <-ch:
			return
		case <-time.After(10 * time.Second):
			cmd.log.Info("Timeout. Force kill cmd process")
			err := ecmd.Process.Kill()
			if err != nil {
				cmd.log.Errorf("Error while killing cmd process: %s", err)
			}
			return
		}
	}
}

func (cmd *runCmd) start(appname string) {
	cmd.log.Infof("Restarting '%s'...", appname)
	if !strings.Contains(appname, "./") {
		appname = "./" + appname
	}
	ecmd = exec.Command(appname)
	ecmd.Env = []string{"APP_ENV=" + cmd.Env}
	ecmd.Stdout = os.Stdout
	ecmd.Stderr = os.Stderr

	if err := ecmd.Run(); err != nil {
		cmd.log.Errorf("running %s failed...", appname)
	}
}

func (cmd *runCmd) shouldIgnoreFile(filename string) bool {
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

func (cmd *runCmd) shouldWatchFileWithExtension(name string) bool {
	for _, s := range watchExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

func addBuildFileToIgnoreIfNotIn(rootPath string) {
	fs, err := os.OpenFile(fmt.Sprintf("%s/.gitignore", rootPath), os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	content, err := ioutil.ReadAll(fs)
	if err != nil {
		panic(err)
	}

	contentArr := strings.Split(string(content), "\n")
	for _, c := range contentArr {
		if c == "app" {
			return
		}
	}
	_, err = fs.WriteString("\napp")
	if err != nil {
		panic(err)
	}
}
