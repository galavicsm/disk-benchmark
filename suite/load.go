package suite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"diskbenchmark/benchmark"
)

const CurrentVersion = 1

type File struct {
	Version   int            `json:"version"`
	Workloads []WorkloadSpec `json:"workloads"`
}

type WorkloadSpec struct {
	Type              string `json:"type"`
	Name              string `json:"name,omitempty"`
	Duration          string `json:"duration,omitempty"`
	RandomReadPercent *int   `json:"random_read_percent,omitempty"`
	RandomSeed        *int64 `json:"random_seed,omitempty"`
}

func Load(path string) ([]benchmark.Workload, error) {
	suite, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return suite.WorkloadList()
}

func LoadFile(path string) (File, error) {
	suite, err := readFile(path)
	if err != nil {
		return File{}, err
	}
	if _, err := suite.WorkloadList(); err != nil {
		return File{}, err
	}
	return suite, nil
}

func readFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read workload suite: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var suite File
	if err := decoder.Decode(&suite); err != nil {
		return File{}, fmt.Errorf("decode workload suite: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return File{}, fmt.Errorf("decode workload suite: multiple JSON values are not allowed")
		}
		return File{}, fmt.Errorf("decode workload suite: %w", err)
	}
	return suite, nil
}

func (s File) WorkloadList() ([]benchmark.Workload, error) {
	if s.Version != CurrentVersion {
		return nil, fmt.Errorf("unsupported workload suite version %d; expected %d", s.Version, CurrentVersion)
	}
	if len(s.Workloads) == 0 {
		return nil, fmt.Errorf("workload suite must contain at least one workload")
	}

	var workloads []benchmark.Workload
	for index, spec := range s.Workloads {
		configured, err := spec.workloads()
		if err != nil {
			return nil, fmt.Errorf("workload %d: %w", index+1, err)
		}
		workloads = append(workloads, configured...)
	}

	names := make(map[string]bool, len(workloads))
	for _, workload := range workloads {
		if names[workload.Name()] {
			return nil, fmt.Errorf(
				"duplicate workload name %q; set a unique name in the suite",
				workload.Name(),
			)
		}
		names[workload.Name()] = true
	}
	return workloads, nil
}

func (s WorkloadSpec) workloads() ([]benchmark.Workload, error) {
	duration, err := s.parseDuration()
	if err != nil {
		return nil, err
	}
	if s.Type != "random" && (s.RandomReadPercent != nil || s.RandomSeed != nil) {
		return nil, fmt.Errorf("random_read_percent and random_seed are valid only for random workloads")
	}
	if s.RandomReadPercent != nil && (*s.RandomReadPercent < 0 || *s.RandomReadPercent > 100) {
		return nil, fmt.Errorf("random_read_percent must be between 0 and 100")
	}

	switch s.Type {
	case "sequential":
		writeLabel, readLabel := "", ""
		if s.Name != "" {
			writeLabel = s.Name + " write"
			readLabel = s.Name + " read"
		}
		return []benchmark.Workload{
			s.configure(benchmark.SequentialWrite{}, duration, writeLabel),
			s.configure(benchmark.SequentialRead{}, duration, readLabel),
		}, nil
	case "sequential-write":
		return []benchmark.Workload{s.configure(benchmark.SequentialWrite{}, duration, s.Name)}, nil
	case "sequential-read":
		return []benchmark.Workload{s.configure(benchmark.SequentialRead{}, duration, s.Name)}, nil
	case "random":
		workload := benchmark.RandomMixed{
			ReadPercent: s.RandomReadPercent,
			Seed:        s.RandomSeed,
		}
		return []benchmark.Workload{s.configure(workload, duration, s.Name)}, nil
	case "overwrite":
		return []benchmark.Workload{s.configure(benchmark.SequentialOverwrite{}, duration, s.Name)}, nil
	case "fsync-write":
		return []benchmark.Workload{s.configure(benchmark.FsyncWrite{}, duration, s.Name)}, nil
	default:
		return nil, fmt.Errorf(
			"unsupported type %q; use sequential, sequential-write, sequential-read, random, overwrite, or fsync-write",
			s.Type,
		)
	}
}

func (s WorkloadSpec) parseDuration() (*time.Duration, error) {
	if s.Duration == "" {
		return nil, nil
	}
	duration, err := time.ParseDuration(s.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q: %w", s.Duration, err)
	}
	if duration < 0 {
		return nil, fmt.Errorf("duration must not be negative")
	}
	return &duration, nil
}

func (WorkloadSpec) configure(
	workload benchmark.Workload,
	duration *time.Duration,
	label string,
) benchmark.Workload {
	if duration == nil && label == "" {
		return workload
	}
	return benchmark.ConfiguredWorkload{
		Workload:         workload,
		DurationOverride: duration,
		Label:            label,
	}
}
