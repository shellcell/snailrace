package report

import (
	"math"
	"strings"
	"testing"

	"github.com/shellcell/snailrace/internal/model"
)

func TestEmptyReportBuildsNoCharts(t *testing.T) {
	if charts := NewRenderer(model.Report{}).reportCharts(); len(charts) != 0 {
		t.Fatalf("empty report charts = %d, want 0", len(charts))
	}
}

func TestChartsOmitNonFiniteCoordinates(t *testing.T) {
	runs := []model.Run{
		{Index: 1, WallSeconds: math.Inf(1)},
		{Index: 2, WallSeconds: math.NaN()},
	}
	report := model.Report{
		Config: model.Config{Baseline: 1, Mode: "command"},
		Host:   model.HostInfo{OS: "linux"},
		Benchmarks: []model.Benchmark{{
			Tool: model.ToolInfo{Name: "invalid", DiskFootprintBytes: 1},
			Runs: runs, Summary: model.Summarize(runs),
		}},
	}
	for _, chart := range NewRenderer(report).reportCharts() {
		output := chart.html()
		if strings.Contains(output, "NaN") || strings.Contains(output, "+Inf") ||
			strings.Contains(output, "-Inf") {
			t.Fatalf("chart %q contains non-finite coordinates", chart.title)
		}
	}
}

func TestComparisonChartsContainDeltaAndRunViews(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("baseline", 10, 10, 10),
			benchmarkWithWallTimes("candidate", 8, 8, 8),
		},
	}
	charts := NewRenderer(report).reportCharts()
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

func TestBaselineChartsSeparateEveryMetric(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("baseline", 1, 1),
			benchmarkWithWallTimes("candidate", 2, 2),
		},
	}
	var titles []string
	for _, chart := range NewRenderer(report).reportCharts() {
		if chart.kind == "baseline" {
			titles = append(titles, chart.title)
		}
	}
	if len(titles) != len(metricCatalog) {
		t.Fatalf("baseline charts = %d, want %d: %v", len(titles), len(metricCatalog), titles)
	}
	for _, expected := range []string{
		"Mean RSS", "Peak RSS", "OS-reported max RSS", "Peak virtual memory",
	} {
		found := false
		for _, title := range titles {
			found = found || title == expected
		}
		if !found {
			t.Fatalf("missing separate baseline chart %q: %v", expected, titles)
		}
	}
}

func TestBaselineChartShowsBaselineConfidenceWhisker(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("baseline", 9, 10, 11),
			benchmarkWithWallTimes("candidate", 8, 9, 10),
		},
	}
	chart := deltaForestChart(NewRenderer(report), chartGroup{
		name: "Wall time", metrics: chartMetrics(metricWall),
	})
	if !strings.Contains(chart.body, `class="baseline-ci"`) {
		t.Fatalf("baseline confidence whisker missing: %s", chart.body)
	}
	if !strings.Contains(chart.description, "baseline whisker") {
		t.Fatalf("baseline whisker explanation missing: %s", chart.description)
	}
}

