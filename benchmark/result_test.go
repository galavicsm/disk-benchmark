package benchmark

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCalculateLatency(t *testing.T) {
	stats := calculateLatency([]time.Duration{
		10 * time.Millisecond,
		time.Millisecond,
		5 * time.Millisecond,
		2 * time.Millisecond,
	})
	if stats.Average != 4500*time.Microsecond {
		t.Fatalf("average = %s, want 4.5ms", stats.Average)
	}
	if stats.P50 < 2*time.Millisecond || stats.P50 > 2100*time.Microsecond ||
		stats.P95 != 10*time.Millisecond ||
		stats.P99 != 10*time.Millisecond || stats.Maximum != 10*time.Millisecond {
		t.Fatalf("unexpected latency stats: %+v", stats)
	}
	if stats.SampleCount != 4 || stats.BucketCount != latencyBucketCount {
		t.Fatalf("histogram metadata = %d samples, %d buckets; want 4, %d",
			stats.SampleCount, stats.BucketCount, latencyBucketCount)
	}
}

func TestLatencyHistogramUsesFixedBucketStorage(t *testing.T) {
	var histogram latencyHistogram
	for sample := 0; sample < 1_000_000; sample++ {
		histogram.record(time.Duration(sample) * time.Nanosecond)
	}
	if len(histogram.buckets) != latencyBucketCount {
		t.Fatalf("histogram has %d buckets, want %d", len(histogram.buckets), latencyBucketCount)
	}
	stats := histogram.stats()
	if stats.SampleCount != 1_000_000 || stats.BucketCount != latencyBucketCount {
		t.Fatalf("stats = %d samples, %d buckets", stats.SampleCount, stats.BucketCount)
	}
	if stats.P50 > stats.P95 || stats.P95 > stats.P99 || stats.P99 > stats.Maximum {
		t.Fatalf("percentiles are not ordered: %+v", stats)
	}
}

func TestMeasurementJSONIncludesDerivedMetrics(t *testing.T) {
	measurement := Measurement{
		Name:       "test",
		Bytes:      2 * 1024 * 1024,
		Operations: 4,
		Duration:   time.Second,
	}
	data, err := json.Marshal(measurement)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["mib_per_second"] != float64(2) || decoded["iops"] != float64(4) {
		t.Fatalf("derived JSON metrics = %v MiB/s, %v IOPS; want 2 and 4",
			decoded["mib_per_second"], decoded["iops"])
	}
}
