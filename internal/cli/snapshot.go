package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/kartikeyyadav/fpm/internal/snapshot"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage environment snapshots (version your environment like git)",
	Long: `Take snapshots of your entire environment state — all packages from all
managers (fpm, pip, conda, uv, poetry, pdm, system). Restore any previous
state instantly.

Snapshots are scoped per-environment: each venv/project has its own
independent snapshot history. This means:
  - Project A's snapshots are separate from Project B's
  - Each venv tracks its own timeline
  - Global environment has its own snapshot history (use --system)

Think of it as git for your Python environment.`,
	GroupID: "environment",
}

var snapshotCreateCmd = &cobra.Command{
	Use:     "create [message]",
	Short:   "Capture the current environment state",
	Aliases: []string{"save"},
	RunE: func(cmd *cobra.Command, args []string) error {
		message := ""
		if len(args) > 0 {
			message = strings.Join(args, " ")
		}

		envInfo, err := resolveSnapshotEnv()
		if err != nil {
			return err
		}

		scanner := env.NewScanner(envInfo.dirs)
		scanResult, _ := scanner.Scan()

		store := snapshot.NewStore(envInfo.storePath)
		snap, err := store.Capture(scanResult, envInfo.pythonVersion, envInfo.pythonPath, envInfo.dirs, message, envInfo.projectDir)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}

		fmt.Printf("Snapshot created: %s\n", snap.ID)
		fmt.Printf("  Packages: %d total (%d fpm, %d pip, %d other)\n",
			len(snap.Packages),
			countByManager(snap.Packages, "fpm"),
			countByManager(snap.Packages, "pip"),
			len(snap.Packages)-countByManager(snap.Packages, "fpm")-countByManager(snap.Packages, "pip"),
		)
		if message != "" {
			fmt.Printf("  Message: %s\n", message)
		}

		return nil
	},
}

var snapshotListCmd = &cobra.Command{
	Use:     "list",
	Short:   "Show all environment snapshots",
	Aliases: []string{"log"},
	RunE: func(cmd *cobra.Command, args []string) error {
		envInfo, err := resolveSnapshotEnv()
		if err != nil {
			return err
		}

		store := snapshot.NewStore(envInfo.storePath)
		snapshots, err := store.List()
		if err != nil || len(snapshots) == 0 {
			fmt.Println("No snapshots found. Run 'fpm snapshot create' to take one.")
			return nil
		}

		current, _ := store.Current()
		currentID := ""
		if current != nil {
			currentID = current.ID
		}

		fmt.Println("Environment snapshots:")
		fmt.Println()
		for _, snap := range snapshots {
			marker := "  "
			if snap.ID == currentID {
				marker = "* "
			}
			msg := snap.Message
			if msg == "" {
				msg = "(no message)"
			}
			fmt.Printf("%s%s  %s  [%d packages]  %s\n",
				marker,
				snap.ID,
				snap.CreatedAt.Format("2006-01-02 15:04"),
				len(snap.Packages),
				msg,
			)
		}

		return nil
	},
}

var snapshotRestoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id>",
	Short: "Restore environment to a previous snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snapID := args[0]

		envInfo, err := resolveSnapshotEnv()
		if err != nil {
			return err
		}

		// Permission check for system restores
		if flagSystem && envInfo.sitePackages != "" {
			if f, err := os.OpenFile(envInfo.sitePackages, os.O_RDONLY, 0); err != nil {
				return fmt.Errorf("no write access to %s\n\n  hint: System snapshot restore affects all users. Use sudo or ensure write permissions.", envInfo.sitePackages)
			} else {
				f.Close()
			}
		}

		store := snapshot.NewStore(envInfo.storePath)
		snap, err := store.Get(snapID)
		if err != nil {
			return fmt.Errorf("snapshot %q not found", snapID)
		}

		scanner := env.NewScanner(envInfo.dirs)
		currentScan, _ := scanner.Scan()

		// Detect system conflicts (only for project-level restores)
		systemStrategy := snapshot.StrategyAbort
		systemSitePackages := ""
		if !flagSystem {
			sysDirs := findSystemSitePackages()
			if len(sysDirs) > 0 {
				systemSitePackages = sysDirs[0]
				sysScanner := env.NewScanner(sysDirs)
				sysScan, _ := sysScanner.Scan()
				conflicts := snapshot.DetectSystemConflicts(snap, sysScan)
				if len(conflicts) > 0 {
					fmt.Printf("\n  \033[33m⚠\033[0m System package conflicts detected:\n")
					for _, c := range conflicts {
						fmt.Printf("    %s: snapshot needs %s, system has %s (%s)\n",
							c.Package, c.SnapshotVersion, c.CurrentVersion, c.Manager)
					}
					fmt.Printf("\n  How to resolve?\n")
					fmt.Printf("    [1] Roll back system packages too\n")
					fmt.Printf("    [2] Install at project level (overrides system)\n")
					fmt.Printf("    [3] Abort (fix system packages manually)\n")
					fmt.Printf("  Choice [1/2/3]: ")

					var choice string
					fmt.Scanln(&choice)
					switch choice {
					case "1":
						systemStrategy = snapshot.StrategyRollbackSystem
					case "2":
						systemStrategy = snapshot.StrategyOverrideLocal
					default:
						fmt.Println("  Aborted.")
						return nil
					}
				}
			}
		}

		fmt.Printf("Restoring snapshot %s (%s)...\n", snap.ID, snap.CreatedAt.Format("2006-01-02 15:04"))

		cfg, _ := config.LoadFromCwd()
		c := cache.New(cfg.Cache.Dir)
		refTracker := cache.NewRefTracker(c)
		pypiClient := client.New(client.ClientOptions{
			Indexes:  cfg.Indexes,
			CacheDir: filepath.Join(cfg.Cache.Dir, "http"),
			Network:  cfg.Network,
		})

		result, err := snapshot.Restore(snap, currentScan, snapshot.RestoreOptions{
			Cache:              c,
			RefTracker:         refTracker,
			PyPIClient:         pypiClient,
			SitePackages:       envInfo.sitePackages,
			SystemSitePackages: systemSitePackages,
			EnvPath:            envInfo.storePath,
			AutoDownload:       true,
			RestoreExternal:    true,
			ProjectDir:         envInfo.projectDir,
			SystemStrategy:     systemStrategy,
		})
		if err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		result.PrintLog()
		fmt.Printf("\nEnvironment restored to snapshot %s.\n", snap.ID)
		return nil
	},
}

