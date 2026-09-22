package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"diskbenchmark/benchmark"
)

func TestServiceRunsSmallBenchmark(t *testing.T) {
	service := NewService()
	request := service.GetDefaults()
	request.Directory = t.TempDir()
	request.Size = "16KiB"
	request.BlockSize = "4KiB"
	request.Workload = WorkloadSequential
	request.VerifyData = true

	validation := service.ValidateBenchmark(request)
	if !validation.Valid || len(validation.Errors) != 0 {
		t.Fatalf("ValidateBenchmark() = %+v, want valid", validation)
	}

	result := service.StartBenchmark(request)
	if result.Error != nil {
		t.Fatalf("StartBenchmark() returned error: %+v", result.Error)
	}
	if result.Report == nil {
		t.Fatal("StartBenchmark() returned no report")
	}
	if result.Report.Config.SizeBytes != "16384" ||
		result.Report.Config.BlockSizeBytes != "4096" {
		t.Fatalf("report config has unsafe or incorrect sizes: %+v", result.Report.Config)
	}
	if len(result.Report.Iterations) != 1 ||
		len(result.Report.Iterations[0].Measurements) != 2 {
		t.Fatalf("unexpected benchmark report: %+v", result.Report)
	}
	if !result.Report.Iterations[0].Verification.Passed {
		t.Fatalf("verification did not pass: %+v", result.Report.Iterations[0].Verification)
	}
}

func TestServiceReturnsStructuredValidationErrors(t *testing.T) {
	service := NewService()
	valid := service.GetDefaults()
	valid.Directory = t.TempDir()
	valid.Size = "16KiB"
	valid.BlockSize = "4KiB"

	tests := []struct {
		name      string
		mutate    func(*BenchmarkRequest)
		wantField string
		wantText  string
	}{
		{
			name:      "size",
			mutate:    func(request *BenchmarkRequest) { request.Size = "large" },
			wantField: "size",
			wantText:  "no numeric value",
		},
		{
			name:      "block size exceeds data size",
			mutate:    func(request *BenchmarkRequest) { request.BlockSize = "32KiB" },
			wantField: "blockSize",
			wantText:  "must not exceed data size",
		},
		{
			name:      "path",
			mutate:    func(request *BenchmarkRequest) { request.Directory = filepath.Join(t.TempDir(), "missing") },
			wantField: "directory",
			wantText:  "inspect benchmark directory",
		},
		{
			name:      "suite",
			mutate:    func(request *BenchmarkRequest) { request.Workload = ""; request.SuiteFile = "missing.json" },
			wantField: "suiteFile",
			wantText:  "read workload suite",
		},
		{
			name:      "workload",
			mutate:    func(request *BenchmarkRequest) { request.Workload = "unknown" },
			wantField: "workload",
			wantText:  "unsupported workload",
		},
		{
			name:      "I/O mode",
			mutate:    func(request *BenchmarkRequest) { request.IOMode = "automatic" },
			wantField: "ioMode",
			wantText:  "unsupported I/O mode",
		},
		{
			name:      "cache control",
			mutate:    func(request *BenchmarkRequest) { request.CacheControl = "automatic" },
			wantField: "cacheControl",
			wantText:  "unsupported cache control",
		},
		{
			name: "conflicting workload sources",
			mutate: func(request *BenchmarkRequest) {
				request.Workload = WorkloadAll
				request.SuiteFile = "suite.json"
			},
			wantField: "suiteFile",
			wantText:  "cannot be used together",
		},
		{
			name:      "negative duration",
			mutate:    func(request *BenchmarkRequest) { request.Duration = "-1s" },
			wantField: "duration",
			wantText:  "must not be negative",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			test.mutate(&request)

			validation := service.ValidateBenchmark(request)
			if validation.Valid || len(validation.Errors) != 1 {
				t.Fatalf("ValidateBenchmark() = %+v, want one error", validation)
			}
			got := validation.Errors[0]
			if got.Code != ErrorCodeValidation ||
				got.Field != test.wantField ||
				!strings.Contains(got.Message, test.wantText) {
				t.Fatalf("validation error = %+v, want field %q containing %q", got, test.wantField, test.wantText)
			}

			result := service.StartBenchmark(request)
			if result.Report != nil || result.Error == nil || *result.Error != got {
				t.Fatalf("StartBenchmark() = %+v, want matching structured error", result)
			}
		})
	}
}

