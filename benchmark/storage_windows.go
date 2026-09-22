//go:build windows

package benchmark

import (
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

type fileStandardInfo struct {
	AllocationSize int64
	EndOfFile      int64
	NumberOfLinks  uint32
	DeletePending  byte
	Directory      byte
	padding        [2]byte
}

func availableSpace(directory string) (int64, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return 0, err
	}
	path, err := windows.UTF16PtrFromString(absolute)
	if err != nil {
		return 0, err
	}
	var available uint64
	if err := windows.GetDiskFreeSpaceEx(path, &available, nil, nil); err != nil {
		return 0, err
	}
	if available > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1), nil
	}
	return int64(available), nil
}

func allocatedFileBytes(file *os.File) (int64, error) {
	var information fileStandardInfo
	if err := windows.GetFileInformationByHandleEx(
		windows.Handle(file.Fd()),
		windows.FileStandardInfo,
		(*byte)(unsafe.Pointer(&information)),
		uint32(unsafe.Sizeof(information)),
	); err != nil {
		return 0, err
	}
	return information.AllocationSize, nil
}