func TestToolColorsAreDistinctAndStable(t *testing.T) {
	seen := make(map[string]bool)
	for index := 0; index < 10; index++ {
		color := toolColor(index)
		if seen[color] {
			t.Fatalf("tool %d repeats color %s", index, color)
		}
		seen[color] = true
	}
	if toolColor(0) != "#88c0d0" {
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
	ranking := rankingCharts(report, calculateRanking(report))[0].body
	for index := range report.Benchmarks {
		color := toolColor(index)
		style := `style="fill:` + color + `"`
		if !strings.Contains(legend, style) || !strings.Contains(ranking, style) {
			t.Fatalf("tool %d color %s missing from chart legend or ranking", index, color)
		}
	}
}

func TestRankingChartsPutBestAtTopAndWorstAtBottom(t *testing.T) {
	report := model.Report{
		Config: model.Config{
			Mode: "command", Baseline: 3,
			IndexDimensions: []string{"time", "cpu"},
		},
		Benchmarks: []model.Benchmark{
			benchmarkWithResourceCosts("fast", 1, 9, 300, 300, 300),
			benchmarkWithResourceCosts("middle", 2, 2, 200, 200, 200),
			benchmarkWithResourceCosts("efficient", 3, 1, 100, 100, 100),
		},
	}
	expected := map[string][]string{
		"BALANCED INDEX": {"#1 efficient", "#2 middle", "#3 fast"},
		"TIME":           {"#1 fast", "#2 middle", "#3 efficient"},
		"CPU COST":       {"#1 efficient", "#2 middle", "#3 fast"},
		"RAM AGGREGATE":  {"#1 efficient", "#2 middle", "#3 fast"},
		"LINKED SIZE":    {"#1 efficient", "#2 middle", "#3 fast"},
	}
	for _, chart := range rankingCharts(report, calculateRanking(report)) {
		order, ok := expected[chart.title]
		if !ok {
			t.Fatalf("missing expected order for %s chart", chart.title)
		}
		best := strings.Index(chart.body, order[0])
		middle := strings.Index(chart.body, order[1])
		worst := strings.Index(chart.body, order[2])
		if best < 0 || middle <= best || worst <= middle {
			t.Fatalf("%s chart does not put best at top and worst at bottom", chart.title)
		}
	}
}

func TestSingleRunDistributionOmitsUndefinedConfidenceWhisker(t *testing.T) {
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{benchmarkWithWallTimes("tool", 1)},
	}
	chart := absoluteDistributionChart(report, metricCatalog[metricWall])
	if strings.Contains(chart.body, `stroke="#eceff4"`) {
		t.Fatal("one observation should not render a confidence whisker")
	}
}

func TestDistributionSummaryPaintsAboveRawRuns(t *testing.T) {
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{benchmarkWithWallTimes("tool", 1, 2)},
	}
	body := absoluteDistributionChart(report, metricCatalog[metricWall]).body
	dot := strings.Index(body, `class="run-dot"`)
	whisker := strings.Index(body, `class="mean-ci"`)
	diamond := strings.Index(body, `class="mean-diamond"`)
	if dot < 0 || whisker <= dot || diamond <= whisker {
		t.Fatalf("SVG z-order should be run dots, then whisker, then diamond: %d/%d/%d", dot, whisker, diamond)
	}
}

func TestFixedTUIRankingChartUsesCPUUnits(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "tui", DurationSeconds: 1, Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("first", 1), benchmarkWithWallTimes("second", 1),
		},
	}
	charts := rankingCharts(report, calculateRanking(report))
	found := false
	for _, chart := range charts {
		if chart.title == "CPU" && strings.Contains(chart.body, "%") {
			found = true
		}
	}
	if !found {
		t.Fatal("fixed TUI primary ranking chart should show average CPU percentage")
	}
}

func TestSingleToolReportOmitsRankingCharts(t *testing.T) {
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{benchmarkWithTrendRuns("tool")},
	}
	for _, chart := range NewRenderer(report).reportCharts() {
		if chart.kind == "ranking" {
			t.Fatalf("single-tool report should not include ranking chart %q", chart.title)
		}
	}
}

func TestBalancedIndexDescriptionListsIncludedCategories(t *testing.T) {
	benchmarks := []model.Benchmark{
		benchmarkWithResourceCosts("first", 1, 0.5, 100, 200, 1000),
		benchmarkWithResourceCosts("second", 2, 0.7, 120, 260, 1200),
	}
	// Default index omits disk footprint, so linked size is not listed.
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: benchmarks,
	}
	description := rankingCharts(report, calculateRanking(report))[0].description
	for _, expected := range []string{
		"Included here: wall time, CPU cost, RAM aggregate.",
		"score = value / best value",
	} {
		if !strings.Contains(description, expected) {
			t.Fatalf("balanced-index description missing %q: %s", expected, description)
		}
	}
	// Selecting disk adds linked size back to the composite description.
	withDisk := model.Report{
		Config: model.Config{
			Mode: "command", Baseline: 1,
			IndexDimensions: []string{"time", "cpu", "ram", "disk"},
		},
		Benchmarks: benchmarks,
	}
	description = rankingCharts(withDisk, calculateRanking(withDisk))[0].description
	if !strings.Contains(description, "wall time, CPU cost, RAM aggregate, linked size") {
		t.Fatalf("disk index missing linked size: %s", description)
	}
}

