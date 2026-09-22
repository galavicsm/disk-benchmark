package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSequentialWorkloadsHandlePartialFinalBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.bin")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	config := Config{Size: 2500, BlockSize: 1024}
	buffer := make([]byte, config.BlockSize)
	fillDeterministic(buffer)

	writeResult, err := (SequentialWrite{}).Run(context.Background(), file, buffer, config)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if writeResult.Bytes != config.Size || writeResult.Operations != 3 {
		t.Fatalf("write result = %d bytes, %d operations; want %d bytes, 3 operations",
			writeResult.Bytes, writeResult.Operations, config.Size)
	}

	readResult, err := (SequentialRead{}).Run(context.Background(), file, buffer, config)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if readResult.Bytes != config.Size || readResult.Operations != 3 {
		t.Fatalf("read result = %d bytes, %d operations; want %d bytes, 3 operations",
			readResult.Bytes, readResult.Operations, config.Size)
	}

	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != config.Size {
		t.Fatalf("file size = %d, want %d", info.Size(), config.Size)
	}
}

func TestRandomPlanIsDeterministicAndPreservesMix(t *testing.T) {
	config := Config{
		Size:              10*1024 + 17,
		BlockSize:         1024,
		Workers:           3,
		QueueDepth:        2,
		RandomReadPercent: 60,
		RandomSeed:        42,
	}
	first := randomPlan(config)
	second := randomPlan(config)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("randomPlan() returned different plans for the same seed")
	}

	var operations, reads, bytes int64
	seenOffsets := make(map[int64]bool)
	for worker, plan := range first {
		for _, operation := range plan {
			operations++
			bytes += int64(operation.size)
			if !operation.write {
				reads++
			}
			if seenOffsets[operation.offset] {
				t.Fatalf("offset %d occurs more than once", operation.offset)
			}
			seenOffsets[operation.offset] = true
			expectedWorker := int((operation.offset / int64(config.BlockSize)) * int64(config.Workers) / blockCount(config))
			if worker != expectedWorker {
				t.Fatalf("offset %d assigned to worker %d, want %d", operation.offset, worker, expectedWorker)
			}
		}
	}
	if operations != 11 || reads != 7 || bytes != config.Size {
		t.Fatalf("plan = %d operations, %d reads, %d bytes; want 11, 7, %d",
			operations, reads, bytes, config.Size)
	}
}

func TestRandomMixedRunsConcurrently(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "random.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	config := Config{
		Size:              12*1024 + 11,
		BlockSize:         1024,
		Workers:           3,
		QueueDepth:        2,
		RandomReadPercent: 40,
		RandomSeed:        7,
	}
	buffer := make([]byte, config.BlockSize)
	fillDeterministic(buffer)
	workload := RandomMixed{}
	if err := workload.Prepare(context.Background(), file, buffer, config); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}
	measurement, err := workload.Run(context.Background(), file, buffer, config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if measurement.Bytes != config.Size || measurement.Operations != 13 {
		t.Fatalf("measurement = %d bytes, %d operations; want %d bytes, 13 operations",
			measurement.Bytes, measurement.Operations, config.Size)
	}
	if measurement.Reads != 5 || measurement.Writes != 8 {
		t.Fatalf("measurement = %d reads, %d writes; want 5 reads, 8 writes",
			measurement.Reads, measurement.Writes)
	}
	if measurement.Latency.Maximum < measurement.Latency.P99 ||
		measurement.Latency.P99 < measurement.Latency.P95 ||
		measurement.Latency.P95 < measurement.Latency.P50 {
		t.Fatalf("latency percentiles are not ordered: %+v", measurement.Latency)
	}
}

func TestProcessBlocksHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	processed, operations, err := processBlocks(ctx, 1024, 256, func(int) error {
		called = true
		return nil
	})
	if err != context.Canceled {
		t.Fatalf("processBlocks() error = %v, want context.Canceled", err)
	}
	if called || processed != 0 || operations != 0 {
		t.Fatalf("operation ran after cancellation: called=%v bytes=%d operations=%d", called, processed, operations)
	}
}

func TestSequentialOverwritePreservesFileSize(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "overwrite.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	config := Config{Size: 4096, BlockSize: 1024, Workers: 2, QueueDepth: 2}
	buffer := make([]byte, config.BlockSize)
	fillDeterministic(buffer)
	workload := SequentialOverwrite{}
	if err := workload.Prepare(context.Background(), file, buffer, config); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}
	measurement, err := workload.Run(context.Background(), file, buffer, config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if measurement.Bytes != config.Size || measurement.Writes != 4 {
		t.Fatalf("measurement = %d bytes, %d writes; want %d and 4",
			measurement.Bytes, measurement.Writes, config.Size)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != config.Size {
		t.Fatalf("file size = %d, want %d", info.Size(), config.Size)
	}
}

func TestFsyncPlanSynchronizesEveryWrite(t *testing.T) {
	plans := fsyncPlan(Config{Size: 4096, BlockSize: 1024, Workers: 2})
	var operations int
	for _, plan := range plans {
		for _, operation := range plan {
			operations++
			if !operation.write || !operation.sync {
				t.Fatalf("operation is not a synchronized write: %+v", operation)
			}
		}
	}
	if operations != 4 {
		t.Fatalf("got %d operations, want 4", operations)
	}
}

func TestDurationRepeatsWorkingSet(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "duration.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	config := Config{
		Size:       4096,
		BlockSize:  1024,
		Workers:    1,
		QueueDepth: 1,
		Duration:   10 * time.Millisecond,
	}
	buffer := make([]byte, config.BlockSize)
	fillDeterministic(buffer)
	measurement, err := (SequentialWrite{}).Run(context.Background(), file, buffer, config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if measurement.Operations <= 4 || measurement.Bytes <= config.Size {
		t.Fatalf("timed workload did not repeat: %d operations, %d bytes",
			measurement.Operations, measurement.Bytes)
	}
	if measurement.Duration < config.Duration {
		t.Fatalf("duration = %s, want at least %s", measurement.Duration, config.Duration)
	}
}

func TestDurationCompletesInitialWorkingSetPass(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "short-duration.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	config := Config{
		Size:       4096,
		BlockSize:  1024,
		Workers:    1,
		QueueDepth: 1,
		Duration:   time.Nanosecond,
	}
	buffer := make([]byte, config.BlockSize)
	fillDeterministic(buffer)
	measurement, err := (SequentialWrite{}).Run(context.Background(), file, buffer, config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if measurement.Operations < 4 || measurement.Bytes < config.Size {
		t.Fatalf("initial pass = %d operations, %d bytes; want at least 4 and %d",
			measurement.Operations, measurement.Bytes, config.Size)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != config.Size {
		t.Fatalf("file size = %d, want %d", info.Size(), config.Size)
	}
}
