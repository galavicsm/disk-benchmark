//go:build windows

package benchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsCacheControlReportsUnsupported(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "cache.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	result, err := applyCacheControl(file, Config{CacheControl: CacheControlAttempt})
	if err != nil {
		t.Fatalf("attempt policy returned error: %v", err)
	}
	if result.Status != cacheStatusUnsupported {
		t.Fatalf("attempt status = %q, want %q", result.Status, cacheStatusUnsupported)
	}

	if _, err := applyCacheControl(file, Config{CacheControl: CacheControlRequire}); err == nil {
		t.Fatal("require policy returned no error")
	}
}
