package benchmark

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"
)

type Workload interface {
	Name() string
	Run(context.Context, *os.File, []byte, Config) (Measurement, error)
}

type workloadPreparer interface {
	Prepare(context.Context, *os.File, []byte, Config) error
}

type SequentialWrite struct{}

func (SequentialWrite) Name() string {
	return "Sequential write"
}

func (w SequentialWrite) Run(ctx context.Context, file *os.File, buffer []byte, config Config) (Measurement, error) {
	if err := file.Truncate(0); err != nil {
		return Measurement{}, fmt.Errorf("truncate test file: %w", err)
	}
	measurement, err := executeOperations(ctx, file, buffer, config, sequentialPlan(config, true), w.Name())
	if err != nil {
		return Measurement{}, fmt.Errorf("write test file: %w", err)
	}
	if config.Sync {
		start := time.Now()
		if err := file.Sync(); err != nil {
			return Measurement{}, fmt.Errorf("sync test file: %w", err)
		}
		measurement.Duration += time.Since(start)
	}
	return measurement, nil
}

type SequentialRead struct{}

func (SequentialRead) Name() string {
	return "Sequential read"
}

func (SequentialRead) Prepare(ctx context.Context, file *os.File, buffer []byte, config Config) error {
	return preparePopulatedFile(ctx, file, buffer, config)
}

func (w SequentialRead) Run(ctx context.Context, file *os.File, buffer []byte, config Config) (Measurement, error) {
	cacheControl, err := applyCacheControl(file, config)
	if err != nil {
		return Measurement{}, err
	}
	measurement, err := executeOperations(ctx, file, buffer, config, sequentialPlan(config, false), w.Name())
	if err != nil {
		return Measurement{}, fmt.Errorf("read test file: %w", err)
	}
	measurement.CacheControl = cacheControl
	return measurement, nil
}

type RandomMixed struct {
	ReadPercent *int
	Seed        *int64
}

func (w RandomMixed) Name() string {
	if w.ReadPercent != nil && w.Seed != nil {
		return fmt.Sprintf("Random mixed (%d%% read, seed %d)", *w.ReadPercent, *w.Seed)
	}
	if w.ReadPercent != nil {
		return fmt.Sprintf("Random mixed (%d%% read)", *w.ReadPercent)
	}
	if w.Seed != nil {
		return fmt.Sprintf("Random mixed (seed %d)", *w.Seed)
	}
	return "Random mixed"
}

func (w RandomMixed) Prepare(ctx context.Context, file *os.File, buffer []byte, config Config) error {
	config = w.apply(config)
	return preparePopulatedFile(ctx, file, buffer, config)
}

func preparePopulatedFile(ctx context.Context, file *os.File, buffer []byte, config Config) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect test file: %w", err)
	}
	if info.Size() == config.Size {
		return nil
	}
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("truncate test file: %w", err)
	}
	preparationConfig := config
	preparationConfig.Duration = 0
	if _, err := executeOperations(
		ctx,
		file,
		buffer,
		preparationConfig,
		sequentialPlan(preparationConfig, true),
		"File preparation",
	); err != nil {
		return fmt.Errorf("populate test file: %w", err)
	}
	if config.Sync {
		if err := file.Sync(); err != nil {
			return fmt.Errorf("sync test file: %w", err)
		}
	}
	return nil
}

func (w RandomMixed) Run(ctx context.Context, file *os.File, buffer []byte, config Config) (Measurement, error) {
	config = w.apply(config)
	cacheControl := cacheControlNotApplicable(config, "workload contains no read operations")
	var err error
	if config.RandomReadPercent > 0 {
		cacheControl, err = applyCacheControl(file, config)
		if err != nil {
			return Measurement{}, err
		}
	}
	measurement, err := executeOperations(ctx, file, buffer, config, randomPlan(config), w.Name())
	if err != nil {
		return Measurement{}, fmt.Errorf("access test file: %w", err)
	}
	if config.Sync && measurement.Writes > 0 {
		start := time.Now()
		if err := file.Sync(); err != nil {
			return Measurement{}, fmt.Errorf("sync test file: %w", err)
		}
		measurement.Duration += time.Since(start)
	}
	measurement.CacheControl = cacheControl
	return measurement, nil
}

