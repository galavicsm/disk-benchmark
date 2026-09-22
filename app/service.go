package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"diskbenchmark/benchmark"
	"diskbenchmark/units"
	"diskbenchmark/workloads"
)

type benchmarkRunner interface {
	Run(context.Context, benchmark.Config) (benchmark.Report, error)
}

type runnerFactory func(benchmark.Observer, ...benchmark.Workload) benchmarkRunner

type ServiceOption func(*Service)

type activeRun struct {
	id     string
	state  RunState
	cancel context.CancelFunc
}

type Service struct {
	mu        sync.Mutex
	eventMu   sync.Mutex
	active    *activeRun
	nextRunID uint64
	emit      EventEmitter
	newRunner runnerFactory
	dialogs   DialogProvider
	reports   map[string]benchmark.Report
}

func NewService(options ...ServiceOption) *Service {
	service := &Service{
		emit:    func(string, any) {},
		reports: make(map[string]benchmark.Report),
		newRunner: func(observer benchmark.Observer, selected ...benchmark.Workload) benchmarkRunner {
			return benchmark.NewRunner(selected...).WithObserver(observer)
		},
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func WithEventEmitter(emitter EventEmitter) ServiceOption {
	return func(service *Service) {
		if emitter != nil {
			service.emit = emitter
		}
	}
}

func (*Service) GetDefaults() BenchmarkRequest {
	return DefaultBenchmarkRequest()
}

func (*Service) InspectDirectory(directory string) DirectoryInspectionResult {
	inspection, err := benchmark.InspectDirectory(directory)
	if err != nil {
		return DirectoryInspectionResult{
			Error: serviceError(ErrorCodeInspection, "directory", err),
		}
	}
	result := newDirectoryInspection(inspection)
	return DirectoryInspectionResult{Inspection: &result}
}

func (*Service) ValidateBenchmark(request BenchmarkRequest) ValidationResult {
	config, selected, validationError := buildBenchmark(request)
	if validationError != nil {
		return ValidationResult{
			Valid:  false,
			Errors: []ServiceError{*validationError},
		}
	}
	return ValidationResult{
		Valid:       true,
		Errors:      []ServiceError{},
		WriteImpact: assessWriteImpact(config, selected),
	}
}

func (s *Service) StartBenchmark(request BenchmarkRequest) BenchmarkResult {
	runID, ctx, conflict := s.beginRun()
	if conflict != nil {
		return BenchmarkResult{
			RunID: conflict.RunID,
			State: conflict.State,
			Error: conflict.Error,
		}
	}

	config, selected, validationError := buildBenchmark(request)
	if validationError != nil {
		state := s.finishRun(runID, RunStateFailed, nil,
			namedEvent{EventBenchmarkFailed, FailedEvent{RunID: runID, Error: *validationError}},
		)
		if state == RunStateCancelled {
			return BenchmarkResult{RunID: runID, State: state}
		}
		return BenchmarkResult{RunID: runID, State: RunStateFailed, Error: validationError}
	}
	writeImpact := assessWriteImpact(config, selected)
	if writeImpact.RequiresConfirmation && !request.WriteImpactConfirmed {
		confirmationError := serviceError(
			ErrorCodeConfirmation,
			"writeImpactConfirmed",
			fmt.Errorf("explicit confirmation is required for this benchmark's write impact"),
		)
		state := s.finishRun(runID, RunStateAwaitingConfirmation, nil)
		return BenchmarkResult{RunID: runID, State: state, Error: confirmationError}
	}
	if !s.setRunState(runID, RunStateRunning) {
		state := s.finishRun(runID, RunStateCancelled, nil)
		return BenchmarkResult{RunID: runID, State: state}
	}
	observer := benchmark.ObserverFunc(func(event benchmark.ProgressEvent) {
		s.emitEvent(EventBenchmarkProgress, newProgressEvent(runID, event))
	})
	report, err := s.newRunner(observer, selected...).Run(ctx, config)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			state := s.finishRun(runID, RunStateCancelled, nil)
			return BenchmarkResult{RunID: runID, State: state}
		}
		executionError := serviceError(ErrorCodeExecution, "", err)
		state := s.finishRun(runID, RunStateFailed, nil,
			namedEvent{EventBenchmarkFailed, FailedEvent{RunID: runID, Error: *executionError}},
		)
		if state == RunStateCancelled {
			return BenchmarkResult{RunID: runID, State: state}
		}
		return BenchmarkResult{RunID: runID, State: RunStateFailed, Error: executionError}
	}
	result := newReport(report)
	state := s.finishRun(runID, RunStateCompleted, &report,
		namedEvent{EventBenchmarkCompleted, CompletedEvent{RunID: runID, Report: result}},
	)
	if state == RunStateCancelled {
		return BenchmarkResult{RunID: runID, State: state}
	}
	return BenchmarkResult{RunID: runID, State: state, Report: &result}
}

