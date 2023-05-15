package pkg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/go-season/ginctl/pkg/util/log"
)

const (
	UpdateYes = "yes"
	UpdateNo  = "no"
)

var reVersion = regexp.MustCompile(`\d+\.\d+\.\d+`)

type VersionChecker struct {
	log log.Logger
}

func NewVersionChecker(log log.Logger) *VersionChecker {
	return &VersionChecker{
		log: log,
	}
}

func (v *VersionChecker) Check() {
	localPkgs, err := FindRequiredModuleWithVersion()
	if err != nil || localPkgs == nil {
		return
	}

	remotePkgs, err := ListWithTag()
	if err != nil {
		return
	}
	for _, remotePkg := range remotePkgs {
		idx := strings.LastIndex(remotePkg.Name, "/")
		if ver, ok := localPkgs[remotePkg.Name]; ok {
			if strings.Contains(remotePkg.BugTag, "<") {
				rever := strings.TrimPrefix(remotePkg.BugTag, "<")
				rever, _ = eraseVersionPrefix(rever)
				locver, _ := eraseVersionPrefix(ver)
				resemver, _ := semver.Parse(rever)
				locsemver, _ := semver.Parse(locver)
				if locsemver.Compare(resemver) == -1 {
					v.log.Warnf("包%s当前版本%s有兼容性问题或者bug, 请使用`ginctl pkg update %s`升级到最新版本:%s", remotePkg.Name, ver, remotePkg.Name[idx+1:], remotePkg.LatestTag)
					sendQuestionForUpdate(v.log, remotePkg.Name[idx+1:])
				}
			} else if strings.Contains(remotePkg.BugTag, ">") {
				rever := strings.TrimPrefix(remotePkg.BugTag, ">")
				rever, _ = eraseVersionPrefix(rever)
				locver, _ := eraseVersionPrefix(ver)
				resemver, _ := semver.Parse(rever)
				locsemver, _ := semver.Parse(locver)
				if locsemver.Compare(resemver) == 1 && ver != remotePkg.LatestTag {
					v.log.Warnf("包%s当前版本%s有兼容性问题或者bug, 请使用`ginctl pkg update %s`升级到最新版本:%s", remotePkg.Name, ver, remotePkg.Name[idx+1:], remotePkg.LatestTag)
					sendQuestionForUpdate(v.log, remotePkg.Name[idx+1:])
				}
			} else if remotePkg.BugTag != "" && ver == remotePkg.BugTag {
				v.log.Warnf("包%s当前版本%s有兼容性问题或者bug, 请使用`ginctl pkg update %s`升级到最新版本:%s", remotePkg.Name, ver, remotePkg.Name[idx+1:], remotePkg.LatestTag)
				sendQuestionForUpdate(v.log, remotePkg.Name[idx+1:])
			}
		}
	}
}

func sendQuestionForUpdate(logger log.Logger, pname string) {
	choice, err := logger.Question(&log.QuestionOptions{
		Question:     "你想要现在去更新依赖吗？",
		Options:      []string{UpdateYes, UpdateNo},
		DefaultValue: UpdateYes,
	})
	if err != nil {
		return
	}
	if choice == UpdateNo {
		return
	}

	wd, _ := os.Getwd()
	cmdStr := fmt.Sprintf("ginctl pkg update %s", pname)
	ecmd := exec.Command("bash", "-c", cmdStr)
	ecmd.Dir = wd
	output, err := ecmd.Output()
	if err != nil {
		panic(err)
	}
	logger.WriteString(string(output))
}

func FindRequiredModuleWithVersion() (map[string]string, error) {
	cmdStr := "cat go.mod | grep 'github.com/go-season' | grep -v 'module' | grep -v 'replace' | awk '{printf \"%s|%s\\n\", $1,$2}'"
	output, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return nil, err
	}

	if strings.TrimRight(string(output), "\n") == "" {
		return nil, nil
	}

	ret := make(map[string]string)
	items := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
	for _, item := range items {
		parts := strings.Split(item, "|")
		ret[strings.TrimPrefix(parts[0], "github.com/go-season/")] = parts[1]
	}

	return ret, nil
}

func eraseVersionPrefix(version string) (string, error) {
	indices := reVersion.FindStringIndex(version)
	if indices == nil {
		return version, errors.New("version not adopting semver")
	}
	if indices[0] > 0 {
		version = version[indices[0]:]
	}

	return version, nil
}
