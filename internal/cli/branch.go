package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

type branchMeta struct {
	Active   string            `json:"active"`
	Branches map[string]string `json:"branches"` // name → snapshot-id or state path
}

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Manage environment branches for parallel experimentation",
	Long: `Create named environment branches to experiment without affecting your main state.

Like git branches, each branch has its own set of packages.
Switch between them instantly via snapshot restore.`,
	GroupID: "environment",
}

var branchCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new branch from current state",
	Args:  cobra.ExactArgs(1),
	RunE:  runBranchCreate,
}

var branchSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch to a different branch",
	Args:  cobra.ExactArgs(1),
	RunE:  runBranchSwitch,
}

var branchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all branches",
	RunE:  runBranchList,
}

var branchDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a branch",
	Args:  cobra.ExactArgs(1),
	RunE:  runBranchDelete,
}

func init() {
	branchCmd.AddCommand(branchCreateCmd)
	branchCmd.AddCommand(branchSwitchCmd)
	branchCmd.AddCommand(branchListCmd)
	branchCmd.AddCommand(branchDeleteCmd)
	rootCmd.AddCommand(branchCmd)
}

func branchMetaPath(envPath string) string {
	return filepath.Join(envPath, ".fpm-branches.json")
}

func loadBranchMeta(envPath string) *branchMeta {
	path := branchMetaPath(envPath)
	lock, _ := fs.LockFileShared(path)
	defer fs.UnlockFile(lock)

	data, err := os.ReadFile(path)
	if err != nil {
		return &branchMeta{Active: "main", Branches: map[string]string{}}
	}
	var meta branchMeta
	json.Unmarshal(data, &meta)
	if meta.Branches == nil {
		meta.Branches = make(map[string]string)
	}
	if meta.Active == "" {
		meta.Active = "main"
	}
	return &meta
}

func saveBranchMeta(envPath string, meta *branchMeta) error {
	path := branchMetaPath(envPath)
	lock, err := fs.LockFile(path)
	if err != nil {
		return err
	}
	defer fs.UnlockFile(lock)

	data, _ := json.MarshalIndent(meta, "", "  ")
	return os.WriteFile(path, data, 0644)
}

func branchStateDir(envPath, branchName string) string {
	return filepath.Join(envPath, ".fpm-branches", branchName)
}

func runBranchCreate(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	branchName := args[0]
	meta := loadBranchMeta(activeVenv.Path)

	if _, exists := meta.Branches[branchName]; exists {
		return fmt.Errorf("branch %q already exists", branchName)
	}

	// Save current state as this branch's snapshot
	stateDir := branchStateDir(activeVenv.Path, branchName)
	os.MkdirAll(stateDir, 0755)

	// Copy the depgraph as branch state
	graphSrc := filepath.Join(activeVenv.Path, ".fpm-depgraph.json")
	graphDst := filepath.Join(stateDir, "depgraph.json")
	if data, err := os.ReadFile(graphSrc); err == nil {
		os.WriteFile(graphDst, data, 0644)
	}

	// Save installed package list
	sitePackagesDirs := env.FindSitePackagesDirs([]string{activeVenv.SitePackages})
	scanner := env.NewScanner(sitePackagesDirs)
	scanResult, _ := scanner.Scan()
	var pkgList []string
	for _, pkg := range scanResult.Packages {
		if pkg.Manager == env.ManagerFpm {
			pkgList = append(pkgList, pkg.Name.Normalized()+"=="+pkg.Version.String())
		}
	}
	listData, _ := json.Marshal(pkgList)
	os.WriteFile(filepath.Join(stateDir, "packages.json"), listData, 0644)

	meta.Branches[branchName] = stateDir
	saveBranchMeta(activeVenv.Path, meta)

	fmt.Printf("Created branch \"%s\" from current state (%d packages)\n", branchName, len(pkgList))
	return nil
}

