package python

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/config"
)

const releaseAPI = "https://api.github.com/repos/indygreg/python-build-standalone/releases?per_page=5"

type ManagedVersion struct {
	Version string
	Path    string
	Current bool
}

func ListManaged() ([]ManagedVersion, error) {
	installDir := config.PythonInstallDir()
	entries, err := os.ReadDir(installDir)
	if err != nil {
		return nil, nil
	}

	currentLink, _ := os.Readlink(filepath.Join(config.BinDir(), "python3"))

	var versions []ManagedVersion
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "cpython-") {
			continue
		}
		ver := strings.TrimPrefix(name, "cpython-")
		pythonPath := filepath.Join(installDir, name, pythonBinaryRelPath())

		isCurrent := false
		if currentLink != "" {
			resolved, _ := filepath.EvalSymlinks(currentLink)
			if resolved == pythonPath {
				isCurrent = true
			}
		}

		versions = append(versions, ManagedVersion{
			Version: ver,
			Path:    filepath.Join(installDir, name),
			Current: isCurrent,
		})
	}

	return versions, nil
}

func InstallVersion(ctx context.Context, version string) (string, error) {
	installDir := config.PythonInstallDir()
	targetDir := filepath.Join(installDir, "cpython-"+version)

	// Check if already installed
	pythonPath := filepath.Join(targetDir, pythonBinaryRelPath())
	if _, err := os.Stat(pythonPath); err == nil {
		return targetDir, nil
	}

	// Determine download URL
	url, err := buildDownloadURL(version)
	if err != nil {
		return "", err
	}

	// Download
	fmt.Printf("Downloading Python %s...\n", version)
	tarball, err := downloadFile(ctx, url)
	if err != nil {
		return "", fmt.Errorf("failed to download Python %s: %w", version, err)
	}
	defer os.Remove(tarball)

	// Extract
	fmt.Printf("Installing Python %s...\n", version)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	if err := extractTarGz(tarball, targetDir); err != nil {
		os.RemoveAll(targetDir)
		return "", fmt.Errorf("failed to extract: %w", err)
	}

	// Ensure bin directory exists and update symlinks
	if err := updateBinSymlinks(version, targetDir); err != nil {
		return "", err
	}

	fmt.Printf("Installed Python %s to %s\n", version, targetDir)
	return targetDir, nil
}

func UseVersion(version string, global bool) error {
	installDir := config.PythonInstallDir()
	targetDir := filepath.Join(installDir, "cpython-"+version)

	pythonPath := filepath.Join(targetDir, pythonBinaryRelPath())
	if _, err := os.Stat(pythonPath); err != nil {
		return fmt.Errorf("Python %s is not installed. Run 'fpm python install %s' first", version, version)
	}

	if global {
		return updateBinSymlinks(version, targetDir)
	}

	// Local: write .python-version
	cwd, _ := os.Getwd()
	return WritePythonVersionFile(cwd, version)
}

func UninstallVersion(version string) error {
	installDir := config.PythonInstallDir()
	targetDir := filepath.Join(installDir, "cpython-"+version)

	if _, err := os.Stat(targetDir); err != nil {
		return fmt.Errorf("Python %s is not installed", version)
	}

	return os.RemoveAll(targetDir)
}

func updateBinSymlinks(version, targetDir string) error {
	binDir := config.BinDir()
	os.MkdirAll(binDir, 0755)

	pythonPath := filepath.Join(targetDir, pythonBinaryRelPath())

	// Create python3 symlink
	link := filepath.Join(binDir, "python3")
	os.Remove(link)
	if err := os.Symlink(pythonPath, link); err != nil {
		return err
	}

	// Create python symlink
	link2 := filepath.Join(binDir, "python")
	os.Remove(link2)
	os.Symlink(pythonPath, link2)

	// Create versioned symlink (e.g., python3.11)
	parts := strings.SplitN(version, ".", 3)
	if len(parts) >= 2 {
		versionedLink := filepath.Join(binDir, fmt.Sprintf("python%s.%s", parts[0], parts[1]))
		os.Remove(versionedLink)
		os.Symlink(pythonPath, versionedLink)
	}

	return nil
}

func buildDownloadURL(version string) (string, error) {
	os_ := runtime.GOOS
	arch := runtime.GOARCH

	var platform string
	switch {
	case os_ == "linux" && arch == "amd64":
		platform = "x86_64-unknown-linux-gnu"
	case os_ == "linux" && arch == "arm64":
		platform = "aarch64-unknown-linux-gnu"
	case os_ == "darwin" && arch == "amd64":
		platform = "x86_64-apple-darwin"
	case os_ == "darwin" && arch == "arm64":
		platform = "aarch64-apple-darwin"
	case os_ == "windows" && arch == "amd64":
		platform = "x86_64-pc-windows-msvc"
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", os_, arch)
	}

	// Query GitHub API for recent releases and find a matching asset
	req, err := http.NewRequest("GET", releaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "fpm")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query python-build-standalone releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var releases []struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse releases: %w", err)
	}

	// Search for matching asset: cpython-{version}*-{platform}-install_only_stripped.tar.gz
	// version can be "3.12" (match any 3.12.x) or "3.12.7" (exact)
	prefix := "cpython-" + version
	suffix := platform + "-install_only_stripped.tar.gz"

	for _, rel := range releases {
		for _, asset := range rel.Assets {
			if strings.HasPrefix(asset.Name, prefix) && strings.HasSuffix(asset.Name, suffix) {
				return asset.BrowserDownloadURL, nil
			}
		}
	}

	return "", fmt.Errorf("no Python %s build found for %s/%s — check available versions at https://github.com/indygreg/python-build-standalone/releases", version, os_, arch)
}

func downloadFile(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	tmpFile, err := os.CreateTemp("", "fpm-python-*.tar.gz")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

func extractTarGz(tarball, destDir string) error {
	cmd := exec.Command("tar", "-xzf", tarball, "-C", destDir, "--strip-components=1")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
