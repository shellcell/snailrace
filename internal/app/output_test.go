package app

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/shellcell/snailrace/internal/model"
)

func TestSavingAlsoWritesCompleteReportToStdout(t *testing.T) {
	directory := t.TempDir()
	var stdout, stderr bytes.Buffer
	runs := []model.Run{{Index: 1, WallSeconds: 1}, {Index: 2, WallSeconds: 2}}
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{{
			Tool: model.ToolInfo{Name: "test-tool"}, Runs: runs, Summary: model.Summarize(runs),
		}},
	}
	if err := writeResult(
		&stdout, &stderr, directory, []string{"text"}, report,
	); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "TIME") {
		t.Fatal("stdout does not contain the report")
	}
	if !strings.Contains(stderr.String(), "Report saved to") {
		t.Fatal("stderr does not contain the saved path")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("saved files = %d, want 1", len(entries))
	}
}

func TestSavedFormatDoesNotReplaceTextStdout(t *testing.T) {
	directory := t.TempDir()
	var stdout, stderr bytes.Buffer
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{{
			Tool: model.ToolInfo{Name: "test-tool"},
		}},
	}
	if err := writeResult(
		&stdout, &stderr, directory, []string{"html"}, report,
	); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "<!doctype html>") {
		t.Fatal("HTML was written to stdout")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("saved entries = %v, error = %v", entries, err)
	}
	data, err := os.ReadFile(directory + "/" + entries[0].Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<!doctype html>") {
		t.Fatal("saved file is not HTML")
	}
}

func TestMarkdownAndSVGSavedAsChartBundle(t *testing.T) {
	directory := t.TempDir()
	var stderr bytes.Buffer
	runs := []model.Run{{Index: 1, WallSeconds: 1}, {Index: 2, WallSeconds: 2}}
	result := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			{Tool: model.ToolInfo{Name: "first"}, Runs: runs, Summary: model.Summarize(runs)},
			{Tool: model.ToolInfo{Name: "second"}, Runs: runs, Summary: model.Summarize(runs)},
		},
	}
	if err := saveReportFormats(
		&stderr, directory, []string{"markdown", "svg"}, result,
	); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("bundle entries = %v, error = %v", entries, err)
	}
	bundle := directory + "/" + entries[0].Name()
	markdown, err := os.ReadFile(bundle + "/report.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(markdown), "<svg") ||
		!strings.Contains(string(markdown), "](charts/") {
		t.Fatal("Markdown should reference separate chart SVGs")
	}
	charts, err := os.ReadDir(bundle + "/charts")
	if err != nil || len(charts) == 0 {
		t.Fatalf("chart files = %v, error = %v", charts, err)
	}
}
