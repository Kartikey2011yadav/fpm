package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/spf13/cobra"
)

func init() {
	repairCmd.RunE = runRepair
	configShowCmd.RunE = runConfigShow
	configSetCmd.RunE = runConfigSet
	configInitCmd.RunE = runConfigInit
}

func runRepair(cmd *cobra.Command, args []string) error {
	fmt.Println("fpm repair — checking installation health...")
	fmt.Println()
	issues := 0
	fixed := 0

	// 1. Check directories exist
	fmt.Println("  Checking directories...")
	dirs := map[string]string{
		"Cache":       config.CacheDir(),
		"Data":        config.DataDir(),
		"Config":      config.ConfigDir(),
		"Python":      config.PythonInstallDir(),
		"Tools":       config.ToolsDir(),
		"Bin":         config.BinDir(),
		"Credentials": config.CredentialsDir(),
	}

	for name, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("    \033[33m●\033[0m %s directory missing: %s\n", name, dir)
			os.MkdirAll(dir, 0755)
			fmt.Printf("      \033[32m✓\033[0m Created\n")
			issues++
			fixed++
		} else {
			fmt.Printf("    \033[32m✓\033[0m %s: %s\n", name, dir)
		}
	}

	// 2. Check cache structure
	fmt.Println("\n  Checking cache structure...")
	pkgCache := cache.New(config.CacheDir())
	if err := pkgCache.Init(); err != nil {
		fmt.Printf("    \033[31m✗\033[0m Cache init failed: %v\n", err)
		issues++
	} else {
		fmt.Printf("    \033[32m✓\033[0m Cache directories intact\n")
	}

	// 3. Check Python availability
	fmt.Println("\n  Checking Python...")
	finder := python.NewFinder()
	interp, err := finder.FindBest("")
	if err != nil {
		fmt.Printf("    \033[31m✗\033[0m No Python found on system\n")
		fmt.Printf("      hint: Install Python or run 'fpm python install 3.12'\n")
		issues++
	} else {
		fmt.Printf("    \033[32m✓\033[0m Python %s at %s\n", interp.VersionString(), interp.Path)
	}

	// 4. Check bin directory is in PATH
	fmt.Println("\n  Checking PATH...")
	binDir := config.BinDir()
	pathEnv := os.Getenv("PATH")
	if !strings.Contains(pathEnv, binDir) {
		fmt.Printf("    \033[33m●\033[0m %s is not in PATH\n", binDir)
		fmt.Printf("      hint: Add to your shell profile: export PATH=\"%s:$PATH\"\n", binDir)
		issues++
	} else {
		fmt.Printf("    \033[32m✓\033[0m Bin directory in PATH\n")
	}

	// 5. Check symlinks in bin dir
	fmt.Println("\n  Checking symlinks...")
	symlinks := []string{"python3", "python"}
	for _, name := range symlinks {
		link := filepath.Join(binDir, name)
		target, err := os.Readlink(link)
		if err != nil {
			continue
		}
		if _, err := os.Stat(target); os.IsNotExist(err) {
			fmt.Printf("    \033[31m✗\033[0m Broken symlink: %s → %s\n", link, target)
			os.Remove(link)
			fmt.Printf("      \033[32m✓\033[0m Removed broken symlink\n")
			issues++
			fixed++
		}
	}

	// 6. Check config file
	fmt.Println("\n  Checking configuration...")
	userCfgPath := filepath.Join(config.ConfigDir(), "config.toml")
	if _, err := os.Stat(userCfgPath); os.IsNotExist(err) {
		fmt.Printf("    \033[2m●\033[0m No user config (using defaults): %s\n", userCfgPath)
	} else {
		fmt.Printf("    \033[32m✓\033[0m User config: %s\n", userCfgPath)
	}

	// 7. Check interpreter cache integrity
	fmt.Println("\n  Checking interpreter cache...")
	interpDir := filepath.Join(config.CacheDir(), "interpreters")
	entries, _ := os.ReadDir(interpDir)
	staleCount := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		// We can't easily verify without re-probing, just count
		staleCount++
	}
	if staleCount > 0 {
		fmt.Printf("    \033[32m✓\033[0m %d cached interpreter entries\n", staleCount)
	} else {
		fmt.Printf("    \033[2m●\033[0m No cached interpreter data (will be populated on first use)\n")
	}

	// 8. Multi-user mode checks
	if config.IsMultiUserMode() {
		fmt.Println("\n  Checking multi-user mode...")
		fmt.Printf("    \033[32m✓\033[0m Multi-user mode active\n")
		fmt.Printf("    Shared cache: %s\n", config.CacheDir())

		fixPerms, _ := cmd.Flags().GetBool("fix-permissions")
		if fixPerms {
			fmt.Println("    Fixing permissions...")
			permFixed := fixMultiUserPermissions(config.CacheDir())
			fixed += permFixed
			if permFixed > 0 {
				fmt.Printf("    \033[32m✓\033[0m Fixed permissions on %d items\n", permFixed)
			} else {
				fmt.Printf("    \033[32m✓\033[0m Permissions already correct\n")
			}
		} else {
			fmt.Printf("    \033[2m●\033[0m Run 'fpm repair --fix-permissions' to fix shared cache permissions\n")
		}
	}

	// Summary
	fmt.Println()
	if issues == 0 {
		fmt.Println("  \033[32m✓ Everything looks good!\033[0m")
	} else {
		fmt.Printf("  Found %d issue(s), fixed %d.\n", issues, fixed)
		if issues > fixed {
			fmt.Printf("  %d issue(s) require manual action (see hints above).\n", issues-fixed)
		}
	}

	return nil
}

