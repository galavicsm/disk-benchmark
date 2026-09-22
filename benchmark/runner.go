package benchmark

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

type Runner struct {
	workloads []Workload
	observer  Observer
}

func NewRunner(workloads ...Workload) *Runner {
	if len(workloads) == 0 {
		workloads = []Workload{SequentialWrite{}, SequentialRead{}}
	}
	return &Runner{workloads: append([]Workload(nil), workloads...)}
}

func (r *Runner) WithObserver(observer Observer) *Runner {
	clone := *r
	clone.observer = observer
	return &clone
}

func (r *Runner) Run(ctx context.Context, config Config) (Report, error) {
	r.emit(ProgressEvent{
		Phase:           PhasePreflight,
		Status:          PhaseStarted,
		WarmupsTotal:    config.WarmupIterations,
		IterationsTotal: config.Iterations,
		WorkloadsTotal:  len(r.workloads),
	})
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	config, err := ValidateConfig(config)
	if err != nil {
		return Report{}, err
	}
	environment, err := collectEnvironment(config.Directory)
	if err != nil {
		return Report{}, err
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	r.emit(ProgressEvent{
		Phase:           PhasePreflight,
		Status:          PhaseCompleted,
		WarmupsTotal:    config.WarmupIterations,
		IterationsTotal: config.Iterations,
		WorkloadsTotal:  len(r.workloads),
	})

	report := Report{
		StartedAt:   time.Now().UTC(),
		Config:      config,
		Environment: environment,
		Iterations:  make([]IterationResult, 0, config.Iterations),
	}
	warmupConfig := config
	warmupConfig.KeepFile = false
	warmupConfig.VerifyData = false
	for warmup := 1; warmup <= config.WarmupIterations; warmup++ {
		r.emit(r.iterationEvent(PhaseWarmup, PhaseStarted, warmupConfig, warmup, true))
		if _, err := r.runIteration(ctx, warmupConfig, warmup, true); err != nil {
			return Report{}, fmt.Errorf("warm-up iteration %d: %w", warmup, err)
		}
		report.WarmupsCompleted++
		r.emit(r.iterationEvent(PhaseWarmup, PhaseCompleted, warmupConfig, warmup, true))
	}
	for iteration := 1; iteration <= config.Iterations; iteration++ {
		r.emit(r.iterationEvent(PhaseIteration, PhaseStarted, config, iteration, false))
		result, err := r.runIteration(ctx, config, iteration, false)
		if err != nil {
			return Report{}, fmt.Errorf("iteration %d: %w", iteration, err)
		}
		report.Iterations = append(report.Iterations, result)
		r.emit(r.iterationEvent(PhaseIteration, PhaseCompleted, config, iteration, false))
	}
	report.Summaries = summarize(report.Iterations)
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	report.Environment.AvailableBytesAfter, err = availableSpace(config.Directory)
	if err != nil {
		return Report{}, fmt.Errorf("inspect available disk space after benchmark: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r *Runner) runIteration(
	ctx context.Context,
	config Config,
	number int,
	warmup bool,
) (result IterationResult, err error) {
	if err := ctx.Err(); err != nil {
		return IterationResult{}, err
	}
	if _, err := preflightSpace(config); err != nil {
		return IterationResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return IterationResult{}, err
	}
	file, path, err := createBenchmarkFile(config)
	if err != nil {
		return IterationResult{}, err
	}
	succeeded := false
	defer func() {
		r.emit(r.progressEvent(PhaseCleanup, PhaseStarted, config, number, warmup, "", 0, false))
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			succeeded = false
		}
		closeErr := file.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close test file: %w", closeErr)
		}
		if config.KeepFile && succeeded && err == nil && closeErr == nil {
			r.emit(r.progressEvent(PhaseCleanup, PhaseCompleted, config, number, warmup, "", 0, false))
			return
		}

		removeErr := os.Remove(path)
		if removeErr != nil {
			removeErr = fmt.Errorf("remove test file %q: %w", path, removeErr)
		}
		err = errors.Join(err, closeErr, removeErr)
		r.emit(r.progressEvent(PhaseCleanup, PhaseCompleted, config, number, warmup, "", 0, false))
	}()

	buffer := makeIOBuffer(config.BlockSize, config.IOAlignment)
	fillDeterministic(buffer)
	result = IterationResult{
		Number:       number,
		Measurements: make([]Measurement, 0, len(r.workloads)),
	}
	for index, workload := range r.workloads {
		workloadNumber := index + 1
		indeterminate := workloadDuration(workload, config) > 0
		if preparer, ok := workload.(workloadPreparer); ok {
			r.emit(r.progressEvent(
				PhaseWorkloadPreparation,
				PhaseStarted,
				config,
				number,
				warmup,
				workload.Name(),
				workloadNumber,
				indeterminate,
			))
			if prepareErr := preparer.Prepare(ctx, file, buffer, config); prepareErr != nil {
				return IterationResult{}, fmt.Errorf("prepare %s: %w", workload.Name(), prepareErr)
			}
			if err := ctx.Err(); err != nil {
				return IterationResult{}, err
			}
			r.emit(r.progressEvent(
				PhaseWorkloadPreparation,
				PhaseCompleted,
				config,
				number,
				warmup,
				workload.Name(),
				workloadNumber,
				indeterminate,
			))
		}
		if err := ctx.Err(); err != nil {
			return IterationResult{}, err
		}
		r.emit(r.progressEvent(
			PhaseWorkload,
			PhaseStarted,
			config,
			number,
			warmup,
			workload.Name(),
			workloadNumber,
			indeterminate,
		))
		measurement, workloadErr := workload.Run(ctx, file, buffer, config)
		if workloadErr != nil {
			return IterationResult{}, fmt.Errorf("%s: %w", workload.Name(), workloadErr)
		}
		if err := ctx.Err(); err != nil {
			return IterationResult{}, err
		}
		result.Measurements = append(result.Measurements, measurement)
		r.emit(r.progressEvent(
			PhaseWorkload,
			PhaseCompleted,
			config,
			number,
			warmup,
			workload.Name(),
			workloadNumber,
			indeterminate,
		))
	}
	r.emit(r.progressEvent(PhaseVerification, PhaseStarted, config, number, warmup, "", 0, false))
	result.Verification, err = verifyFile(ctx, file, buffer, config)
	if err != nil {
		return IterationResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return IterationResult{}, err
	}
	r.emit(r.progressEvent(PhaseVerification, PhaseCompleted, config, number, warmup, "", 0, false))
	result.Storage, err = inspectFileStorage(file)
	if err != nil {
		return IterationResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return IterationResult{}, err
	}
	if config.KeepFile {
		result.FilePath = path
	}
	succeeded = true
	return result, nil
}

func (r *Runner) emit(event ProgressEvent) {
	if r.observer != nil {
		r.observer.ObserveProgress(event)
	}
}

func (r *Runner) iterationEvent(
	phase Phase,
	status PhaseStatus,
	config Config,
	number int,
	warmup bool,
) ProgressEvent {
	return r.progressEvent(phase, status, config, number, warmup, "", 0, false)
}

func (r *Runner) progressEvent(
	phase Phase,
	status PhaseStatus,
	config Config,
	number int,
	warmup bool,
	workload string,
	workloadNumber int,
	indeterminate bool,
) ProgressEvent {
	event := ProgressEvent{
		Phase:           phase,
		Status:          status,
		WarmupsTotal:    config.WarmupIterations,
		IterationsTotal: config.Iterations,
		Workload:        workload,
		WorkloadNumber:  workloadNumber,
		WorkloadsTotal:  len(r.workloads),
		Indeterminate:   indeterminate,
	}
	if warmup {
		event.Warmup = number
	} else {
		event.Iteration = number
	}
	return event
}

func workloadDuration(workload Workload, config Config) time.Duration {
	if configured, ok := workload.(ConfiguredWorkload); ok {
		return configured.apply(config).Duration
	}
	return config.Duration
}

func fillDeterministic(buffer []byte) {
	var state uint64 = 0x9e3779b97f4a7c15
	for index := range buffer {
		state ^= state << 7
		state ^= state >> 9
		state ^= state << 8
		buffer[index] = byte(state)
	}
}
