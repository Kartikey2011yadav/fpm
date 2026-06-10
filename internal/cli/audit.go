package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/kartikeyyadav/fpm/internal/audit"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:     "audit",
	Short:   "Scan dependencies for known vulnerabilities",
	Long:    "Query the OSV (Open Source Vulnerabilities) database for known security issues in installed packages.",
	GroupID: "package",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		activeVenv, _ := venv.Detect(cwd)

		var dirs []string
		if activeVenv != nil && !flagSystem {
			dirs = []string{activeVenv.SitePackages}
		} else if flagSystem {
			sysDirs := findSystemSitePackages()
			if len(sysDirs) == 0 {
				return fmt.Errorf("no Python environment found")
			}
			dirs = sysDirs
		} else {
			return fmt.Errorf("no virtual environment found. Use --system to audit system packages, or run from a project directory")
		}

		scanner := env.NewScanner(dirs)
		result, err := scanner.Scan()
		if err != nil {
			return err
		}

		if len(result.Packages) == 0 {
			fmt.Println("No packages installed to audit.")
			return nil
		}

		fmt.Printf("Auditing %d packages for vulnerabilities...\n\n", len(result.Packages))

		ctx := context.Background()
		report, err := audit.Scan(ctx, result.Packages)
		if err != nil {
			return fmt.Errorf("audit failed: %w", err)
		}

		report.Print()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
