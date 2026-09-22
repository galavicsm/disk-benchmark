package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

func TestParseIOMode(t *testing.T) {
	for _, value := range []string{"buffered", "BUFFERED", "direct", "DIRECT"} {
		if _, err := ParseIOMode(value); err != nil {
			t.Fatalf("ParseIOMode(%q) returned error: %v", value, err)
		}
	}
	if _, err := ParseIOMode("invalid"); err == nil {
		t.Fatal("ParseIOMode(invalid) returned no error")
	}
}

func TestMakeIOBufferAlignment(t *testing.T) {
	for _, alignment := range []int{1, 512, 4096} {
		buffer := makeIOBuffer(8192, alignment)
		if len(buffer) != 8192 {
			t.Fatalf("alignment %d produced length %d, want 8192", alignment, len(buffer))
		}
		if uintptr(unsafe.Pointer(&buffer[0]))%uintptr(alignment) != 0 {
			t.Fatalf("buffer address is not aligned to %d bytes", alignment)
		}
		for index := range buffer {
			buffer[index] = byte(index)
		}
		clone := cloneIOBuffer(buffer, alignment)
		if uintptr(unsafe.Pointer(&clone[0]))%uintptr(alignment) != 0 {
			t.Fatalf("clone address is not aligned to %d bytes", alignment)
		}
		for index := range clone {
			if clone[index] != byte(index) {
				t.Fatalf("clone differs at byte %d", index)
			}
		}
	}
}

func TestConfigureDirectIOModeValidatesAlignment(t *testing.T) {
	directory := t.TempDir()
	alignment, err := directIOAlignment(directory)
	if err != nil {
		t.Fatalf("directIOAlignment() returned error: %v", err)
	}
	config := Config{
		Directory:         directory,
		Size:              int64(alignment * 4),
		BlockSize:         alignment,
		Iterations:        1,
		Workers:           1,
		QueueDepth:        1,
		RandomReadPercent: 50,
		IOMode:            DirectIO,
	}
	config, err = configureIOMode(config)
	if err != nil {
		t.Fatalf("configureIOMode() returned error: %v", err)
	}
	if config.IOAlignment != alignment {
		t.Fatalf("I/O alignment = %d, want %d", config.IOAlignment, alignment)
	}

	config.BlockSize = alignment + 1
	if _, err := configureIOMode(config); err == nil {
		t.Fatal("unaligned block size returned no error")
	}
	config.BlockSize = alignment
	config.Size++
	if _, err := configureIOMode(config); err == nil {
		t.Fatal("unaligned data size returned no error")
	}
}

func TestRunnerDirectIO(t *testing.T) {
	directory := t.TempDir()
	alignment, err := directIOAlignment(directory)
	if err != nil {
		t.Fatalf("directIOAlignment() returned error: %v", err)
	}
	blockSize := alignment * ((4096 + alignment - 1) / alignment)
	config := Config{
		Directory:         directory,
		Size:              int64(blockSize * 8),
		BlockSize:         blockSize,
		Iterations:        1,
		Sync:              true,
		Workers:           2,
		QueueDepth:        2,
		RandomReadPercent: 50,
		RandomSeed:        1,
		IOMode:            DirectIO,
		VerifyData:        true,
	}
	report, err := NewRunner(
		SequentialWrite{},
		SequentialRead{},
		RandomMixed{},
		SequentialOverwrite{},
		FsyncWrite{},
	).Run(context.Background(), config)
	if err != nil {
		t.Fatalf("direct I/O benchmark returned error: %v", err)
	}
	if report.Config.IOAlignment != alignment {
		t.Fatalf("report alignment = %d, want %d", report.Config.IOAlignment, alignment)
	}
	if len(report.Iterations[0].Measurements) != 5 {
		t.Fatalf("got %d measurements, want 5", len(report.Iterations[0].Measurements))
	}
	if !report.Iterations[0].Verification.Passed ||
		report.Iterations[0].Verification.Bytes != config.Size {
		t.Fatalf("direct I/O verification failed: %+v", report.Iterations[0].Verification)
	}
	matches, err := filepath.Glob(filepath.Join(directory, "diskbenchmark-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("direct I/O temporary files were not removed: %v", matches)
	}
}

func TestCreateDirectFileCanBeRetained(t *testing.T) {
	directory := t.TempDir()
	alignment, err := directIOAlignment(directory)
	if err != nil {
		t.Fatal(err)
	}
	config := Config{
		Directory:   directory,
		Size:        int64(alignment),
		BlockSize:   alignment,
		Iterations:  1,
		KeepFile:    true,
		Workers:     1,
		QueueDepth:  1,
		IOMode:      DirectIO,
		IOAlignment: alignment,
	}
	report, err := NewRunner(SequentialWrite{}).Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	path := report.Iterations[0].FilePath
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("retained direct I/O file is unavailable: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove retained file: %v", err)
	}
}
