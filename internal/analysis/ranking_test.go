package analysis

import (
	"math"
	"testing"

	"github.com/shellcell/snailrace/internal/model"
)

func TestDefaultIndexExcludesDiskFootprint(t *testing.T) {
	benchmarks := []model.Benchmark{
		indexBenchmark("compact-but-slow", 2, 2, 200, 200, 1),
		indexBenchmark("fast-but-fat", 1, 1, 100, 100, 1_000_000),
	}
	// Default index (time, cpu, ram) ranks the fast tool first despite its size.
	if got, ok := AutomaticBaseline(model.Config{Mode: "command"}, benchmarks); !ok || got != 2 {
		t.Fatalf("default baseline = %d, want fast tool (2)", got)
	}
	// Including disk lets the huge footprint sink the fast tool.
	withDisk := model.Config{
		Mode: "command", IndexDimensions: []string{"time", "cpu", "ram", "disk"},
	}
	if got, ok := AutomaticBaseline(withDisk, benchmarks); !ok || got != 1 {
		t.Fatalf("disk baseline = %d, want compact tool (1)", got)
	}
}

func TestUnavailableRankingUsesExplicitState(t *testing.T) {
	benchmarks := []model.Benchmark{indexBenchmark("short", 1, 1, 100, 100, 10)}
	config := model.Config{
		Mode: "command", IntervalMS: 10, IndexDimensions: []string{"ram"},
	}
	ranking := Calculate(config, benchmarks)
	if ranking.Available || ranking.UnavailableReason == "" {
		t.Fatalf("ranking = %+v, want explicit unavailable state", ranking)
	}
	if len(ranking.Rows) != 1 || ranking.Rows[0].OverallRank != 0 ||
		math.IsInf(ranking.Rows[0].OverallScore, 0) || math.IsNaN(ranking.Rows[0].OverallScore) {
		t.Fatalf("unavailable row contains invalid ranking values: %+v", ranking.Rows)
	}
	if _, ok := AutomaticBaseline(config, benchmarks); ok {
		t.Fatal("unavailable ranking should not select an automatic baseline")
	}
}

func TestFailedBenchmarkIsExcludedFromRanking(t *testing.T) {
	failed := indexBenchmark("failed", 0.001, 0.001, 100, 100, 10)
	failed.Runs[0].ExitCode = 7
	failed.Summary = model.Summarize(failed.Runs)
	success := indexBenchmark("success", 1, 1, 200, 200, 20)
	config := model.Config{
		Mode: "command", IndexDimensions: []string{"time", "cpu"},
	}
	ranking := Calculate(config, []model.Benchmark{failed, success})
	if !ranking.Available || len(ranking.Rows) != 1 || ranking.Rows[0].Benchmark != 1 {
		t.Fatalf("failed tool was not excluded: %+v", ranking)
	}
	if baseline, ok := AutomaticBaseline(config, []model.Benchmark{failed, success}); !ok || baseline != 2 {
		t.Fatalf("baseline = %d/%v, want successful tool 2", baseline, ok)
	}
}

func indexBenchmark(
	name string,
	wall, cpu, meanRAM, peakRAM float64,
	footprint int64,
) model.Benchmark {
	runs := []model.Run{{
		Index: 1, WallSeconds: wall, CPUUserSeconds: cpu,
		MeanResidentBytes: meanRAM, PeakResidentBytes: peakRAM,
	}}
	return model.Benchmark{
		Tool: model.ToolInfo{Name: name, DiskFootprintBytes: footprint},
		Runs: runs, Summary: model.Summarize(runs),
	}
}
