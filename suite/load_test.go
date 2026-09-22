package suite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"diskbenchmark/benchmark"
)

func TestWorkloadListBuildsConfiguredSuite(t *testing.T) {
	readPercent := 70
	seed := int64(42)
	file := File{
		Version: CurrentVersion,
		Workloads: []WorkloadSpec{
			{Type: "sequential", Duration: "5ms"},
			{
				Type:              "random",
				Name:              "database mix",
				RandomReadPercent: &readPercent,
				RandomSeed:        &seed,
			},
			{Type: "overwrite"},
			{Type: "fsync-write"},
		},
	}
	workloads, err := file.WorkloadList()
	if err != nil {
		t.Fatalf("WorkloadList() returned error: %v", err)
	}
	if len(workloads) != 5 {
		t.Fatalf("got %d workloads, want 5", len(workloads))
	}
	if workloads[0].Name() != "Sequential write (5ms)" ||
		workloads[1].Name() != "Sequential read (5ms)" ||
		workloads[2].Name() != "database mix" {
		t.Fatalf("unexpected workload names: %q, %q, %q",
			workloads[0].Name(), workloads[1].Name(), workloads[2].Name())
	}
	configured, ok := workloads[0].(benchmark.ConfiguredWorkload)
	if !ok || configured.DurationOverride == nil || *configured.DurationOverride != 5*time.Millisecond {
		t.Fatalf("duration override was not retained: %#v", workloads[0])
	}
}

func TestLoadRejectsInvalidSuites(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "unknown field",
			content: `{"version":1,"unknown":true,"workloads":[{"type":"overwrite"}]}`,
			want:    "unknown field",
		},
		{
			name:    "unsupported version",
			content: `{"version":2,"workloads":[{"type":"overwrite"}]}`,
			want:    "unsupported workload suite version",
		},
		{
			name:    "empty",
			content: `{"version":1,"workloads":[]}`,
			want:    "at least one workload",
		},
		{
			name:    "invalid type",
			content: `{"version":1,"workloads":[{"type":"unknown"}]}`,
			want:    "unsupported type",
		},
		{
			name:    "invalid duration",
			content: `{"version":1,"workloads":[{"type":"overwrite","duration":"later"}]}`,
			want:    "invalid duration",
		},
		{
			name:    "random option on overwrite",
			content: `{"version":1,"workloads":[{"type":"overwrite","random_read_percent":50}]}`,
			want:    "valid only for random",
		},
		{
			name:    "duplicate names",
			content: `{"version":1,"workloads":[{"type":"overwrite"},{"type":"overwrite"}]}`,
			want:    "duplicate workload name",
		},
		{
			name:    "trailing value",
			content: `{"version":1,"workloads":[{"type":"overwrite"}]} {}`,
			want:    "multiple JSON values",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "suite.json")
			if err := os.WriteFile(path, []byte(test.content), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestSuiteRunsWithPerWorkloadDuration(t *testing.T) {
	file := File{
		Version: CurrentVersion,
		Workloads: []WorkloadSpec{
			{Type: "sequential-read", Duration: "2ms"},
			{Type: "overwrite"},
			{Type: "fsync-write"},
		},
	}
	workloads, err := file.WorkloadList()
	if err != nil {
		t.Fatal(err)
	}
	config := benchmark.Config{
		Directory:         t.TempDir(),
		Size:              4096,
		BlockSize:         1024,
		Iterations:        1,
		VerifyData:        true,
		Workers:           1,
		QueueDepth:        1,
		RandomReadPercent: 50,
	}
	report, err := benchmark.NewRunner(workloads...).Run(context.Background(), config)
	if err != nil {
		t.Fatalf("suite benchmark returned error: %v", err)
	}
	measurements := report.Iterations[0].Measurements
	if len(measurements) != 3 {
		t.Fatalf("got %d measurements, want 3", len(measurements))
	}
	if measurements[0].TargetDuration != 2*time.Millisecond ||
		measurements[0].Duration < 2*time.Millisecond {
		t.Fatalf("duration override was not executed: %+v", measurements[0])
	}
	if !report.Iterations[0].Verification.Passed {
		t.Fatalf("suite verification failed: %+v", report.Iterations[0].Verification)
	}
}

func TestExampleSuiteLoads(t *testing.T) {
	workloads, err := Load(filepath.Join("..", "examples", "workload-suite.json"))
	if err != nil {
		t.Fatalf("example suite is invalid: %v", err)
	}
	if len(workloads) != 5 {
		t.Fatalf("example suite expands to %d workloads, want 5", len(workloads))
	}
}
