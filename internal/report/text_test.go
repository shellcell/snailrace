package report

import (
	"bytes"
	"strings"
	"testing"

	"perftool/internal/model"
)

func TestFixedTUIComparisonEmphasizesResources(t *testing.T) {
	input := model.Report{
		Config:  model.Config{Mode: "tui", DurationSeconds: 5},
		Verbose: true,
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

func TestCompactReportShowsBarsAndOmitsDetail(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			rankingBenchmark("first", 1, 1, 100, 200, 1000),
			rankingBenchmark("second", 2, 2, 120, 260, 1200),
		},
		Notes: []string{"a methodology note"},
	}
	var output bytes.Buffer
	if err := writeText(&output, report); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"BALANCED", "TIME", "█"} {
		if !strings.Contains(text, want) {
			t.Fatalf("compact report missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"Statistical detail", "95% CI mean", "Note"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("compact report should omit %q:\n%s", unwanted, text)
		}
	}
}

func TestVerboseReportKeepsStatisticalDetail(t *testing.T) {
	report := model.Report{
		Config:  model.Config{Mode: "command", Baseline: 1},
		Verbose: true,
		Benchmarks: []model.Benchmark{
			rankingBenchmark("first", 1, 1, 100, 200, 1000),
			rankingBenchmark("second", 2, 2, 120, 260, 1200),
		},
		Notes: []string{"a methodology note"},
	}
	var output bytes.Buffer
	if err := writeText(&output, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Statistical detail") {
		t.Fatal("verbose report should keep the statistical detail")
	}
}

func TestCompactSingleToolListsKeyMetrics(t *testing.T) {
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{rankingBenchmark("solo", 1, 1, 100, 200, 1000)},
	}
	var output bytes.Buffer
	if err := writeText(&output, report); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "TIME") || strings.Contains(text, "█") {
		t.Fatalf("single-tool compact report should list metrics without bars:\n%s", text)
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
