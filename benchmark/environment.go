package benchmark

import (
	"fmt"
	"os"
	"runtime"
)

func collectEnvironment(directory string) (Environment, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return Environment{}, fmt.Errorf("read hostname: %w", err)
	}
	filesystem, volume, err := filesystemDetails(directory)
	if err != nil {
		return Environment{}, fmt.Errorf("inspect filesystem: %w", err)
	}
	available, err := availableSpace(directory)
	if err != nil {
		return Environment{}, fmt.Errorf("inspect available disk space: %w", err)
	}
	return Environment{
		Hostname:             hostname,
		OperatingSystem:      runtime.GOOS,
		Architecture:         runtime.GOARCH,
		CPUs:                 runtime.NumCPU(),
		GoVersion:            runtime.Version(),
		Filesystem:           filesystem,
		Volume:               volume,
		AvailableBytesBefore: available,
	}, nil
}
