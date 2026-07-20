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
	row metricDefinition,
	intervalSeconds float64,
) deltaResult {
	return compareMetricWithBaselineIndex(
		baseline, indexMetricValues(baseline.Runs, row), candidate, row, intervalSeconds,
	)
}

func compareMetricWithBaselineIndex(
	baseline model.Benchmark,
	baselineValues map[int]float64,
	candidate model.Benchmark,
	row metricDefinition,
	intervalSeconds float64,
) deltaResult {
	baseMean, candidateMean, differences := pairedDifferences(
		baselineValues, candidate.Runs, row,
	)
	result := deltaResult{baselineMean: baseMean, candidateMean: candidateMean}
	if baseMean != 0 {
		percent := (candidateMean/baseMean - 1) * 100
		if finiteNumber(percent) {
			result.percent = percent
			result.percentAvailable = true
		}
	}
	result.difference = model.CalculateStats(differences)
	result.status, result.class = comparisonStatus(result.difference, row.direction)
	if row.sampled && !samplingReliable(baseline, candidate, intervalSeconds) {
		result.status, result.class = "sampling-limited", "uncertain"
	}
	return result
}

func indexMetricValues(runs []model.Run, row metricDefinition) map[int]float64 {
	values := make(map[int]float64, len(runs))
	for _, run := range runs {
		if row.id == metricPhysicalFootprint && !run.PhysicalFootprintValid {
			continue
		}
		value := row.run(run)
		if finiteNumber(value) {
			values[run.Index] = value
		}
	}
	return values
}

func samplingReliable(
	baseline, candidate model.Benchmark,
	intervalSeconds float64,
) bool {
	if intervalSeconds <= 0 {
		return true
	}
	for _, benchmark := range []model.Benchmark{baseline, candidate} {
		if !model.SamplingReliable(benchmark, intervalSeconds) {
			return false
		}
	}
	return true
}

func pairedDifferences(
	baseline map[int]float64,
	candidate []model.Run,
	row metricDefinition,
) (float64, float64, []float64) {
	differences := make([]float64, 0, len(candidate))
	baseMean, candidateMean := 0.0, 0.0
	count := 0
	for _, run := range candidate {
		if row.id == metricPhysicalFootprint && !run.PhysicalFootprintValid {
			continue
		}
		value, ok := baseline[run.Index]
		if ok {
			candidateValue := row.run(run)
			if !finiteNumber(candidateValue) {
				continue
			}
			count++
			baseMean = runningMean(baseMean, value, count)
			candidateMean = runningMean(candidateMean, candidateValue, count)
			differences = append(differences, candidateValue-value)
		}
	}
	return baseMean, candidateMean, differences
}

func runningMean(mean, value float64, count int) float64 {
	n := float64(count)
	mean = mean*(1-1/n) + value/n
	if math.IsInf(mean, 0) {
		return math.Copysign(math.MaxFloat64, mean)
	}
	return mean
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

func formatDelta(delta deltaResult) string {
	change := "Δ n/a"
	if delta.percentAvailable {
		change = "Δ " + formatSignedPercent(delta.percent)
	}
	return change + " · " + delta.status
}

func formatDeltaInterval(delta deltaResult, row metricDefinition) string {
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
