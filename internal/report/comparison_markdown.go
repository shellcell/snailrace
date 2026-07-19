package report

import (
	"fmt"
	"io"

	"github.com/shellcell/snailrace/internal/model"
)

func writeMarkdownComparison(writer io.Writer, report model.Report) {
	baselinePosition := baselineIndex(report)
	baseline := report.Benchmarks[baselinePosition]
	fmt.Fprintf(
		writer, "## Comparison\n\nBaseline: **%s**\n\n",
		escapeMarkdown(baseline.Tool.Name),
	)
	for index, candidate := range report.Benchmarks {
		if index == baselinePosition {
			continue
		}
		fmt.Fprintf(
			writer, "### %s\n\n", escapeMarkdown(candidate.Tool.Name),
		)
		fmt.Fprintln(
			writer,
			"| Metric | Baseline mean | Candidate mean | Δ mean | Paired confidence |",
		)
		fmt.Fprintln(writer, "|---|---:|---:|---:|---:|")
		staticDelta, _ := compareStaticCost(
			float64(baseline.Tool.DiskFootprintBytes),
			float64(candidate.Tool.DiskFootprintBytes),
		)
		fmt.Fprintf(
			writer, "| Linked footprint | %s | %s | %s | static discovery |\n",
			formatBytes(float64(baseline.Tool.DiskFootprintBytes)),
			formatBytes(float64(candidate.Tool.DiskFootprintBytes)), staticDelta,
		)
		for _, row := range metricRows {
			if !available(row, report.Host.OS) {
				fmt.Fprintf(writer, "| %s | N/A | N/A | N/A | N/A |\n", row.name)
				continue
			}
			delta := compareMetric(
				baseline, candidate, row, report.Config.IntervalMS/1000,
			)
			fmt.Fprintf(
				writer, "| %s | %s | %s | %s | %s |\n", row.name,
				row.format(row.stats(baseline.Summary).Mean),
				row.format(row.stats(candidate.Summary).Mean),
				formatDelta(delta, row), formatDeltaInterval(delta, row),
			)
		}
		fmt.Fprintln(writer)
	}
}
