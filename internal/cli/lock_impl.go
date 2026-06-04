package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/lock"
	"github.com/kartikeyyadav/fpm/internal/pep508"
	"github.com/kartikeyyadav/fpm/internal/resolver"
	"github.com/kartikeyyadav/fpm/internal/workspace"
	"github.com/spf13/cobra"
)

func init() {
	lockCmd.RunE = runLock
}

func runLock(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	cfg, _ := config.LoadFromCwd()

	// Read pyproject.toml for dependencies
	pyproject, err := workspace.ReadPyProjectToml(cwd)
	if err != nil {
		return fmt.Errorf("no pyproject.toml found. Run 'fpm init' first")
	}

	// Parse dependencies
	var requirements []pep508.Requirement
	for _, dep := range pyproject.Project.Dependencies {
		req, err := pep508.ParseRequirement(dep)
		if err != nil {
			fmt.Printf("  Warning: invalid dependency %q: %v\n", dep, err)
			continue
		}
		requirements = append(requirements, req)
	}

	if len(requirements) == 0 {
		fmt.Println("No dependencies to resolve.")
		return nil
	}

	// Resolve
	pypiClient := client.New(client.ClientOptions{
		Indexes:     cfg.Indexes,
		CacheDir:    filepath.Join(cfg.Cache.Dir, "http"),
		Concurrency: cfg.Tool.Concurrency,
	})

	fmt.Printf("Resolving %d dependencies...\n", len(requirements))

	res, err := resolver.New(resolver.ResolverOptions{
		Client:     pypiClient,
		Immutables: cfg.Immutable.Packages,
	}).Resolve(requirements)
	if err != nil {
		return fmt.Errorf("resolution failed: %w", err)
	}

	// Write lockfile
	lf := lock.Generate(res, pyproject.Project.RequiresPython)
	lockPath := filepath.Join(cwd, lock.LockFileName)

	// Check for changes
	oldLf, oldErr := lock.Read(lockPath)
	if err := lf.Write(lockPath); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	if oldErr == nil {
		diff := lock.Diff(oldLf, lf)
		if diff.IsEmpty() {
			fmt.Println("Lockfile is up to date.")
		} else {
			fmt.Printf("Updated lockfile:\n%s", diff.String())
		}
	} else {
		fmt.Printf("Wrote lockfile with %d packages.\n", len(lf.Packages))
	}

	return nil
}
