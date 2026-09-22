package units

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

var multipliers = map[string]float64{
	"":    1,
	"B":   1,
	"KIB": 1024,
	"MIB": 1024 * 1024,
	"GIB": 1024 * 1024 * 1024,
	"TIB": 1024 * 1024 * 1024 * 1024,
}

func ParseBytes(value string) (int64, error) {
	input := strings.TrimSpace(value)
	if input == "" {
		return 0, fmt.Errorf("size must not be empty")
	}

	numberEnd := 0
	for numberEnd < len(input) {
		character := input[numberEnd]
		if (character >= '0' && character <= '9') || character == '.' {
			numberEnd++
			continue
		}
		break
	}
	if numberEnd == 0 {
		return 0, fmt.Errorf("size %q has no numeric value", value)
	}

	number, err := strconv.ParseFloat(input[:numberEnd], 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
		return 0, fmt.Errorf("size %q must contain a positive number", value)
	}
	unit := strings.ToUpper(strings.TrimSpace(input[numberEnd:]))
	multiplier, ok := multipliers[unit]
	if !ok {
		return 0, fmt.Errorf("size %q uses unsupported unit %q", value, input[numberEnd:])
	}

	bytes := number * multiplier
	if bytes > math.MaxInt64 {
		return 0, fmt.Errorf("size %q is too large", value)
	}
	rounded := math.Round(bytes)
	if math.Abs(bytes-rounded) > 1e-9 {
		return 0, fmt.Errorf("size %q does not resolve to a whole number of bytes", value)
	}
	return int64(rounded), nil
}

func FormatBytes(bytes int64) string {
	type unit struct {
		suffix string
		size   int64
	}
	units := []unit{
		{suffix: "TiB", size: 1024 * 1024 * 1024 * 1024},
		{suffix: "GiB", size: 1024 * 1024 * 1024},
		{suffix: "MiB", size: 1024 * 1024},
		{suffix: "KiB", size: 1024},
	}
	for _, candidate := range units {
		if bytes >= candidate.size {
			return fmt.Sprintf("%.2f %s", float64(bytes)/float64(candidate.size), candidate.suffix)
		}
	}
	return fmt.Sprintf("%d B", bytes)
}
