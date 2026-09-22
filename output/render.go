package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"diskbenchmark/benchmark"
	"diskbenchmark/units"
)

type Format string

const (
	Text Format = "text"
	JSON Format = "json"
	CSV  Format = "csv"
)

func ParseFormat(value string) (Format, error) {
	format := Format(value)
	switch format {
	case Text, JSON, CSV:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported output format %q; use text, json, or csv", value)
	}
}

func Render(writer io.Writer, format Format, report benchmark.Report) error {
	switch format {
	case Text:
		return renderText(writer, report)
	case JSON:
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return fmt.Errorf("encode JSON report: %w", err)
		}
		return nil
	case CSV:
		return renderCSV(writer, report)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func renderText(writer io.Writer, report benchmark.Report) error {
	if _, err := fmt.Fprintln(writer, "Disk Benchmark"); err != nil {
		return err
	}
	lines := []struct {
		label string
		value any
	}{
		{label: "Path:", value: report.Config.Directory},
		{label: "Filesystem:", value: report.Environment.Filesystem},
		{label: "Data size:", value: units.FormatBytes(report.Config.Size)},
		{label: "Block size:", value: units.FormatBytes(int64(report.Config.BlockSize))},
		{label: "I/O mode:", value: report.Config.IOMode},
		{label: "I/O alignment:", value: units.FormatBytes(int64(report.Config.IOAlignment))},
		{label: "Cache control:", value: report.Config.CacheControl},
		{label: "Warm-ups:", value: report.WarmupsCompleted},
		{label: "Verify data:", value: report.Config.VerifyData},
		{label: "Duration:", value: report.Config.Duration},
		{label: "Suite:", value: report.Config.SuiteFile},
		{label: "Suite version:", value: report.Config.SuiteVersion},
		{label: "Free before:", value: units.FormatBytes(report.Environment.AvailableBytesBefore)},
		{label: "Free after:", value: units.FormatBytes(report.Environment.AvailableBytesAfter)},
		{label: "Workers:", value: report.Config.Workers},
		{label: "Queue depth:", value: report.Config.QueueDepth},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(writer, "%-12s %v\n", line.label, line.value); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(writer); err != nil {
		return err
	}

	for _, iteration := range report.Iterations {
		if len(report.Iterations) > 1 {
			if _, err := fmt.Fprintf(writer, "Iteration %d\n", iteration.Number); err != nil {
				return err
			}
		}
		for _, measurement := range iteration.Measurements {
			if _, err := fmt.Fprintf(writer,
				"%-18s %10.2f MiB/s  %10.2f IOPS  (%s, %d operations)\n",
				measurement.Name+":",
				measurement.MiBPerSecond(),
				measurement.IOPS(),
				formatDuration(measurement.Duration),
				measurement.Operations,
			); err != nil {
				return err
			}
			if measurement.TargetDuration > 0 {
				if _, err := fmt.Fprintf(
					writer,
					"  target duration %s\n",
					formatDuration(measurement.TargetDuration),
				); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(writer,
				"  latency avg %s  p50 %s  p95 %s  p99 %s  max %s; reads %d, writes %d\n",
				formatDuration(measurement.AverageLatency()),
				formatDuration(measurement.Latency.P50),
				formatDuration(measurement.Latency.P95),
				formatDuration(measurement.Latency.P99),
				formatDuration(measurement.Latency.Maximum),
				measurement.Reads,
				measurement.Writes,
			); err != nil {
				return err
			}
			if measurement.CacheControl.Status != "" {
				if _, err := fmt.Fprintf(
					writer,
					"  cache control %s",
					measurement.CacheControl.Status,
				); err != nil {
					return err
				}
				if measurement.CacheControl.API != "" {
					if _, err := fmt.Fprintf(writer, " via %s", measurement.CacheControl.API); err != nil {
						return err
					}
				}
				if measurement.CacheControl.Detail != "" {
					if _, err := fmt.Fprintf(writer, ": %s", measurement.CacheControl.Detail); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintln(writer); err != nil {
					return err
				}
			}
		}
		if iteration.FilePath != "" {
			if _, err := fmt.Fprintf(writer, "Test file retained: %s\n", iteration.FilePath); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(
			writer,
			"File storage: %s logical, %s allocated, sparse=%t\n",
			units.FormatBytes(iteration.Storage.LogicalBytes),
			units.FormatBytes(iteration.Storage.AllocatedBytes),
			iteration.Storage.Sparse,
		); err != nil {
			return err
		}
		if iteration.Verification.Enabled {
			if _, err := fmt.Fprintf(
				writer,
				"Verification: passed=%t, %s checked in %s\n",
				iteration.Verification.Passed,
				units.FormatBytes(iteration.Verification.Bytes),
				formatDuration(iteration.Verification.Duration),
			); err != nil {
				return err
			}
		}
		if len(report.Iterations) > 1 {
			if _, err := fmt.Fprintln(writer); err != nil {
				return err
			}
		}
	}

	if len(report.Iterations) > 1 {
		if _, err := fmt.Fprintln(writer, "Summary"); err != nil {
			return err
		}
		for _, summary := range report.Summaries {
			if _, err := fmt.Fprintf(writer,
				"%-18s MiB/s min %10.2f avg %10.2f max %10.2f; IOPS min %.2f avg %.2f max %.2f\n",
				summary.Name+":",
				summary.MinimumMiBPerSec,
				summary.AverageMiBPerSec,
				summary.MaximumMiBPerSec,
				summary.MinimumIOPS,
				summary.AverageIOPS,
				summary.MaximumIOPS,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderCSV(writer io.Writer, report benchmark.Report) error {
	csvWriter := csv.NewWriter(writer)
	header := []string{
		"record_type", "started_at", "hostname", "operating_system", "architecture",
		"cpus", "go_version", "filesystem", "volume", "directory", "size_bytes",
		"block_size_bytes", "iterations", "warmup_iterations", "warmups_completed",
		"verify_data", "configured_duration_ns", "suite_file", "suite_version",
		"available_bytes_before",
		"available_bytes_after", "workers", "queue_depth",
		"random_read_percent", "random_seed", "io_mode", "io_alignment_bytes",
		"cache_control", "iteration", "workload", "bytes",
		"operations", "reads", "writes", "duration_ns", "target_duration_ns",
		"mib_per_second", "iops",
		"latency_average_ns", "latency_p50_ns", "latency_p95_ns", "latency_p99_ns",
		"latency_maximum_ns", "latency_sample_count", "latency_bucket_count",
		"cache_control_api", "cache_control_status", "cache_control_detail",
		"file_logical_bytes", "file_allocated_bytes", "file_sparse",
		"verification_enabled", "verification_passed", "verification_bytes",
		"verification_duration_ns",
		"minimum_mib_per_second", "average_mib_per_second",
		"maximum_mib_per_second", "minimum_iops", "average_iops", "maximum_iops",
		"file_path",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	writeRecord := func(values map[string]string) error {
		record := make([]string, len(header))
		for index, name := range header {
			record[index] = values[name]
		}
		return csvWriter.Write(record)
	}
	base := csvBase(report)
	for _, iteration := range report.Iterations {
		for _, measurement := range iteration.Measurements {
			values := cloneValues(base)
			values["record_type"] = "measurement"
			values["iteration"] = strconv.Itoa(iteration.Number)
			values["workload"] = measurement.Name
			values["bytes"] = strconv.FormatInt(measurement.Bytes, 10)
			values["operations"] = strconv.FormatInt(measurement.Operations, 10)
			values["reads"] = strconv.FormatInt(measurement.Reads, 10)
			values["writes"] = strconv.FormatInt(measurement.Writes, 10)
			values["duration_ns"] = strconv.FormatInt(int64(measurement.Duration), 10)
			values["target_duration_ns"] = strconv.FormatInt(int64(measurement.TargetDuration), 10)
			values["mib_per_second"] = formatFloat(measurement.MiBPerSecond())
			values["iops"] = formatFloat(measurement.IOPS())
			values["latency_average_ns"] = strconv.FormatInt(int64(measurement.AverageLatency()), 10)
			values["latency_p50_ns"] = strconv.FormatInt(int64(measurement.Latency.P50), 10)
			values["latency_p95_ns"] = strconv.FormatInt(int64(measurement.Latency.P95), 10)
			values["latency_p99_ns"] = strconv.FormatInt(int64(measurement.Latency.P99), 10)
			values["latency_maximum_ns"] = strconv.FormatInt(int64(measurement.Latency.Maximum), 10)
			values["latency_sample_count"] = strconv.FormatUint(measurement.Latency.SampleCount, 10)
			values["latency_bucket_count"] = strconv.Itoa(measurement.Latency.BucketCount)
			values["cache_control_api"] = measurement.CacheControl.API
			values["cache_control_status"] = measurement.CacheControl.Status
			values["cache_control_detail"] = measurement.CacheControl.Detail
			values["file_logical_bytes"] = strconv.FormatInt(iteration.Storage.LogicalBytes, 10)
			values["file_allocated_bytes"] = strconv.FormatInt(iteration.Storage.AllocatedBytes, 10)
			values["file_sparse"] = strconv.FormatBool(iteration.Storage.Sparse)
			values["verification_enabled"] = strconv.FormatBool(iteration.Verification.Enabled)
			values["verification_passed"] = strconv.FormatBool(iteration.Verification.Passed)
			values["verification_bytes"] = strconv.FormatInt(iteration.Verification.Bytes, 10)
			values["verification_duration_ns"] = strconv.FormatInt(int64(iteration.Verification.Duration), 10)
			values["file_path"] = iteration.FilePath
			if err := writeRecord(values); err != nil {
				return fmt.Errorf("write CSV measurement: %w", err)
			}
		}
	}
	for _, summary := range report.Summaries {
		values := cloneValues(base)
		values["record_type"] = "summary"
		values["workload"] = summary.Name
		values["minimum_mib_per_second"] = formatFloat(summary.MinimumMiBPerSec)
		values["average_mib_per_second"] = formatFloat(summary.AverageMiBPerSec)
		values["maximum_mib_per_second"] = formatFloat(summary.MaximumMiBPerSec)
		values["minimum_iops"] = formatFloat(summary.MinimumIOPS)
		values["average_iops"] = formatFloat(summary.AverageIOPS)
		values["maximum_iops"] = formatFloat(summary.MaximumIOPS)
		if err := writeRecord(values); err != nil {
			return fmt.Errorf("write CSV summary: %w", err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush CSV report: %w", err)
	}
	return nil
}

func csvBase(report benchmark.Report) map[string]string {
	return map[string]string{
		"started_at":             report.StartedAt.Format(time.RFC3339Nano),
		"hostname":               report.Environment.Hostname,
		"operating_system":       report.Environment.OperatingSystem,
		"architecture":           report.Environment.Architecture,
		"cpus":                   strconv.Itoa(report.Environment.CPUs),
		"go_version":             report.Environment.GoVersion,
		"filesystem":             report.Environment.Filesystem,
		"volume":                 report.Environment.Volume,
		"directory":              report.Config.Directory,
		"size_bytes":             strconv.FormatInt(report.Config.Size, 10),
		"block_size_bytes":       strconv.Itoa(report.Config.BlockSize),
		"iterations":             strconv.Itoa(report.Config.Iterations),
		"warmup_iterations":      strconv.Itoa(report.Config.WarmupIterations),
		"warmups_completed":      strconv.Itoa(report.WarmupsCompleted),
		"verify_data":            strconv.FormatBool(report.Config.VerifyData),
		"configured_duration_ns": strconv.FormatInt(int64(report.Config.Duration), 10),
		"suite_file":             report.Config.SuiteFile,
		"suite_version":          strconv.Itoa(report.Config.SuiteVersion),
		"available_bytes_before": strconv.FormatInt(report.Environment.AvailableBytesBefore, 10),
		"available_bytes_after":  strconv.FormatInt(report.Environment.AvailableBytesAfter, 10),
		"workers":                strconv.Itoa(report.Config.Workers),
		"queue_depth":            strconv.Itoa(report.Config.QueueDepth),
		"random_read_percent":    strconv.Itoa(report.Config.RandomReadPercent),
		"random_seed":            strconv.FormatInt(report.Config.RandomSeed, 10),
		"io_mode":                string(report.Config.IOMode),
		"io_alignment_bytes":     strconv.Itoa(report.Config.IOAlignment),
		"cache_control":          string(report.Config.CacheControl),
	}
}

func cloneValues(values map[string]string) map[string]string {
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func formatDuration(duration time.Duration) string {
	if duration >= time.Second {
		return duration.Round(time.Millisecond).String()
	}
	if duration >= time.Millisecond {
		return duration.Round(time.Microsecond).String()
	}
	return duration.Round(time.Nanosecond).String()
}
