package benchmark

import "testing"

func TestConfigValidate(t *testing.T) {
	valid := Config{
		Directory:  t.TempDir(),
		Size:       4096,
		BlockSize:  1024,
		Iterations: 1,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config returned error: %v", err)
	}

	tests := []struct {
		name   string
		modify func(*Config)
	}{
		{name: "empty directory", modify: func(config *Config) { config.Directory = "" }},
		{name: "zero size", modify: func(config *Config) { config.Size = 0 }},
		{name: "zero block size", modify: func(config *Config) { config.BlockSize = 0 }},
		{name: "block exceeds size", modify: func(config *Config) { config.BlockSize = 8192 }},
		{name: "zero iterations", modify: func(config *Config) { config.Iterations = 0 }},
		{name: "negative warm-ups", modify: func(config *Config) { config.WarmupIterations = -1 }},
		{name: "negative duration", modify: func(config *Config) { config.Duration = -1 }},
		{name: "negative workers", modify: func(config *Config) { config.Workers = -1 }},
		{name: "negative queue depth", modify: func(config *Config) { config.QueueDepth = -1 }},
		{name: "negative read percentage", modify: func(config *Config) { config.RandomReadPercent = -1 }},
		{name: "read percentage over 100", modify: func(config *Config) { config.RandomReadPercent = 101 }},
		{name: "invalid I/O mode", modify: func(config *Config) { config.IOMode = "invalid" }},
		{name: "invalid cache control", modify: func(config *Config) { config.CacheControl = "invalid" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.modify(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("Validate() returned no error")
			}
		})
	}
}
