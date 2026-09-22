//go:build windows

package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var getDiskFreeSpace = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceW")

func directIOAlignment(directory string) (int, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return 0, err
	}
	volume := filepath.VolumeName(absolute)
	if volume == "" {
		return 0, fmt.Errorf("could not determine volume for %q", directory)
	}
	root, err := syscall.UTF16PtrFromString(volume + `\`)
	if err != nil {
		return 0, err
	}
	var sectorsPerCluster uint32
	var bytesPerSector uint32
	success, _, callErr := getDiskFreeSpace.Call(
		uintptr(unsafe.Pointer(root)),
		uintptr(unsafe.Pointer(&sectorsPerCluster)),
		uintptr(unsafe.Pointer(&bytesPerSector)),
		0,
		0,
	)
	if success == 0 {
		return 0, callErr
	}
	if bytesPerSector == 0 {
		return 0, fmt.Errorf("volume %s reported a zero-byte sector size", volume)
	}
	return int(bytesPerSector), nil
}

func openDirectFile(path string) (*os.File, error) {
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		pathPointer,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_TEMPORARY|
			windows.FILE_FLAG_NO_BUFFERING|
			windows.FILE_FLAG_WRITE_THROUGH|
			windows.FILE_FLAG_OVERLAPPED,
		0,
	)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), path), nil
}
