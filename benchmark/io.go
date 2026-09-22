package benchmark

import (
	"errors"
	"fmt"
	"os"
	"unsafe"
)

func configureIOMode(config Config) (Config, error) {
	if config.IOMode == BufferedIO {
		config.IOAlignment = 1
		return config, nil
	}

	alignment, err := directIOAlignment(config.Directory)
	if err != nil {
		return Config{}, fmt.Errorf("determine direct I/O alignment: %w", err)
	}
	if alignment <= 0 {
		return Config{}, fmt.Errorf("direct I/O returned invalid alignment %d", alignment)
	}
	if config.BlockSize%alignment != 0 {
		return Config{}, fmt.Errorf(
			"direct I/O block size %d must be a multiple of %d bytes",
			config.BlockSize,
			alignment,
		)
	}
	if config.Size%int64(alignment) != 0 {
		return Config{}, fmt.Errorf(
			"direct I/O data size %d must be a multiple of %d bytes",
			config.Size,
			alignment,
		)
	}
	config.IOAlignment = alignment
	return config, nil
}

func createBenchmarkFile(config Config) (*os.File, string, error) {
	file, err := os.CreateTemp(config.Directory, "diskbenchmark-*.tmp")
	if err != nil {
		return nil, "", fmt.Errorf("create test file: %w", err)
	}
	path := file.Name()
	if config.IOMode == BufferedIO {
		return file, path, nil
	}

	if err := file.Close(); err != nil {
		removeErr := os.Remove(path)
		return nil, "", errors.Join(
			fmt.Errorf("close test file before enabling direct I/O: %w", err),
			wrapRemoveError(path, removeErr),
		)
	}
	directFile, err := openDirectFile(path)
	if err != nil {
		removeErr := os.Remove(path)
		return nil, "", errors.Join(
			fmt.Errorf("open test file for direct I/O: %w", err),
			wrapRemoveError(path, removeErr),
		)
	}
	return directFile, path, nil
}

func wrapRemoveError(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("remove test file %q: %w", path, err)
}

func makeIOBuffer(size, alignment int) []byte {
	if alignment <= 1 {
		return make([]byte, size)
	}
	storage := make([]byte, size+alignment-1)
	address := uintptr(unsafe.Pointer(&storage[0]))
	padding := int((uintptr(alignment) - address%uintptr(alignment)) % uintptr(alignment))
	return storage[padding : padding+size]
}

func cloneIOBuffer(source []byte, alignment int) []byte {
	clone := makeIOBuffer(len(source), alignment)
	copy(clone, source)
	return clone
}
