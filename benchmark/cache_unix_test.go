//go:build linux || darwin

package benchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnixRequiredCacheControlIsApplied(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "cache.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(make([]byte, 4096)); err != nil {
		t.Fatal(err)
	}

	result, err := applyCacheControl(file, Config{CacheControl: CacheControlRequire})
	if err != nil {
		t.Fatalf("require policy returned error: %v", err)
	}
	if result.Status != cacheStatusApplied || result.API == "" {
		t.Fatalf("unexpected cache-control result: %+v", result)
	}
}
