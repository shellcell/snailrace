package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shellcell/snailrace/internal/model"
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

func samplingLimitedRAMReport() model.Report {
	// Positive RAM values but IntervalMS>0 with runs lacking >=2 samples make RAM
	// present yet sampling-limited: shown with a remark, excluded from the index.
	return model.Report{
		Config: model.Config{Mode: "command", Baseline: 1, IntervalMS: 10},
		Benchmarks: []model.Benchmark{
			rankingBenchmark("first", 1, 1, 100, 200, 1000),
			rankingBenchmark("second", 2, 2, 120, 260, 1200),
		},
	}
}

func TestSamplingLimitedRAMIsShownButExcludedFromIndex(t *testing.T) {
	report := samplingLimitedRAMReport()
	ranking := calculateRanking(report)
	if ranking.ramAvailable || !ranking.ramPresent {
		t.Fatalf("want RAM present but sampling-limited, got present=%v available=%v",
			ranking.ramPresent, ranking.ramAvailable)
	}
	if strings.Contains(balancedIndexCategories(ranking), "RAM") {
		t.Fatal("sampling-limited RAM must stay out of the balanced index")
	}
	// Compact stdout shows the RAM bars with a remark.
	var compact bytes.Buffer
	if err := writeText(&compact, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compact.String(), "RAM") ||
		!strings.Contains(compact.String(), "sampling-limited") {
		t.Fatalf("compact report should show RAM with a remark:\n%s", compact.String())
	}
	// HTML ranking table and the RAM chart show it with a remark, not N/A.
	var htmlRanking bytes.Buffer
	writeHTMLRanking(&htmlRanking, report)
	if !strings.Contains(htmlRanking.String(), "sampling-limited") {
		t.Fatal("HTML ranking should mark RAM sampling-limited instead of N/A")
	}
	foundRAMChart := false
	for _, chart := range rankingCharts(report) {
		if chart.title == "RAM AGGREGATE" {
			foundRAMChart = true
			if !strings.Contains(chart.description, "sampling-limited") {
				t.Fatal("RAM chart description should note the sampling limitation")
			}
		}
	}
	if !foundRAMChart {
		t.Fatal("RAM aggregate chart should still render when sampling-limited")
	}
}

func TestAllFormatsExposeActualDimensionsAndReliability(t *testing.T) {
	report := samplingLimitedRAMReport()
	for _, format := range []string{"text", "markdown", "html", "svg"} {
		var output bytes.Buffer
		if err := Write(&output, format, report); err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		text := strings.ToLower(output.String())
		if !strings.Contains(text, "time, cpu") {
			t.Fatalf("%s omits actual included dimensions", format)
		}
		if !strings.Contains(text, "reliability") {
			t.Fatalf("%s omits reliability caveat", format)
		}
		if !strings.Contains(text, "first:") || !strings.Contains(text, "second:") {
			t.Fatalf("%s omits a per-tool reliability caveat", format)
		}
	}
	var output bytes.Buffer
	if err := Write(&output, "json", report); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Ranking struct {
			Included []string `json:"included_dimensions"`
		} `json:"ranking"`
		Reliability []string `json:"reliability_caveats"`
	}
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if strings.Join(decoded.Ranking.Included, ",") != "time,cpu" ||
		len(decoded.Reliability) == 0 {
		t.Fatalf("JSON semantics = %+v", decoded)
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
