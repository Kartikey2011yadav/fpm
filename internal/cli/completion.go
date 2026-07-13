package cli

import (
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for fpm.

To load completions:

Bash:
  $ source <(fpm completion bash)
  # To load for each session, execute once:
  # Linux:
  $ fpm completion bash > /etc/bash_completion.d/fpm
  # macOS:
  $ fpm completion bash > $(brew --prefix)/etc/bash_completion.d/fpm

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Execute once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  $ fpm completion zsh > "${fpath[1]}/_fpm"
  # You will need to start a new shell for this to take effect.

Fish:
  $ fpm completion fish | source
  $ fpm completion fish > ~/.config/fish/completions/fpm.fish

PowerShell:
  PS> fpm completion powershell | Out-String | Invoke-Expression
  # To load for every new session, add to your profile:
  PS> fpm completion powershell > fpm.ps1 && . fpm.ps1`,
	GroupID:               "advanced",
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
