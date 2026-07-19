package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/shellcell/snailrace/internal/model"
)

func TestReportPathContainsToolAndMeasurementTime(t *testing.T) {
	report := model.Report{
		MeasuredAt: time.Date(2026, 7, 11, 14, 5, 9, 0, time.UTC),
		Benchmarks: []model.Benchmark{{
			Tool: model.ToolInfo{Name: "My Tool!"},
		}},
	}
	got := reportPath("reports", "markdown", report)
	want := filepath.Join("reports", "snail-my-tool-20260711-140509.md")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestComparisonReportPathNamesMeasuredTools(t *testing.T) {
	report := model.Report{
		MeasuredAt: time.Date(2026, 7, 11, 14, 5, 9, 0, time.UTC),
		Benchmarks: []model.Benchmark{
			{Tool: model.ToolInfo{Name: "grep"}},
			{Tool: model.ToolInfo{Name: "ripgrep"}},
			{Tool: model.ToolInfo{Name: "awk"}},
		},
	}
	got := filepath.Base(reportPath(".", "json", report))
	want := "snail-grep-vs-ripgrep-and-1-20260711-140509.json"
	if got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
}
