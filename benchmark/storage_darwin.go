//go:build darwin

package benchmark

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func availableSpace(directory string) (int64, error) {
	var statistics unix.Statfs_t
	if err := unix.Statfs(directory, &statistics); err != nil {
		return 0, err
	}
	if statistics.Bsize <= 0 {
		return 0, fmt.Errorf("filesystem reported invalid block size %d", statistics.Bsize)
	}
	return saturatingProduct(uint64(statistics.Bavail), uint64(statistics.Bsize)), nil
}

func allocatedFileBytes(file *os.File) (int64, error) {
	var statistics unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &statistics); err != nil {
		return 0, err
	}
	if statistics.Blocks < 0 {
		return 0, fmt.Errorf("filesystem reported invalid allocated block count %d", statistics.Blocks)
	}
	return saturatingProduct(uint64(statistics.Blocks), 512), nil
}
