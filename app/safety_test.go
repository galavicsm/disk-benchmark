package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"diskbenchmark/benchmark"
)

func TestWriteImpactPolicy(t *testing.T) {
	duration := time.Second
	tests := []struct {
		name      string
		config    benchmark.Config
		workloads []benchmark.Workload
		want      string
	}{
		{
			name:      "below thresholds",
			config:    benchmark.Config{Size: LargeWorkingSetThreshold - 1, Iterations: 1},
			workloads: []benchmark.Workload{benchmark.SequentialWrite{}},
		},
		{
			name:      "large working set at threshold",
			config:    benchmark.Config{Size: LargeWorkingSetThreshold, Iterations: 1},
			workloads: []benchmark.Workload{benchmark.SequentialWrite{}},
			want:      "at least 10 GiB",
		},
		{
			name:      "global duration",
			config:    benchmark.Config{Size: 4096, Iterations: 1, Duration: time.Second},
			workloads: []benchmark.Workload{benchmark.SequentialWrite{}},
			want:      "Duration-based",
		},
		{
			name:   "suite duration override",
			config: benchmark.Config{Size: 4096, Iterations: 1},
			workloads: []benchmark.Workload{benchmark.ConfiguredWorkload{
				Workload:         benchmark.SequentialWrite{},
				DurationOverride: &duration,
			}},
			want: "Duration-based",
		},
		{
			name:      "multiple iterations",
			config:    benchmark.Config{Size: 4096, Iterations: 2},
			workloads: []benchmark.Workload{benchmark.SequentialWrite{}},
			want:      "2 measured iterations",
		},
		{
			name:      "fsync workload",
			config:    benchmark.Config{Size: 4096, Iterations: 1},
			workloads: []benchmark.Workload{benchmark.FsyncWrite{}},
			want:      "Fsync-per-block",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assessment := assessWriteImpact(test.config, test.workloads)
			if test.want == "" {
				if assessment.RequiresConfirmation || len(assessment.Reasons) != 0 {
					t.Fatalf("assessment = %+v, want no confirmation", assessment)
				}
				return
			}
			if !assessment.RequiresConfirmation || len(assessment.Reasons) != 1 ||
				!strings.Contains(assessment.Reasons[0], test.want) {
				t.Fatalf("assessment = %+v, want reason containing %q", assessment, test.want)
			}
		})
	}
}

func TestServiceRequiresWriteImpactConfirmation(t *testing.T) {
	service := NewService()
	request := service.GetDefaults()
	request.Directory = t.TempDir()
	request.Size = "4KiB"
	request.BlockSize = "1KiB"
	request.Workload = WorkloadFsync

	validation := service.ValidateBenchmark(request)
	if !validation.Valid || !validation.WriteImpact.RequiresConfirmation {
		t.Fatalf("ValidateBenchmark() = %+v, want confirmation requirement", validation)
	}
	result := service.StartBenchmark(request)
	if result.State != RunStateAwaitingConfirmation ||
		result.Error == nil ||
		result.Error.Code != ErrorCodeConfirmation {
		t.Fatalf("StartBenchmark() = %+v, want confirmation error", result)
	}

	request.WriteImpactConfirmed = true
	result = service.StartBenchmark(request)
	if result.State != RunStateCompleted || result.Error != nil || result.Report == nil {
		t.Fatalf("confirmed StartBenchmark() = %+v, want completed report", result)
	}
}

func TestServiceValidationBlocksInsufficientSpace(t *testing.T) {
	request := DefaultBenchmarkRequest()
	request.Directory = t.TempDir()
	request.Size = "100TiB"
	request.BlockSize = "1MiB"

	validation := NewService().ValidateBenchmark(request)
	if validation.Valid || len(validation.Errors) != 1 ||
		validation.Errors[0].Field != "size" ||
		!strings.Contains(validation.Errors[0].Message, "insufficient disk space") {
		t.Fatalf("ValidateBenchmark() = %+v, want insufficient-space size error", validation)
	}
}

func TestSelectSuiteFileReturnsOrderedSummary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "suite.json")
	content := `{
		"version": 1,
		"workloads": [
			{"type":"sequential","duration":"2s"},
			{"type":"random","name":"database mix","random_read_percent":70,"random_seed":42}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	service := NewService(WithDialogProvider(DialogProvider{
		SelectSuiteFile: func() (string, error) { return path, nil },
	}))

	result := service.SelectSuiteFile()
	if result.Error != nil || result.Suite == nil || len(result.Suite.Workloads) != 2 {
		t.Fatalf("SelectSuiteFile() = %+v", result)
	}
	first := result.Suite.Workloads[0]
	second := result.Suite.Workloads[1]
	if first.Name != "Sequential" || first.Duration != "2s" ||
		second.Name != "database mix" ||
		second.RandomReadPercent == nil || *second.RandomReadPercent != 70 ||
		second.RandomSeed != "42" {
		t.Fatalf("suite summary has incorrect order or overrides: %+v", result.Suite)
	}
}
