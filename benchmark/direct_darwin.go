//go:build darwin

package benchmark

import (
	"os"

	"golang.org/x/sys/unix"
)

func directIOAlignment(string) (int, error) {
	// F_NOCACHE bypasses the unified buffer cache but does not impose the
	// sector-alignment restrictions of Windows or Linux direct I/O.
	return 1, nil
}

func openDirectFile(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	if _, err := unix.FcntlInt(uintptr(descriptor), unix.F_NOCACHE, 1); err != nil {
		_ = unix.Close(descriptor)
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}
