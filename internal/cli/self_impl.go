package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

func init() {
	selfUpdateCmd.RunE = runSelfUpdate
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("Checking for updates...")

	// Check latest release
	resp, err := http.Get("https://api.github.com/repos/Kartikey2011yadav/fpm/releases/latest")
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release info: %w", err)
	}

	latest := release.TagName
	if latest == "" {
		return fmt.Errorf("no releases found")
	}

	if latest == "v"+Version || latest == Version {
		fmt.Printf("Already on latest version: %s\n", Version)
		return nil
	}

	fmt.Printf("Current: %s → Latest: %s\n", Version, latest)

	// Find matching asset
	assetName := fmt.Sprintf("fpm-%s-%s-%s", latest, runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName || asset.Name == assetName+".tar.gz" || asset.Name == assetName+".zip" {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, latest)
	}

	// Download
	fmt.Printf("Downloading %s...\n", latest)
	binResp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer binResp.Body.Close()

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "fpm-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, binResp.Body); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Make executable
	os.Chmod(tmpFile.Name(), 0755)

	// Atomic replace
	if err := os.Rename(tmpFile.Name(), execPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w (try with sudo)", err)
	}

	fmt.Printf("Updated to %s.\n", latest)
	return nil
}