func (s *Service) CancelBenchmark(runID string) CancelResult {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()

	s.mu.Lock()
	if s.active == nil {
		s.mu.Unlock()
		return CancelResult{
			State: RunStateIdle,
			Error: serviceError(ErrorCodeConflict, "", fmt.Errorf("no benchmark is currently running")),
		}
	}
	if runID != "" && s.active.id != runID {
		activeID := s.active.id
		activeState := s.active.state
		s.mu.Unlock()
		return CancelResult{
			RunID: activeID,
			State: activeState,
			Error: serviceError(
				ErrorCodeConflict,
				"",
				fmt.Errorf("benchmark run %q is not active", runID),
			),
		}
	}
	active := s.active
	active.state = RunStateCancelling
	active.cancel()
	s.mu.Unlock()

	s.emit(EventBenchmarkState, StateEvent{RunID: active.id, State: RunStateCancelling})
	return CancelResult{RunID: active.id, State: RunStateCancelling}
}

type namedEvent struct {
	name    string
	payload any
}

func (s *Service) beginRun() (string, context.Context, *CancelResult) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()

	s.mu.Lock()
	if s.active != nil {
		conflict := &CancelResult{
			RunID: s.active.id,
			State: s.active.state,
			Error: serviceError(
				ErrorCodeConflict,
				"",
				fmt.Errorf("benchmark run %q is already active", s.active.id),
			),
		}
		s.mu.Unlock()
		return "", nil, conflict
	}
	s.nextRunID++
	runID := strconv.FormatUint(s.nextRunID, 10)
	ctx, cancel := context.WithCancel(context.Background())
	s.active = &activeRun{id: runID, state: RunStateValidating, cancel: cancel}
	s.mu.Unlock()

	s.emit(EventBenchmarkState, StateEvent{RunID: runID, State: RunStateValidating})
	return runID, ctx, nil
}

func (s *Service) setRunState(runID string, state RunState) bool {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()

	s.mu.Lock()
	if s.active == nil || s.active.id != runID || s.active.state == RunStateCancelling {
		s.mu.Unlock()
		return false
	}
	s.active.state = state
	s.mu.Unlock()
	s.emit(EventBenchmarkState, StateEvent{RunID: runID, State: state})
	return true
}

func (s *Service) finishRun(
	runID string,
	state RunState,
	report *benchmark.Report,
	events ...namedEvent,
) RunState {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()

	s.mu.Lock()
	if s.active != nil && s.active.id == runID {
		if s.active.state == RunStateCancelling {
			state = RunStateCancelled
		}
		s.active.cancel()
		s.active = nil
	}
	if state == RunStateCompleted && report != nil {
		clear(s.reports)
		s.reports[runID] = *report
	}
	s.mu.Unlock()
	if state == RunStateCancelled {
		s.emit(EventBenchmarkState, StateEvent{RunID: runID, State: RunStateCancelled})
		s.emit(EventBenchmarkCancelled, CancelledEvent{RunID: runID})
		s.emit(EventBenchmarkState, StateEvent{RunID: runID, State: RunStateIdle})
		return state
	}
	s.emit(EventBenchmarkState, StateEvent{RunID: runID, State: state})
	for _, event := range events {
		s.emit(event.name, event.payload)
	}
	return state
}

func (s *Service) emitEvent(name string, payload any) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.emit(name, payload)
}

