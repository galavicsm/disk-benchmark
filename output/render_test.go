package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"diskbenchmark/benchmark"
)

func testReport() benchmark.Report {
	measurement := benchmark.Measurement{
		Name:           "Random mixed",
		Bytes:          4096,
		Operations:     4,
		Reads:          2,
		Writes:         2,
		Duration:       time.Millisecond,
		TargetDuration: 5 * time.Millisecond,
		Latency: benchmark.LatencyStats{
			Average:     100 * time.Microsecond,
			P50:         90 * time.Microsecond,
			P95:         150 * time.Microsecond,
			P99:         150 * time.Microsecond,
			Maximum:     150 * time.Microsecond,
			SampleCount: 4,
			BucketCount: 2017,
		},
		CacheControl: benchmark.CacheControlResult{
			Mode:   benchmark.CacheControlAttempt,
			API:    "test_api",
			Status: "applied",
		},
	}
	return benchmark.Report{
		StartedAt: time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC),
		Config: benchmark.Config{
			Directory:         `C:\Temp`,
			Size:              4096,
			BlockSize:         1024,
			Iterations:        1,
			Workers:           2,
			QueueDepth:        2,
			RandomReadPercent: 50,
			RandomSeed:        1,
			IOMode:            benchmark.DirectIO,
			IOAlignment:       4096,
			CacheControl:      benchmark.CacheControlAttempt,
			WarmupIterations:  1,
			VerifyData:        true,
			Duration:          5 * time.Millisecond,
			SuiteFile:         "suite.json",
			SuiteVersion:      1,
		},
		Environment: benchmark.Environment{
			Hostname:             "host",
			OperatingSystem:      "windows",
			Architecture:         "amd64",
			CPUs:                 8,
			GoVersion:            "go1.23",
			Filesystem:           "NTFS",
			Volume:               `C:\`,
			AvailableBytesBefore: 10 * 1024 * 1024,
			AvailableBytesAfter:  9 * 1024 * 1024,
		},
		Iterations: []benchmark.IterationResult{{
			Number:       1,
			Measurements: []benchmark.Measurement{measurement},
			Storage: benchmark.FileStorage{
				LogicalBytes:   4096,
				AllocatedBytes: 4096,
			},
			Verification: benchmark.VerificationResult{
				Enabled:  true,
				Passed:   true,
				Bytes:    4096,
				Duration: time.Millisecond,
			},
		}},
		Summaries: []benchmark.Summary{{
			Name:             "Random mixed",
			MinimumMiBPerSec: measurement.MiBPerSecond(),
			MaximumMiBPerSec: measurement.MiBPerSecond(),
			AverageMiBPerSec: measurement.MiBPerSecond(),
			MinimumIOPS:      measurement.IOPS(),
			MaximumIOPS:      measurement.IOPS(),
			AverageIOPS:      measurement.IOPS(),
		}},
		WarmupsCompleted: 1,
	}
}

func TestRenderJSON(t *testing.T) {
	var buffer bytes.Buffer
	if err := Render(&buffer, JSON, testReport()); err != nil {
		t.Fatalf("Render() returned error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded["environment"].(map[string]any)["filesystem"] != "NTFS" {
		t.Fatalf("filesystem metadata missing from JSON: %s", buffer.String())
	}
	config := decoded["config"].(map[string]any)
	if config["cache_control"] != "attempt" {
		t.Fatalf("cache-control configuration missing from JSON: %s", buffer.String())
	}
	if config["duration_ns"] != float64(5*time.Millisecond) ||
		config["suite_version"] != float64(1) {
		t.Fatalf("suite configuration missing from JSON: %s", buffer.String())
	}
	iteration := decoded["iterations"].([]any)[0].(map[string]any)
	if iteration["verification"].(map[string]any)["passed"] != true ||
		iteration["storage"].(map[string]any)["allocated_bytes"] != float64(4096) {
		t.Fatalf("safety metadata missing from JSON: %s", buffer.String())
	}
}

func TestRenderCSVHasConsistentRecords(t *testing.T) {
	var buffer bytes.Buffer
	if err := Render(&buffer, CSV, testReport()); err != nil {
		t.Fatalf("Render() returned error: %v", err)
	}
	records, err := csv.NewReader(strings.NewReader(buffer.String())).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d CSV records, want header, measurement, and summary", len(records))
	}
	for index, record := range records[1:] {
		if len(record) != len(records[0]) {
			t.Fatalf("record %d has %d fields, header has %d", index+1, len(record), len(records[0]))
		}
	}
	if records[1][0] != "measurement" || records[2][0] != "summary" {
		t.Fatalf("unexpected record types: %q, %q", records[1][0], records[2][0])
	}
	columns := make(map[string]string, len(records[0]))
	for index, name := range records[0] {
		columns[name] = records[1][index]
	}
	if columns["cache_control_status"] != "applied" ||
		columns["latency_sample_count"] != "4" ||
		columns["latency_bucket_count"] != "2017" {
		t.Fatalf("CSV cache/histogram fields are incomplete: %v", columns)
	}
	if columns["warmups_completed"] != "1" ||
		columns["verification_passed"] != "true" ||
		columns["file_allocated_bytes"] != "4096" {
		t.Fatalf("CSV safety fields are incomplete: %v", columns)
	}
	if columns["configured_duration_ns"] != "5000000" ||
		columns["target_duration_ns"] != "5000000" ||
		columns["suite_version"] != "1" {
		t.Fatalf("CSV suite fields are incomplete: %v", columns)
	}
}
