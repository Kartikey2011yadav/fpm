package installer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/pkg/types"
)

type UninstallResult struct {
	RemovedFiles int
	RemovedDirs  int
	Package      string
	Version      string
}

func Uninstall(name types.PackageName, sitePackages string) (*UninstallResult, error) {
	// Find the .dist-info directory for this package
	distInfoDir, err := findDistInfo(name, sitePackages)
	if err != nil {
		return nil, fmt.Errorf("package %s not found: %w", name.Raw(), err)
	}

	result := &UninstallResult{
		Package: name.Raw(),
	}

	// Read RECORD file
	recordPath := filepath.Join(distInfoDir, "RECORD")
	files, err := readRecord(recordPath)
	if err != nil {
		// No RECORD file — try to remove just the dist-info and top-level package dir
		return removeByConvention(name, distInfoDir, sitePackages)
	}

	// Remove all files listed in RECORD
	for _, file := range files {
		fullPath := filepath.Join(sitePackages, file)
		if err := os.Remove(fullPath); err == nil {
			result.RemovedFiles++
		}
	}

	// Remove the dist-info directory itself
	if err := os.RemoveAll(distInfoDir); err == nil {
		result.RemovedDirs++
	}

	// Clean up empty parent directories
	cleanEmptyDirs(sitePackages)

	return result, nil
}

func findDistInfo(name types.PackageName, sitePackages string) (string, error) {
	entries, err := os.ReadDir(sitePackages)
	if err != nil {
		return "", err
	}

	normalized := name.Normalized()
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dist-info") {
			continue
		}

		// Extract package name from dist-info dir name (e.g., "requests-2.31.0.dist-info")
		dirName := entry.Name()
		dashIdx := strings.LastIndex(strings.TrimSuffix(dirName, ".dist-info"), "-")
		if dashIdx < 0 {
			continue
		}
		pkgPart := dirName[:dashIdx]
		// Normalize for comparison
		pkgNormalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(pkgPart, "_", "-"), ".", "-"))
		if pkgNormalized == normalized {
			return filepath.Join(sitePackages, dirName), nil
		}
	}

	return "", fmt.Errorf("no .dist-info found for %s", name.Raw())
}

func readRecord(recordPath string) ([]string, error) {
	f, err := os.Open(recordPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var files []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// RECORD format: path,hash,size
		parts := strings.SplitN(line, ",", 3)
		if len(parts) >= 1 && parts[0] != "" {
			files = append(files, parts[0])
		}
	}
	return files, scanner.Err()
}

func removeByConvention(name types.PackageName, distInfoDir, sitePackages string) (*UninstallResult, error) {
	result := &UninstallResult{Package: name.Raw()}

	// Remove dist-info
	os.RemoveAll(distInfoDir)
	result.RemovedDirs++

	// Try removing the package directory (e.g., "requests/")
	pkgDir := filepath.Join(sitePackages, strings.ReplaceAll(name.Normalized(), "-", "_"))
	if info, err := os.Stat(pkgDir); err == nil && info.IsDir() {
		os.RemoveAll(pkgDir)
		result.RemovedDirs++
	}

	// Try single-file module (e.g., "six.py")
	singleFile := filepath.Join(sitePackages, strings.ReplaceAll(name.Normalized(), "-", "_")+".py")
	if _, err := os.Stat(singleFile); err == nil {
		os.Remove(singleFile)
		result.RemovedFiles++
	}

	return result, nil
}

func cleanEmptyDirs(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == root {
			return nil
		}
		entries, _ := os.ReadDir(path)
		if len(entries) == 0 {
			os.Remove(path)
		}
		return nil
	})
}
