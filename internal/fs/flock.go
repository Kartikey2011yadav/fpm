//go:build !windows

package fs

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func LockFile(path string) (*os.File, error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	const timeout = 30 * time.Second
	const retryInterval = 100 * time.Millisecond

	deadline := time.Now().Add(timeout)
	firstAttempt := true

	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return f, nil
		}

		if time.Now().After(deadline) {
			f.Close()
			return nil, fmt.Errorf("timed out waiting for lock on %s after %s", lockPath, timeout)
		}

		if firstAttempt {
			fmt.Printf("Waiting for lock on %s...\n", lockPath)
			firstAttempt = false
		}

		time.Sleep(retryInterval)
	}
}

func LockFileShared(path string) (*os.File, error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func UnlockFile(f *os.File) error {
	if f == nil {
		return nil
	}
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return f.Close()
}
