package benchmark

import (
	"math"
	"math/bits"
	"time"
)

const (
	latencySubBuckets      = 32
	latencyExponentBuckets = 63
	latencyBucketCount     = 1 + latencySubBuckets*latencyExponentBuckets
)

type latencyHistogram struct {
	buckets [latencyBucketCount]uint64
	count   uint64
	sumNS   float64
	maximum time.Duration
}

func (h *latencyHistogram) record(latency time.Duration) {
	if latency < 0 {
		latency = 0
	}
	h.buckets[latencyBucketIndex(latency)]++
	h.count++
	h.sumNS += float64(latency)
	if latency > h.maximum {
		h.maximum = latency
	}
}

func (h *latencyHistogram) stats() LatencyStats {
	if h.count == 0 {
		return LatencyStats{BucketCount: latencyBucketCount}
	}
	return LatencyStats{
		Average:     time.Duration(h.sumNS / float64(h.count)),
		P50:         h.percentile(0.50),
		P95:         h.percentile(0.95),
		P99:         h.percentile(0.99),
		Maximum:     h.maximum,
		SampleCount: h.count,
		BucketCount: latencyBucketCount,
	}
}

func (h *latencyHistogram) percentile(percentile float64) time.Duration {
	rank := uint64(math.Ceil(float64(h.count) * percentile))
	if rank < 1 {
		rank = 1
	}
	var observed uint64
	for index, count := range h.buckets {
		observed += count
		if observed >= rank {
			upperBound := latencyBucketUpperBound(index)
			if upperBound > h.maximum {
				return h.maximum
			}
			return upperBound
		}
	}
	return h.maximum
}

func latencyBucketIndex(latency time.Duration) int {
	nanoseconds := uint64(latency)
	if nanoseconds == 0 {
		return 0
	}
	exponent := bits.Len64(nanoseconds) - 1
	base := uint64(1) << exponent
	subBucket := int(float64(nanoseconds-base) / float64(base) * latencySubBuckets)
	if subBucket >= latencySubBuckets {
		subBucket = latencySubBuckets - 1
	}
	return 1 + exponent*latencySubBuckets + subBucket
}

func latencyBucketUpperBound(index int) time.Duration {
	if index == 0 {
		return 0
	}
	index--
	exponent := index / latencySubBuckets
	subBucket := index % latencySubBuckets
	base := float64(uint64(1) << exponent)
	upperBound := math.Ceil(base*(1+float64(subBucket+1)/latencySubBuckets)) - 1
	if upperBound > math.MaxInt64 {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(upperBound)
}
