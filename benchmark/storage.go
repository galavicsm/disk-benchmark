package benchmark

import (
	"fmt"
	"os"
)

func preflightSpace(config Config) (int64, error) {
	available, err := availableSpace(config.Directory)
	if err != nil {
		return 0, fmt.Errorf("inspect available disk space: %w", err)
	}
	if available < config.Size {
		return available, fmt.Errorf(
			"insufficient disk space in %q: need %d bytes, have %d bytes available",
			config.Directory,
			config.Size,
			available,
		)
	}
	return available, nil
}

func inspectFileStorage(file *os.File) (FileStorage, error) {
	info, err := file.Stat()
	if err != nil {
		return FileStorage{}, fmt.Errorf("inspect test file size: %w", err)
	}
	allocated, err := allocatedFileBytes(file)
	if err != nil {
		return FileStorage{}, fmt.Errorf("inspect allocated file space: %w", err)
	}
	return classifyFileStorage(info.Size(), allocated), nil
}

func classifyFileStorage(logical, allocated int64) FileStorage {
	return FileStorage{
		LogicalBytes:   logical,
		AllocatedBytes: allocated,
		Sparse:         logical > 0 && allocated < logical,
	}
}
