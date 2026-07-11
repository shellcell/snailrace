package report

import (
	"fmt"
	"io"

	"perftool/internal/model"
)

func writeTextComparison(writer io.Writer, report model.Report) {
	baselinePosition := baselineIndex(report)
	baseline := report.Benchmarks[baselinePosition]
	for index, candidate := range report.Benchmarks {
		if index == baselinePosition {
			continue
		}
		fmt.Fprintf(
			writer, "Comparison\t%s vs %s [BASELINE]\n",
			candidate.Tool.Name, baseline.Tool.Name,
		)
		fmt.Fprintln(
			writer,
			"Metric\tBaseline mean\tCandidate mean\tΔ mean\tPaired confidence",
		)
		staticDelta, _ := compareStaticCost(
			float64(baseline.Tool.DiskFootprintBytes),
			float64(candidate.Tool.DiskFootprintBytes),
		)
		fmt.Fprintf(
			writer, "Linked footprint\t%s\t%s\t%s\texact metadata\n",
			formatBytes(float64(baseline.Tool.DiskFootprintBytes)),
			formatBytes(float64(candidate.Tool.DiskFootprintBytes)), staticDelta,
		)
		for _, row := range metricRows {
			if !available(row, report.Host.OS) {
				fmt.Fprintf(writer, "%s\tN/A\tN/A\tN/A\tN/A\n", row.name)
				continue
			}
			delta := compareMetric(
				baseline, candidate, row, report.Config.IntervalMS/1000,
			)
			fmt.Fprintf(
				writer, "%s\t%s\t%s\t%s\t%s\n", row.name,
				row.format(row.stats(baseline.Summary).Mean),
				row.format(row.stats(candidate.Summary).Mean),
				formatDelta(delta, row), formatDeltaInterval(delta, row),
			)
		}
		fmt.Fprintln(writer)
	}
}
