package app

type Workload string

const (
	WorkloadSequential Workload = "sequential"
	WorkloadRandom     Workload = "random"
	WorkloadMixed      Workload = "mixed"
	WorkloadOverwrite  Workload = "overwrite"
	WorkloadFsync      Workload = "fsync"
	WorkloadAll        Workload = "all"
)

type IOMode string

const (
	IOModeBuffered IOMode = "buffered"
	IOModeDirect   IOMode = "direct"
)

type CacheControlMode string

const (
	CacheControlOff     CacheControlMode = "off"
	CacheControlAttempt CacheControlMode = "attempt"
	CacheControlRequire CacheControlMode = "require"
)

type RunState string

const (
	RunStateIdle                 RunState = "idle"
	RunStateValidating           RunState = "validating"
	RunStateAwaitingConfirmation RunState = "awaiting-confirmation"
	RunStateRunning              RunState = "running"
	RunStateCancelling           RunState = "cancelling"
	RunStateCompleted            RunState = "completed"
	RunStateFailed               RunState = "failed"
	RunStateCancelled            RunState = "cancelled"
)

type BenchmarkRequest struct {
	Directory            string           `json:"directory"`
	Size                 string           `json:"size"`
	BlockSize            string           `json:"blockSize"`
	Iterations           int              `json:"iterations"`
	WarmupIterations     int              `json:"warmupIterations"`
	KeepFile             bool             `json:"keepFile"`
	VerifyData           bool             `json:"verifyData"`
	SyncWrites           bool             `json:"syncWrites"`
	Workload             Workload         `json:"workload"`
	SuiteFile            string           `json:"suiteFile"`
	Duration             string           `json:"duration"`
	Workers              int              `json:"workers"`
	QueueDepth           int              `json:"queueDepth"`
	RandomReadPercent    int              `json:"randomReadPercent"`
	RandomSeed           string           `json:"randomSeed"`
	IOMode               IOMode           `json:"ioMode"`
	CacheControl         CacheControlMode `json:"cacheControl"`
	WriteImpactConfirmed bool             `json:"writeImpactConfirmed"`
}

func DefaultBenchmarkRequest() BenchmarkRequest {
	return BenchmarkRequest{
		Directory:         ".",
		Size:              "1GiB",
		BlockSize:         "1MiB",
		Iterations:        1,
		SyncWrites:        true,
		Workload:          WorkloadSequential,
		Duration:          "0s",
		Workers:           1,
		QueueDepth:        1,
		RandomReadPercent: 50,
		RandomSeed:        "1",
		IOMode:            IOModeBuffered,
		CacheControl:      CacheControlOff,
	}
}
