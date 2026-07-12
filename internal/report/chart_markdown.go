package report

import (
	"fmt"
	"io"
	"path/filepath"
)

func writeMarkdownCharts(
	writer io.Writer,
	charts []ChartArtifact,
	directory string,
) {
	sections := []struct{ kind, title string }{
		{"ranking", "Ranking Charts"},
		{"baseline", "From Baseline"},
		{"tradeoff", "Tradeoffs"},
		{"distribution", "Distributions"},
		{"trend", "Measurement Trends"},
	}
	for _, section := range sections {
		if section.kind == "trend" {
			writeMarkdownTrendCharts(writer, charts, directory, section.title)
			continue
		}
		wroteHeading := false
		for _, chart := range charts {
			if chart.Section != section.kind {
				continue
			}
			if !wroteHeading {
				fmt.Fprintf(writer, "## %s\n\n", escapeMarkdown(section.title))
				wroteHeading = true
			}
			path := filepath.ToSlash(filepath.Join(directory, chart.Filename))
			fmt.Fprintf(writer, "![%s](%s)\n\n", escapeMarkdown(chart.Title), path)
		}
	}
}

func writeMarkdownTrendCharts(
	writer io.Writer,
	charts []ChartArtifact,
	directory string,
	title string,
) {
	sections := make(map[string]bool)
	for _, chart := range charts {
		if chart.Section != "trend" {
			continue
		}
		sectionTitle := title
		if chart.Subsection != "" {
			sectionTitle += " · " + chart.Subsection
		}
		if !sections[sectionTitle] {
			fmt.Fprintf(writer, "## %s\n\n", escapeMarkdown(sectionTitle))
			sections[sectionTitle] = true
		}
		path := filepath.ToSlash(filepath.Join(directory, chart.Filename))
		fmt.Fprintf(writer, "![%s](%s)\n\n", escapeMarkdown(chart.Title), path)
	}
}
