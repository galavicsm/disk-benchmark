package app

import "diskbenchmark/benchmark"

const (
	EventBenchmarkState     = "benchmark:state"
	EventBenchmarkProgress  = "benchmark:progress"
	EventBenchmarkCompleted = "benchmark:completed"
	EventBenchmarkFailed    = "benchmark:failed"
	EventBenchmarkCancelled = "benchmark:cancelled"
)

type EventEmitter func(name string, payload any)

type StateEvent struct {
	RunID string   `json:"runId"`
	State RunState `json:"state"`
}

type ProgressEvent struct {
	RunID           string `json:"runId"`
	Phase           string `json:"phase"`
	Status          string `json:"status"`
	Warmup          int    `json:"warmup"`
	WarmupsTotal    int    `json:"warmupsTotal"`
	Iteration       int    `json:"iteration"`
	IterationsTotal int    `json:"iterationsTotal"`
	Workload        string `json:"workload,omitempty"`
	WorkloadNumber  int    `json:"workloadNumber"`
	WorkloadsTotal  int    `json:"workloadsTotal"`
	Indeterminate   bool   `json:"indeterminate"`
}

type CompletedEvent struct {
	RunID  string `json:"runId"`
	Report Report `json:"report"`
}

type FailedEvent struct {
	RunID string       `json:"runId"`
	Error ServiceError `json:"error"`
}

type CancelledEvent struct {
	RunID string `json:"runId"`
}

func newProgressEvent(runID string, source benchmark.ProgressEvent) ProgressEvent {
	return ProgressEvent{
		RunID:           runID,
		Phase:           string(source.Phase),
		Status:          string(source.Status),
		Warmup:          source.Warmup,
		WarmupsTotal:    source.WarmupsTotal,
		Iteration:       source.Iteration,
		IterationsTotal: source.IterationsTotal,
		Workload:        source.Workload,
		WorkloadNumber:  source.WorkloadNumber,
		WorkloadsTotal:  source.WorkloadsTotal,
		Indeterminate:   source.Indeterminate,
	}
}
