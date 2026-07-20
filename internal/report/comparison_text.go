package report

import (
	"fmt"
	"io"
)

func writeTextComparison(writer io.Writer, renderer *Renderer) {
	report := renderer.displayReport
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
			writer, "Linked footprint\t%s\t%s\t%s\tstatic discovery\n",
			formatBytes(float64(baseline.Tool.DiskFootprintBytes)),
			formatBytes(float64(candidate.Tool.DiskFootprintBytes)), staticDelta,
		)
		for _, row := range metricCatalog {
			if !availableFor(
				row, report.Host.OS, baseline.Summary, candidate.Summary,
			) {
				fmt.Fprintf(writer, "%s\tN/A\tN/A\tN/A\tN/A\n", row.name)
				continue
			}
			delta := renderer.comparison(index, row.id)
			fmt.Fprintf(
				writer, "%s\t%s\t%s\t%s\t%s\n", row.name,
				row.format(delta.baselineMean), row.format(delta.candidateMean),
				formatDelta(delta), formatDeltaInterval(delta, row),
			)
		}
		fmt.Fprintln(writer)
	}
}
