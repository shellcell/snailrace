package report

import (
	"bytes"
	"io"
	"testing"

	"github.com/shellcell/snailrace/internal/model"
)

func TestCachedRendererMatchesIndependentFormats(t *testing.T) {
	report := benchmarkReport(3, 4)
	renderer := NewRenderer(report)
	for _, format := range []string{"html", "svg", "markdown", "json", "text"} {
		var cached, independent bytes.Buffer
		if err := renderer.Write(&cached, format); err != nil {
			t.Fatal(err)
		}
		if err := NewRenderer(report).Write(&independent, format); err != nil {
			t.Fatal(err)
		}
		if cached.String() != independent.String() {
			t.Fatalf("cached %s output differs", format)
		}
	}
}

func TestRendererOutputIsOrderIndependentAndRepeatable(t *testing.T) {
	report := benchmarkReport(3, 4)
	formats := []string{"html", "svg", "markdown", "json", "text"}
	want := make(map[string]string, len(formats))
	for _, format := range formats {
		var output bytes.Buffer
		if err := NewRenderer(report).Write(&output, format); err != nil {
			t.Fatal(err)
		}
		want[format] = output.String()
	}
	for _, order := range [][]string{
		formats,
		{"text", "json", "markdown", "svg", "html", "text"},
	} {
		renderer := NewRenderer(report)
		for _, format := range order {
			var output bytes.Buffer
			if err := renderer.Write(&output, format); err != nil {
				t.Fatal(err)
			}
			if output.String() != want[format] {
				t.Fatalf("%s output depends on rendering order", format)
			}
		}
	}
}

func TestRendererMetadataIsIndependent(t *testing.T) {
	report := benchmarkReport(2, 2)
	originalName := report.Benchmarks[0].Tool.Name
	renderer := NewRenderer(report)
	report.Benchmarks[0].Tool.Name = "changed"
	metadata := renderer.Metadata()
	if metadata.ToolNames[0] != originalName {
		t.Fatalf("renderer metadata changed with source: %+v", metadata)
	}
	metadata.ToolNames[0] = "also changed"
	if renderer.Metadata().ToolNames[0] != originalName {
		t.Fatal("renderer returned mutable metadata")
	}
}

func BenchmarkWriteHTML10Tools100Runs(b *testing.B) {
	report := benchmarkReport(10, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := NewRenderer(report).Write(io.Discard, "html"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReportCharts10Tools100Runs(b *testing.B) {
	report := benchmarkReport(10, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if charts := NewRenderer(report).reportCharts(); len(charts) == 0 {
			b.Fatal("no charts")
		}
	}
}

func BenchmarkWriteAllFormatsCached10Tools100Runs(b *testing.B) {
	report := benchmarkReport(10, 100)
	formats := []string{"html", "svg", "markdown", "json", "text"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		renderer := NewRenderer(report)
		for _, format := range formats {
			if err := renderer.Write(io.Discard, format); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkWriteAllFormatsIndependent10Tools100Runs(b *testing.B) {
	report := benchmarkReport(10, 100)
	formats := []string{"html", "svg", "markdown", "json", "text"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for _, format := range formats {
			if err := NewRenderer(report).Write(io.Discard, format); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func benchmarkReport(toolCount, runCount int) model.Report {
	benchmarks := make([]model.Benchmark, toolCount)
	for tool := range toolCount {
		runs := make([]model.Run, runCount)
		for run := range runCount {
			scale := float64(tool+1) * (1 + float64(run%7)/100)
			runs[run] = model.Run{
				Index: run + 1, StopReason: "exited", WallSeconds: scale / 100,
				CPUUserSeconds: scale / 200, CPUSystemSeconds: scale / 500,
				AverageCPUPercent: 70 + scale, MeanResidentBytes: scale * 1024 * 1024,
				PeakResidentBytes: scale * 2 * 1024 * 1024,
				OSMaxRSSBytes:     scale * 2 * 1024 * 1024,
				PeakVirtualBytes:  scale * 4 * 1024 * 1024,
				PeakProcesses:     2, PeakThreads: 4, PeakFileDescriptors: 8,
				SampleCount: 10, SampleCoverageSeconds: scale / 100,
			}
		}
		benchmarks[tool] = model.Benchmark{
			Tool: model.ToolInfo{
				Name: "tool", Command: []string{"tool"},
				DiskFootprintBytes: int64(tool+1) * 1024 * 1024,
			},
			Runs: runs, Summary: model.Summarize(runs),
		}
	}
	return model.Report{
		Config: model.Config{
			Runs: runCount, Baseline: 1, Mode: "command", IntervalMS: 10,
			IndexDimensions: []string{"time", "cpu", "ram", "disk"},
		},
		Host: model.HostInfo{OS: "linux", LogicalCPUs: 8}, Benchmarks: benchmarks,
	}
}
