//go:build !windows && !linux && !darwin

package benchmark

import (
	"fmt"
	"os"
	"runtime"
)

func directIOAlignment(string) (int, error) {
	return 0, fmt.Errorf("direct I/O is not supported on %s", runtime.GOOS)
}

func openDirectFile(string) (*os.File, error) {
	return nil, fmt.Errorf("direct I/O is not supported on %s", runtime.GOOS)
}
