package benchmark

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerRemovesTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	config := Config{
		Directory:  directory,
		Size:       8193,
		BlockSize:  1024,
		Iterations: 2,
		Sync:       true,
	}

	report, err := NewRunner().Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(report.Iterations) != config.Iterations {
		t.Fatalf("got %d iterations, want %d", len(report.Iterations), config.Iterations)
	}
	for _, iteration := range report.Iterations {
		if iteration.FilePath != "" {
			t.Fatalf("temporary file path exposed when keep-file is false: %q", iteration.FilePath)
		}
		for _, measurement := range iteration.Measurements {
			if measurement.Bytes != config.Size {
				t.Fatalf("%s processed %d bytes, want %d", measurement.Name, measurement.Bytes, config.Size)
			}
		}
	}

	matches, err := filepath.Glob(filepath.Join(directory, "diskbenchmark-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files were not removed: %v", matches)
	}
}

func TestRunnerRetainsRequestedFile(t *testing.T) {
	directory := t.TempDir()
	config := Config{
		Directory:  directory,
		Size:       4097,
		BlockSize:  1024,
		Iterations: 1,
		KeepFile:   true,
	}

	report, err := NewRunner().Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	path := report.Iterations[0].FilePath
	if path == "" {
		t.Fatal("retained file path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read retained file: %v", err)
	}
	if int64(len(data)) != config.Size {
		t.Fatalf("retained file size = %d, want %d", len(data), config.Size)
	}
	allZero := true
	for _, value := range data {
		if value != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("retained file contains only zero bytes")
	}
}

func TestSummarize(t *testing.T) {
	iterations := []IterationResult{
		{Measurements: []Measurement{{Name: "write", Bytes: 10 * 1024 * 1024, Duration: 2 * time.Second}}},
		{Measurements: []Measurement{{Name: "write", Bytes: 30 * 1024 * 1024, Duration: 2 * time.Second}}},
	}

	summaries := summarize(iterations)
	if len(summaries) != 1 {
		t.Fatalf("got %d summaries, want 1", len(summaries))
	}
	got := summaries[0]
	if got.MinimumMiBPerSec != 5 || got.AverageMiBPerSec != 10 || got.MaximumMiBPerSec != 15 {
		t.Fatalf("summary = min %.2f avg %.2f max %.2f, want 5, 10, 15",
			got.MinimumMiBPerSec, got.AverageMiBPerSec, got.MaximumMiBPerSec)
	}
}

type countingWorkload struct {
	count *atomic.Int64
}

func (countingWorkload) Name() string {
	return "counting"
}

func (w countingWorkload) Run(
	context.Context,
	*os.File,
	[]byte,
	Config,
) (Measurement, error) {
	w.count.Add(1)
	return Measurement{Name: w.Name(), Duration: time.Nanosecond}, nil
}

func TestRunnerExecutesUnmeasuredWarmups(t *testing.T) {
	var count atomic.Int64
	config := Config{
		Directory:         t.TempDir(),
		Size:              4096,
		BlockSize:         1024,
		Iterations:        3,
		WarmupIterations:  2,
		Workers:           1,
		QueueDepth:        1,
		RandomReadPercent: 50,
	}
	report, err := NewRunner(countingWorkload{count: &count}).Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if count.Load() != 5 {
		t.Fatalf("workload ran %d times, want 5", count.Load())
	}
	if report.WarmupsCompleted != 2 || len(report.Iterations) != 3 {
		t.Fatalf("report has %d warmups and %d measured iterations, want 2 and 3",
			report.WarmupsCompleted, len(report.Iterations))
	}
}

func TestRunnerVerifiesDataAndReportsStorage(t *testing.T) {
	config := Config{
		Directory:         t.TempDir(),
		Size:              8192,
		BlockSize:         1024,
		Iterations:        1,
		VerifyData:        true,
		Workers:           2,
		QueueDepth:        2,
		RandomReadPercent: 50,
	}
	report, err := NewRunner().Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	iteration := report.Iterations[0]
	if !iteration.Verification.Passed || iteration.Verification.Bytes != config.Size {
		t.Fatalf("verification result = %+v", iteration.Verification)
	}
	if iteration.Storage.LogicalBytes != config.Size ||
		iteration.Storage.AllocatedBytes < config.Size ||
		iteration.Storage.Sparse {
		t.Fatalf("storage result = %+v", iteration.Storage)
	}
	if report.Environment.AvailableBytesBefore <= 0 || report.Environment.AvailableBytesAfter <= 0 {
		t.Fatalf("available-space metadata is missing: %+v", report.Environment)
	}
}