func (w RandomMixed) apply(config Config) Config {
	if w.ReadPercent != nil {
		config.RandomReadPercent = *w.ReadPercent
	}
	if w.Seed != nil {
		config.RandomSeed = *w.Seed
	}
	return config
}

type SequentialOverwrite struct{}

func (SequentialOverwrite) Name() string {
	return "Sequential overwrite"
}

func (SequentialOverwrite) Prepare(ctx context.Context, file *os.File, buffer []byte, config Config) error {
	return preparePopulatedFile(ctx, file, buffer, config)
}

func (w SequentialOverwrite) Run(
	ctx context.Context,
	file *os.File,
	buffer []byte,
	config Config,
) (Measurement, error) {
	measurement, err := executeOperations(ctx, file, buffer, config, sequentialPlan(config, true), w.Name())
	if err != nil {
		return Measurement{}, fmt.Errorf("overwrite test file: %w", err)
	}
	if config.Sync {
		start := time.Now()
		if err := file.Sync(); err != nil {
			return Measurement{}, fmt.Errorf("sync test file: %w", err)
		}
		measurement.Duration += time.Since(start)
	}
	return measurement, nil
}

type FsyncWrite struct{}

func (FsyncWrite) Name() string {
	return "Fsync-per-block write"
}

func (w FsyncWrite) Run(
	ctx context.Context,
	file *os.File,
	buffer []byte,
	config Config,
) (Measurement, error) {
	if err := file.Truncate(0); err != nil {
		return Measurement{}, fmt.Errorf("truncate test file: %w", err)
	}
	measurement, err := executeOperations(ctx, file, buffer, config, fsyncPlan(config), w.Name())
	if err != nil {
		return Measurement{}, fmt.Errorf("fsync-per-block test file: %w", err)
	}
	return measurement, nil
}

type ConfiguredWorkload struct {
	Workload         Workload
	DurationOverride *time.Duration
	Label            string
}

func (w ConfiguredWorkload) Name() string {
	if w.Label != "" {
		return w.Label
	}
	if w.DurationOverride != nil && *w.DurationOverride > 0 {
		return fmt.Sprintf("%s (%s)", w.Workload.Name(), *w.DurationOverride)
	}
	return w.Workload.Name()
}

func (w ConfiguredWorkload) Prepare(
	ctx context.Context,
	file *os.File,
	buffer []byte,
	config Config,
) error {
	preparer, ok := w.Workload.(workloadPreparer)
	if !ok {
		return nil
	}
	return preparer.Prepare(ctx, file, buffer, w.apply(config))
}

func (w ConfiguredWorkload) Run(
	ctx context.Context,
	file *os.File,
	buffer []byte,
	config Config,
) (Measurement, error) {
	measurement, err := w.Workload.Run(ctx, file, buffer, w.apply(config))
	if err != nil {
		return Measurement{}, err
	}
	measurement.Name = w.Name()
	return measurement, nil
}

func (w ConfiguredWorkload) apply(config Config) Config {
	if w.DurationOverride != nil {
		config.Duration = *w.DurationOverride
	}
	return config
}

type operation struct {
	offset int64
	size   int
	write  bool
	sync   bool
}

type operationResult struct {
	bytes   int
	write   bool
	latency time.Duration
	err     error
}

func sequentialPlan(config Config, write bool) [][]operation {
	config = config.normalized()
	plans := make([][]operation, config.Workers)
	blockCount := blockCount(config)
	for block := int64(0); block < blockCount; block++ {
		offset := block * int64(config.BlockSize)
		size := config.BlockSize
		if remaining := config.Size - offset; remaining < int64(size) {
			size = int(remaining)
		}
		worker := int(block * int64(config.Workers) / blockCount)
		plans[worker] = append(plans[worker], operation{offset: offset, size: size, write: write})
	}
	return plans
}

func randomPlan(config Config) [][]operation {
	config = config.normalized()
	count := blockCount(config)
	plans := make([][]operation, config.Workers)
	random := rand.New(rand.NewSource(config.RandomSeed))
	order := random.Perm(int(count))

	readCount := int((count*int64(config.RandomReadPercent) + 50) / 100)
	readOperations := make([]bool, count)
	for index := 0; index < readCount; index++ {
		readOperations[index] = true
	}
	random.Shuffle(len(readOperations), func(i, j int) {
		readOperations[i], readOperations[j] = readOperations[j], readOperations[i]
	})

	for index, block := range order {
		offset := int64(block) * int64(config.BlockSize)
		size := config.BlockSize
		if remaining := config.Size - offset; remaining < int64(size) {
			size = int(remaining)
		}
		worker := int(int64(block) * int64(config.Workers) / count)
		plans[worker] = append(plans[worker], operation{
			offset: offset,
			size:   size,
			write:  !readOperations[index],
		})
	}
	return plans
}

