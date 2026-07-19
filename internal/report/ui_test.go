package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shellcell/snailrace/internal/model"
)

func TestHumanFormatsAvoidScientificNotationForLargeUnits(t *testing.T) {
	if got := formatBytes(1023 * 1024); strings.Contains(got, "e+") {
		t.Fatalf("bytes formatted as scientific notation: %q", got)
	}
	if got := formatSignedPercent(2070); strings.Contains(got, "e+") {
		t.Fatalf("percentage formatted as scientific notation: %q", got)
	}
	for value, want := range map[float64]string{920: "920", 1000: "1000", 20: "20"} {
		if got := formatNumber(value); got != want {
			t.Fatalf("formatNumber(%g) = %q, want %q", value, got, want)
		}
	}
}

func TestJSONIncludesComputedRanking(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("baseline", 1, 1),
			benchmarkWithWallTimes("candidate", 2, 2),
		},
	}
	var output bytes.Buffer
	if err := writeJSON(&output, NewRenderer(report)); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"\"ranking\"", "\"winners\"", "balanced_index"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("JSON does not contain %s", expected)
		}
	}
}

func TestUnavailableRankingSerializesWithoutNonFiniteValues(t *testing.T) {
	report := model.Report{
		Config: model.Config{
			Mode: "command", Baseline: 1, IntervalMS: 10,
			IndexDimensions: []string{"ram"},
		},
		Benchmarks: []model.Benchmark{benchmarkWithWallTimes("short", 0.001)},
	}
	var output bytes.Buffer
	if err := writeJSON(&output, NewRenderer(report)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"available": false`) ||
		strings.Contains(output.String(), "+Inf") || strings.Contains(output.String(), "NaN") {
		t.Fatalf("unexpected unavailable ranking JSON: %s", output.String())
	}
}

func TestFailedCommandsAreProminentInEveryFormat(t *testing.T) {
	failed := benchmarkWithWallTimes("failed", 0.001)
	failed.Runs[0].ExitCode = 7
	failed.Summary = model.Summarize(failed.Runs)
	report := model.Report{
		Config: model.Config{
			Mode: "command", Baseline: 2, IndexDimensions: []string{"time"},
		},
		Benchmarks: []model.Benchmark{failed, benchmarkWithWallTimes("success", 1)},
	}
	for _, format := range []string{"text", "html", "markdown", "svg", "json"} {
		t.Run(format, func(t *testing.T) {
			var output bytes.Buffer
			if err := NewRenderer(report).Write(&output, format); err != nil {
				t.Fatal(err)
			}
			value := strings.ToLower(output.String())
			if !strings.Contains(value, "failed") || !strings.Contains(value, "7") {
				t.Fatalf("%s does not prominently report failure: %s", format, output.String())
			}
		})
	}
}

func TestHTMLPreservesLongCommandInWrappingBlock(t *testing.T) {
	command := "tool --value=" + strings.Repeat("x", 300)
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{{
			Tool: model.ToolInfo{Name: "tool", Command: []string{command}},
		}},
	}
	var output bytes.Buffer
	if err := writeHTML(&output, NewRenderer(report)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), command) ||
		!strings.Contains(output.String(), `class="command"`) {
		t.Fatal("HTML does not preserve the command in a wrapping block")
	}
}
