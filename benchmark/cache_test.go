package benchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCacheControlMode(t *testing.T) {
	for _, value := range []string{"off", "OFF", "attempt", "ATTEMPT", "require", "REQUIRE"} {
		if _, err := ParseCacheControlMode(value); err != nil {
			t.Fatalf("ParseCacheControlMode(%q) returned error: %v", value, err)
		}
	}
	if _, err := ParseCacheControlMode("invalid"); err == nil {
		t.Fatal("ParseCacheControlMode(invalid) returned no error")
	}
}

func TestApplyCacheControlOff(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "cache.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	result, err := applyCacheControl(file, Config{CacheControl: CacheControlOff})
	if err != nil {
		t.Fatalf("applyCacheControl() returned error: %v", err)
	}
	if result.Status != cacheStatusDisabled {
		t.Fatalf("status = %q, want %q", result.Status, cacheStatusDisabled)
	}
}

func TestRequiredCacheControlIsSatisfiedByDirectIO(t *testing.T) {
	result, err := applyCacheControl(nil, Config{
		IOMode:       DirectIO,
		CacheControl: CacheControlRequire,
	})
	if err != nil {
		t.Fatalf("applyCacheControl() returned error: %v", err)
	}
	if result.Status != cacheStatusBypassed || result.API != "direct_io" {
		t.Fatalf("unexpected direct I/O result: %+v", result)
	}
}