func fsyncPlan(config Config) [][]operation {
	plans := sequentialPlan(config, true)
	for worker := range plans {
		for index := range plans[worker] {
			plans[worker][index].sync = true
		}
	}
	return plans
}

func executeOperations(
	ctx context.Context,
	file *os.File,
	template []byte,
	config Config,
	plans [][]operation,
	name string,
) (Measurement, error) {
	config = config.normalized()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan operationResult)
	var waitGroup sync.WaitGroup
	start := time.Now()
	deadline := start.Add(config.Duration)

	for _, workerPlan := range plans {
		for lane := 0; lane < config.QueueDepth; lane++ {
			laneOperations := operationsForLane(workerPlan, lane, config.QueueDepth)
			if len(laneOperations) == 0 {
				continue
			}
			waitGroup.Add(1)
			go func(operations []operation) {
				defer waitGroup.Done()
				buffer := cloneIOBuffer(template, config.IOAlignment)
				firstCycle := true
				for {
					for _, current := range operations {
						if err := ctx.Err(); err != nil {
							results <- operationResult{err: err}
							return
						}
						if !firstCycle && config.Duration > 0 && !time.Now().Before(deadline) {
							return
						}
						operationStart := time.Now()
						bytes, err := executeOperation(file, buffer, current)
						results <- operationResult{
							bytes:   bytes,
							write:   current.write,
							latency: time.Since(operationStart),
							err:     err,
						}
						if err != nil {
							cancel()
							return
						}
					}
					firstCycle = false
					if config.Duration <= 0 {
						return
					}
					if !time.Now().Before(deadline) {
						return
					}
				}
			}(laneOperations)
		}
	}

	var duration time.Duration
	go func() {
		waitGroup.Wait()
		duration = time.Since(start)
		close(results)
	}()

	measurement := Measurement{Name: name, TargetDuration: config.Duration}
	var histogram latencyHistogram
	var firstError error
	for result := range results {
		if result.err != nil {
			if firstError == nil {
				firstError = result.err
			}
			continue
		}
		measurement.Bytes += int64(result.bytes)
		measurement.Operations++
		if result.write {
			measurement.Writes++
		} else {
			measurement.Reads++
		}
		histogram.record(result.latency)
	}
	if firstError != nil {
		return Measurement{}, firstError
	}
	measurement.Duration = duration
	measurement.Latency = histogram.stats()
	return measurement, nil
}

func operationsForLane(operations []operation, lane, queueDepth int) []operation {
	count := (len(operations) + queueDepth - 1 - lane) / queueDepth
	if count <= 0 {
		return nil
	}
	result := make([]operation, 0, count)
	for index := lane; index < len(operations); index += queueDepth {
		result = append(result, operations[index])
	}
	return result
}

func executeOperation(file *os.File, buffer []byte, operation operation) (int, error) {
	if operation.write {
		written, err := file.WriteAt(buffer[:operation.size], operation.offset)
		if err != nil {
			return written, err
		}
		if written != operation.size {
			return written, io.ErrShortWrite
		}
		if operation.sync {
			if err := file.Sync(); err != nil {
				return written, err
			}
		}
		return written, nil
	}

	read, err := file.ReadAt(buffer[:operation.size], operation.offset)
	if err != nil {
		return read, err
	}
	if read != operation.size {
		return read, io.ErrUnexpectedEOF
	}
	return read, nil
}

func blockCount(config Config) int64 {
	return 1 + (config.Size-1)/int64(config.BlockSize)
}

func processBlocks(ctx context.Context, total int64, blockSize int, operation func(int) error) (int64, int64, error) {
	var processed int64
	var operations int64
	for processed < total {
		if err := ctx.Err(); err != nil {
			return processed, operations, err
		}
		currentSize := int64(blockSize)
		if remaining := total - processed; remaining < currentSize {
			currentSize = remaining
		}
		if err := operation(int(currentSize)); err != nil {
			return processed, operations, err
		}
		processed += currentSize
		operations++
	}
	return processed, operations, nil
}
