package units

import "testing"

func TestParseBytes(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{input: "4096", want: 4096},
		{input: "4KiB", want: 4 * 1024},
		{input: "1.5 MiB", want: 1572864},
		{input: "2gib", want: 2 * 1024 * 1024 * 1024},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := ParseBytes(test.input)
			if err != nil {
				t.Fatalf("ParseBytes(%q) returned error: %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("ParseBytes(%q) = %d, want %d", test.input, got, test.want)
			}
		})
	}
}

func TestParseBytesRejectsInvalidValues(t *testing.T) {
	inputs := []string{"", "0", "-1MiB", "one MiB", "1MB", "0.1B", "999999999999TiB"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseBytes(input); err == nil {
				t.Fatalf("ParseBytes(%q) returned no error", input)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	if got, want := FormatBytes(1572864), "1.50 MiB"; got != want {
		t.Fatalf("FormatBytes() = %q, want %q", got, want)
	}
}
