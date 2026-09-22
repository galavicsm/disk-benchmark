//go:build darwin

package benchmark

import (
	"os"

	"golang.org/x/sys/unix"
)

func platformCacheControl(file *os.File) (string, bool, error) {
	const api = "fcntl_f_nocache"
	if err := file.Sync(); err != nil {
		return api, true, err
	}
	if _, err := unix.FcntlInt(file.Fd(), unix.F_NOCACHE, 1); err != nil {
		return api, true, err
	}
	return api, true, nil
}