func TestServiceInspectsDirectory(t *testing.T) {
	directory := t.TempDir()
	result := NewService().InspectDirectory(directory)
	if result.Error != nil {
		t.Fatalf("InspectDirectory() returned error: %+v", result.Error)
	}
	if result.Inspection == nil {
		t.Fatal("InspectDirectory() returned no inspection")
	}
	if !filepath.IsAbs(result.Inspection.ResolvedPath) {
		t.Fatalf("resolved path is not absolute: %q", result.Inspection.ResolvedPath)
	}
	available, err := strconv.ParseInt(result.Inspection.AvailableBytes, 10, 64)
	if err != nil || available <= 0 {
		t.Fatalf("available bytes = %q, want a positive decimal string", result.Inspection.AvailableBytes)
	}
	if result.Inspection.DirectIO.Capability == DirectIOAvailable &&
		result.Inspection.DirectIO.AlignmentBytes == "" {
		t.Fatalf("direct I/O alignment is missing: %+v", result.Inspection.DirectIO)
	}

	file := filepath.Join(directory, "file")
	if err := os.WriteFile(file, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	invalid := NewService().InspectDirectory(file)
	if invalid.Error == nil ||
		invalid.Error.Code != ErrorCodeInspection ||
		invalid.Error.Field != "directory" {
		t.Fatalf("InspectDirectory(file) = %+v, want structured directory error", invalid)
	}
}

func TestReportUsesStringsForLargeIntegers(t *testing.T) {
	source := benchmark.Report{
		StartedAt: time.Unix(1, 2).UTC(),
		Config: benchmark.Config{
			Size:         math.MaxInt64,
			RandomSeed:   math.MaxInt64,
			Duration:     time.Duration(math.MaxInt64),
			IOMode:       benchmark.BufferedIO,
			CacheControl: benchmark.CacheControlOff,
		},
		Environment: benchmark.Environment{
			AvailableBytesBefore: math.MaxInt64,
			AvailableBytesAfter:  math.MaxInt64,
		},
		Iterations: []benchmark.IterationResult{{
			Measurements: []benchmark.Measurement{{
				Bytes:      math.MaxInt64,
				Operations: math.MaxInt64,
				Latency: benchmark.LatencyStats{
					SampleCount: math.MaxUint64,
				},
			}},
			Storage: benchmark.FileStorage{LogicalBytes: math.MaxInt64},
		}},
	}

	report := newReport(source)
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	maxInt64 := strconv.FormatInt(math.MaxInt64, 10)
	maxUint64 := strconv.FormatUint(math.MaxUint64, 10)
	for _, value := range []string{maxInt64, maxUint64} {
		if !strings.Contains(string(encoded), `"`+value+`"`) {
			t.Fatalf("report JSON does not contain quoted integer %s: %s", value, encoded)
		}
	}
}

type recordedEvent struct {
	name    string
	payload any
}

type eventRecorder struct {
	mu     sync.Mutex
	events []recordedEvent
}

func (r *eventRecorder) emit(name string, payload any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, recordedEvent{name: name, payload: payload})
}

func (r *eventRecorder) snapshot() []recordedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedEvent(nil), r.events...)
}

func TestServicePublishesRunScopedEvents(t *testing.T) {
	var recorder eventRecorder
	service := NewService(WithEventEmitter(recorder.emit))
	request := service.GetDefaults()
	request.Directory = t.TempDir()
	request.Size = "4KiB"
	request.BlockSize = "1KiB"
	request.Workload = WorkloadSequential

	result := service.StartBenchmark(request)
	if result.Error != nil || result.Report == nil || result.RunID == "" ||
		result.State != RunStateCompleted {
		t.Fatalf("StartBenchmark() = %+v, want completed report", result)
	}

	events := recorder.snapshot()
	if len(events) == 0 {
		t.Fatal("StartBenchmark() published no events")
	}
	names := make(map[string]bool)
	for _, event := range events {
		names[event.name] = true
		switch payload := event.payload.(type) {
		case StateEvent:
			if payload.RunID != result.RunID {
				t.Fatalf("state event run ID = %q, want %q", payload.RunID, result.RunID)
			}
		case ProgressEvent:
			if payload.RunID != result.RunID {
				t.Fatalf("progress event run ID = %q, want %q", payload.RunID, result.RunID)
			}
		case CompletedEvent:
			if payload.RunID != result.RunID {
				t.Fatalf("completed event run ID = %q, want %q", payload.RunID, result.RunID)
			}
		default:
			t.Fatalf("unexpected event payload %T", event.payload)
		}
	}
	for _, name := range []string{
		EventBenchmarkState,
		EventBenchmarkProgress,
		EventBenchmarkCompleted,
	} {
		if !names[name] {
			t.Fatalf("event %q was not published: %+v", name, events)
		}
	}
}

