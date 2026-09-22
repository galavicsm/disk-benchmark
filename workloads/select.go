package workloads

import (
	"fmt"
	"strings"

	"diskbenchmark/benchmark"
	"diskbenchmark/suite"
)

const Default = "sequential"

type Selection struct {
	BuiltIn   string
	SuiteFile string
}

type Result struct {
	Workloads    []benchmark.Workload
	SuiteVersion int
}

func Select(selection Selection) (Result, error) {
	if selection.BuiltIn != "" && selection.SuiteFile != "" {
		return Result{}, fmt.Errorf("built-in workload and workload suite cannot be used together")
	}
	if selection.SuiteFile != "" {
		selected, err := suite.Load(selection.SuiteFile)
		if err != nil {
			return Result{}, err
		}
		return Result{Workloads: selected, SuiteVersion: suite.CurrentVersion}, nil
	}

	name := selection.BuiltIn
	if name == "" {
		name = Default
	}
	selected, err := BuiltIn(name)
	if err != nil {
		return Result{}, err
	}
	return Result{Workloads: selected}, nil
}

func BuiltIn(value string) ([]benchmark.Workload, error) {
	switch strings.ToLower(value) {
	case "sequential":
		return []benchmark.Workload{benchmark.SequentialWrite{}, benchmark.SequentialRead{}}, nil
	case "random", "mixed":
		return []benchmark.Workload{benchmark.RandomMixed{}}, nil
	case "overwrite":
		return []benchmark.Workload{benchmark.SequentialOverwrite{}}, nil
	case "fsync":
		return []benchmark.Workload{benchmark.FsyncWrite{}}, nil
	case "all":
		return []benchmark.Workload{
			benchmark.SequentialWrite{},
			benchmark.SequentialRead{},
			benchmark.RandomMixed{},
			benchmark.SequentialOverwrite{},
			benchmark.FsyncWrite{},
		}, nil
	default:
		return nil, fmt.Errorf(
			"unsupported workload %q; use sequential, random, mixed, overwrite, fsync, or all",
			value,
		)
	}
}
