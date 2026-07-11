package report

import (
	"strings"
	"testing"

	"perftool/internal/model"
)

func TestComparisonChartsContainDeltaAndRunViews(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("baseline", 10, 10, 10),
			benchmarkWithWallTimes("candidate", 8, 8, 8),
		},
	}
	charts := reportCharts(report)
	if len(charts) < 2 {
		t.Fatalf("charts = %d, want delta and distribution charts", len(charts))
	}
	var combined strings.Builder
	for _, chart := range charts {
		combined.WriteString(chart.body)
	}
	for _, expected := range []string{
		"Δ FROM BASELINE", "RUN DISTRIBUTION", "TRADEOFF · Time × Total CPU",
	} {
		if !strings.Contains(combined.String(), expected) {
			t.Fatalf("charts do not contain %q", expected)
		}
	}
	if !strings.Contains(combined.String(), "10 s · baseline") {
		t.Fatal("delta chart does not show the absolute baseline value")
	}
}

func TestToolColorsAreDistinctAndStable(t *testing.T) {
	seen := make(map[string]bool)
	for index := 0; index < 10; index++ {
		color := chartColor(index, 0)
		if seen[color] {
			t.Fatalf("tool %d repeats color %s", index, color)
		}
		seen[color] = true
		if chartColor(index, 3) != color || chartColor(index, 7) != color {
			t.Fatalf("tool %d color changes with baseline", index)
		}
	}
	if chartColor(0, 0) != "#88c0d0" {
		t.Fatal("first tool should begin with the Frost color")
	}
}

func TestChartLegendsUseToolIdentityColors(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 2},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("first", 1, 1),
			benchmarkWithWallTimes("baseline", 2, 2),
		},
	}
	legend := commandLegendChart(report).body
	ranking := rankingCharts(report)[0].body
	for index := range report.Benchmarks {
		color := toolColor(report, index)
		style := `style="fill:` + color + `"`
		if !strings.Contains(legend, style) || !strings.Contains(ranking, style) {
			t.Fatalf("tool %d color %s missing from chart legend or ranking", index, color)
		}
	}
}

func TestSingleRunDistributionOmitsUndefinedConfidenceWhisker(t *testing.T) {
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{benchmarkWithWallTimes("tool", 1)},
	}
	chart := absoluteDistributionChart(report, chartMetricDefinitions()[0])
	if strings.Contains(chart.body, `stroke="#eceff4"`) {
		t.Fatal("one observation should not render a confidence whisker")
	}
}

func TestFixedTUIRankingChartUsesCPUUnits(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "tui", DurationSeconds: 1, Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("first", 1), benchmarkWithWallTimes("second", 1),
		},
	}
	charts := rankingCharts(report)
	if len(charts) < 2 || charts[1].title != "CPU" ||
		!strings.Contains(charts[1].body, "%") {
		t.Fatal("fixed TUI primary ranking chart should show average CPU percentage")
	}
}
