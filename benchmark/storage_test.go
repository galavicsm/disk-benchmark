package benchmark

import (
	"math"
	"testing"
)

func TestPreflightSpaceRejectsOversizedFile(t *testing.T) {
	available, err := preflightSpace(Config{
		Directory: t.TempDir(),
		Size:      math.MaxInt64,
	})
	if err == nil {
		t.Fatalf("preflightSpace() returned no error with %d bytes available", available)
	}
}

func TestClassifyFileStorage(t *testing.T) {
	sparse := classifyFileStorage(4096, 1024)
	if !sparse.Sparse {
		t.Fatalf("expected sparse classification: %+v", sparse)
	}
	allocated := classifyFileStorage(4096, 4096)
	if allocated.Sparse {
		t.Fatalf("expected fully allocated classification: %+v", allocated)
	}
}

func TestSaturatingProduct(t *testing.T) {
	if got := saturatingProduct(math.MaxUint64, 2); got != math.MaxInt64 {
		t.Fatalf("saturatingProduct() = %d, want %d", got, int64(math.MaxInt64))
	}
	if got := saturatingProduct(100, 512); got != 51200 {
		t.Fatalf("saturatingProduct() = %d, want 51200", got)
	}
}
