package app

import (
	"fmt"
	"strconv"
	"time"

	"diskbenchmark/benchmark"
)

const LargeWorkingSetThreshold int64 = 10 * 1024 * 1024 * 1024

type WriteImpactAssessment struct {
	RequiresConfirmation bool     `json:"requiresConfirmation"`
	Reasons              []string `json:"reasons"`
	LargeSizeThreshold   string   `json:"largeSizeThresholdBytes"`
}

func assessWriteImpact(
	config benchmark.Config,
	selected []benchmark.Workload,
) WriteImpactAssessment {
	assessment := WriteImpactAssessment{
		Reasons:            []string{},
		LargeSizeThreshold: strconv.FormatInt(LargeWorkingSetThreshold, 10),
	}
	if config.Size >= LargeWorkingSetThreshold {
		assessment.Reasons = append(
			assessment.Reasons,
			fmt.Sprintf("The working set is at least 10 GiB (%d bytes).", config.Size),
		)
	}
	if hasDurationWorkload(selected, config.Duration) {
		assessment.Reasons = append(
			assessment.Reasons,
			"Duration-based workloads can write substantially more than the working-set size.",
		)
	}
	if config.Iterations > 1 {
		assessment.Reasons = append(
			assessment.Reasons,
			fmt.Sprintf("The benchmark will run %d measured iterations.", config.Iterations),
		)
	}
	if hasFsyncWorkload(selected) {
		assessment.Reasons = append(
			assessment.Reasons,
			"Fsync-per-block workloads perform durable writes for every block.",
		)
	}
	assessment.RequiresConfirmation = len(assessment.Reasons) > 0
	return assessment
}

func hasDurationWorkload(selected []benchmark.Workload, defaultDuration time.Duration) bool {
	if defaultDuration > 0 {
		return true
	}
	for _, workload := range selected {
		configured, ok := workload.(benchmark.ConfiguredWorkload)
		if ok && configured.DurationOverride != nil && *configured.DurationOverride > 0 {
			return true
		}
	}
	return false
}

func hasFsyncWorkload(selected []benchmark.Workload) bool {
	for _, workload := range selected {
		switch current := workload.(type) {
		case benchmark.FsyncWrite:
			return true
		case benchmark.ConfiguredWorkload:
			if hasFsyncWorkload([]benchmark.Workload{current.Workload}) {
				return true
			}
		}
	}
	return false
}
