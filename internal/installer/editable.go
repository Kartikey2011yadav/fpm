package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EditableInstall struct {
	SourcePath  string
	TargetDir   string
	PackageName string
}

func InstallEditable(opts EditableInstall) error {
	absSource, err := filepath.Abs(opts.SourcePath)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}

	if _, err := os.Stat(absSource); err != nil {
		return fmt.Errorf("source path does not exist: %s", absSource)
	}

	// Write .egg-link file (legacy editable install format)
	eggLinkName := strings.ReplaceAll(opts.PackageName, "-", "_") + ".egg-link"
	eggLinkPath := filepath.Join(opts.TargetDir, eggLinkName)
	content := absSource + "\n.\n"
	if err := os.WriteFile(eggLinkPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .egg-link: %w", err)
	}

	// Add to easy-install.pth (legacy) or create a .pth file
	pthPath := filepath.Join(opts.TargetDir, "__editable__."+strings.ReplaceAll(opts.PackageName, "-", "_")+".pth")
	if err := os.WriteFile(pthPath, []byte(absSource+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write .pth: %w", err)
	}

	return nil
}

func UninstallEditable(packageName, sitePackages string) error {
	normalized := strings.ReplaceAll(packageName, "-", "_")

	// Remove .egg-link
	eggLink := filepath.Join(sitePackages, normalized+".egg-link")
	os.Remove(eggLink)

	// Remove .pth file
	pthFile := filepath.Join(sitePackages, "__editable__."+normalized+".pth")
	os.Remove(pthFile)

	return nil
}
