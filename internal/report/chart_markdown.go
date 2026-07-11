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
		wroteHeading := false
		for _, chart := range charts {
			if chart.Section != section.kind {
				continue
			}
			if !wroteHeading {
				fmt.Fprintf(writer, "## %s\n\n", section.title)
				wroteHeading = true
			}
			path := filepath.ToSlash(filepath.Join(directory, chart.Filename))
			fmt.Fprintf(writer, "![%s](%s)\n\n", escapeMarkdown(chart.Title), path)
		}
	}
}
