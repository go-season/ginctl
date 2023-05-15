package sdk

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-season/ginctl/pkg/util/log"

	"github.com/go-season/ginctl/pkg/util/file"

	"gopkg.in/src-d/go-git.v4/config"

	"github.com/go-season/ginctl/pkg/util"

	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/src-d/go-git.v4"
)

const (
	sdkRepoURL        = "https://gitlab.idc.xiaozhu.com/xz/lib/sdk.git"
	sdkRepoAccessUser = "scaffold"
	sdkRepoAccessPass = "scaffold@laysheng"
)

func CloneRepoToLocal(path string) error {
	if path == "" {
		return errors.New("local path is not exists")
	}
	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL: sdkRepoURL,
		Auth: &http.BasicAuth{
			Username: sdkRepoAccessUser,
			Password: sdkRepoAccessPass,
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func PublishRepo(path string, canSwitch bool, oldVersion bool, debug bool, logger log.Logger) error {
	cwd, _ := os.Getwd()

	localPath := GetLocalPath()
	if localPath == "" {
		return errors.New("get local path failed")
	}

	modname := util.GetModuleName(cwd)
	pos := strings.LastIndex(modname, "/")
	prjName := modname[pos+1:]
	sdkPath := createPathIfNotExists(localPath)
	if sdkPath == "" {
		return errors.New("create local map path failed")
	}

	curBranch, err := util.GetProjectCurrentBranch()
	if err != nil {
		return err
	}
	if strings.TrimSpace(curBranch) == "trunk" {
		curBranch = "candidate"
	}

	curBranch = strings.Trim(curBranch, "\n")

	// switch branch
	if canSwitch {
		cmdStr := fmt.Sprintf("git branch -r | grep %s | wc -l", curBranch)
		if debug {
			logger.Info("执行：", cmdStr)
		}
		ecmd := exec.Command("bash", "-c", cmdStr)
		ecmd.Dir = localPath
		output, err := ecmd.Output()
		if err != nil {
			return err
		}
		cnt, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			return err
		}
		cmdStr = fmt.Sprintf("git checkout -b %s", curBranch)
		if cnt > 0 {
			cmdStr = fmt.Sprintf("git checkout %s && git pull --no-edit origin %s", curBranch, curBranch)
		}

		if debug {
			logger.Info("执行：", cmdStr)
		}

		ecmd = exec.Command("bash", "-c", cmdStr)
		ecmd.Dir = localPath
		_, err = ecmd.Output()
		if err != nil {
			return err
		}
	}

	filePath := sdkPath
	if oldVersion {
		filePath = fmt.Sprintf("%s/%s", localPath, prjName)
	}
	os.RemoveAll(filePath)

	cmdStr := fmt.Sprintf("cp -r %s %s", path, sdkPath)
	if oldVersion {
		cmdStr = fmt.Sprintf("cp -r %s %s/%s", path, localPath, prjName)
	}

	if debug {
		logger.Info("执行：", cmdStr)
	}

	_, err = exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return err
	}

	cmdStr = fmt.Sprintf("cd %s && git add . && git commit -m 'add %s sdk'", localPath, modname)

	if debug {
		logger.Info("执行：", cmdStr)
	}

	_, _ = exec.Command("bash", "-c", cmdStr).Output()

	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return err
	}

	_, _ = repo.CreateRemote(&config.RemoteConfig{
		Name: curBranch,
		URLs: []string{sdkRepoURL},
	})

	err = repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: sdkRepoAccessUser,
			Password: sdkRepoAccessPass,
		},
		RemoteName: curBranch,
		Progress:   os.Stdout,
	})

	if debug {
		logger.Info("执行发布...")
	}

	if err != nil && err.Error() != "already up-to-date" {
		return err
	}

	return nil
}

func createPathIfNotExists(path string) string {
	prjPath, err := util.GetProjectPath()
	if err != nil {
		return ""
	}

	prjPath = strings.TrimPrefix(prjPath, "/")
	found, err := file.PathExists(fmt.Sprintf("%s/%s", path, prjPath))
	if err != nil {
		return ""
	}
	if !found {
		err = os.MkdirAll(fmt.Sprintf("%s/%s", path, prjPath), 0755)
		if err != nil {
			return ""
		}
	}

	return fmt.Sprintf("%s/%s", path, strings.Replace(prjPath, "-", "", -1))
}

func GetLocalPath() string {
	home, err := homedir.Dir()
	if err != nil {
		return ""
	}
	localPath := fmt.Sprintf("%s/.ginctl/sdk", home)

	return localPath
}

func GetLocallyPath() string {
	home, err := homedir.Dir()
	if err != nil {
		return ""
	}
	locallyPath := fmt.Sprintf("%s/.ginctl/sdklocally", home)

	return locallyPath
}
