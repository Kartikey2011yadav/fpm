package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/journal"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show operation history",
	Long: `Show the history of install, remove, upgrade, and snapshot operations.

Each operation is recorded with a timestamp and affected packages.
Use --oneline for compact output.`,
	GroupID: "advanced",
	RunE:    runLog,
}

func init() {
	logCmd.Flags().BoolP("oneline", "1", false, "Compact one-line-per-entry format")
	logCmd.Flags().IntP("limit", "n", 20, "Number of entries to show")
	rootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	oneline, _ := cmd.Flags().GetBool("oneline")
	limit, _ := cmd.Flags().GetInt("limit")

	envPath := ""
	if v, _ := venv.Detect(cwd); v != nil {
		envPath = v.Path
	}

	entries, err := journal.Read(envPath, limit)
	if err != nil {
		fmt.Println("No operation history yet.")
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("No operation history yet.")
		return nil
	}

	if oneline {
		for _, e := range entries {
			pkgSummary := ""
			if len(e.Packages) > 0 {
				pkgSummary = e.Packages[0]
				if len(e.Packages) > 1 {
					pkgSummary += fmt.Sprintf(" (+%d more)", len(e.Packages)-1)
				}
			} else if e.Message != "" {
				pkgSummary = e.Message
			}
			fmt.Printf("\033[33m%s\033[0m  %-10s %s\n", e.ID, e.Operation, pkgSummary)
		}
		return nil
	}

	for i, e := range entries {
		ts := e.Timestamp.Local().Format("2006-01-02 15:04")
		fmt.Printf("\033[33m%s\033[0m %s  \033[1m%s\033[0m", e.ID, ts, e.Operation)
		if e.Message != "" {
			fmt.Printf("  \033[2m(%s)\033[0m", e.Message)
		}
		fmt.Println()
		if len(e.Packages) > 0 {
			fmt.Printf("    %s\n", strings.Join(e.Packages, ", "))
		}
		if i < len(entries)-1 {
			fmt.Println()
		}
	}

	return nil
}
