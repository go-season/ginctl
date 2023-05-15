package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-season/ginctl/pkg/util/factory"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

const GinCtlGitLabPeer = "github.com/go-season/ginctl"

type SelfUpdateCmd struct {
}

func NewSelfUpdateCmd(f factory.Factory) *cobra.Command {
	cmd := SelfUpdateCmd{}

	selfUpdateCmd := &cobra.Command{
		Use:   "selfupdate",
		Short: "更新ginctl脚手架",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	return selfUpdateCmd
}

func (cmd *SelfUpdateCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	cmdstr := "go get " + GinCtlGitLabPeer
	ecmd := exec.Command("bash", "-c", cmdstr)
	f.GetLog().StartWait("start downloading...")
	home, _ := homedir.Dir()
	ecmd.Dir = home
	if err := ecmd.Run(); err != nil {
		panic(err)
	}
	ecmd = exec.Command("bash", "-c", "ginctl -v")
	var stdout, stderr strings.Builder
	ecmd.Stdout = &stdout
	ecmd.Stderr = &stderr
	if err := ecmd.Run(); err != nil {
		panic(fmt.Errorf("execute go list command, %s, stdout:%s, stderr:%s", err, stdout.String(), stderr.String()))
	}
	outStr, _ := stdout.String(), stderr.String()
	version := strings.TrimPrefix(outStr, "ginctl version")
	f.GetLog().Donef("update ginctl successful, current version is:%s", version)

	return nil
}
