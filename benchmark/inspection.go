package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
)

type DirectoryInspection struct {
	ResolvedPath      string
	Filesystem        string
	Volume            string
	AvailableBytes    int64
	DirectIOSupported bool
	DirectIOAlignment int
	DirectIODetail    string
}

func InspectDirectory(directory string) (DirectoryInspection, error) {
	if directory == "" {
		return DirectoryInspection{}, fmt.Errorf("benchmark directory must not be empty")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return DirectoryInspection{}, fmt.Errorf("resolve benchmark directory: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return DirectoryInspection{}, fmt.Errorf("resolve benchmark directory: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return DirectoryInspection{}, fmt.Errorf("inspect benchmark directory: %w", err)
	}
	if !info.IsDir() {
		return DirectoryInspection{}, fmt.Errorf("benchmark path %q is not a directory", resolved)
	}
	filesystem, volume, err := filesystemDetails(resolved)
	if err != nil {
		return DirectoryInspection{}, fmt.Errorf("inspect filesystem: %w", err)
	}
	available, err := availableSpace(resolved)
	if err != nil {
		return DirectoryInspection{}, fmt.Errorf("inspect available disk space: %w", err)
	}

	inspection := DirectoryInspection{
		ResolvedPath:   resolved,
		Filesystem:     filesystem,
		Volume:         volume,
		AvailableBytes: available,
	}
	alignment, directErr := directIOAlignment(resolved)
	if directErr != nil {
		inspection.DirectIODetail = directErr.Error()
		return inspection, nil
	}
	if alignment <= 0 {
		inspection.DirectIODetail = fmt.Sprintf("direct I/O returned invalid alignment %d", alignment)
		return inspection, nil
	}
	inspection.DirectIOSupported = true
	inspection.DirectIOAlignment = alignment
	return inspection, nil
}
