package report

import (
	"testing"

	"perftool/internal/analysis"
	"perftool/internal/model"
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
	if ranking.winners[1].tool != "fast" || ranking.winners[2].tool != "efficient" {
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
