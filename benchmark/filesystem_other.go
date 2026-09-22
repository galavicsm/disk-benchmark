//go:build !windows

package benchmark

import "path/filepath"

func filesystemDetails(directory string) (string, string, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", "", err
	}
	return "unknown", filepath.VolumeName(absolute), nil
}
