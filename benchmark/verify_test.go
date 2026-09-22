package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyFileDetectsCorruption(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "verify.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	config := Config{
		Size:        4096,
		BlockSize:   1024,
		VerifyData:  true,
		IOAlignment: 1,
	}
	expected := make([]byte, config.BlockSize)
	fillDeterministic(expected)
	for offset := int64(0); offset < config.Size; offset += int64(config.BlockSize) {
		if _, err := file.WriteAt(expected, offset); err != nil {
			t.Fatal(err)
		}
	}

	result, err := verifyFile(context.Background(), file, expected, config)
	if err != nil {
		t.Fatalf("verifyFile() returned error: %v", err)
	}
	if !result.Passed || result.Bytes != config.Size {
		t.Fatalf("verification result = %+v", result)
	}

	corrupt := expected[0] ^ 0xff
	if _, err := file.WriteAt([]byte{corrupt}, int64(config.BlockSize)); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyFile(context.Background(), file, expected, config); err == nil ||
		!strings.Contains(err.Error(), "offset 1024") {
		t.Fatalf("corruption error = %v, want offset 1024", err)
	}
}

func TestVerifyFileCanBeDisabled(t *testing.T) {
	result, err := verifyFile(context.Background(), nil, nil, Config{})
	if err != nil {
		t.Fatalf("verifyFile() returned error: %v", err)
	}
	if result.Enabled || result.Passed || result.Bytes != 0 {
		t.Fatalf("disabled verification result = %+v", result)
	}
}
