package cmd

import (
	"os"

	"github.com/go-season/ginctl/cmd/apitest"
	"github.com/go-season/ginctl/cmd/faker"

	"github.com/go-season/ginctl/cmd/add"
	"github.com/go-season/ginctl/cmd/cc"
	"github.com/go-season/ginctl/cmd/pkg"
	"github.com/go-season/ginctl/cmd/polyfill"
	"github.com/go-season/ginctl/cmd/route"
	"github.com/go-season/ginctl/cmd/sdk"
	pkg2 "github.com/go-season/ginctl/pkg/ginctl/pkg"
	"github.com/go-season/ginctl/pkg/util/factory"
	"github.com/go-season/ginctl/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var version = "1.0.76"

func NewRootCmd(log log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:           "ginctl",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Happying development to use gin framework",
		PersistentPreRun: func(cobraCmd *cobra.Command, args []string) {
			if cobraCmd.Parent().Name() == "pkg" && cobraCmd.Name() == "update" {
				return
			}
			if cobraCmd.Name() == "new" {
				return
			}

			pkg2.NewVersionChecker(log).Check()
		},
		Long: `Ginctl help you to build gin framework skeleton easily. Get started by running the new command in anywhere:
	
	ginctl new [project-name]`,
	}
}

func Execute() {
	f := factory.DefaultFactory()
	rootCmd := BuildRoot(f)
	if err := rootCmd.Execute(); err != nil {
		f.GetLog().Error(err.Error())
		os.Exit(1)
	}
}

func BuildRoot(f factory.Factory) *cobra.Command {
	rootCmd := NewRootCmd(f.GetLog())

	rootCmd.Version = version

	rootCmd.AddCommand(NewNewCmd(f))
	rootCmd.AddCommand(NewRunCmd(f))
	rootCmd.AddCommand(NewDocCmd(f))
	rootCmd.AddCommand(NewTagCmd(f))
	rootCmd.AddCommand(NewCleanCmd(f))
	rootCmd.AddCommand(NewCompletionCmd())
	rootCmd.AddCommand(NewSelfUpdateCmd(f))
	rootCmd.AddCommand(NewGitHookCmd(f))
	rootCmd.AddCommand(NewCronCmd(f))
	rootCmd.AddCommand(pkg.NewPkgCmd(f))
	rootCmd.AddCommand(sdk.NewSDKCmd(f))
	rootCmd.AddCommand(add.NewAddCmd(f))
	rootCmd.AddCommand(route.NewRouteCmd(f))
	rootCmd.AddCommand(polyfill.NewPolyfillCmd(f))
	rootCmd.AddCommand(cc.NewCCCmd(f))
	rootCmd.AddCommand(faker.NewFakerCmd(f))
	rootCmd.AddCommand(apitest.NewAPITestCmd(f))

	cobra.OnInitialize(func() {
		initConfig(f.GetLog())
	})

	return rootCmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig(log log.Logger) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Panic(err)
		}

		// Search config in home directory with name ".ginctl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".ginctl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
