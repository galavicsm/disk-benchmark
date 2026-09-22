package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"diskbenchmark/benchmark"
	"diskbenchmark/output"
	"diskbenchmark/units"
	"diskbenchmark/workloads"
)

const (
	defaultSize      = "1GiB"
	defaultBlockSize = "1MiB"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "diskbenchmark: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("diskbenchmark", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	directory := flags.String("path", ".", "directory in which to create the temporary test file")
	sizeValue := flags.String("size", defaultSize, "total data size (for example 1GiB)")
	blockSizeValue := flags.String("block-size", defaultBlockSize, "I/O block size (for example 1MiB)")
	iterations := flags.Int("iterations", 1, "number of benchmark iterations")
	warmupIterations := flags.Int("warmup-iterations", 0, "number of unmeasured warm-up iterations")
	keepFile := flags.Bool("keep-file", false, "keep generated test files")
	verifyData := flags.Bool("verify-data", false, "verify test file contents after measured workloads")
	syncWrites := flags.Bool("sync", true, "flush written data before completing write measurements")
	workloadValue := flags.String(
		"workload",
		"sequential",
		"workloads to run: sequential, random, mixed, overwrite, fsync, or all",
	)
	suiteFile := flags.String("suite", "", "path to a versioned JSON workload suite")
	duration := flags.Duration("duration", 0, "duration for each workload; zero processes the data size once")
	workers := flags.Int("workers", 1, "number of workers assigned to separate file regions")
	queueDepth := flags.Int("queue-depth", 1, "maximum concurrent operations per worker")
	randomReadPercent := flags.Int("random-read-percent", 50, "percentage of random operations that are reads")
	randomSeed := flags.Int64("random-seed", 1, "seed used to generate the random operation plan")
	ioModeValue := flags.String("io-mode", "buffered", "I/O mode: buffered or direct")
	cacheControlValue := flags.String("cache-control", "off", "cache policy before reads: off, attempt, or require")
	outputValue := flags.String("output", "text", "output format: text, json, or csv")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	size, err := units.ParseBytes(*sizeValue)
	if err != nil {
		return fmt.Errorf("invalid -size: %w", err)
	}
	blockSize, err := units.ParseBytes(*blockSizeValue)
	if err != nil {
		return fmt.Errorf("invalid -block-size: %w", err)
	}
	maxInt := int64(^uint(0) >> 1)
	if blockSize > maxInt {
		return fmt.Errorf("invalid -block-size: value is too large for this platform")
	}
	format, err := output.ParseFormat(strings.ToLower(*outputValue))
	if err != nil {
		return err
	}
	ioMode, err := benchmark.ParseIOMode(*ioModeValue)
	if err != nil {
		return err
	}
	cacheControl, err := benchmark.ParseCacheControlMode(*cacheControlValue)
	if err != nil {
		return err
	}
	workloadExplicit := false
	flags.Visit(func(current *flag.Flag) {
		if current.Name == "workload" {
			workloadExplicit = true
		}
	})
	builtInWorkload := strings.ToLower(*workloadValue)
	if *suiteFile != "" {
		if workloadExplicit {
			return fmt.Errorf("-suite and -workload cannot be used together")
		}
		builtInWorkload = ""
	}
	selection, err := workloads.Select(workloads.Selection{
		BuiltIn:   builtInWorkload,
		SuiteFile: *suiteFile,
	})
	if err != nil {
		return err
	}

	config := benchmark.Config{
		Directory:         *directory,
		Size:              size,
		BlockSize:         int(blockSize),
		Iterations:        *iterations,
		WarmupIterations:  *warmupIterations,
		KeepFile:          *keepFile,
		VerifyData:        *verifyData,
		Sync:              *syncWrites,
		Workers:           *workers,
		QueueDepth:        *queueDepth,
		RandomReadPercent: *randomReadPercent,
		RandomSeed:        *randomSeed,
		IOMode:            ioMode,
		CacheControl:      cacheControl,
		Duration:          *duration,
		SuiteFile:         *suiteFile,
		SuiteVersion:      selection.SuiteVersion,
	}

	fmt.Fprintln(os.Stderr, "WARNING: This benchmark writes data and may affect SSD endurance.")
	report, err := benchmark.NewRunner(selection.Workloads...).Run(ctx, config)
	if err != nil {
		return err
	}
	if err := output.Render(os.Stdout, format, report); err != nil {
		return err
	}
	return nil
}