func fixMultiUserPermissions(cacheRoot string) int {
	fixed := 0
	filepath.Walk(cacheRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			want := os.FileMode(0775) | os.ModeSetgid
			if info.Mode().Perm() != want.Perm() || info.Mode()&os.ModeSetgid == 0 {
				os.Chmod(path, want)
				fixed++
			}
		} else {
			want := os.FileMode(0664)
			if info.Mode().Perm() != want.Perm() {
				os.Chmod(path, want)
				fixed++
			}
		}
		return nil
	})
	return fixed
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, _ := config.LoadFromCwd()

	fmt.Println("Current fpm configuration:")
	fmt.Println()

	fmt.Println("  Directories:")
	fmt.Printf("    cache:       %s\n", config.CacheDir())
	fmt.Printf("    data:        %s\n", config.DataDir())
	fmt.Printf("    config:      %s\n", config.ConfigDir())
	fmt.Printf("    python:      %s\n", config.PythonInstallDir())
	fmt.Printf("    tools:       %s\n", config.ToolsDir())
	fmt.Printf("    bin:         %s\n", config.BinDir())
	fmt.Printf("    credentials: %s\n", config.CredentialsDir())

	fmt.Println("\n  Settings:")
	fmt.Printf("    concurrency:           %d\n", cfg.Tool.Concurrency)
	fmt.Printf("    link-mode:             %s\n", cfg.Tool.LinkMode)
	fmt.Printf("    cross-manager-policy:  %s\n", cfg.Tool.CrossManagerPolicy)

	fmt.Println("\n  Logging:")
	logLevel := cfg.Log.Level
	if logLevel == "" {
		logLevel = "off"
	}
	logFile := cfg.Log.File
	if logFile == "" {
		logFile = "(default: " + config.DefaultLogFile() + ")"
	}
	fmt.Printf("    level:  %s\n", logLevel)
	fmt.Printf("    file:   %s\n", logFile)

	fmt.Println("\n  Indexes:")
	for _, idx := range cfg.Indexes {
		def := ""
		if idx.Default {
			def = " (default)"
		}
		fmt.Printf("    %s: %s%s\n", idx.Name, idx.URL, def)
	}

	if len(cfg.Network.AllowInsecureHost) > 0 {
		fmt.Println("\n  Network:")
		fmt.Printf("    allow-insecure-host: %s\n", strings.Join(cfg.Network.AllowInsecureHost, ", "))
	}

	fmt.Println("\n  Config files (highest priority first):")
	cwd, _ := os.Getwd()
	projectCfg := filepath.Join(cwd, "fpm.toml")
	userCfg := filepath.Join(config.ConfigDir(), "config.toml")
	systemCfg := filepath.Join(config.SystemConfigDir(), "config.toml")

	for _, path := range []string{projectCfg, userCfg, systemCfg} {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("    \033[32m✓\033[0m %s\n", path)
		} else {
			fmt.Printf("    \033[2m-\033[0m %s\n", path)
		}
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	userCfgPath := filepath.Join(config.ConfigDir(), "config.toml")
	os.MkdirAll(filepath.Dir(userCfgPath), 0755)

	// Load existing config or create empty
	existing := make(map[string]interface{})
	if data, err := os.ReadFile(userCfgPath); err == nil {
		toml.Unmarshal(data, &existing)
	}

	// Set the value in nested map
	parts := strings.Split(key, ".")
	setNestedValue(existing, parts, value)

	// Write back
	f, err := os.Create(userCfgPath)
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(existing); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	fmt.Printf("Saved to %s\n", userCfgPath)

	// Handle cache.dir change — offer to move
	if key == "cache.dir" {
		oldDir := config.CacheDir()
		if oldDir != value {
			fmt.Printf("\nTo move existing cache data:\n  mv %s/* %s/\n", oldDir, value)
		}
	}

	// Handle mode change to multi-user
	if key == "tool.mode" && value == "multi-user" {
		sharedDir := config.SharedCacheDir()
		fmt.Printf("\nMulti-user mode enabled.\n")
		fmt.Printf("  Shared cache: %s\n", sharedDir)
		fmt.Printf("\n  Setup (run as root):\n")
		fmt.Printf("    mkdir -p %s\n", sharedDir)
		fmt.Printf("    groupadd fpm 2>/dev/null\n")
		fmt.Printf("    chgrp -R fpm %s\n", sharedDir)
		fmt.Printf("    chmod -R 2775 %s\n", sharedDir)
		fmt.Printf("    usermod -aG fpm <username>\n")
	}

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	userCfgPath := filepath.Join(config.ConfigDir(), "config.toml")

	if _, err := os.Stat(userCfgPath); err == nil {
		return fmt.Errorf("config file already exists: %s\nUse 'fpm config set' to modify values", userCfgPath)
	}

	os.MkdirAll(filepath.Dir(userCfgPath), 0755)

	defaultContent := `# fpm user configuration
# This file is loaded for all projects. Project-level fpm.toml takes priority.
# See: fpm config show

[tool]
# cross-manager-policy = "ask"  # ask | install | skip
# link-mode = "auto"            # auto | hardlink | copy
# concurrency = 50

[python]
# version = "3.12"
# preference = "managed"        # managed | system | only-managed

[network]
# allow-insecure-host = []
# system-certs = false

[log]
# level = "off"                 # debug | info | warn | error | off
# file = ""                     # empty = default (~/.local/share/fpm/logs/fpm.log)

# [cache]
# dir = ""                      # override cache directory
`

	if err := os.WriteFile(userCfgPath, []byte(defaultContent), 0644); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	fmt.Printf("Created user config: %s\n", userCfgPath)
	fmt.Println("Edit this file to customize fpm behavior globally.")
	return nil
}

func setNestedValue(m map[string]interface{}, keys []string, value string) {
	if len(keys) == 1 {
		m[keys[0]] = value
		return
	}

	child, ok := m[keys[0]].(map[string]interface{})
	if !ok {
		child = make(map[string]interface{})
		m[keys[0]] = child
	}
	setNestedValue(child, keys[1:], value)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
