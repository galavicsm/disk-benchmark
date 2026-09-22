package app

import (
	"strconv"

	"diskbenchmark/benchmark"
)

type DirectIOCapability string

const (
	DirectIOAvailable   DirectIOCapability = "available"
	DirectIOUnsupported DirectIOCapability = "unsupported"
)

type DirectoryInspection struct {
	ResolvedPath   string             `json:"resolvedPath"`
	Filesystem     string             `json:"filesystem"`
	Volume         string             `json:"volume"`
	AvailableBytes string             `json:"availableBytes"`
	DirectIO       DirectIOInspection `json:"directIo"`
}

type DirectIOInspection struct {
	Capability     DirectIOCapability `json:"capability"`
	AlignmentBytes string             `json:"alignmentBytes,omitempty"`
	Detail         string             `json:"detail,omitempty"`
}

func newDirectoryInspection(source benchmark.DirectoryInspection) DirectoryInspection {
	directIO := DirectIOInspection{
		Capability: DirectIOUnsupported,
		Detail:     source.DirectIODetail,
	}
	if source.DirectIOSupported {
		directIO.Capability = DirectIOAvailable
		directIO.AlignmentBytes = strconv.Itoa(source.DirectIOAlignment)
	}
	return DirectoryInspection{
		ResolvedPath:   source.ResolvedPath,
		Filesystem:     source.Filesystem,
		Volume:         source.Volume,
		AvailableBytes: formatInt64(source.AvailableBytes),
		DirectIO:       directIO,
	}
}
