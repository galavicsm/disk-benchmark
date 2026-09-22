package benchmark

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type IOMode string

const (
	BufferedIO IOMode = "buffered"
	DirectIO   IOMode = "direct"
)

func ParseIOMode(value string) (IOMode, error) {
	mode := IOMode(strings.ToLower(value))
	switch mode {
	case BufferedIO, DirectIO:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported I/O mode %q; use buffered or direct", value)
	}
}

type CacheControlMode string

const (
	CacheControlOff     CacheControlMode = "off"
	CacheControlAttempt CacheControlMode = "attempt"
	CacheControlRequire CacheControlMode = "require"
)

func ParseCacheControlMode(value string) (CacheControlMode, error) {
	mode := CacheControlMode(strings.ToLower(value))
	switch mode {
	case CacheControlOff, CacheControlAttempt, CacheControlRequire:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported cache control %q; use off, attempt, or require", value)
	}
}

type Config struct {
	Directory         string           `json:"directory"`
	Size              int64            `json:"size_bytes"`
	BlockSize         int              `json:"block_size_bytes"`
	Iterations        int              `json:"iterations"`
	KeepFile          bool             `json:"keep_file"`
	Sync              bool             `json:"sync_writes"`
	Workers           int              `json:"workers"`
	QueueDepth        int              `json:"queue_depth"`
	RandomReadPercent int              `json:"random_read_percent"`
	RandomSeed        int64            `json:"random_seed"`
	IOMode            IOMode           `json:"io_mode"`
	IOAlignment       int              `json:"io_alignment_bytes"`
	CacheControl      CacheControlMode `json:"cache_control"`
	WarmupIterations  int              `json:"warmup_iterations"`
	VerifyData        bool             `json:"verify_data"`
	Duration          time.Duration    `json:"duration_ns"`
	SuiteFile         string           `json:"suite_file,omitempty"`
	SuiteVersion      int              `json:"suite_version,omitempty"`
}

func (c Config) Validate() error {
	if c.Directory == "" {
		return fmt.Errorf("benchmark directory must not be empty")
	}
	if c.Size <= 0 {
		return fmt.Errorf("data size must be greater than zero")
	}
	if c.BlockSize <= 0 {
		return fmt.Errorf("block size must be greater than zero")
	}
	if int64(c.BlockSize) > c.Size {
		return fmt.Errorf("block size must not exceed data size")
	}
	if c.Iterations <= 0 {
		return fmt.Errorf("iterations must be greater than zero")
	}
	if c.WarmupIterations < 0 {
		return fmt.Errorf("warm-up iterations must not be negative")
	}
	if c.Duration < 0 {
		return fmt.Errorf("workload duration must not be negative")
	}
	if c.Workers < 0 {
		return fmt.Errorf("workers must be greater than zero")
	}
	if c.QueueDepth < 0 {
		return fmt.Errorf("queue depth must be greater than zero")
	}
	if c.RandomReadPercent < 0 || c.RandomReadPercent > 100 {
		return fmt.Errorf("random read percentage must be between 0 and 100")
	}
	if c.IOMode != "" && c.IOMode != BufferedIO && c.IOMode != DirectIO {
		return fmt.Errorf("I/O mode must be buffered or direct")
	}
	if c.CacheControl != "" &&
		c.CacheControl != CacheControlOff &&
		c.CacheControl != CacheControlAttempt &&
		c.CacheControl != CacheControlRequire {
		return fmt.Errorf("cache control must be off, attempt, or require")
	}

	info, err := os.Stat(c.Directory)
	if err != nil {
		return fmt.Errorf("inspect benchmark directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("benchmark path %q is not a directory", c.Directory)
	}
	return nil
}

func (c Config) normalized() Config {
	if c.Workers == 0 {
		c.Workers = 1
	}
	if c.QueueDepth == 0 {
		c.QueueDepth = 1
	}
	if c.IOMode == "" {
		c.IOMode = BufferedIO
	}
	if c.CacheControl == "" {
		c.CacheControl = CacheControlOff
	}
	return c
}

func ValidateConfig(config Config) (Config, error) {
	config = config.normalized()
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return configureIOMode(config)
}
