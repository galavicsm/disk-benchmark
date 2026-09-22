package benchmark

type Phase string

const (
	PhasePreflight           Phase = "preflight"
	PhaseWarmup              Phase = "warmup"
	PhaseIteration           Phase = "iteration"
	PhaseWorkloadPreparation Phase = "workload-preparation"
	PhaseWorkload            Phase = "workload"
	PhaseVerification        Phase = "verification"
	PhaseCleanup             Phase = "cleanup"
)

type PhaseStatus string

const (
	PhaseStarted   PhaseStatus = "started"
	PhaseCompleted PhaseStatus = "completed"
)

type ProgressEvent struct {
	Phase           Phase
	Status          PhaseStatus
	Warmup          int
	WarmupsTotal    int
	Iteration       int
	IterationsTotal int
	Workload        string
	WorkloadNumber  int
	WorkloadsTotal  int
	Indeterminate   bool
}

type Observer interface {
	ObserveProgress(ProgressEvent)
}

type ObserverFunc func(ProgressEvent)

func (f ObserverFunc) ObserveProgress(event ProgressEvent) {
	f(event)
}
