//go:build windows

package fs

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32    = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx = modkernel32.NewProc("LockFileEx")
	procUnlockFile = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock = 0x00000002
	lockfileFailImmediately = 0x00000001
)

func LockFile(path string) (*os.File, error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if err := lockFileWindows(f, lockfileExclusiveLock); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func LockFileShared(path string) (*os.File, error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	if err := lockFileWindows(f, 0); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func UnlockFile(f *os.File) error {
	if f == nil {
		return nil
	}
	ol := new(syscall.Overlapped)
	r1, _, _ := procUnlockFile.Call(
		uintptr(f.Fd()),
		uintptr(0),
		uintptr(unsafe.Pointer(ol)),
		uintptr(1), uintptr(0),
	)
	_ = r1
	return f.Close()
}

func lockFileWindows(f *os.File, flags uint32) error {
	ol := new(syscall.Overlapped)
	r1, _, err := procLockFileEx.Call(
		uintptr(f.Fd()),
		uintptr(flags),
		uintptr(0),
		uintptr(1), uintptr(0),
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}