func TestSingleToolReportHasMetricGroupTrendCharts(t *testing.T) {
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{benchmarkWithTrendRuns("tool")},
	}
	var titles []string
	var combined strings.Builder
	for _, chart := range NewRenderer(report).reportCharts() {
		if chart.kind != "trend" {
			continue
		}
		titles = append(titles, chart.title)
		combined.WriteString(chart.body)
		combined.WriteString(chart.description)
	}
	for _, expected := range []string{
		"Performance trends", "CPU cost trends", "Memory cost trends",
		"Process structure trends",
		`class="trend-line"`, `class="grid"`, `text-anchor="end"`,
		"CPU time (Total CPU, User CPU, System CPU); Average CPU (Average CPU)",
		"Resident memory (OS-reported max RSS, Peak RSS, Mean RSS)",
		"Processes and threads (Peak threads, Peak processes)",
	} {
		if !strings.Contains(strings.Join(titles, " ")+combined.String(), expected) {
			t.Fatalf("single-tool trend charts missing %q: titles=%v", expected, titles)
		}
	}
}

func TestTrendChartsAreSectionedByTool(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithTrendRuns("first"),
			benchmarkWithTrendRuns("second"),
		},
	}
	var titles []string
	for _, section := range chartSections(NewRenderer(report).reportCharts()) {
		if strings.HasPrefix(section.title, "Measurement trends") {
			titles = append(titles, section.title)
		}
	}
	for _, expected := range []string{
		"Measurement trends · first [BASELINE]",
		"Measurement trends · second",
	} {
		if !containsString(titles, expected) {
			t.Fatalf("trend section missing %q: %v", expected, titles)
		}
	}
}

func TestTrendLegendValuesDoNotOverlapLabels(t *testing.T) {
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{benchmarkWithTrendRuns("tool")},
	}
	chart := measurementTrendChart(report, 0, chartGroup{
		name: "MEMORY COST", metrics: chartMetrics(
			metricMeanResident, metricPeakResident, metricOSMaxRSS,
			metricPhysicalFootprint,
		),
	})
	if strings.Contains(chart.body, `x="704"`) {
		t.Fatalf("trend legend values should not use overlapping right column: %s", chart.body)
	}
	for _, expected := range []string{
		`OS-reported max RSS`, `x="92"`, `x="520"`,
		`x="92" y="88" class="label">Resident memory`,
		`x="92" y="158" class="value">run 1`,
		`x="500" y="158" text-anchor="end" class="value">run 3`,
		`100 B ... 130 B`,
	} {
		if !strings.Contains(chart.body, expected) {
			t.Fatalf("trend legend missing stacked value layout marker %q: %s", expected, chart.body)
		}
	}
}

func TestComparisonReportHasPerToolTrendCharts(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithTrendRuns("first"),
			benchmarkWithTrendRuns("second"),
		},
	}
	titles := trendChartTitles(NewRenderer(report).reportCharts())
	for _, expected := range []string{
		"CPU cost trends · first [BASELINE]",
		"Memory cost trends · first [BASELINE]",
		"CPU cost trends · second",
		"Memory cost trends · second",
	} {
		if !containsString(titles, expected) {
			t.Fatalf("comparison trend charts missing %q: %v", expected, titles)
		}
	}
}

