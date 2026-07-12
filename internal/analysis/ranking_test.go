package analysis

import (
	"testing"

	"perftool/internal/model"
)

func TestDefaultIndexExcludesDiskFootprint(t *testing.T) {
	benchmarks := []model.Benchmark{
		indexBenchmark("compact-but-slow", 2, 2, 200, 200, 1),
		indexBenchmark("fast-but-fat", 1, 1, 100, 100, 1_000_000),
	}
	// Default index (time, cpu, ram) ranks the fast tool first despite its size.
	if got := AutomaticBaseline(model.Config{Mode: "command"}, benchmarks); got != 2 {
		t.Fatalf("default baseline = %d, want fast tool (2)", got)
	}
	// Including disk lets the huge footprint sink the fast tool.
	withDisk := model.Config{
		Mode: "command", IndexDimensions: []string{"time", "cpu", "ram", "disk"},
	}
	if got := AutomaticBaseline(withDisk, benchmarks); got != 1 {
		t.Fatalf("disk baseline = %d, want compact tool (1)", got)
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
