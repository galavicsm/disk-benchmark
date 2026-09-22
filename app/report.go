package app

import (
	"strconv"
	"time"

	"diskbenchmark/benchmark"
)

type Report struct {
	StartedAt        string            `json:"startedAt"`
	Config           ReportConfig      `json:"config"`
	Environment      Environment       `json:"environment"`
	Iterations       []IterationResult `json:"iterations"`
	Summaries        []Summary         `json:"summaries"`
	WarmupsCompleted int               `json:"warmupsCompleted"`
}

type ReportConfig struct {
	Directory         string           `json:"directory"`
	SizeBytes         string           `json:"sizeBytes"`
	BlockSizeBytes    string           `json:"blockSizeBytes"`
	Iterations        int              `json:"iterations"`
	KeepFile          bool             `json:"keepFile"`
	SyncWrites        bool             `json:"syncWrites"`
	Workers           int              `json:"workers"`
	QueueDepth        int              `json:"queueDepth"`
	RandomReadPercent int              `json:"randomReadPercent"`
	RandomSeed        string           `json:"randomSeed"`
	IOMode            IOMode           `json:"ioMode"`
	IOAlignmentBytes  string           `json:"ioAlignmentBytes"`
	CacheControl      CacheControlMode `json:"cacheControl"`
	WarmupIterations  int              `json:"warmupIterations"`
	VerifyData        bool             `json:"verifyData"`
	DurationNS        string           `json:"durationNs"`
	SuiteFile         string           `json:"suiteFile,omitempty"`
	SuiteVersion      int              `json:"suiteVersion,omitempty"`
}

type Environment struct {
	Hostname             string `json:"hostname"`
	OperatingSystem      string `json:"operatingSystem"`
	Architecture         string `json:"architecture"`
	CPUs                 int    `json:"cpus"`
	GoVersion            string `json:"goVersion"`
	Filesystem           string `json:"filesystem"`
	Volume               string `json:"volume"`
	AvailableBytesBefore string `json:"availableBytesBefore"`
	AvailableBytesAfter  string `json:"availableBytesAfter"`
}

type IterationResult struct {
	Number       int                `json:"number"`
	FilePath     string             `json:"filePath,omitempty"`
	Measurements []Measurement      `json:"measurements"`
	Storage      FileStorage        `json:"storage"`
	Verification VerificationResult `json:"verification"`
}

type Measurement struct {
	Name             string             `json:"name"`
	Bytes            string             `json:"bytes"`
	Operations       string             `json:"operations"`
	Reads            string             `json:"reads"`
	Writes           string             `json:"writes"`
	DurationNS       string             `json:"durationNs"`
	TargetDurationNS string             `json:"targetDurationNs"`
	Latency          LatencyStats       `json:"latency"`
	CacheControl     CacheControlResult `json:"cacheControl"`
	MiBPerSecond     float64            `json:"mibPerSecond"`
	IOPS             float64            `json:"iops"`
}

type LatencyStats struct {
	AverageNS   string `json:"averageNs"`
	P50NS       string `json:"p50Ns"`
	P95NS       string `json:"p95Ns"`
	P99NS       string `json:"p99Ns"`
	MaximumNS   string `json:"maximumNs"`
	SampleCount string `json:"sampleCount"`
	BucketCount int    `json:"bucketCount"`
}

type CacheControlResult struct {
	Mode   CacheControlMode `json:"mode"`
	API    string           `json:"api,omitempty"`
	Status string           `json:"status"`
	Detail string           `json:"detail,omitempty"`
}

type FileStorage struct {
	LogicalBytes   string `json:"logicalBytes"`
	AllocatedBytes string `json:"allocatedBytes"`
	Sparse         bool   `json:"sparse"`
}

type VerificationResult struct {
	Enabled    bool   `json:"enabled"`
	Passed     bool   `json:"passed"`
	Bytes      string `json:"bytes"`
	DurationNS string `json:"durationNs"`
}

type Summary struct {
	Name             string  `json:"name"`
	MinimumMiBPerSec float64 `json:"minimumMibPerSecond"`
	MaximumMiBPerSec float64 `json:"maximumMibPerSecond"`
	AverageMiBPerSec float64 `json:"averageMibPerSecond"`
	MinimumIOPS      float64 `json:"minimumIops"`
	MaximumIOPS      float64 `json:"maximumIops"`
	AverageIOPS      float64 `json:"averageIops"`
}

