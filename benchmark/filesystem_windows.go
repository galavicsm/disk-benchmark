//go:build windows

package benchmark

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

var getVolumeInformation = syscall.NewLazyDLL("kernel32.dll").NewProc("GetVolumeInformationW")

func filesystemDetails(directory string) (string, string, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", "", err
	}
	volume := filepath.VolumeName(absolute)
	if volume == "" {
		return "", "", fmt.Errorf("could not determine volume for %q", directory)
	}
	root := volume + `\`
	rootPointer, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return "", "", err
	}
	filesystemBuffer := make([]uint16, 256)
	success, _, callErr := getVolumeInformation.Call(
		uintptr(unsafe.Pointer(rootPointer)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&filesystemBuffer[0])),
		uintptr(len(filesystemBuffer)),
	)
	if success == 0 {
		return "", "", callErr
	}
	return syscall.UTF16ToString(filesystemBuffer), root, nil
}
