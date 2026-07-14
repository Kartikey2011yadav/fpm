package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs [topic]",
	Short: "Open documentation or show help topics",
	Long: `Open fpm documentation in your browser, or show inline help for a topic.

Without arguments, opens the full documentation in your default browser.

Available topics:
  quickstart   Getting started with fpm
  config       Configuration reference (fpm.toml)
  caching      Content-addressable storage and cache
  snapshots    Environment snapshots (create, restore, diff)
  resolver     Dependency resolution algorithm
  venv         Virtual environment management
  install      Installation methods
  workflow     Git-like environment workflow (stash, branch, bisect)`,
	GroupID: "advanced",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runDocs,
}

func init() {
	rootCmd.AddCommand(docsCmd)
}

var docTopics = map[string]string{
	"quickstart": `Getting Started with fpm
========================

  fpm init myproject && cd myproject   # Create project + venv
  fpm install requests pandas          # Install packages
  fpm run python main.py               # Run in managed env
  fpm list                             # See installed packages
  fpm list -a                          # See ALL packages (pip, conda, etc.)

No activation needed. fpm detects your project by directory.
Run 'fpm docs' to open full documentation in your browser.`,

	"config": `Configuration Reference (fpm.toml)
====================================

fpm uses fpm.toml in your project root. Priority order:
  CLI flags > env vars > project fpm.toml > user config > defaults

Key sections:
  [project]      name, requires-python, dependencies
  [tool.fpm]     cross-manager-policy, concurrency, link-mode
  [python]       version, preference (managed|system|only-managed)
  [immutable]    packages pinned to exact versions
  [[index]]      package index URLs (PyPI, private registries)

Environment variables:
  FPM_CACHE_DIR, FPM_DATA_DIR, FPM_CONFIG_DIR
  FPM_INSECURE=1, FPM_ALLOW_INSECURE_HOST=host1,host2

Run 'fpm config show' to see resolved configuration.`,

	"caching": `Content-Addressable Storage (CAS)
==================================

fpm stores every package exactly once, keyed by SHA256 hash:
  ~/.cache/fpm/cas/sha256/<prefix>/<hash>/

Packages are linked into environments via reflink (CoW) or hardlink.
10 projects using requests = one copy on disk.

Cache layout:
  cas/       Content-addressable package storage
  wheels/    Downloaded .whl files
  http/      Metadata cache (10-min TTL)
  refs/      Environment-to-package reference tracking

Commands:
  fpm cache gc      Remove unreferenced packages
  fpm cache clean   Clear entire cache`,

	"snapshots": `Environment Snapshots
======================

Capture and restore your entire Python environment state:

  fpm snapshot create "before experiment"   # Save state
  fpm snapshot list                         # List snapshots
  fpm snapshot diff <id>                    # Compare with current
  fpm snapshot restore <id>                 # Restore instantly

Snapshots capture:
  - All packages from all managers (fpm, pip, uv, conda, poetry, pdm)
  - Python version and paths
  - fpm.toml configuration

Restore uses CAS re-linking for instant recovery.`,

	"resolver": `Dependency Resolution (PubGrub)
================================

fpm uses the PubGrub algorithm for dependency resolution:
  - Modern backtracking solver
  - Handles version conflicts with clear error messages
  - Supports platform markers and environment constraints

Strategies:
  highest    Install highest compatible versions (default)
  lowest     Install lowest compatible versions
  installed  Prefer already-installed versions

Immutable packages are injected as hard constraints.
The resolver fails with a clear message if resolution is impossible.`,

	"venv": `Virtual Environment Management
================================

fpm auto-creates and detects virtual environments:

  fpm venv             Create .venv in current directory
  fpm venv --python 3.11   Use specific Python version

Detection walks up from cwd looking for pyvenv.cfg.
No 'source activate' needed — fpm detects by directory.

Python version management:
  fpm python list      Show available versions
  fpm python install   Download and install a version
  fpm python use       Set default version for project`,

	"install": `Installation Methods
=====================

Interactive installer (recommended):
  curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | bash

Windows PowerShell:
  irm https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.ps1 | iex

Other methods:
  pip install fpm-cli                                      # PyPI
  brew install kartikeyyadav/tap/fpm                       # Homebrew
  go install github.com/kartikeyyadav/fpm/cmd/fpm@latest   # Go
  docker run ghcr.io/kartikey2011yadav/fpm --version       # Docker

Update:
  fpm self update`,

	"workflow": `Git-Like Environment Workflow
==============================

fpm treats environments like git treats code — versioned, diffable, reversible.

Status & History:
  fpm status              Show what changed vs lockfile (added/removed/changed)
  fpm log                 Show operation history (install/remove/upgrade)
  fpm log --oneline       Compact view with operation IDs
  fpm blame <pkg>         Why was this package installed? By whom?

Save & Restore:
  fpm stash               Save unlocked packages, restore clean lockfile state
  fpm stash pop           Bring stashed packages back
  fpm stash list          See stash stack
  fpm snapshot create     Capture full environment state
  fpm snapshot restore    Roll back to any point in time
  fpm tag <name>          Name a snapshot for easy reference

Branching & Experimentation:
  fpm branch create <n>   Create a named env branch
  fpm branch switch <n>   Switch to a different branch
  fpm branch list         See all branches
  fpm cherry-pick <snap> <pkg>   Restore one package from a snapshot

Debugging:
  fpm bisect start        Begin binary search for breaking change
  fpm bisect good/bad     Mark snapshots as working/broken
  fpm revert <id>         Undo a past operation by journal ID`,
}

func runDocs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return openBrowser("https://github.com/Kartikey2011yadav/fpm#readme")
	}

	topic := args[0]
	content, ok := docTopics[topic]
	if !ok {
		return fmt.Errorf("unknown topic %q\n\nAvailable: quickstart, config, caching, snapshots, resolver, venv, install", topic)
	}

	fmt.Println(content)
	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Open in your browser: %s\n", url)
		return nil
	}
	fmt.Printf("Opened documentation in browser.\n")
	return nil
}
