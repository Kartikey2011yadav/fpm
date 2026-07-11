package cache

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MaxExtractSize is the maximum total bytes that can be extracted from a wheel.
// 2 GB should accommodate even the largest legitimate wheels (e.g. PyTorch).
const MaxExtractSize = 2 * 1024 * 1024 * 1024

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	var totalExtracted int64
	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)

		// Security: prevent zip slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		n, err := extractZipFile(f, target)
		if err != nil {
			return err
		}
		totalExtracted += n
		if totalExtracted > MaxExtractSize {
			return fmt.Errorf("extraction exceeded %d bytes limit (possible zip bomb)", MaxExtractSize)
		}
	}
	return nil
}

func extractZipFile(f *zip.File, target string) (int64, error) {
	rc, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return 0, err
	}
	defer out.Close()

	n, err := io.Copy(out, io.LimitReader(rc, MaxExtractSize))
	return n, err
}
