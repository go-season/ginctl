package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
)

const BackendURI = "http://service-backend.xiaozhu.com"

func Publish(module string) error {
	v := url.Values{}
	v.Set("module", module)

	resp, err := http.PostForm(fmt.Sprintf("%s/pkg/add", BackendURI), v)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}

		return errors.New(fmt.Sprintf("发布失败，原因: %s", result["errorMsg"]))
	}

	return nil
}

type Pkg struct {
	Name      string `json:"name"`
	Module    string `json:"module"`
	BugTag    string `json:"bugTag"`
	LatestTag string `json:"latestTag"`
}

type Pkgs struct {
	List []Pkg `json:"list"`
}

type Response struct {
	Status   int    `json:"status"`
	Content  Pkgs   `json:"content"`
	ErrorMsg string `json:"errorMsg"`
}

func List() ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/pkg/list", BackendURI))
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	modules := make([]string, 0, len(response.Content.List))
	for _, pkg := range response.Content.List {
		modules = append(modules, pkg.Module)
	}

	return modules, nil
}

func ListWithTag() ([]Pkg, error) {
	resp, err := http.Get(fmt.Sprintf("%s/pkg/list", BackendURI))
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	pkgs := make([]Pkg, 0, len(response.Content.List))
	for _, pkg := range response.Content.List {
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

func FindRequiredModule() ([]string, error) {
	cmdStr := fmt.Sprintf("cat go.mod | grep 'github.com/go-season' | grep -v 'module' | grep -v 'replace' | awk '{print $1}'")
	output, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return nil, err
	}

	items := strings.Split(strings.TrimRight(string(output), "\n"), "\n")

	return items, nil
}

func FindDiffModule(modules map[string]bool) ([]string, error) {
	cmdStr := fmt.Sprintf("cat go.mod | grep 'github.com/go-season' | grep -v 'module' | awk '{print $1}'")
	output, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		return nil, err
	}

	items := strings.Split(string(output), "\n")
	for _, item := range items {
		if _, ok := modules[item]; ok {
			delete(modules, item)
		}
	}

	diff := make([]string, 0)
	for module := range modules {
		diff = append(diff, module)
	}

	return diff, nil
}
