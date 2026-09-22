package app

import (
	"strconv"
	"strings"

	"diskbenchmark/suite"
)

type SuiteSummary struct {
	Version   int                    `json:"version"`
	Workloads []SuiteWorkloadSummary `json:"workloads"`
}

type SuiteWorkloadSummary struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Duration          string `json:"duration,omitempty"`
	RandomReadPercent *int   `json:"randomReadPercent,omitempty"`
	RandomSeed        string `json:"randomSeed,omitempty"`
}

func newSuiteSummary(file suite.File) SuiteSummary {
	summary := SuiteSummary{
		Version:   file.Version,
		Workloads: make([]SuiteWorkloadSummary, len(file.Workloads)),
	}
	for index, workload := range file.Workloads {
		name := workload.Name
		if name == "" {
			name = defaultSuiteWorkloadName(workload.Type)
		}
		current := SuiteWorkloadSummary{
			Name:              name,
			Type:              workload.Type,
			Duration:          workload.Duration,
			RandomReadPercent: workload.RandomReadPercent,
		}
		if workload.RandomSeed != nil {
			current.RandomSeed = strconv.FormatInt(*workload.RandomSeed, 10)
		}
		summary.Workloads[index] = current
	}
	return summary
}

func defaultSuiteWorkloadName(workloadType string) string {
	words := strings.ReplaceAll(workloadType, "-", " ")
	if words == "" {
		return "Unnamed workload"
	}
	return strings.ToUpper(words[:1]) + words[1:]
}
