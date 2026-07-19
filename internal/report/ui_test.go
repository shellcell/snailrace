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
	if err := writeJSON(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"\"ranking\"", "\"winners\"", "balanced_index"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("JSON does not contain %s", expected)
		}
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
	if err := writeHTML(&output, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), command) ||
		!strings.Contains(output.String(), `class="command"`) {
		t.Fatal("HTML does not preserve the command in a wrapping block")
	}
}