type blockingRunner struct {
	started chan struct{}
}

func (r blockingRunner) Run(ctx context.Context, _ benchmark.Config) (benchmark.Report, error) {
	close(r.started)
	<-ctx.Done()
	return benchmark.Report{}, ctx.Err()
}

func TestServiceRejectsConcurrentRunAndCancelsActiveRun(t *testing.T) {
	var recorder eventRecorder
	started := make(chan struct{})
	service := NewService(WithEventEmitter(recorder.emit))
	service.newRunner = func(benchmark.Observer, ...benchmark.Workload) benchmarkRunner {
		return blockingRunner{started: started}
	}
	request := service.GetDefaults()
	request.Directory = t.TempDir()
	request.Size = "4KiB"
	request.BlockSize = "1KiB"

	resultChannel := make(chan BenchmarkResult, 1)
	go func() {
		resultChannel <- service.StartBenchmark(request)
	}()
	<-started

	conflict := service.StartBenchmark(request)
	if conflict.Error == nil ||
		conflict.Error.Code != ErrorCodeConflict ||
		conflict.RunID != "1" {
		t.Fatalf("concurrent StartBenchmark() = %+v, want active-run conflict", conflict)
	}
	staleCancel := service.CancelBenchmark("stale")
	if staleCancel.Error == nil || staleCancel.Error.Code != ErrorCodeConflict {
		t.Fatalf("CancelBenchmark(stale) = %+v, want conflict", staleCancel)
	}
	cancelled := service.CancelBenchmark("1")
	if cancelled.Error != nil || cancelled.RunID != "1" ||
		cancelled.State != RunStateCancelling {
		t.Fatalf("CancelBenchmark() = %+v, want cancelling run 1", cancelled)
	}

	result := <-resultChannel
	if result.Error != nil || result.RunID != "1" || result.State != RunStateCancelled {
		t.Fatalf("cancelled StartBenchmark() = %+v, want cancelled result", result)
	}
	idleCancel := service.CancelBenchmark("1")
	if idleCancel.Error == nil || idleCancel.State != RunStateIdle {
		t.Fatalf("CancelBenchmark() after completion = %+v, want idle conflict", idleCancel)
	}

	var states []RunState
	var cancelledEvent bool
	for _, event := range recorder.snapshot() {
		if event.name == EventBenchmarkState {
			states = append(states, event.payload.(StateEvent).State)
		}
		if event.name == EventBenchmarkCancelled {
			payload := event.payload.(CancelledEvent)
			cancelledEvent = payload.RunID == "1"
		}
	}
	expectedStates := []RunState{
		RunStateValidating,
		RunStateRunning,
		RunStateCancelling,
		RunStateCancelled,
		RunStateIdle,
	}
	if fmt.Sprint(states) != fmt.Sprint(expectedStates) {
		t.Fatalf("state events = %v, want %v", states, expectedStates)
	}
	if !cancelledEvent {
		t.Fatal("service did not publish benchmark:cancelled")
	}
}

