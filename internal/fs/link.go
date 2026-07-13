package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

type LinkMode int

const (
	LinkModeAuto     LinkMode = iota
	LinkModeReflink           // CoW clone (APFS, btrfs, xfs)
	LinkModeHardlink          // os.Link
	LinkModeCopy              // full copy
	LinkModeSymlink           // os.Symlink
)

func (m LinkMode) String() string {
	switch m {
	case LinkModeReflink:
		return "reflink"
	case LinkModeHardlink:
		return "hardlink"
	case LinkModeCopy:
		return "copy"
	case LinkModeSymlink:
		return "symlink"
	default:
		return "auto"
	}
}

func BestLinkMode() LinkMode {
	switch runtime.GOOS {
	case "darwin":
		return LinkModeReflink
	case "linux":
		return LinkModeReflink
	default:
		return LinkModeHardlink
	}
}

func LinkFile(src, dst string, mode LinkMode) error {
	if mode == LinkModeAuto {
		mode = BestLinkMode()
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	switch mode {
	case LinkModeReflink:
		err := reflink(src, dst)
		if err != nil {
			// Fall back to hardlink
			err = os.Link(src, dst)
			if err != nil {
				// Fall back to copy
				return copyFile(src, dst)
			}
		}
		return nil

	case LinkModeHardlink:
		err := os.Link(src, dst)
		if err != nil {
			return copyFile(src, dst)
		}
		return nil

	case LinkModeCopy:
		return copyFile(src, dst)

	case LinkModeSymlink:
		return os.Symlink(src, dst)

	default:
		return fmt.Errorf("unknown link mode: %d", mode)
	}
}

func LinkDir(srcDir, dstDir string, mode LinkMode) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dstDir, rel)

		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}

		return LinkFile(path, dst, mode)
	})
}

func AtomicReplace(srcDir, targetDir, pkgName string, mode LinkMode) error {
	parent := filepath.Dir(targetDir)
	tmpDst := filepath.Join(parent, ".fpm-install-"+pkgName)
	os.RemoveAll(tmpDst)

	if err := LinkDir(srcDir, tmpDst, mode); err != nil {
		os.RemoveAll(tmpDst)
		return err
	}

	finalPath := filepath.Join(targetDir, pkgName)
	oldPath := finalPath + ".old"

	os.Rename(finalPath, oldPath)
	if err := os.Rename(filepath.Join(tmpDst, pkgName), finalPath); err != nil {
		os.Rename(oldPath, finalPath)
		os.RemoveAll(tmpDst)
		return err
	}

	os.RemoveAll(oldPath)
	os.RemoveAll(tmpDst)
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func AtomicRename(src, dst string) error {
	return os.Rename(src, dst)
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
