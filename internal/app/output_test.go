package app

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
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

func TestStdoutFailureDoesNotPreventSaving(t *testing.T) {
	directory := t.TempDir()
	report := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{{Tool: model.ToolInfo{Name: "tool"}}},
	}
	if err := writeResult(
		failingWriter{}, io.Discard, directory, []string{"html"}, report,
	); err == nil {
		t.Fatal("stdout failure should be returned")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("saved entries = %v, error = %v", entries, err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
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

func TestRunIgnoresNonZeroCommandExit(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	if err := Run(
		[]string{"-n", "2", "-warmups", "0", "-c", "exit 7"},
		&stdout, &stderr,
	); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "FAILED") ||
		!strings.Contains(stdout.String(), "excluded from rankings") {
		t.Fatal("stdout does not report the failed command")
	}
}

func TestRunSavesDefaultHTMLInCurrentDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	var stdout, stderr bytes.Buffer
	if err := Run(
		[]string{"-n", "1", "-warmups", "0", "--", "/bin/true"},
		&stdout, &stderr,
	); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".html") {
		t.Fatalf("saved entries = %v, want one HTML report", entries)
	}
	if !strings.Contains(stderr.String(), "Report saved to") {
		t.Fatal("stderr does not announce the default report path")
	}
}

func TestRunNoSaveLeavesCurrentDirectoryEmpty(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	if err := Run(
		[]string{"-no-save", "-n", "1", "-warmups", "0", "--", "/bin/true"},
		io.Discard, io.Discard,
	); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("no-save entries = %v, error = %v", entries, err)
	}
}

func TestSavingSameReportUsesUniqueNames(t *testing.T) {
	directory := t.TempDir()
	result := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{{
			Tool: model.ToolInfo{Name: "same"},
		}},
	}
	const saves = 6
	errors := make(chan error, saves)
	var wait sync.WaitGroup
	for index := 0; index < saves; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			errors <- saveReportFormats(io.Discard, directory, []string{"html"}, result)
		}()
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != saves {
		t.Fatalf("saved entries = %d, want %d unique reports", len(entries), saves)
	}
}

func TestSavingDifferentFormatsDoesNotReuseExistingStem(t *testing.T) {
	directory := t.TempDir()
	result := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{{Tool: model.ToolInfo{Name: "same"}}},
	}
	if err := saveReportFormats(io.Discard, directory, []string{"html"}, result); err != nil {
		t.Fatal(err)
	}
	if err := saveReportFormats(io.Discard, directory, []string{"markdown"}, result); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() == entries[1].Name() {
		t.Fatalf("saved entries reused a report stem: %v", entries)
	}
}

func TestFailedSavePublishesNoPartialReports(t *testing.T) {
	directory := t.TempDir()
	result := model.Report{
		Config:     model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{{Tool: model.ToolInfo{Name: "tool"}}},
	}
	if err := saveReportFormats(
		io.Discard, directory, []string{"html", "unknown"}, result,
	); err == nil {
		t.Fatal("invalid staged format should fail")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed save left partial entries: %v", entries)
	}
}

func TestPublishingNeverReplacesExistingDestination(t *testing.T) {
	directory := t.TempDir()
	staged := directory + "/staged"
	final := directory + "/final"
	if err := os.WriteFile(staged, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(final, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := renameNoReplace(staged, final); err == nil {
		t.Fatal("publication replaced an existing destination")
	}
	data, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Fatalf("destination content = %q, want existing", data)
	}
}

func TestSavedSVGBundleIncludesCommandFailures(t *testing.T) {
	directory := t.TempDir()
	failedRun := model.Run{ExitCode: 7, StopReason: "exited", WallSeconds: 0.001}
	result := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{{
			Tool: model.ToolInfo{Name: "failed"},
			Runs: []model.Run{failedRun}, Summary: model.Summarize([]model.Run{failedRun}),
		}},
	}
	if err := saveReportFormats(io.Discard, directory, []string{"svg"}, result); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("bundle entries = %v, error = %v", entries, err)
	}
	charts, err := os.ReadDir(directory + "/" + entries[0].Name() + "/charts")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, chart := range charts {
		if strings.Contains(chart.Name(), "command-failures") {
			found = true
		}
	}
	if !found {
		t.Fatalf("SVG charts do not include command failures: %v", charts)
	}
}

func TestFormatAliasesAreCanonicalizedBeforeSaving(t *testing.T) {
	formats := uniqueFormats([]string{"text", "txt", "markdown", "md"})
	if len(formats) != 2 || formats[0] != "text" || formats[1] != "markdown" {
		t.Fatalf("canonical formats = %v", formats)
	}
}