func runBranchSwitch(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	// Acquire venv lock to prevent concurrent install/remove during switch
	venvLock, lockErr := fs.LockFile(filepath.Join(activeVenv.SitePackages, ".fpm"))
	if lockErr != nil {
		return fmt.Errorf("could not acquire environment lock (another operation in progress?): %w", lockErr)
	}
	defer fs.UnlockFile(venvLock)

	targetBranch := args[0]
	meta := loadBranchMeta(activeVenv.Path)

	if targetBranch == meta.Active {
		fmt.Printf("Already on branch \"%s\"\n", targetBranch)
		return nil
	}

	if _, exists := meta.Branches[targetBranch]; !exists {
		return fmt.Errorf("branch %q does not exist. Create it with 'fpm branch create %s'", targetBranch, targetBranch)
	}

	// Save current state to current branch before switching
	currentStateDir := branchStateDir(activeVenv.Path, meta.Active)
	os.MkdirAll(currentStateDir, 0755)

	graphSrc := filepath.Join(activeVenv.Path, ".fpm-depgraph.json")
	graphDst := filepath.Join(currentStateDir, "depgraph.json")
	if data, err := os.ReadFile(graphSrc); err == nil {
		os.WriteFile(graphDst, data, 0644)
	}

	sitePackagesDirs := env.FindSitePackagesDirs([]string{activeVenv.SitePackages})
	scanner := env.NewScanner(sitePackagesDirs)
	scanResult, _ := scanner.Scan()
	var pkgList []string
	for _, pkg := range scanResult.Packages {
		if pkg.Manager == env.ManagerFpm {
			pkgList = append(pkgList, pkg.Name.Normalized()+"=="+pkg.Version.String())
		}
	}
	listData, _ := json.Marshal(pkgList)
	os.WriteFile(filepath.Join(currentStateDir, "packages.json"), listData, 0644)
	meta.Branches[meta.Active] = currentStateDir

	// Restore target branch state
	targetStateDir := meta.Branches[targetBranch]
	graphData, err := os.ReadFile(filepath.Join(targetStateDir, "depgraph.json"))
	if err == nil {
		os.WriteFile(graphSrc, graphData, 0644)
	}

	// Clean current site-packages of fpm packages and re-link from target
	for _, pkg := range scanResult.Packages {
		if pkg.Manager == env.ManagerFpm {
			uninstallPackage(activeVenv.SitePackages, pkg.Name)
		}
	}

	// Read target package list — packages will be restored by 'fpm sync'
	targetPkgData, _ := os.ReadFile(filepath.Join(targetStateDir, "packages.json"))
	var targetPkgs []string
	json.Unmarshal(targetPkgData, &targetPkgs)

	meta.Active = targetBranch
	saveBranchMeta(activeVenv.Path, meta)

	fmt.Printf("Switched to branch \"%s\"\n", targetBranch)
	if len(targetPkgs) > 0 {
		fmt.Printf("  Run 'fpm sync' to restore %d packages from this branch.\n", len(targetPkgs))
	}
	return nil
}

func runBranchList(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	meta := loadBranchMeta(activeVenv.Path)

	// Always show main
	if meta.Active == "main" {
		fmt.Printf("* \033[32mmain\033[0m\n")
	} else {
		fmt.Printf("  main\n")
	}

	for name := range meta.Branches {
		if name == "main" {
			continue
		}
		if name == meta.Active {
			fmt.Printf("* \033[32m%s\033[0m\n", name)
		} else {
			fmt.Printf("  %s\n", name)
		}
	}
	return nil
}

func runBranchDelete(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	branchName := args[0]
	meta := loadBranchMeta(activeVenv.Path)

	if branchName == meta.Active {
		return fmt.Errorf("cannot delete the active branch. Switch to another branch first")
	}
	if branchName == "main" {
		return fmt.Errorf("cannot delete the main branch")
	}

	stateDir, exists := meta.Branches[branchName]
	if !exists {
		return fmt.Errorf("branch %q not found", branchName)
	}

	os.RemoveAll(stateDir)
	delete(meta.Branches, branchName)
	saveBranchMeta(activeVenv.Path, meta)

	fmt.Printf("Deleted branch \"%s\"\n", branchName)
	return nil
}
