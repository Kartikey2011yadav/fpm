package env

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type ConflictAction int

const (
	ActionInstall ConflictAction = iota
	ActionSkip
	ActionAbort
)

type CrossManagerPolicy string

const (
	PolicyAsk     CrossManagerPolicy = "ask"
	PolicyInstall CrossManagerPolicy = "install"
	PolicySkip    CrossManagerPolicy = "skip"
)

type CrossManagerResult struct {
	Action   ConflictAction
	Existing *InstalledPackage
	Message  string
}

type CrossManagerChecker struct {
	scan   *ScanResult
	policy CrossManagerPolicy
	writer io.Writer
	reader io.Reader
}

func NewCrossManagerChecker(scan *ScanResult, policy CrossManagerPolicy) *CrossManagerChecker {
	return &CrossManagerChecker{
		scan:   scan,
		policy: policy,
		writer: os.Stdout,
		reader: os.Stdin,
	}
}

func (c *CrossManagerChecker) Check(name types.PackageName, requestedVersion pep440.Version) CrossManagerResult {
	existing := c.scan.FindByName(name)
	if len(existing) == 0 {
		return CrossManagerResult{Action: ActionInstall}
	}

	// Check for exact version match in another manager
	for _, pkg := range existing {
		if pkg.Manager == ManagerFpm {
			continue
		}

		if pkg.Version.Equal(requestedVersion) {
			// Same version exists in another manager — inform and skip
			msg := fmt.Sprintf("%s %s is already installed via %s — skipping download",
				name.Raw(), requestedVersion.String(), pkg.Manager)
			return CrossManagerResult{
				Action:   ActionSkip,
				Existing: &pkg,
				Message:  msg,
			}
		}
	}

	// Different version exists — needs user decision
	for _, pkg := range existing {
		if pkg.Manager == ManagerFpm {
			continue
		}

		return c.handleVersionConflict(name, requestedVersion, pkg)
	}

	return CrossManagerResult{Action: ActionInstall}
}

func (c *CrossManagerChecker) handleVersionConflict(
	name types.PackageName,
	requested pep440.Version,
	existing InstalledPackage,
) CrossManagerResult {
	switch c.policy {
	case PolicyInstall:
		return CrossManagerResult{
			Action:   ActionInstall,
			Existing: &existing,
			Message: fmt.Sprintf("%s %s exists via %s, installing %s (fpm's version will take priority)",
				name.Raw(), existing.Version.String(), existing.Manager, requested.String()),
		}

	case PolicySkip:
		return CrossManagerResult{
			Action:   ActionSkip,
			Existing: &existing,
			Message: fmt.Sprintf("%s %s exists via %s, skipping installation of %s (policy: skip)",
				name.Raw(), existing.Version.String(), existing.Manager, requested.String()),
		}

	default: // PolicyAsk
		return c.promptUser(name, requested, existing)
	}
}

func (c *CrossManagerChecker) promptUser(
	name types.PackageName,
	requested pep440.Version,
	existing InstalledPackage,
) CrossManagerResult {
	fmt.Fprintf(c.writer, "\n")
	fmt.Fprintf(c.writer, "  %s %s is installed via %s, but you're requesting %s.\n",
		name.Raw(), existing.Version.String(), existing.Manager, requested.String())
	fmt.Fprintf(c.writer, "  After installation, fpm's version will take priority based on path order.\n")
	fmt.Fprintf(c.writer, "\n")
	fmt.Fprintf(c.writer, "  [1] Skip installation (keep %s's %s)\n", existing.Manager, existing.Version.String())
	fmt.Fprintf(c.writer, "  [2] Install anyway (fpm's %s will shadow %s's)\n", requested.String(), existing.Manager)
	fmt.Fprintf(c.writer, "  [3] Abort\n")
	fmt.Fprintf(c.writer, "\n")
	fmt.Fprintf(c.writer, "  Choice [1/2/3]: ")

	var input string
	fmt.Fscanln(c.reader, &input)
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		return CrossManagerResult{
			Action:   ActionSkip,
			Existing: &existing,
			Message:  fmt.Sprintf("Keeping %s's %s %s", existing.Manager, name.Raw(), existing.Version.String()),
		}
	case "2":
		return CrossManagerResult{
			Action:   ActionInstall,
			Existing: &existing,
			Message:  fmt.Sprintf("Installing %s %s (will shadow %s's version)", name.Raw(), requested.String(), existing.Manager),
		}
	default:
		return CrossManagerResult{
			Action:  ActionAbort,
			Message: "Installation aborted by user",
		}
	}
}
