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
	ranking := Calculate(model.Config{Mode: "command"}, benchmarks)
	if !ranking.Available || ranking.Rows[0].Benchmark+1 != 2 {
		t.Fatalf("default baseline = %d, want fast tool (2)", ranking.Rows[0].Benchmark+1)
	}
	// Including disk lets the huge footprint sink the fast tool.
	withDisk := model.Config{
		Mode: "command", IndexDimensions: []string{"time", "cpu", "ram", "disk"},
	}
	ranking = Calculate(withDisk, benchmarks)
	if !ranking.Available || ranking.Rows[0].Benchmark+1 != 1 {
		t.Fatalf("disk baseline = %d, want compact tool (1)", ranking.Rows[0].Benchmark+1)
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
	if ranking.Rows[0].Benchmark+1 != 2 {
		t.Fatalf("baseline = %d, want successful tool 2", ranking.Rows[0].Benchmark+1)
	}
}

func TestDimensionDefaultsCannotBeMutatedByCallers(t *testing.T) {
	dimensions := DefaultIndexDimensions()
	dimensions[0] = "disk"
	if got := DefaultIndexDimensions()[0]; got != "time" {
		t.Fatalf("mutated default dimension = %q", got)
	}
	all := IndexDimensions()
	all[0] = "disk"
	if got := IndexDimensions()[0]; got != "time" {
		t.Fatalf("mutated known dimension = %q", got)
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
