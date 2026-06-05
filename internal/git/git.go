package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/config"
)

type GitSource struct {
	URL       string
	Reference string // branch, tag, or commit
	Subdirectory string
}

type CloneResult struct {
	Path   string
	Commit string
}

func Clone(source GitSource) (*CloneResult, error) {
	cacheDir := filepath.Join(config.CacheDir(), "git")
	os.MkdirAll(cacheDir, 0755)

	// Generate unique directory name from URL
	dirName := urlToDir(source.URL)
	repoDir := filepath.Join(cacheDir, dirName)

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		// Already cloned — fetch latest
		if err := gitFetch(repoDir); err != nil {
			// Fetch failed, re-clone
			os.RemoveAll(repoDir)
			if err := gitClone(source.URL, repoDir); err != nil {
				return nil, err
			}
		}
	} else {
		if err := gitClone(source.URL, repoDir); err != nil {
			return nil, err
		}
	}

	// Checkout the specified reference
	ref := source.Reference
	if ref == "" {
		ref = "HEAD"
	}
	if err := gitCheckout(repoDir, ref); err != nil {
		return nil, fmt.Errorf("failed to checkout %s: %w", ref, err)
	}

	// Get current commit
	commit, err := gitRevParse(repoDir, "HEAD")
	if err != nil {
		return nil, err
	}

	resultPath := repoDir
	if source.Subdirectory != "" {
		resultPath = filepath.Join(repoDir, source.Subdirectory)
	}

	return &CloneResult{
		Path:   resultPath,
		Commit: commit,
	}, nil
}

func ParseGitURL(rawURL string) (*GitSource, error) {
	source := &GitSource{}

	// Strip git+ prefix
	url := rawURL
	if strings.HasPrefix(url, "git+") {
		url = url[4:]
	}

	// Check for #subdirectory fragment FIRST (before @ref parsing)
	if hashIdx := strings.Index(url, "#"); hashIdx > 0 {
		fragment := url[hashIdx+1:]
		url = url[:hashIdx]
		if strings.HasPrefix(fragment, "subdirectory=") {
			source.Subdirectory = strings.TrimPrefix(fragment, "subdirectory=")
		}
	}

	// Check for @ref suffix
	if atIdx := strings.LastIndex(url, "@"); atIdx > 0 {
		// Make sure it's not part of ssh URL (git@github.com)
		if !strings.Contains(url[:atIdx], ":") || strings.Contains(url[:atIdx], "://") {
			source.Reference = url[atIdx+1:]
			url = url[:atIdx]
		}
	}

	source.URL = url
	return source, nil
}

func IsGitURL(url string) bool {
	return strings.HasPrefix(url, "git+") ||
		strings.HasPrefix(url, "git://") ||
		(strings.HasPrefix(url, "https://") && strings.HasSuffix(url, ".git"))
}

// Internal git operations

func gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", "--depth=1", url, dest)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %s", stderr.String())
	}
	return nil
}

func gitFetch(repoDir string) error {
	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = repoDir
	return cmd.Run()
}

func gitCheckout(repoDir, ref string) error {
	cmd := exec.Command("git", "checkout", ref)
	cmd.Dir = repoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s", stderr.String())
	}
	return nil
}

func gitRevParse(repoDir, ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func urlToDir(url string) string {
	// Convert URL to safe directory name
	name := url
	name = strings.TrimPrefix(name, "https://")
	name = strings.TrimPrefix(name, "http://")
	name = strings.TrimPrefix(name, "git://")
	name = strings.TrimSuffix(name, ".git")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return name
}
