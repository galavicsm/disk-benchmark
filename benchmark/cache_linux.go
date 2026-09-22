//go:build linux

package benchmark

import (
	"os"

	"golang.org/x/sys/unix"
)

func platformCacheControl(file *os.File) (string, bool, error) {
	const api = "posix_fadvise_dontneed"
	if err := file.Sync(); err != nil {
		return api, true, err
	}
	if err := unix.Fadvise(int(file.Fd()), 0, 0, unix.FADV_DONTNEED); err != nil {
		return api, true, err
	}
	return api, true, nil
}
