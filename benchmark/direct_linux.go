//go:build linux

package benchmark

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func directIOAlignment(directory string) (int, error) {
	var statistics unix.Statfs_t
	if err := unix.Statfs(directory, &statistics); err != nil {
		return 0, err
	}
	if statistics.Bsize <= 0 {
		return 0, fmt.Errorf("filesystem reported invalid block size %d", statistics.Bsize)
	}
	return int(statistics.Bsize), nil
}

func openDirectFile(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDWR|unix.O_DIRECT, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}