func TestServiceCancellationRemovesIncompleteKeptFile(t *testing.T) {
	directory := t.TempDir()
	workloadStarted := make(chan struct{})
	var startedOnce sync.Once
	service := NewService(WithEventEmitter(func(name string, payload any) {
		if name != EventBenchmarkProgress {
			return
		}
		progress := payload.(ProgressEvent)
		if progress.Phase == string(benchmark.PhaseWorkload) &&
			progress.Status == string(benchmark.PhaseStarted) {
			startedOnce.Do(func() { close(workloadStarted) })
		}
	}))
	request := service.GetDefaults()
	request.Directory = directory
	request.Size = "16KiB"
	request.BlockSize = "4KiB"
	request.Duration = "10s"
	request.KeepFile = true
	request.WriteImpactConfirmed = true

	resultChannel := make(chan BenchmarkResult, 1)
	go func() {
		resultChannel <- service.StartBenchmark(request)
	}()
	select {
	case <-workloadStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("benchmark workload did not start")
	}
	cancelled := service.CancelBenchmark("1")
	if cancelled.Error != nil {
		t.Fatalf("CancelBenchmark() returned error: %+v", cancelled.Error)
	}
	result := <-resultChannel
	if result.State != RunStateCancelled || result.Error != nil {
		t.Fatalf("StartBenchmark() = %+v, want cancelled result", result)
	}
	matches, err := filepath.Glob(filepath.Join(directory, "diskbenchmark-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("cancelled service run retained files: %v", matches)
	}
}

func TestServicePublishesStructuredFailureEvent(t *testing.T) {
	var recorder eventRecorder
	service := NewService(WithEventEmitter(recorder.emit))
	request := service.GetDefaults()
	request.Size = "invalid"

	result := service.StartBenchmark(request)
	if result.Error == nil || result.State != RunStateFailed || result.RunID != "1" {
		t.Fatalf("StartBenchmark() = %+v, want failed run 1", result)
	}
	var failure *FailedEvent
	for _, event := range recorder.snapshot() {
		if event.name == EventBenchmarkFailed {
			payload := event.payload.(FailedEvent)
			failure = &payload
		}
	}

	if failure == nil ||
		failure.RunID != result.RunID ||
		failure.Error.Code != ErrorCodeValidation ||
		failure.Error.Field != "size" {
		t.Fatalf("failure event = %+v, want structured size validation error", failure)
	}
}

func TestServiceNativeDialogResults(t *testing.T) {
	service := NewService(WithDialogProvider(DialogProvider{
		SelectDirectory: func() (string, error) {
			return `C:\benchmark`, nil
		},
		SelectSuiteFile: func() (string, error) {
			return "", nil
		},
	}))
	directory := service.SelectDirectory()
	if directory.Error != nil || directory.Cancelled || directory.Path != `C:\benchmark` {
		t.Fatalf("SelectDirectory() = %+v", directory)
	}
	suite := service.SelectSuiteFile()
	if suite.Error != nil || !suite.Cancelled || suite.Path != "" {
		t.Fatalf("SelectSuiteFile() = %+v", suite)
	}

	unavailable := NewService().SelectDirectory()
	if unavailable.Error == nil || unavailable.Error.Code != ErrorCodeDialog {
		t.Fatalf("SelectDirectory() without provider = %+v, want dialog error", unavailable)
	}
}

func TestServiceExportsCompletedReportThroughNativeDialog(t *testing.T) {
	directory := t.TempDir()
	var destination string
	service := NewService(WithDialogProvider(DialogProvider{
		SelectSaveFile: func(request SaveDialogRequest) (string, error) {
			destination = filepath.Join(directory, "report."+strings.TrimPrefix(request.Pattern, "*."))
			return destination, nil
		},
	}))
	request := service.GetDefaults()
	request.Directory = directory
	request.Size = "4KiB"
	request.BlockSize = "1KiB"
	request.Workload = WorkloadSequential
	run := service.StartBenchmark(request)
	if run.Error != nil || run.State != RunStateCompleted {
		t.Fatalf("StartBenchmark() = %+v", run)
	}

	jsonResult := service.ExportReport(ExportRequest{RunID: run.RunID, Format: ExportJSON})
	if jsonResult.Error != nil || jsonResult.Cancelled || jsonResult.Path != destination {
		t.Fatalf("ExportReport(json) = %+v", jsonResult)
	}
	data, err := os.ReadFile(jsonResult.Path)
	if err != nil {
		t.Fatal(err)
	}
	var exported benchmark.Report
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("exported JSON is invalid: %v", err)
	}
	if exported.Config.Size != 4096 || len(exported.Iterations) != 1 {
		t.Fatalf("exported JSON report is incomplete: %+v", exported)
	}

	csvResult := service.ExportReport(ExportRequest{RunID: run.RunID, Format: ExportCSV})
	if csvResult.Error != nil || csvResult.Cancelled {
		t.Fatalf("ExportReport(csv) = %+v", csvResult)
	}
	data, err = os.ReadFile(csvResult.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "record_type,started_at,") {
		t.Fatalf("exported CSV has unexpected header: %q", data)
	}

	missing := service.ExportReport(ExportRequest{RunID: "missing", Format: ExportJSON})
	if missing.Error == nil || missing.Error.Code != ErrorCodeExport {
		t.Fatalf("ExportReport(missing) = %+v, want export error", missing)
	}
}
