package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "生成自动补全脚本",
		Long: `To load completions:

Bash:

$ source <(ginctl completion bash)

# To load completions for each session, execute once:
Linux:
  $ ginctl completion bash > /etc/bash_completion.d/yourprogram
MacOS:
  $ ginctl completion bash > /usr/local/etc/bash_completion.d/yourprogram

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ ginctl completion zsh > "${fpath[1]}/_ginctl"

# You will need to start a new shell for this setup to take effect.

Fish:

$ ginctl completion fish | source

# To load completions for each session, execute once:
$ ginctl completion fish > ~/.config/fish/completions/ginctl.fish
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			switch args[0] {
			case "bash":
				err = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				err = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				err = cmd.Root().GenFishCompletion(os.Stdout, true)
			}
			return err
		},
	}
}
