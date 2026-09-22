package benchmark

import (
	"encoding/json"
	"time"
)

type Measurement struct {
	Name           string             `json:"name"`
	Bytes          int64              `json:"bytes"`
	Operations     int64              `json:"operations"`
	Reads          int64              `json:"reads"`
	Writes         int64              `json:"writes"`
	Duration       time.Duration      `json:"duration_ns"`
	TargetDuration time.Duration      `json:"target_duration_ns"`
	Latency        LatencyStats       `json:"latency"`
	CacheControl   CacheControlResult `json:"cache_control"`
}

type LatencyStats struct {
	Average     time.Duration `json:"average_ns"`
	P50         time.Duration `json:"p50_ns"`
	P95         time.Duration `json:"p95_ns"`
	P99         time.Duration `json:"p99_ns"`
	Maximum     time.Duration `json:"maximum_ns"`
	SampleCount uint64        `json:"sample_count"`
	BucketCount int           `json:"bucket_count"`
}

type CacheControlResult struct {
	Mode   CacheControlMode `json:"mode"`
	API    string           `json:"api,omitempty"`
	Status string           `json:"status"`
	Detail string           `json:"detail,omitempty"`
}

func (m Measurement) MiBPerSecond() float64 {
	if m.Duration <= 0 {
		return 0
	}
	return float64(m.Bytes) / (1024 * 1024) / m.Duration.Seconds()
}

func (m Measurement) AverageLatency() time.Duration {
	if m.Latency.Average > 0 {
		return m.Latency.Average
	}
	if m.Operations > 0 {
		return m.Duration / time.Duration(m.Operations)
	}
	return 0
}

func (m Measurement) IOPS() float64 {
	if m.Duration <= 0 {
		return 0
	}
	return float64(m.Operations) / m.Duration.Seconds()
}

func (m Measurement) MarshalJSON() ([]byte, error) {
	type measurement Measurement
	return json.Marshal(struct {
		measurement
		MiBPerSecond float64 `json:"mib_per_second"`
		IOPS         float64 `json:"iops"`
	}{
		measurement:  measurement(m),
		MiBPerSecond: m.MiBPerSecond(),
		IOPS:         m.IOPS(),
	})
}

type IterationResult struct {
	Number       int                `json:"number"`
	FilePath     string             `json:"file_path,omitempty"`
	Measurements []Measurement      `json:"measurements"`
	Storage      FileStorage        `json:"storage"`
	Verification VerificationResult `json:"verification"`
}

type FileStorage struct {
	LogicalBytes   int64 `json:"logical_bytes"`
	AllocatedBytes int64 `json:"allocated_bytes"`
	Sparse         bool  `json:"sparse"`
}

type VerificationResult struct {
	Enabled  bool          `json:"enabled"`
	Passed   bool          `json:"passed"`
	Bytes    int64         `json:"bytes"`
	Duration time.Duration `json:"duration_ns"`
}

type Summary struct {
	Name             string  `json:"name"`
	MinimumMiBPerSec float64 `json:"minimum_mib_per_second"`
	MaximumMiBPerSec float64 `json:"maximum_mib_per_second"`
	AverageMiBPerSec float64 `json:"average_mib_per_second"`
	MinimumIOPS      float64 `json:"minimum_iops"`
	MaximumIOPS      float64 `json:"maximum_iops"`
	AverageIOPS      float64 `json:"average_iops"`
}

type Report struct {
	StartedAt        time.Time         `json:"started_at"`
	Config           Config            `json:"config"`
	Environment      Environment       `json:"environment"`
	Iterations       []IterationResult `json:"iterations"`
	Summaries        []Summary         `json:"summaries"`
	WarmupsCompleted int               `json:"warmups_completed"`
}

type Environment struct {
	Hostname             string `json:"hostname"`
	OperatingSystem      string `json:"operating_system"`
	Architecture         string `json:"architecture"`
	CPUs                 int    `json:"cpus"`
	GoVersion            string `json:"go_version"`
	Filesystem           string `json:"filesystem"`
	Volume               string `json:"volume"`
	AvailableBytesBefore int64  `json:"available_bytes_before"`
	AvailableBytesAfter  int64  `json:"available_bytes_after"`
}

func summarize(iterations []IterationResult) []Summary {
	if len(iterations) == 0 {
		return nil
	}

	type aggregate struct {
		name      string
		min       float64
		max       float64
		total     float64
		minIOPS   float64
		maxIOPS   float64
		totalIOPS float64
		count     int
	}
	order := make([]string, 0, len(iterations[0].Measurements))
	aggregates := make(map[string]*aggregate)

	for _, iteration := range iterations {
		for _, measurement := range iteration.Measurements {
			value := measurement.MiBPerSecond()
			iops := measurement.IOPS()
			current, ok := aggregates[measurement.Name]
			if !ok {
				current = &aggregate{
					name:    measurement.Name,
					min:     value,
					max:     value,
					minIOPS: iops,
					maxIOPS: iops,
				}
				aggregates[measurement.Name] = current
				order = append(order, measurement.Name)
			}
			current.min = min(current.min, value)
			current.max = max(current.max, value)
			current.total += value
			current.minIOPS = min(current.minIOPS, iops)
			current.maxIOPS = max(current.maxIOPS, iops)
			current.totalIOPS += iops
			current.count++
		}
	}

	summaries := make([]Summary, 0, len(order))
	for _, name := range order {
		current := aggregates[name]
		summaries = append(summaries, Summary{
			Name:             current.name,
			MinimumMiBPerSec: current.min,
			MaximumMiBPerSec: current.max,
			AverageMiBPerSec: current.total / float64(current.count),
			MinimumIOPS:      current.minIOPS,
			MaximumIOPS:      current.maxIOPS,
			AverageIOPS:      current.totalIOPS / float64(current.count),
		})
	}
	return summaries
}

func calculateLatency(latencies []time.Duration) LatencyStats {
	var histogram latencyHistogram
	for _, latency := range latencies {
		histogram.record(latency)
	}
	return histogram.stats()
}
