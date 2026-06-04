package fs

import (
	"os"
	"syscall"
	"unsafe"
)

func reflink(src, dst string) error {
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

	// APFS clonefile via fcopyfile with COPYFILE_CLONE flag
	// Use fclonefileat syscall for CoW copy
	srcFd := srcFile.Fd()
	dstFd := dstFile.Fd()

	// FICLONE equivalent on macOS: use fclonefileat
	// Actually on macOS, we use clonefile(2) syscall
	dstFile.Close()
	os.Remove(dst)

	srcPath, err := syscall.BytePtrFromString(src)
	if err != nil {
		return err
	}
	dstPath, err := syscall.BytePtrFromString(dst)
	if err != nil {
		return err
	}

	// clonefile(src, dst, 0) - syscall 462 on macOS
	_, _, errno := syscall.Syscall(
		462, // SYS_clonefile
		uintptr(unsafe.Pointer(srcPath)),
		uintptr(unsafe.Pointer(dstPath)),
		0,
	)

	_ = srcFd
	_ = dstFd

	if errno != 0 {
		return errno
	}
	return nil
}
