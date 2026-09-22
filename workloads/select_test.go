package workloads

import (
	"os"
	"path/filepath"
	"testing"

	"diskbenchmark/benchmark"
	"diskbenchmark/suite"
)

func TestBuiltIn(t *testing.T) {
	tests := []struct {
		value string
		count int
	}{
		{value: "sequential", count: 2},
		{value: "random", count: 1},
		{value: "mixed", count: 1},
		{value: "overwrite", count: 1},
		{value: "fsync", count: 1},
		{value: "all", count: 5},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			selected, err := BuiltIn(test.value)
			if err != nil {
				t.Fatalf("BuiltIn() returned error: %v", err)
			}
			if len(selected) != test.count {
				t.Fatalf("BuiltIn() returned %d workloads, want %d", len(selected), test.count)
			}
		})
	}

	if selected, err := BuiltIn("invalid"); err == nil || selected != nil {
		t.Fatalf("BuiltIn(invalid) = %v, %v; want nil and error", selected, err)
	}
}

func TestBuiltInAllPreservesOrder(t *testing.T) {
	selected, err := BuiltIn("all")
	if err != nil {
		t.Fatal(err)
	}
	expected := []any{
		benchmark.SequentialWrite{},
		benchmark.SequentialRead{},
		benchmark.RandomMixed{},
		benchmark.SequentialOverwrite{},
		benchmark.FsyncWrite{},
	}
	for index := range expected {
		if selected[index].Name() != expected[index].(benchmark.Workload).Name() {
			t.Fatalf("workload %d is %q, want %q", index, selected[index].Name(), expected[index].(benchmark.Workload).Name())
		}
	}
}

func TestSelectLoadsSuite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "suite.json")
	content := []byte(`{"version":1,"workloads":[{"type":"overwrite"}]}`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	result, err := Select(Selection{SuiteFile: path})
	if err != nil {
		t.Fatalf("Select() returned error: %v", err)
	}
	if result.SuiteVersion != suite.CurrentVersion || len(result.Workloads) != 1 {
		t.Fatalf("Select() = %+v, want one version %d suite workload", result, suite.CurrentVersion)
	}
}

func TestSelectRejectsConflictingSources(t *testing.T) {
	result, err := Select(Selection{BuiltIn: "all", SuiteFile: "suite.json"})
	if err == nil || result.Workloads != nil {
		t.Fatalf("Select() = %+v, %v; want empty result and error", result, err)
	}
}