func newReport(source benchmark.Report) Report {
	report := Report{
		StartedAt:        source.StartedAt.Format(time.RFC3339Nano),
		Config:           newReportConfig(source.Config),
		Environment:      newEnvironment(source.Environment),
		Iterations:       make([]IterationResult, len(source.Iterations)),
		Summaries:        make([]Summary, len(source.Summaries)),
		WarmupsCompleted: source.WarmupsCompleted,
	}
	for index, iteration := range source.Iterations {
		report.Iterations[index] = newIterationResult(iteration)
	}
	for index, summary := range source.Summaries {
		report.Summaries[index] = Summary{
			Name:             summary.Name,
			MinimumMiBPerSec: summary.MinimumMiBPerSec,
			MaximumMiBPerSec: summary.MaximumMiBPerSec,
			AverageMiBPerSec: summary.AverageMiBPerSec,
			MinimumIOPS:      summary.MinimumIOPS,
			MaximumIOPS:      summary.MaximumIOPS,
			AverageIOPS:      summary.AverageIOPS,
		}
	}
	return report
}

func newReportConfig(config benchmark.Config) ReportConfig {
	return ReportConfig{
		Directory:         config.Directory,
		SizeBytes:         formatInt64(config.Size),
		BlockSizeBytes:    strconv.Itoa(config.BlockSize),
		Iterations:        config.Iterations,
		KeepFile:          config.KeepFile,
		SyncWrites:        config.Sync,
		Workers:           config.Workers,
		QueueDepth:        config.QueueDepth,
		RandomReadPercent: config.RandomReadPercent,
		RandomSeed:        formatInt64(config.RandomSeed),
		IOMode:            IOMode(config.IOMode),
		IOAlignmentBytes:  strconv.Itoa(config.IOAlignment),
		CacheControl:      CacheControlMode(config.CacheControl),
		WarmupIterations:  config.WarmupIterations,
		VerifyData:        config.VerifyData,
		DurationNS:        formatDuration(config.Duration),
		SuiteFile:         config.SuiteFile,
		SuiteVersion:      config.SuiteVersion,
	}
}

func newEnvironment(environment benchmark.Environment) Environment {
	return Environment{
		Hostname:             environment.Hostname,
		OperatingSystem:      environment.OperatingSystem,
		Architecture:         environment.Architecture,
		CPUs:                 environment.CPUs,
		GoVersion:            environment.GoVersion,
		Filesystem:           environment.Filesystem,
		Volume:               environment.Volume,
		AvailableBytesBefore: formatInt64(environment.AvailableBytesBefore),
		AvailableBytesAfter:  formatInt64(environment.AvailableBytesAfter),
	}
}

func newIterationResult(iteration benchmark.IterationResult) IterationResult {
	result := IterationResult{
		Number:       iteration.Number,
		FilePath:     iteration.FilePath,
		Measurements: make([]Measurement, len(iteration.Measurements)),
		Storage: FileStorage{
			LogicalBytes:   formatInt64(iteration.Storage.LogicalBytes),
			AllocatedBytes: formatInt64(iteration.Storage.AllocatedBytes),
			Sparse:         iteration.Storage.Sparse,
		},
		Verification: VerificationResult{
			Enabled:    iteration.Verification.Enabled,
			Passed:     iteration.Verification.Passed,
			Bytes:      formatInt64(iteration.Verification.Bytes),
			DurationNS: formatDuration(iteration.Verification.Duration),
		},
	}
	for index, measurement := range iteration.Measurements {
		result.Measurements[index] = newMeasurement(measurement)
	}
	return result
}

func newMeasurement(measurement benchmark.Measurement) Measurement {
	return Measurement{
		Name:             measurement.Name,
		Bytes:            formatInt64(measurement.Bytes),
		Operations:       formatInt64(measurement.Operations),
		Reads:            formatInt64(measurement.Reads),
		Writes:           formatInt64(measurement.Writes),
		DurationNS:       formatDuration(measurement.Duration),
		TargetDurationNS: formatDuration(measurement.TargetDuration),
		Latency: LatencyStats{
			AverageNS:   formatDuration(measurement.Latency.Average),
			P50NS:       formatDuration(measurement.Latency.P50),
			P95NS:       formatDuration(measurement.Latency.P95),
			P99NS:       formatDuration(measurement.Latency.P99),
			MaximumNS:   formatDuration(measurement.Latency.Maximum),
			SampleCount: strconv.FormatUint(measurement.Latency.SampleCount, 10),
			BucketCount: measurement.Latency.BucketCount,
		},
		CacheControl: CacheControlResult{
			Mode:   CacheControlMode(measurement.CacheControl.Mode),
			API:    measurement.CacheControl.API,
			Status: measurement.CacheControl.Status,
			Detail: measurement.CacheControl.Detail,
		},
		MiBPerSecond: measurement.MiBPerSecond(),
		IOPS:         measurement.IOPS(),
	}
}

func formatInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}

func formatDuration(value time.Duration) string {
	return strconv.FormatInt(int64(value), 10)
}
