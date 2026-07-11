package report

import (
	"bytes"
	"strings"
	"testing"

	"perftool/internal/model"
)

func TestFixedTUIComparisonEmphasizesResources(t *testing.T) {
	input := model.Report{
		Config: model.Config{Mode: "tui", DurationSeconds: 5},
		Benchmarks: []model.Benchmark{
			{Tool: model.ToolInfo{Name: "htop"}},
			{Tool: model.ToolInfo{Name: "btop"}},
		},
	}
	var output bytes.Buffer
	if err := writeText(&output, input); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Average CPU") {
		t.Fatal("TUI comparison does not contain average CPU")
	}
	if strings.Contains(output.String(), "Relative") {
		t.Fatal("TUI comparison should not rank fixed wall time")
	}
}

func TestColumnGapsFindsRunsOfAtLeastTwoSpaces(t *testing.T) {
	gaps := columnGaps("a  bb   c d")
	want := [][2]int{{1, 3}, {5, 8}}
	if len(gaps) != len(want) {
		t.Fatalf("gaps = %v, want %v", gaps, want)
	}
	for index := range want {
		if gaps[index] != want[index] {
			t.Fatalf("gap %d = %v, want %v", index, gaps[index], want[index])
		}
	}
}

func TestSingleObservationReportsUndefinedConfidenceInterval(t *testing.T) {
	benchmark := benchmarkWithWallTimes("tool", 1)
	var output bytes.Buffer
	writeTextBenchmark(&output, "linux", 0, benchmark, false)
	if !strings.Contains(output.String(), "N/A (n < 2)") {
		t.Fatal("single observation should report an undefined confidence interval")
	}
}

func TestStatisticalDetailWarnsAboutLimitedSampling(t *testing.T) {
	benchmark := benchmarkWithWallTimes("tool", 1)
	var output bytes.Buffer
	writeTextBenchmark(&output, "linux", 0.1, benchmark, false)
	if !strings.Contains(output.String(), "Sampling quality") {
		t.Fatal("sampling-limited detail should carry a visible warning")
	}
}
