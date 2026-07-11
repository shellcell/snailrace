package report

import (
	"fmt"
	"os"
	"path/filepath"

	"perftool/internal/model"
)

type ChartArtifact struct {
	Title    string
	Section  string
	Filename string
}

func WriteChartFiles(
	directory string,
	report model.Report,
	includeCommands bool,
) ([]ChartArtifact, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	charts := reportCharts(report)
	if includeCommands {
		charts = append([]svgChart{commandLegendChart(report)}, charts...)
	}
	artifacts := make([]ChartArtifact, 0, len(charts))
	for index, chart := range charts {
		filename := fmt.Sprintf("%02d-%s.svg", index+1, chart.slug)
		path := filepath.Join(directory, filename)
		if err := os.WriteFile(path, []byte(chart.html()), 0o644); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, ChartArtifact{
			Title: chart.title, Section: chart.kind, Filename: filename,
		})
	}
	return artifacts, nil
}
