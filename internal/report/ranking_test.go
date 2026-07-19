package report

import (
	"testing"

	"github.com/shellcell/snailrace/internal/analysis"
	"github.com/shellcell/snailrace/internal/model"
)

func TestAutomaticBaselineUsesBalancedWinner(t *testing.T) {
	benchmarks := []model.Benchmark{
		rankingBenchmark("fast", 1, 3, 300, 300, 300),
		rankingBenchmark("middle", 2, 2, 200, 200, 200),
		rankingBenchmark("efficient", 3, 1, 100, 100, 100),
	}
	config := model.Config{Mode: "command"}
	if got := analysis.AutomaticBaseline(config, benchmarks); got != 3 {
		t.Fatalf("baseline = %d, want efficient tool at index 3", got)
	}
	ranking := calculateRanking(model.Report{Config: config, Benchmarks: benchmarks})
	if ranking.winners[1].benchmarks[0] != 0 || ranking.winners[2].benchmarks[0] != 2 {
		t.Fatalf("unexpected category winners: %+v", ranking.winners)
	}
	if ranking.rows[0].overallScore/ranking.bestOverall != 1 {
		t.Fatal("best overall row should have zero difference from best")
	}
}

func TestCompositeRAMWinnerIsNormalizedToOne(t *testing.T) {
	benchmarks := []model.Benchmark{
		rankingBenchmark("low mean", 1, 1, 100, 400, 1),
		rankingBenchmark("balanced", 1, 1, 150, 150, 1),
		rankingBenchmark("low peak", 1, 1, 400, 100, 1),
	}
	ranking := calculateRanking(model.Report{
		Config: model.Config{Mode: "command"}, Benchmarks: benchmarks,
	})
	for _, row := range ranking.rows {
		if row.ramRank == 1 && row.ramScore != 1 {
			t.Fatalf("RAM winner score = %g, want 1", row.ramScore)
		}
	}
}

func TestRankingReportsAllExactTies(t *testing.T) {
	benchmarks := []model.Benchmark{
		rankingBenchmark("first", 1, 1, 100, 100, 100),
		rankingBenchmark("second", 1, 1, 100, 100, 100),
	}
	ranking := calculateRanking(model.Report{
		Config: model.Config{Mode: "command"}, Benchmarks: benchmarks,
	})
	if len(ranking.winners[0].benchmarks) != 2 {
		t.Fatalf("overall tied leaders = %v, want both tools", ranking.winners[0].benchmarks)
	}
}

func TestZeroBestCategoryHasNoRatioDelta(t *testing.T) {
	benchmarks := []model.Benchmark{
		rankingBenchmark("zero CPU", 1, 0, 100, 100, 100),
		rankingBenchmark("positive CPU", 1, 1, 100, 100, 100),
	}
	ranking := calculateRanking(model.Report{
		Config: model.Config{Mode: "command"}, Benchmarks: benchmarks,
	})
	if ranking.cpuRatio {
		t.Fatal("CPU ratio should be unavailable when the best value is zero")
	}
	if got := rankingCell("0 ns", ranking.rows[0].cpuScore, 1, false); got != "0 ns · Δ n/a (#1)" {
		t.Fatalf("zero-best ranking cell = %q", got)
	}
}

func rankingBenchmark(
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