func buildBenchmark(request BenchmarkRequest) (benchmark.Config, []benchmark.Workload, *ServiceError) {
	size, err := units.ParseBytes(request.Size)
	if err != nil {
		return benchmark.Config{}, nil, serviceError(ErrorCodeValidation, "size", err)
	}
	blockSize, err := units.ParseBytes(request.BlockSize)
	if err != nil {
		return benchmark.Config{}, nil, serviceError(ErrorCodeValidation, "blockSize", err)
	}
	maxInt := int64(^uint(0) >> 1)
	if blockSize > maxInt {
		return benchmark.Config{}, nil, serviceError(
			ErrorCodeValidation,
			"blockSize",
			fmt.Errorf("block size is too large for this platform"),
		)
	}
	duration := time.Duration(0)
	if request.Duration != "" {
		duration, err = time.ParseDuration(request.Duration)
		if err != nil {
			return benchmark.Config{}, nil, serviceError(
				ErrorCodeValidation,
				"duration",
				fmt.Errorf("invalid duration %q: %w", request.Duration, err),
			)
		}
	}
	randomSeed, err := strconv.ParseInt(request.RandomSeed, 10, 64)
	if err != nil {
		return benchmark.Config{}, nil, serviceError(
			ErrorCodeValidation,
			"randomSeed",
			fmt.Errorf("invalid random seed %q: %w", request.RandomSeed, err),
		)
	}
	ioMode, err := benchmark.ParseIOMode(string(request.IOMode))
	if err != nil {
		return benchmark.Config{}, nil, serviceError(ErrorCodeValidation, "ioMode", err)
	}
	cacheControl, err := benchmark.ParseCacheControlMode(string(request.CacheControl))
	if err != nil {
		return benchmark.Config{}, nil, serviceError(ErrorCodeValidation, "cacheControl", err)
	}
	selection, err := workloads.Select(workloads.Selection{
		BuiltIn:   string(request.Workload),
		SuiteFile: request.SuiteFile,
	})
	if err != nil {
		field := "workload"
		if request.SuiteFile != "" {
			field = "suiteFile"
		}
		return benchmark.Config{}, nil, serviceError(ErrorCodeValidation, field, err)
	}

	config := benchmark.Config{
		Directory:         request.Directory,
		Size:              size,
		BlockSize:         int(blockSize),
		Iterations:        request.Iterations,
		WarmupIterations:  request.WarmupIterations,
		KeepFile:          request.KeepFile,
		VerifyData:        request.VerifyData,
		Sync:              request.SyncWrites,
		Workers:           request.Workers,
		QueueDepth:        request.QueueDepth,
		RandomReadPercent: request.RandomReadPercent,
		RandomSeed:        randomSeed,
		IOMode:            ioMode,
		CacheControl:      cacheControl,
		Duration:          duration,
		SuiteFile:         request.SuiteFile,
		SuiteVersion:      selection.SuiteVersion,
	}
	config, err = benchmark.ValidateConfig(config)
	if err != nil {
		return benchmark.Config{}, nil, serviceError(
			ErrorCodeValidation,
			configErrorField(err),
			err,
		)
	}
	inspection, err := benchmark.InspectDirectory(config.Directory)
	if err != nil {
		return benchmark.Config{}, nil, serviceError(
			ErrorCodeValidation,
			"directory",
			err,
		)
	}
	if config.Size > inspection.AvailableBytes {
		return benchmark.Config{}, nil, serviceError(
			ErrorCodeValidation,
			"size",
			fmt.Errorf(
				"insufficient disk space in %q: need %d bytes, have %d bytes available",
				config.Directory,
				config.Size,
				inspection.AvailableBytes,
			),
		)
	}
	return config, selection.Workloads, nil
}

func configErrorField(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "directory"), strings.Contains(message, "benchmark path"):
		return "directory"
	case strings.Contains(message, "block size"):
		return "blockSize"
	case strings.Contains(message, "data size"):
		return "size"
	case strings.Contains(message, "warm-up"):
		return "warmupIterations"
	case strings.Contains(message, "iterations"):
		return "iterations"
	case strings.Contains(message, "duration"):
		return "duration"
	case strings.Contains(message, "workers"):
		return "workers"
	case strings.Contains(message, "queue depth"):
		return "queueDepth"
	case strings.Contains(message, "random read"):
		return "randomReadPercent"
	case strings.Contains(message, "I/O"):
		return "ioMode"
	case strings.Contains(message, "cache control"):
		return "cacheControl"
	default:
		return ""
	}
}

func serviceError(code ErrorCode, field string, err error) *ServiceError {
	return &ServiceError{
		Code:    code,
		Field:   field,
		Message: err.Error(),
	}
}
