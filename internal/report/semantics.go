package report

import (
	"fmt"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

func includedDimensions(report model.Report) []string {
	ranking := calculateRanking(report)
	var result []string
	if ranking.indexPrimary {
		result = append(result, "time")
	}
	if ranking.indexCPU {
		result = append(result, "cpu")
	}
	if ranking.indexRAM {
		result = append(result, "ram")
	}
	if ranking.indexFootprint {
		result = append(result, "disk")
	}
	return result
}

func includedDimensionsText(report model.Report) string {
	dimensions := includedDimensions(report)
	if len(dimensions) == 0 {
		return "none"
	}
	return strings.Join(dimensions, ", ")
}

func reliabilityCaveats(report model.Report) []string {
	interval := report.Config.IntervalMS / 1000
	var result []string
	for _, benchmark := range report.Benchmarks {
		if benchmark.Tool.SHA256 != "" && !benchmark.Tool.ProvenanceVerified {
			result = append(result, benchmark.Tool.Name+
				": post-run executable verification was not completed")
		}
		if !benchmarkSamplesReliable(benchmark, interval) {
			result = append(result, fmt.Sprintf(
				"%s: sampled process metrics are limited by sparse observations",
				benchmark.Tool.Name,
			))
		}
		if report.Host.OS != "darwin" {
			continue
		}
		physical := benchmark.Summary.PeakPhysicalFootprintBytes
		switch {
		case physical == nil:
			result = append(result, benchmark.Tool.Name+": physical footprint unavailable")
		case physical.N < len(benchmark.Runs):
			result = append(result, fmt.Sprintf(
				"%s: physical footprint available for %d of %d runs",
				benchmark.Tool.Name, physical.N, len(benchmark.Runs),
			))
		}
	}
	return result
}

func physicalFootprintAvailable(report model.Report) bool {
	if report.Host.OS != "darwin" || len(report.Benchmarks) == 0 {
		return false
	}
	for _, benchmark := range report.Benchmarks {
		if benchmark.Summary.PeakPhysicalFootprintBytes == nil {
			return false
		}
	}
	return true
}