func TestChartsHaveExplicitDescriptions(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("baseline", 10, 10),
			benchmarkWithWallTimes("candidate", 8, 8),
		},
	}
	single := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{benchmarkWithWallTimes("tool", 1, 2)},
	}
	chartSets := [][]svgChart{
		append([]svgChart{commandLegendChart(report)}, NewRenderer(report).reportCharts()...),
		NewRenderer(single).reportCharts(),
	}
	for _, charts := range chartSets {
		for _, chart := range charts {
			if chart.height > 0 && chart.description == "" {
				t.Fatalf("chart %q is missing a description", chart.title)
			}
		}
	}
}

func trendChartTitles(charts []svgChart) []string {
	titles := make([]string, 0)
	for _, chart := range charts {
		if chart.kind == "trend" {
			titles = append(titles, chart.title)
		}
	}
	return titles
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func benchmarkWithTrendRuns(name string) model.Benchmark {
	runs := []model.Run{
		trendRun(1, 1.00, 0.20, 0.10, 30, 100, 200, 210, 1000, 1, 2, 3),
		trendRun(2, 1.10, 0.24, 0.11, 32, 130, 230, 240, 1100, 1, 3, 4),
		trendRun(3, 0.95, 0.19, 0.09, 29, 120, 220, 230, 1050, 1, 2, 3),
	}
	return model.Benchmark{
		Tool:    model.ToolInfo{Name: name, DiskFootprintBytes: 1000},
		Runs:    runs,
		Summary: model.Summarize(runs),
	}
}

func trendRun(
	index int,
	wall, userCPU, systemCPU, averageCPU, meanRSS, peakRSS float64,
	osMaxRSS, virtualMemory, processes, threads, fds float64,
) model.Run {
	return model.Run{
		Index: index, WallSeconds: wall, CPUUserSeconds: userCPU,
		CPUSystemSeconds: systemCPU, AverageCPUPercent: averageCPU,
		MeanResidentBytes: meanRSS, PeakResidentBytes: peakRSS,
		OSMaxRSSBytes: osMaxRSS, PeakVirtualBytes: virtualMemory,
		PeakProcesses: processes, PeakThreads: threads, PeakFileDescriptors: fds,
	}
}

func benchmarkWithResourceCosts(
	name string,
	wallSeconds, cpuSeconds, meanResident, peakResident, footprint float64,
) model.Benchmark {
	runs := []model.Run{{
		Index: 1, WallSeconds: wallSeconds,
		CPUUserSeconds: cpuSeconds / 2, CPUSystemSeconds: cpuSeconds / 2,
		MeanResidentBytes: meanResident, PeakResidentBytes: peakResident,
	}}
	return model.Benchmark{
		Tool:    model.ToolInfo{Name: name, DiskFootprintBytes: int64(footprint)},
		Runs:    runs,
		Summary: model.Summarize(runs),
	}
}

func TestChartHTMLIncludesMetadataAndTooltip(t *testing.T) {
	chart := svgChart{
		kind: "distribution", title: "Speed & cost", slug: "speed-cost",
		description: `Explains "quoted" chart text.`,
		body:        svgChartStyle, height: 60,
	}
	svg := chart.html()
	if !strings.Contains(svg, `<title>Speed &amp; cost</title>`) {
		t.Fatalf("SVG title metadata missing: %s", svg)
	}
	if !strings.Contains(svg, `<desc>Explains &#34;quoted&#34; chart text.</desc>`) {
		t.Fatalf("SVG description metadata missing: %s", svg)
	}
	var output strings.Builder
	writeHTMLChartColumn(&output, []svgChart{chart})
	html := output.String()
	if !strings.Contains(html, `class="chart-frame"`) ||
		!strings.Contains(html, `class="chart-help"`) ||
		!strings.Contains(html, `role="tooltip"`) {
		t.Fatalf("HTML tooltip markup missing: %s", html)
	}
	if !strings.Contains(html, `Explains &#34;quoted&#34; chart text.`) {
		t.Fatalf("HTML tooltip text missing or unescaped: %s", html)
	}
}