var snapshotDiffCmd = &cobra.Command{
	Use:   "diff <snapshot-1> [snapshot-2]",
	Short: "Compare two snapshots (or snapshot vs current)",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		envInfo, err := resolveSnapshotEnv()
		if err != nil {
			return err
		}

		store := snapshot.NewStore(envInfo.storePath)

		var diff *snapshot.SnapshotDiff
		if len(args) == 2 {
			diff, err = store.Diff(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Printf("Diff: %s → %s\n\n", args[0], args[1])
		} else {
			snap, err := store.Get(args[0])
			if err != nil {
				return fmt.Errorf("snapshot %q not found", args[0])
			}

			scanner := env.NewScanner(envInfo.dirs)
			currentScan, _ := scanner.Scan()

			currentSnap := &snapshot.Snapshot{}
			for _, pkg := range currentScan.Packages {
				currentSnap.Packages = append(currentSnap.Packages, snapshot.SnapshotPackage{
					Name:    pkg.Name.Normalized(),
					Version: pkg.Version.String(),
					Manager: pkg.Manager.String(),
				})
			}
			diff = snapshot.DiffSnapshots(snap, currentSnap)
			fmt.Printf("Diff: %s → current\n\n", args[0])
		}

		if len(diff.Added) == 0 && len(diff.Removed) == 0 && len(diff.Changed) == 0 {
			fmt.Println("No differences.")
			return nil
		}

		for _, p := range diff.Added {
			fmt.Printf("  + %s %s (%s)\n", p.Name, p.Version, p.Manager)
		}
		for _, p := range diff.Removed {
			fmt.Printf("  - %s %s (%s)\n", p.Name, p.Version, p.Manager)
		}
		for _, c := range diff.Changed {
			fmt.Printf("  ~ %s %s → %s (%s)\n", c.Name, c.OldVersion, c.NewVersion, c.Manager)
		}

		return nil
	},
}

var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <snapshot-id>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		envInfo, err := resolveSnapshotEnv()
		if err != nil {
			return err
		}

		store := snapshot.NewStore(envInfo.storePath)
		if err := store.Delete(args[0]); err != nil {
			return fmt.Errorf("failed to delete snapshot: %w", err)
		}

		fmt.Printf("Deleted snapshot %s.\n", args[0])
		return nil
	},
}

func init() {
	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotRestoreCmd)
	snapshotCmd.AddCommand(snapshotDiffCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	rootCmd.AddCommand(snapshotCmd)
}

type snapshotEnvInfo struct {
	storePath     string
	sitePackages  string
	dirs          []string
	pythonVersion string
	pythonPath    string
	projectDir    string
}

func resolveSnapshotEnv() (*snapshotEnvInfo, error) {
	cwd, _ := os.Getwd()

	if flagSystem {
		finder := python.NewFinder()
		interp, err := finder.FindBest("")
		if err != nil {
			return nil, fmt.Errorf("no Python found for system snapshots")
		}
		sysDirs := env.FindSitePackagesDirs(interp.SysPaths)
		sitePackages := interp.SitePackages
		if sitePackages == "" {
			for _, p := range interp.SysPaths {
				if strings.Contains(p, "site-packages") || strings.Contains(p, "dist-packages") {
					sitePackages = p
					break
				}
			}
		}
		return &snapshotEnvInfo{
			storePath:     "global",
			sitePackages:  sitePackages,
			dirs:          sysDirs,
			pythonVersion: interp.VersionString(),
			pythonPath:    interp.Path,
			projectDir:    "",
		}, nil
	}

	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return nil, fmt.Errorf("no virtual environment found. Use --system for system-level snapshots")
	}

	allDirs := []string{}
	if activeVenv.SitePackages != "" {
		allDirs = append(allDirs, activeVenv.SitePackages)
	}
	if activeVenv.Interpreter != nil {
		allDirs = append(allDirs, env.FindSitePackagesDirs(activeVenv.Interpreter.SysPaths)...)
	}

	pythonVersion := ""
	pythonPath := ""
	if activeVenv.Interpreter != nil {
		pythonVersion = activeVenv.Interpreter.VersionString()
		pythonPath = activeVenv.Interpreter.Path
	}

	return &snapshotEnvInfo{
		storePath:     activeVenv.Path,
		sitePackages:  activeVenv.SitePackages,
		dirs:          allDirs,
		pythonVersion: pythonVersion,
		pythonPath:    pythonPath,
		projectDir:    cwd,
	}, nil
}

func countByManager(packages []snapshot.SnapshotPackage, manager string) int {
	count := 0
	for _, p := range packages {
		if p.Manager == manager {
			count++
		}
	}
	return count
}

