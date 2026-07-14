package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// DirPerm returns the appropriate directory permission mode.
// In multi-user mode: 0775 with setgid (group-writable, inherits group).
// In single-user mode: 0755.
func DirPerm(multiUser bool) os.FileMode {
	if multiUser {
		return 0775 | os.ModeSetgid
	}
	return 0755
}

// FilePerm returns the appropriate file permission mode.
// In multi-user mode: 0664 (group-writable).
// In single-user mode: 0644.
func FilePerm(multiUser bool) os.FileMode {
	if multiUser {
		return 0664
	}
	return 0644
}

// CheckWritable verifies that the given path (or its nearest existing parent)
// is writable by the current user. Returns a user-friendly error with hints
// if permission is denied.
func CheckWritable(path string) error {
	// Walk up to find nearest existing directory
	dir := path
	for {
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				dir = filepath.Dir(dir)
				continue
			}
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Try to create a temp file to verify write access
	tmpPath := filepath.Join(dir, ".fpm-write-test")
	f, err := os.Create(tmpPath)
	if err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:    path,
				Dir:     dir,
				Wrapped: err,
			}
		}
		return err
	}
	f.Close()
	os.Remove(tmpPath)
	return nil
}

// IsPermissionError returns true if the error is a permission denial.
func IsPermissionError(err error) bool {
	return isPermissionError(err)
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		if errors.Is(pathErr.Err, syscall.EACCES) {
			return true
		}
		if runtime.GOOS == "windows" && errors.Is(pathErr.Err, syscall.Errno(5)) {
			return true
		}
	}
	return false
}

// PermissionError provides context and hints for permission failures.
type PermissionError struct {
	Path    string
	Dir     string
	Wrapped error
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission denied: cannot write to %s", e.Path)
}

func (e *PermissionError) Unwrap() error {
	return e.Wrapped
}

func (e *PermissionError) Hint() string {
	if runtime.GOOS == "windows" {
		return "Run as Administrator, or choose a different location."
	}
	return "Try: sudo fpm <command>\n          Or use a virtual environment (fpm venv) to install without root."
}
