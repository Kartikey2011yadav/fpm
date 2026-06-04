//go:build !darwin && !linux

package fs

import "fmt"

func reflink(src, dst string) error {
	return fmt.Errorf("reflink not supported on this platform")
}
