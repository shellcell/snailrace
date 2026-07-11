package report

import (
	"math"
	"testing"

	"perftool/internal/model"
)

func TestPairedComparisonRequiresConfidence(t *testing.T) {
	row := metricRows[0]
	baseline := benchmarkWithWallTimes("baseline", 10, 10, 10)
	better := benchmarkWithWallTimes("better", 8, 8, 8)
	delta := compareMetric(baseline, better, row, 0)
	if delta.status != "better" || math.Abs(delta.percent+20) > 1e-9 {
		t.Fatalf("delta = %+v, want -20%% better", delta)
	}

	noisy := benchmarkWithWallTimes("noisy", 8, 12, 10)
	delta = compareMetric(baseline, noisy, row, 0)
	if delta.status != "inconclusive" {
		t.Fatalf("status = %q, want inconclusive", delta.status)
	}
}

func TestSinglePairIsInconclusive(t *testing.T) {
	delta := compareMetric(
		benchmarkWithWallTimes("baseline", 10),
		benchmarkWithWallTimes("candidate", 5),
		metricRows[0], 0,
	)
	if delta.status != "inconclusive" {
		t.Fatalf("status = %q, want inconclusive", delta.status)
	}
}

func TestZeroBaselineDeltaIsUnavailable(t *testing.T) {
	delta := compareMetric(
		benchmarkWithWallTimes("baseline", 0, 0),
		benchmarkWithWallTimes("candidate", 1, 1),
		metricRows[0], 0,
	)
	if got := formatDelta(delta, metricRows[0]); got != "Δ n/a · worse" {
		t.Fatalf("delta = %q, want unavailable percentage", got)
	}
	if got, _ := compareStaticCost(0, 100); got != "Δ n/a · worse" {
		t.Fatalf("static delta = %q, want unavailable percentage", got)
	}
}

func TestShortRunsMarkSampledMetricsAsLimited(t *testing.T) {
	baseline := benchmarkWithWallTimes("baseline", 0.005, 0.005, 0.005)
	candidate := benchmarkWithWallTimes("candidate", 0.006, 0.006, 0.006)
	for index := range baseline.Runs {
		baseline.Runs[index].MeanResidentBytes = 100
		candidate.Runs[index].MeanResidentBytes = 200
	}
	baseline.Summary = model.Summarize(baseline.Runs)
	candidate.Summary = model.Summarize(candidate.Runs)
	delta := compareMetric(baseline, candidate, metricRows[5], 0.01)
	if delta.status != "sampling-limited" {
		t.Fatalf("status = %q, want sampling-limited", delta.status)
	}
}

func benchmarkWithWallTimes(name string, values ...float64) model.Benchmark {
	runs := make([]model.Run, len(values))
	for index, value := range values {
		runs[index] = model.Run{Index: index + 1, WallSeconds: value}
	}
	return model.Benchmark{
		Tool: model.ToolInfo{Name: name}, Runs: runs,
		Summary: model.Summarize(runs),
	}
}
