package cli

import "github.com/spf13/cobra"

func init() {
	listCmd.RunE = runList
}

func runList(cmd *cobra.Command, args []string) error {
	showAll, _ := cmd.Flags().GetBool("all")
	managerFilter, _ := cmd.Flags().GetString("manager")

	// Default: show only fpm-managed packages unless --all or --manager is set
	if !showAll && managerFilter == "" {
		cmd.Flags().Set("manager", "fpm")
	}

	return runPipList(cmd, args)
}
