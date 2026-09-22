//go:build windows

package benchmark

import "os"

func platformCacheControl(*os.File) (string, bool, error) {
	return "unavailable", false, nil
}