func TestRunnerObserverReportsLifecycleInOrder(t *testing.T) {
	var events []ProgressEvent
	config := Config{
		Directory:         t.TempDir(),
		Size:              4096,
		BlockSize:         1024,
		Iterations:        1,
		WarmupIterations:  1,
		VerifyData:        true,
		Workers:           1,
		QueueDepth:        1,
		RandomReadPercent: 50,
	}
	runner := NewRunner(SequentialRead{}).WithObserver(ObserverFunc(func(event ProgressEvent) {
		events = append(events, event)
	}))
	if _, err := runner.Run(context.Background(), config); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	var sequence []string
	for _, event := range events {
		current := fmt.Sprintf("%s:%s", event.Phase, event.Status)
		if event.Warmup > 0 {
			current += fmt.Sprintf(":warmup-%d", event.Warmup)
		}
		if event.Iteration > 0 {
			current += fmt.Sprintf(":iteration-%d", event.Iteration)
		}
		if event.Workload != "" {
			current += ":" + event.Workload
		}
		sequence = append(sequence, current)
		if event.WarmupsTotal != 1 || event.IterationsTotal != 1 || event.WorkloadsTotal != 1 {
			t.Fatalf("event totals are incomplete: %+v", event)
		}
	}
	expected := []string{
		"preflight:started",
		"preflight:completed",
		"warmup:started:warmup-1",
		"workload-preparation:started:warmup-1:Sequential read",
		"workload-preparation:completed:warmup-1:Sequential read",
		"workload:started:warmup-1:Sequential read",
		"workload:completed:warmup-1:Sequential read",
		"verification:started:warmup-1",
		"verification:completed:warmup-1",
		"cleanup:started:warmup-1",
		"cleanup:completed:warmup-1",
		"warmup:completed:warmup-1",
		"iteration:started:iteration-1",
		"workload-preparation:started:iteration-1:Sequential read",
		"workload-preparation:completed:iteration-1:Sequential read",
		"workload:started:iteration-1:Sequential read",
		"workload:completed:iteration-1:Sequential read",
		"verification:started:iteration-1",
		"verification:completed:iteration-1",
		"cleanup:started:iteration-1",
		"cleanup:completed:iteration-1",
		"iteration:completed:iteration-1",
	}
	if strings.Join(sequence, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("event sequence:\n%s\nwant:\n%s", strings.Join(sequence, "\n"), strings.Join(expected, "\n"))
	}
}

func TestRunnerMarksDurationWorkloadProgressIndeterminate(t *testing.T) {
	duration := time.Millisecond
	var workloadEvents []ProgressEvent
	runner := NewRunner(ConfiguredWorkload{
		Workload:         SequentialWrite{},
		DurationOverride: &duration,
	}).WithObserver(ObserverFunc(func(event ProgressEvent) {
		if event.Phase == PhaseWorkload {
			workloadEvents = append(workloadEvents, event)
		}
	}))
	config := Config{
		Directory:  t.TempDir(),
		Size:       4096,
		BlockSize:  1024,
		Iterations: 1,
	}
	if _, err := runner.Run(context.Background(), config); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(workloadEvents) != 2 {
		t.Fatalf("got %d workload events, want 2", len(workloadEvents))
	}
	for _, event := range workloadEvents {
		if !event.Indeterminate {
			t.Fatalf("duration workload event is not indeterminate: %+v", event)
		}
	}
}

func TestRunnerCancellationRemovesIncompleteKeptFile(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
	}{
		{name: "preparation", phase: PhaseWorkloadPreparation},
		{name: "workload", phase: PhaseWorkload},
		{name: "verification", phase: PhaseVerification},
		{name: "cleanup", phase: PhaseCleanup},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			runner := NewRunner(SequentialRead{}).WithObserver(ObserverFunc(func(event ProgressEvent) {
				if event.Phase == test.phase && event.Status == PhaseStarted {
					cancel()
				}
			}))
			config := Config{
				Directory:  directory,
				Size:       4096,
				BlockSize:  1024,
				Iterations: 1,
				KeepFile:   true,
				VerifyData: true,
			}
			if _, err := runner.Run(ctx, config); !errors.Is(err, context.Canceled) {
				t.Fatalf("Run() error = %v, want context.Canceled", err)
			}
			matches, err := filepath.Glob(filepath.Join(directory, "diskbenchmark-*.tmp"))
			if err != nil {
				t.Fatal(err)
			}
			if len(matches) != 0 {
				t.Fatalf("cancelled run retained temporary files: %v", matches)
			}
		})
	}
}

func TestRunnerWithoutObserverRetainsExistingBehavior(t *testing.T) {
	config := Config{
		Directory:  t.TempDir(),
		Size:       4096,
		BlockSize:  1024,
		Iterations: 1,
	}
	report, err := NewRunner(SequentialWrite{}).Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(report.Iterations) != 1 || len(report.Iterations[0].Measurements) != 1 {
		t.Fatalf("Run() returned unexpected report without observer: %+v", report)
	}
}
