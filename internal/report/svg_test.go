package report

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/shellcell/snailrace/internal/model"
)

func TestSVGIsStandaloneCompleteReport(t *testing.T) {
	report := model.Report{
		MeasuredAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC),
		Config:     model.Config{Mode: "command", Runs: 3, Baseline: 1},
		Host:       model.HostInfo{OS: "linux", Architecture: "amd64", CPU: "Test CPU"},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("baseline", 10, 10, 10),
			benchmarkWithWallTimes("candidate", 8, 8, 8),
		},
		Notes: []string{"No outliers removed."},
	}
	var output bytes.Buffer
	if err := writeSVG(&output, report); err != nil {
		t.Fatal(err)
	}
	decoder := xml.NewDecoder(bytes.NewReader(output.Bytes()))
	for {
		if _, err := decoder.Token(); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("invalid SVG: %v", err)
		}
	}
	for _, expected := range []string{"SNAILRACE CHARTS", "Wall time"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("SVG does not contain %q", expected)
		}
	}
}

func TestMarkdownReferencesSeparateSVGCharts(t *testing.T) {
	report := model.Report{
		Config: model.Config{Mode: "command", Baseline: 1},
		Benchmarks: []model.Benchmark{
			benchmarkWithWallTimes("baseline", 10, 10),
			benchmarkWithWallTimes("candidate", 9, 9),
		},
	}
	charts, err := WriteChartFiles(t.TempDir(), report, false)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := WriteMarkdownWithCharts(&output, report, charts, "charts"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "<svg") || !strings.Contains(output.String(), ".svg)") {
		t.Fatal("Markdown should reference, not embed, SVG charts")
	}
}
