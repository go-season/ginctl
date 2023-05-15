package util

import (
	"errors"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func ExtractSpecifiedPathFromCommon(path, search string) string {
	commonPkgPath := GetCommonPkgPath()
	cmd := exec.Command("bash", "-c", fmt.Sprintf("grep 'type %s struct' -nr * | awk -F':' '{print $1}'", search))
	cmd.Dir = commonPkgPath + "/" + path

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("execute go list command, %s, stdout:%s, stderr:%s", err, stdout.String(), stderr.String()))
	}

	outStr, _ := stdout.String(), stderr.String()

	if outStr[0] == '_' { // will shown like _/{GOPATH}/src/{YOUR_PACKAGE} when NOT enable GO MODULE.
		outStr = strings.TrimPrefix(outStr, "_"+build.Default.GOPATH+"/src/")
	}
	f := strings.Split(outStr, "\n")
	outStr = f[0]

	return fmt.Sprintf("%s/%s/%s", commonPkgPath, path, outStr)
}

func GetCommonPkgPath() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("bash", "-c", "go list -m all | grep 'xz-go/common' | awk '{printf \"%s@%s\", $1, $2}'")
	cmd.Dir = wd
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("execute go list command, %s, stdout:%s, stderr:%s", err, stdout.String(), stderr.String()))
	}

	outStr, _ := stdout.String(), stderr.String()

	if outStr[0] == '_' { // will shown like _/{GOPATH}/src/{YOUR_PACKAGE} when NOT enable GO MODULE.
		outStr = strings.TrimPrefix(outStr, "_"+build.Default.GOPATH+"/src/")
	}
	f := strings.Split(outStr, "\n")
	outStr = f[0]

	goPath := build.Default.GOPATH

	return fmt.Sprintf("%s/pkg/mod/%s", goPath, outStr)
}

func GetPkgName(dir string) (string, error) {
	cmd := exec.Command("go", "list", "-f={{.ImportPath}}")
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("execute go list command, %s, stdout:%s, stderr:%s", err, stdout.String(), stderr.String())
	}

	outStr, _ := stdout.String(), stderr.String()

	if outStr[0] == '_' { // will shown like _/{GOPATH}/src/{YOUR_PACKAGE} when NOT enable GO MODULE.
		outStr = strings.TrimPrefix(outStr, "_"+build.Default.GOPATH+"/src/")
	}
	f := strings.Split(outStr, "\n")
	outStr = f[0]

	return outStr, nil
}

func GetPkgBaseName(dir string) string {
	pkgName, _ := GetPkgName(dir)

	return GetImportBaseName(pkgName)
}

func GetImportBaseName(importName string) string {
	pos := strings.LastIndex(importName, "/")
	return importName[pos+1:]
}

func GetModuleName(dir string) string {
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return ""
	}

	outStr, _ := stdout.String(), stderr.String()

	if outStr[0] == '_' { // will shown like _/{GOPATH}/src/{YOUR_PACKAGE} when NOT enable GO MODULE.
		outStr = strings.TrimPrefix(outStr, "_"+build.Default.GOPATH+"/src/")
	}
	f := strings.Split(outStr, "\n")
	outStr = f[0]

	return outStr
}

func GetModeBaseName(name string) string {
	modname := GetModuleName(name)
	pos := strings.LastIndex(modname, "/")
	return modname[pos+1:]
}

func GetProjectPath() (string, error) {
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

	return match[1], nil
}

func GetProjectCurrentBranch() (string, error) {
	cmdStr := "git rev-parse --abbrev-ref HEAD"

	output, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func ReloadModule() error {
	cmd := "go mod tidy"
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	return nil
}
