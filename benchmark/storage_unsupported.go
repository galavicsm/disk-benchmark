//go:build !windows && !linux && !darwin

package benchmark

import (
	"fmt"
	"os"
	"runtime"
)

func availableSpace(string) (int64, error) {
	return 0, fmt.Errorf("available-space checks are not supported on %s", runtime.GOOS)
}

func allocatedFileBytes(*os.File) (int64, error) {
	return 0, fmt.Errorf("file allocation checks are not supported on %s", runtime.GOOS)
}
