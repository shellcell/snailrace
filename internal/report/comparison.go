package report

import (
	"fmt"
	"math"

	"github.com/shellcell/snailrace/internal/model"
)

type metricDirection int

const (
	lowerIsBetter metricDirection = iota
	neutralDirection
)

type deltaResult struct {
	percent          float64
	percentAvailable bool
	baselineMean     float64
	candidateMean    float64
	difference       model.Stats
	status           string
	class            string
}

func baselineIndex(report model.Report) int {
	index := report.Config.Baseline - 1
	if index < 0 || index >= len(report.Benchmarks) {
		return 0
	}
	return index
}

func compareMetric(
	baseline model.Benchmark,
	candidate model.Benchmark,
	row metricRow,
	intervalSeconds float64,
) deltaResult {
	valid := func(model.Run) bool { return true }
	if row.darwinOnly {
		valid = func(run model.Run) bool { return run.PhysicalFootprintValid }
	}
	baseValues, candidateValues, differences := pairedValues(
		baseline.Runs, candidate.Runs, row.run, valid,
	)
	baseMean := model.CalculateStats(baseValues).Mean
	candidateMean := model.CalculateStats(candidateValues).Mean
	result := deltaResult{baselineMean: baseMean, candidateMean: candidateMean}
	if baseMean != 0 {
		result.percent = (candidateMean/baseMean - 1) * 100
		result.percentAvailable = true
	}
	result.difference = model.CalculateStats(differences)
	result.status, result.class = comparisonStatus(result.difference, row.direction)
	if row.sampled && !samplingReliable(baseline, candidate, intervalSeconds) {
		result.status, result.class = "sampling-limited", "uncertain"
	}
	return result
}

func samplingReliable(
	baseline, candidate model.Benchmark,
	intervalSeconds float64,
) bool {
	if intervalSeconds <= 0 {
		return true
	}
	for _, benchmark := range []model.Benchmark{baseline, candidate} {
		if !benchmarkSamplesReliable(benchmark, intervalSeconds) {
			return false
		}
	}
	return true
}

func benchmarkSamplesReliable(benchmark model.Benchmark, intervalSeconds float64) bool {
	if intervalSeconds <= 0 {
		return true
	}
	minimum := intervalSeconds * 2
	for _, run := range benchmark.Runs {
		if run.WallSeconds < minimum || run.SampleCount < 2 ||
			run.SampleCoverageSeconds < intervalSeconds {
			return false
		}
	}
	return true
}

func pairedValues(
	baseline, candidate []model.Run,
	pick func(model.Run) float64,
	valid func(model.Run) bool,
) ([]float64, []float64, []float64) {
	baselineByIndex := make(map[int]float64, len(baseline))
	for _, run := range baseline {
		if valid(run) {
			baselineByIndex[run.Index] = pick(run)
		}
	}
	baseValues := make([]float64, 0, len(candidate))
	candidateValues := make([]float64, 0, len(candidate))
	differences := make([]float64, 0, len(candidate))
	for _, run := range candidate {
		if !valid(run) {
			continue
		}
		value, ok := baselineByIndex[run.Index]
		if ok {
			candidateValue := pick(run)
			baseValues = append(baseValues, value)
			candidateValues = append(candidateValues, candidateValue)
			differences = append(differences, candidateValue-value)
		}
	}
	return baseValues, candidateValues, differences
}

func comparisonStatus(stats model.Stats, direction metricDirection) (string, string) {
	if stats.N < 2 {
		return "inconclusive", "uncertain"
	}
	if math.Abs(stats.Mean) < 1e-15 && stats.StdDev == 0 {
		return "same", "neutral"
	}
	if direction == neutralDirection {
		if math.Abs(stats.Mean) < 1e-15 {
			return "same point estimate", "neutral"
		}
		if stats.Mean < 0 {
			return "lower", "neutral"
		}
		return "higher", "neutral"
	}
	if stats.CI95High < 0 {
		return "better", "good"
	}
	if stats.CI95Low > 0 {
		return "worse", "bad"
	}
	return "inconclusive", "uncertain"
}

func formatDelta(delta deltaResult, row metricRow) string {
	change := "Δ n/a"
	if delta.percentAvailable {
		change = "Δ " + formatSignedPercent(delta.percent)
	}
	return change + " · " + delta.status
}

func formatDeltaInterval(delta deltaResult, row metricRow) string {
	if !delta.difference.CI95Valid {
		return "insufficient paired runs"
	}
	return fmt.Sprintf(
		"95%% CI Δ [%s, %s]", row.format(delta.difference.CI95Low),
		row.format(delta.difference.CI95High),
	)
}

func compareStaticCost(baseline, candidate float64) (string, string) {
	status, class := "same", "neutral"
	if candidate < baseline {
		status, class = "better", "good"
	} else if candidate > baseline {
		status, class = "worse", "bad"
	}
	if baseline == 0 {
		return "Δ n/a · " + status, class
	}
	percent := (candidate/baseline - 1) * 100
	return "Δ " + formatSignedPercent(percent) + " · " + status, class
}
