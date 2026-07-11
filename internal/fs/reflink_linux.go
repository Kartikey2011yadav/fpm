package fs

import (
	"os"
	"syscall"
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

	// FICLONE ioctl for CoW copy — arg is the source fd
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		dstFile.Fd(),
		FICLONE,
		srcFile.Fd(),
	)

	if errno != 0 {
		return errno
	}
	return nil
}
