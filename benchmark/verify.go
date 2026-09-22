package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

func verifyFile(
	ctx context.Context,
	file *os.File,
	expected []byte,
	config Config,
) (VerificationResult, error) {
	result := VerificationResult{Enabled: config.VerifyData}
	if !config.VerifyData {
		return result, nil
	}

	buffer := makeIOBuffer(config.BlockSize, config.IOAlignment)
	start := time.Now()
	for offset := int64(0); offset < config.Size; {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		size := config.BlockSize
		if remaining := config.Size - offset; remaining < int64(size) {
			size = int(remaining)
		}
		read, err := file.ReadAt(buffer[:size], offset)
		if err != nil {
			return result, fmt.Errorf("read verification data at offset %d: %w", offset, err)
		}
		if read != size {
			return result, fmt.Errorf("read verification data at offset %d: %w", offset, io.ErrUnexpectedEOF)
		}
		if mismatch := firstMismatch(buffer[:size], expected[:size]); mismatch >= 0 {
			return result, fmt.Errorf(
				"data verification failed at offset %d: got 0x%02x, want 0x%02x",
				offset+int64(mismatch),
				buffer[mismatch],
				expected[mismatch],
			)
		}
		offset += int64(size)
		result.Bytes += int64(size)
	}
	result.Duration = time.Since(start)
	result.Passed = true
	return result, nil
}

func firstMismatch(actual, expected []byte) int {
	if bytes.Equal(actual, expected) {
		return -1
	}
	limit := min(len(actual), len(expected))
	for index := 0; index < limit; index++ {
		if actual[index] != expected[index] {
			return index
		}
	}
	return limit
}
