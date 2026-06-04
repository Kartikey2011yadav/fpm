package fs

import (
	"os"
	"syscall"
	"unsafe"
)

const FICLONE = 0x40049409

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

	// Try FICLONE ioctl for CoW copy
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		dstFile.Fd(),
		FICLONE,
		uintptr(unsafe.Pointer(&srcFile)),
	)

	if errno != 0 {
		return errno
	}
	return nil
}
