package benchmark

import "math"

func saturatingProduct(left, right uint64) int64 {
	if right != 0 && left > math.MaxInt64/right {
		return math.MaxInt64
	}
	return int64(left * right)
}
